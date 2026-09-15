package antispam

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/rivo/uniseg"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	randomTextMessage = "Comment looks like automated text spam. Please rewrite it in normal words."
	repetitiveMessage = "Comment is too repetitive."
	symbolsMessage    = "Please reduce repeated symbols or emoji."
)

var (
	urlMarkerPattern = regexp.MustCompile(`(?i)https?://|www\.`)
	digitPattern     = regexp.MustCompile(`[0-9]+`)
	lowerCaser       = cases.Lower(language.Und)
)

type ValidationError struct {
	Message string
	Mask    bool
}

func (e *ValidationError) Error() string { return e.Message }

func (s *Service) ValidateContent(raw string) (string, error) {
	content := trimJSWhitespace(raw)
	if content == "" {
		return "", validationError("Comment cannot be empty.")
	}
	if UTF16Length(content) > s.cfg.MaxLength {
		return "", validationError("Comment is too long. Maximum is " + strconv.Itoa(s.cfg.MaxLength) + " characters.")
	}
	if countURLs(content) > s.cfg.MaxURLCount {
		return "", validationError("Comment has too many links. Please reduce links in your message.")
	}
	whitespaceTokens := splitJSWhitespace(content)
	for _, token := range whitespaceTokens {
		if UTF16Length(token) > s.cfg.MaxTokenLength {
			return "", validationError("Comment has an excessively long token. Please shorten it.")
		}
	}
	if UTF16Length(content) >= s.cfg.RandomTextMinLength && len(whitespaceTokens) >= s.cfg.RandomTokenMinCount {
		random := 0
		for _, token := range whitespaceTokens {
			if s.likelyRandomToken(token) {
				random++
			}
		}
		if random >= s.cfg.RandomTokenMinCount && float64(random)/float64(len(whitespaceTokens)) >= s.cfg.RandomTokenMinShare {
			return "", spamValidationError(randomTextMessage)
		}
	}

	graphemes := nonWhitespaceGraphemes(content)
	repeated := 1
	for index := 1; index < len(graphemes); index++ {
		if graphemes[index] != graphemes[index-1] {
			repeated = 1
			continue
		}
		repeated++
		if isSymbol(graphemes[index]) && repeated > s.cfg.MaxRepeatedSymbolRun {
			return "", spamValidationError(symbolsMessage)
		}
		if repeated > s.cfg.MaxRepeatedCharRun {
			return "", spamValidationError(repetitiveMessage)
		}
	}

	tokens := dominanceTokens(content)
	repeated = 1
	for index := 1; index < len(tokens); index++ {
		if tokens[index] != tokens[index-1] {
			repeated = 1
			continue
		}
		repeated++
		if repeated > s.cfg.MaxRepeatedTokenRun {
			if isSymbol(tokens[index]) {
				return "", spamValidationError(symbolsMessage)
			}
			return "", spamValidationError(repetitiveMessage)
		}
	}
	wordTokens := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if isWordLike(token) {
			wordTokens = append(wordTokens, token)
		}
	}
	analyzed := tokens
	if len(wordTokens) >= 20 {
		analyzed = wordTokens
	}
	if len(analyzed) >= 20 {
		counts := map[string]int{}
		dominant, dominantCount := "", 0
		for _, token := range analyzed {
			counts[token]++
			if counts[token] > dominantCount {
				dominant, dominantCount = token, counts[token]
			}
		}
		if float64(dominantCount)/float64(len(analyzed)) >= 0.72 {
			if isSymbol(dominant) {
				return "", spamValidationError(symbolsMessage)
			}
			return "", spamValidationError(repetitiveMessage)
		}
	}
	if UTF16Length(content) >= s.cfg.LowTokenDiversityContentMinLength && len(wordTokens) >= s.cfg.LowTokenDiversityMinTokenCount {
		if float64(uniqueCount(wordTokens))/float64(len(wordTokens)) < s.cfg.LowTokenDiversityThreshold {
			return "", spamValidationError(repetitiveMessage)
		}
	}
	if len(graphemes) >= 120 && float64(uniqueCount(graphemes))/float64(len(graphemes)) < 0.08 {
		return "", spamValidationError(repetitiveMessage)
	}
	if len(graphemes) >= 60 {
		symbols, words := 0, 0
		for _, grapheme := range graphemes {
			if isWordLike(grapheme) {
				words++
			} else if isSymbol(grapheme) {
				symbols++
			}
		}
		if float64(symbols)/float64(len(graphemes)) >= 0.85 && words < 8 {
			return "", spamValidationError(symbolsMessage)
		}
	}
	return content, nil
}

func TextHash(content string) string { return hash(normalizeText(content)) }

func Fingerprint(content string) string {
	value := normalizeText(content)
	value = replaceURLs(value)
	value = digitPattern.ReplaceAllString(value, "0")
	value = replaceNoise(value)
	value = collapseWhitespace(value)
	return hash(truncateUTF16(value, 300))
}

func normalizeText(value string) string {
	return lowerCaser.String(strings.Join(splitJSWhitespace(trimJSWhitespace(value)), " "))
}

func countURLs(value string) int { return len(urlMarkerPattern.FindAllString(value, -1)) }

func (s *Service) likelyRandomToken(value string) bool {
	if UTF16Length(value) < s.cfg.RandomTokenMinLength {
		return false
	}
	letters, digits, symbols, vowels := 0, 0, 0, 0
	unique := map[rune]struct{}{}
	hasLower, hasUpper, connector := false, false, false
	for _, r := range value {
		lower := unicode.ToLower(r)
		unique[lower] = struct{}{}
		if unicode.IsLetter(r) {
			letters++
			if strings.ContainsRune("aeiouyаеёиоуыэюя", lower) {
				vowels++
			}
		} else if r >= '0' && r <= '9' {
			digits++
		} else if !unicode.IsNumber(r) {
			symbols++
		}
		hasLower = hasLower || r >= 'a' && r <= 'z'
		hasUpper = hasUpper || r >= 'A' && r <= 'Z'
		connector = connector || r == '_' || r == '-'
	}
	if letters >= 2 && letters == len([]rune(value)) {
		return false
	}
	length := UTF16Length(value)
	if letters == 0 || float64(len(unique))/float64(length) < 0.45 || float64(vowels)/float64(length) > 0.45 {
		return false
	}
	return digits > 0 || hasLower && hasUpper || connector || float64(symbols)/float64(length) > 0.18
}

func graphemes(value string) []string {
	iterator := uniseg.NewGraphemes(value)
	result := []string{}
	for iterator.Next() {
		result = append(result, iterator.Str())
	}
	return result
}

func nonWhitespaceGraphemes(value string) []string {
	result := []string{}
	for _, grapheme := range graphemes(value) {
		if !allRunes(grapheme, isJSWhitespace) {
			result = append(result, grapheme)
		}
	}
	return result
}

func dominanceTokens(value string) []string {
	result := []string{}
	var word strings.Builder
	flush := func() {
		if word.Len() != 0 {
			result = append(result, strings.ToLower(word.String()))
			word.Reset()
		}
	}
	for _, grapheme := range graphemes(value) {
		if allRunes(grapheme, isJSWhitespace) {
			flush()
		} else if isWordLike(grapheme) {
			word.WriteString(grapheme)
		} else {
			flush()
			if strings.TrimSpace(grapheme) != "" {
				result = append(result, grapheme)
			}
		}
	}
	flush()
	return result
}

func isWordLike(value string) bool {
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return true
		}
	}
	return false
}

func isSymbol(value string) bool {
	if isWordLike(value) {
		return false
	}
	for _, r := range value {
		if unicode.IsSymbol(r) || isExtendedPictographic(r) {
			return true
		}
	}
	return false
}

func allRunes(value string, predicate func(rune) bool) bool {
	for _, r := range value {
		if !predicate(r) {
			return false
		}
	}
	return value != ""
}

func uniqueCount(values []string) int {
	unique := make(map[string]struct{}, len(values))
	for _, value := range values {
		unique[value] = struct{}{}
	}
	return len(unique)
}

func UTF16Length(value string) int { return len(utf16.Encode([]rune(value))) }

func truncateUTF16(value string, limit int) string {
	encoded := utf16.Encode([]rune(value))
	if len(encoded) <= limit {
		return value
	}
	return string(utf16.Decode(encoded[:limit]))
}

func collapseWhitespace(value string) string {
	var result strings.Builder
	space := false
	for _, r := range value {
		if isJSWhitespace(r) {
			space = true
			continue
		}
		if space {
			result.WriteByte(' ')
			space = false
		}
		result.WriteRune(r)
	}
	if space {
		result.WriteByte(' ')
	}
	return result.String()
}

func isJSWhitespace(r rune) bool {
	return r == '\t' || r == '\n' || r == '\v' || r == '\f' || r == '\r' || r == ' ' || r == '\u00a0' || r == '\ufeff' ||
		r == '\u2028' || r == '\u2029' || unicode.In(r, unicode.Zs)
}

func trimJSWhitespace(value string) string { return strings.TrimFunc(value, isJSWhitespace) }

func splitJSWhitespace(value string) []string {
	return strings.FieldsFunc(value, isJSWhitespace)
}

func replaceURLs(value string) string {
	matches := urlMarkerPattern.FindAllStringIndex(value, -1)
	if len(matches) == 0 {
		return value
	}
	var result strings.Builder
	start := 0
	for _, match := range matches {
		if match[0] < start {
			continue
		}
		result.WriteString(value[start:match[0]])
		end := match[1]
		for end < len(value) {
			r, size := utf8.DecodeRuneInString(value[end:])
			if isJSWhitespace(r) {
				break
			}
			end += size
		}
		result.WriteString(" <url> ")
		start = end
	}
	result.WriteString(value[start:])
	return result.String()
}

func replaceNoise(value string) string {
	var result strings.Builder
	noise := false
	for _, r := range value {
		allowed := unicode.IsLetter(r) || unicode.IsNumber(r) || r == '<' || r == '>' || isJSWhitespace(r)
		if !allowed {
			noise = true
			continue
		}
		if noise {
			result.WriteByte(' ')
			noise = false
		}
		result.WriteRune(r)
	}
	if noise {
		result.WriteByte(' ')
	}
	return result.String()
}

func isExtendedPictographic(r rune) bool {
	return r == 0x00a9 || r == 0x00ae || r == 0x203c || r == 0x2049 || r == 0x2122 || r == 0x2139 ||
		r >= 0x2194 && r <= 0x2199 || r >= 0x21a9 && r <= 0x21aa || r >= 0x231a && r <= 0x231b ||
		r == 0x2328 || r == 0x2388 || r == 0x23cf || r >= 0x23e9 && r <= 0x23f3 || r >= 0x23f8 && r <= 0x23fa ||
		r == 0x24c2 || r >= 0x25aa && r <= 0x25ab || r == 0x25b6 || r == 0x25c0 || r >= 0x25fb && r <= 0x25fe ||
		r >= 0x2600 && r <= 0x27bf || r >= 0x2934 && r <= 0x2935 || r >= 0x2b05 && r <= 0x2b07 ||
		r >= 0x2b1b && r <= 0x2b1c || r == 0x2b50 || r == 0x2b55 || r == 0x3030 || r == 0x303d ||
		r == 0x3297 || r == 0x3299 || r >= 0x1f000 && r <= 0x1faff
}

func validationError(message string) error     { return &ValidationError{Message: message} }
func spamValidationError(message string) error { return &ValidationError{Message: message, Mask: true} }
