package content

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
)

type BlockType string

const (
	Paragraph BlockType = "paragraph"
	Heading   BlockType = "heading"
	Quote     BlockType = "quote"
	Divider   BlockType = "divider"
	Media     BlockType = "media"
	Unknown   BlockType = "unknown"
)

type MediaKind string

const (
	Image MediaKind = "image"
	GIF   MediaKind = "gif"
	Video MediaKind = "video"
	Audio MediaKind = "audio"
	File  MediaKind = "file"
)

type MarkType string

const (
	Bold       MarkType = "bold"
	Italic     MarkType = "italic"
	Link       MarkType = "link"
	InlineCode MarkType = "code"
)

type Mark struct {
	Type MarkType `json:"type"`
	Href string   `json:"href,omitempty"`
}

type InlineNode struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Marks []Mark `json:"marks,omitempty"`
}

type Block struct {
	Type      BlockType       `json:"type"`
	Text      string          `json:"text,omitempty"`
	Content   []InlineNode    `json:"content,omitempty"`
	Level     int             `json:"level,omitempty"`
	MediaKind MediaKind       `json:"mediaKind,omitempty"`
	Src       string          `json:"src,omitempty"`
	Spoiler   bool            `json:"spoiler,omitempty"`
	Name      string          `json:"name,omitempty"`
	Alt       string          `json:"alt,omitempty"`
	Caption   string          `json:"caption,omitempty"`
	Raw       json.RawMessage `json:"-"`
}

type PreviewMedia struct {
	MediaKind MediaKind `json:"mediaKind"`
	Src       string    `json:"src"`
	Alt       string    `json:"alt"`
	Caption   string    `json:"caption"`
	Name      string    `json:"name"`
}

var wordPattern = regexp.MustCompile(`[\p{L}\p{N}]+`)

type Limits struct {
	MaxBlocks    int
	MaxText      int
	MaxMediaText int
}

func (block Block) MarshalJSON() ([]byte, error) {
	if block.Type == Unknown {
		return []byte(`{"type":"unknown"}`), nil
	}
	value, err := storageBlock(block)
	if err != nil {
		return nil, err
	}
	return json.Marshal(value)
}

func ParseBlocksJSON(raw string) ([]Block, error) {
	messages, err := decodeDocument(raw)
	if err != nil {
		return nil, err
	}
	blocks := make([]Block, 0, len(messages))
	for _, message := range messages {
		block, err := parseBlock(message, false)
		if err != nil {
			blocks = append(blocks, Block{Type: Unknown, Raw: append(json.RawMessage(nil), message...)})
			continue
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

func ValidateBlocksJSON(raw string, limits Limits) ([]Block, error) {
	if limits.MaxBlocks <= 0 || limits.MaxText <= 0 || limits.MaxMediaText <= 0 {
		return nil, errors.New("content limits must be positive")
	}
	messages, err := decodeDocument(raw)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, errors.New("blocks must be a non-empty array")
	}
	if len(messages) > limits.MaxBlocks {
		return nil, fmt.Errorf("blocks must contain at most %d items", limits.MaxBlocks)
	}
	blocks := make([]Block, 0, len(messages))
	for index, message := range messages {
		block, err := parseBlock(message, true)
		if err != nil {
			return nil, fmt.Errorf("blocks[%d]: %w", index, err)
		}
		if isTextBlock(block.Type) && len(utf16.Encode([]rune(block.Text))) > limits.MaxText {
			return nil, fmt.Errorf("blocks[%d].text must be at most %d characters", index, limits.MaxText)
		}
		if block.Type == Media {
			for field, value := range map[string]string{"name": block.Name, "alt": block.Alt, "caption": block.Caption} {
				if len(utf16.Encode([]rune(value))) > limits.MaxMediaText {
					return nil, fmt.Errorf("blocks[%d].%s must be at most %d characters", index, field, limits.MaxMediaText)
				}
			}
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

func MarshalBlocksJSON(blocks []Block) (string, error) {
	values := make([]any, 0, len(blocks))
	for index, block := range blocks {
		value, err := storageBlock(block)
		if err != nil {
			return "", fmt.Errorf("blocks[%d]: %w", index, err)
		}
		values = append(values, value)
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("marshal blocks: %w", err)
	}
	return string(encoded), nil
}

func ParsePreviewMediaJSON(raw string) (*PreviewMedia, error) {
	var preview PreviewMedia
	if err := json.Unmarshal([]byte(raw), &preview); err != nil {
		return nil, err
	}
	if _, err := storageBlock(Block{
		Type: Media, MediaKind: preview.MediaKind, Src: preview.Src,
		Name: preview.Name, Alt: preview.Alt, Caption: preview.Caption,
	}); err != nil {
		return nil, err
	}
	preview.Src = strings.TrimSpace(preview.Src)
	preview.Name = strings.TrimSpace(preview.Name)
	preview.Alt = strings.TrimSpace(preview.Alt)
	preview.Caption = strings.TrimSpace(preview.Caption)
	return &preview, nil
}

func FeedSummary(blocks []Block) (previewText string, readingMinutes int, previewMedia *PreviewMedia) {
	var readingText strings.Builder
	appendText := func(value string) {
		if value == "" {
			return
		}
		if readingText.Len() != 0 {
			readingText.WriteByte(' ')
		}
		readingText.WriteString(value)
	}
	for _, block := range blocks {
		switch block.Type {
		case Paragraph, Heading, Quote:
			if previewText == "" && block.Type == Paragraph {
				previewText = block.Text
			}
			appendText(block.Text)
		case Media:
			appendText(strings.Join(nonEmpty(block.Name, block.Alt, block.Caption), " "))
			media := &PreviewMedia{MediaKind: block.MediaKind, Src: block.Src, Alt: block.Alt, Caption: block.Caption, Name: block.Name}
			if previewMedia == nil || block.MediaKind == Audio && previewMedia.MediaKind != Audio {
				previewMedia = media
			}
		}
	}
	text := readingText.String()
	if text != "" {
		words := len(wordPattern.FindAllString(text, -1))
		if words == 0 {
			words = (len(utf16.Encode([]rune(text))) + 4) / 5
		}
		readingMinutes = max(1, (words+179)/180)
	}
	return previewText, readingMinutes, previewMedia
}

func nonEmpty(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func decodeDocument(raw string) ([]json.RawMessage, error) {
	var messages []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &messages); err != nil {
		return nil, fmt.Errorf("parse blocks JSON: %w", err)
	}
	if messages == nil {
		return nil, errors.New("blocks JSON must be an array")
	}
	return messages, nil
}

func parseBlock(raw json.RawMessage, strict bool) (Block, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return Block{}, errors.New("block must be an object")
	}
	typeName, ok := textField(fields, "type")
	if !ok {
		return Block{}, errors.New("type is required")
	}
	block := Block{Type: BlockType(typeName)}
	switch block.Type {
	case Paragraph, Heading, Quote:
		if block.Type == Heading {
			level, err := headingLevel(fields["level"], strict)
			if err != nil {
				return Block{}, err
			}
			block.Level = level
		}
		text, _ := textField(fields, "text")
		contentRaw, hasContent := fields["content"]
		if hasContent {
			content, fallback, err := parseInline(contentRaw)
			if err == nil {
				block.Content = content
				block.Text = fallback
				return block, nil
			}
			if strict || text == "" {
				return Block{}, err
			}
		}
		if text == "" {
			return Block{}, errors.New("text is required")
		}
		block.Text = text
		return block, nil
	case Divider:
		return block, nil
	case Media:
		kind, ok := textField(fields, "mediaKind")
		if !ok || !validMediaKind(MediaKind(kind)) {
			return Block{}, errors.New("mediaKind is invalid")
		}
		src, ok := textField(fields, "src")
		if !ok || !safeMediaSource(src) {
			return Block{}, errors.New("src must be a local /uploads/... path")
		}
		block.MediaKind = MediaKind(kind)
		block.Src = src
		block.Name, _ = textField(fields, "name")
		block.Alt, _ = textField(fields, "alt")
		block.Caption, _ = textField(fields, "caption")
		if spoiler, exists := fields["spoiler"]; exists {
			if err := json.Unmarshal(spoiler, &block.Spoiler); err != nil && strict {
				return Block{}, errors.New("spoiler must be a boolean")
			}
		}
		return block, nil
	default:
		return Block{}, fmt.Errorf("unsupported type %q", typeName)
	}
}

func parseInline(raw json.RawMessage) ([]InlineNode, string, error) {
	if string(raw) == "null" {
		return nil, "", errors.New("content must be an array")
	}
	var nodes []InlineNode
	if err := json.Unmarshal(raw, &nodes); err != nil || nodes == nil {
		return nil, "", errors.New("content must be an array")
	}
	var fallback strings.Builder
	for index, node := range nodes {
		if node.Type != "text" {
			return nil, "", fmt.Errorf("content[%d].type is unsupported", index)
		}
		seen := map[MarkType]bool{}
		for markIndex := range node.Marks {
			mark := &node.Marks[markIndex]
			mark.Href = strings.TrimSpace(mark.Href)
			if seen[mark.Type] {
				return nil, "", fmt.Errorf("content[%d].marks[%d] is duplicated", index, markIndex)
			}
			seen[mark.Type] = true
			switch mark.Type {
			case Bold, Italic, InlineCode:
				if mark.Href != "" {
					return nil, "", fmt.Errorf("content[%d].marks[%d].href is not allowed", index, markIndex)
				}
			case Link:
				if !safeLink(mark.Href) {
					return nil, "", fmt.Errorf("content[%d].marks[%d].href is unsafe", index, markIndex)
				}
			default:
				return nil, "", fmt.Errorf("content[%d].marks[%d].type is unsupported", index, markIndex)
			}
		}
		fallback.WriteString(node.Text)
	}
	text := strings.TrimSpace(fallback.String())
	if text == "" {
		return nil, "", errors.New("content must contain text")
	}
	return nodes, text, nil
}

func textField(fields map[string]json.RawMessage, name string) (string, bool) {
	value, exists := fields[name]
	if !exists {
		return "", false
	}
	var text string
	if json.Unmarshal(value, &text) != nil {
		return "", false
	}
	text = strings.TrimSpace(text)
	return text, text != ""
}

func headingLevel(raw json.RawMessage, strict bool) (int, error) {
	var level int
	if err := json.Unmarshal(raw, &level); err != nil && !strict {
		var number float64
		if json.Unmarshal(raw, &number) == nil && number == float64(int(number)) {
			level = int(number)
		}
		var text string
		if json.Unmarshal(raw, &text) == nil {
			number, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
			if err == nil && number == float64(int(number)) {
				level = int(number)
			}
		}
	}
	if level < 1 || level > 3 {
		return 0, errors.New("level must be 1, 2, or 3")
	}
	return level, nil
}

func validMediaKind(kind MediaKind) bool {
	return kind == Image || kind == GIF || kind == Video || kind == Audio || kind == File
}

func safeMediaSource(src string) bool {
	if strings.Contains(src, "\\") || strings.IndexFunc(src, unicode.IsControl) >= 0 {
		return false
	}
	parsed, err := url.Parse(src)
	if err != nil || parsed.Scheme != "" || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	decoded := parsed.EscapedPath()
	stable := false
	for range 8 {
		lower := strings.ToLower(decoded)
		if strings.Contains(lower, "%2f") || strings.Contains(lower, "%5c") {
			return false
		}
		next, err := url.PathUnescape(decoded)
		if err != nil || strings.Contains(next, "\\") || strings.IndexFunc(next, unicode.IsControl) >= 0 {
			return false
		}
		if next == decoded {
			stable = true
			break
		}
		decoded = next
	}
	if !stable || !strings.HasPrefix(decoded, "/uploads/") || decoded == "/uploads/" {
		return false
	}
	return path.Clean(decoded) == decoded && !strings.Contains(decoded, "\\")
}

func safeLink(href string) bool {
	href = strings.TrimSpace(href)
	if href == "" || strings.Contains(href, "\\") || strings.IndexFunc(href, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) >= 0 {
		return false
	}
	parsed, err := url.Parse(href)
	if err != nil || parsed.User != nil {
		return false
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return parsed.Host != ""
	case "mailto":
		return parsed.Opaque != "" && parsed.Host == ""
	case "":
		return (strings.HasPrefix(href, "/") && !strings.HasPrefix(href, "//") && parsed.Host == "") || strings.HasPrefix(href, "#")
	default:
		return false
	}
}

func isTextBlock(blockType BlockType) bool {
	return blockType == Paragraph || blockType == Heading || blockType == Quote
}

func storageBlock(block Block) (any, error) {
	if isTextBlock(block.Type) {
		if block.Content != nil {
			content, text, err := parseInline(mustJSON(block.Content))
			if err != nil {
				return nil, err
			}
			block.Content, block.Text = content, text
		}
		if strings.TrimSpace(block.Text) == "" {
			return nil, errors.New("text is required")
		}
		if block.Type == Heading && (block.Level < 1 || block.Level > 3) {
			return nil, errors.New("level must be 1, 2, or 3")
		}
		return struct {
			Type    BlockType    `json:"type"`
			Level   int          `json:"level,omitempty"`
			Text    string       `json:"text"`
			Content []InlineNode `json:"content,omitempty"`
		}{block.Type, block.Level, strings.TrimSpace(block.Text), block.Content}, nil
	}
	switch block.Type {
	case Divider:
		return struct {
			Type BlockType `json:"type"`
		}{Divider}, nil
	case Media:
		if !validMediaKind(block.MediaKind) || !safeMediaSource(block.Src) {
			return nil, errors.New("invalid media block")
		}
		return struct {
			Type      BlockType `json:"type"`
			MediaKind MediaKind `json:"mediaKind"`
			Src       string    `json:"src"`
			Spoiler   bool      `json:"spoiler"`
			Name      string    `json:"name,omitempty"`
			Alt       string    `json:"alt,omitempty"`
			Caption   string    `json:"caption,omitempty"`
		}{Media, block.MediaKind, block.Src, block.Spoiler, strings.TrimSpace(block.Name), strings.TrimSpace(block.Alt), strings.TrimSpace(block.Caption)}, nil
	default:
		return nil, fmt.Errorf("unsupported type %q cannot be stored", block.Type)
	}
}

func mustJSON(value any) json.RawMessage {
	encoded, _ := json.Marshal(value)
	return encoded
}
