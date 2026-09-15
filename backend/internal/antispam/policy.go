package antispam

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const Pending = "pending"
const Visible = "visible"
const Rejected = "rejected"
const Muted = "muted"

type Stats struct {
	LastAttemptAt          *time.Time
	IPRecentCount          int
	IPRejectedCount        int
	PostRecentCount        int
	GlobalRecentCount      int
	DuplicateAt            *time.Time
	FingerprintRecentCount int
}

type RateDecision struct {
	Limited    bool
	Reason     string
	RetryAfter time.Duration
}

type RequestSource struct {
	Host    string
	Origin  string
	Referer string
}

type Signal struct {
	Reason string
	Score  int
}

type ModerationDecision struct {
	Status  string
	Score   int
	Reasons []string
	Reason  string
}

func (s *Service) RateLimit(stats Stats) RateDecision {
	now := s.now()
	if stats.LastAttemptAt != nil && now.Sub(*stats.LastAttemptAt) < s.cfg.Cooldown {
		return RateDecision{true, "cooldown", max(time.Second, s.cfg.Cooldown-now.Sub(*stats.LastAttemptAt))}
	}
	if stats.IPRecentCount >= s.cfg.BurstMax {
		return RateDecision{true, "ip_burst", max(time.Second, s.cfg.BurstWindow)}
	}
	if stats.PostRecentCount >= s.cfg.PostRateLimitMax {
		return RateDecision{true, "post_flood", max(time.Second, s.cfg.PostRateLimitWindow)}
	}
	if stats.GlobalRecentCount >= s.cfg.GlobalRateLimitMax {
		return RateDecision{true, "global_flood", max(time.Second, s.cfg.GlobalRateLimitWindow)}
	}
	if stats.DuplicateAt != nil {
		return RateDecision{true, "duplicate", max(time.Second, stats.DuplicateAt.Add(s.cfg.DuplicateWindow).Sub(now))}
	}
	return RateDecision{}
}

func SourceSignals(source RequestSource) []Signal {
	host := strings.TrimSpace(source.Host)
	if host == "" {
		return nil
	}
	origin := strings.TrimSpace(source.Origin)
	referer := strings.TrimSpace(source.Referer)
	result := []Signal{}
	if origin != "" && !urlHostMatches(origin, host) {
		result = append(result, Signal{"cross_origin", 3})
	}
	if referer != "" && !urlHostMatches(referer, host) {
		result = append(result, Signal{"cross_referer", 2})
	}
	if origin == "" && referer == "" {
		result = append(result, Signal{"missing_origin_referer", 1})
	}
	return result
}

func (s *Service) Moderate(source RequestSource, name, content string, stats Stats) ModerationDecision {
	decision := ModerationDecision{Status: Visible, Reasons: []string{}}
	add := func(reason string, score int) {
		decision.Score += score
		decision.Reasons = append(decision.Reasons, reason)
	}
	for _, signal := range SourceSignals(source) {
		add(signal.Reason, signal.Score)
	}
	urls := countURLs(content)
	if urls >= 2 {
		add("multiple_links", 2)
	}
	if urls >= 1 && UTF16Length(content) < 80 {
		add("short_link_comment", 2)
	}
	if countURLs(name) > 0 {
		add("link_in_name", 2)
	}
	if stats.FingerprintRecentCount > 0 {
		add("similar_recent_comment", 3)
	}
	if stats.IPRecentCount >= max(2, s.cfg.BurstMax-2) {
		add("near_ip_burst_limit", 2)
	}
	if stats.IPRejectedCount >= 3 {
		add("recent_rejections", 3)
	}
	if decision.Score >= 5 {
		decision.Status = Pending
		decision.Reason = fmt.Sprintf("score:%d; %s", decision.Score, strings.Join(decision.Reasons, ","))
	}
	return decision
}

func urlHostMatches(value, host string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.IsAbs() && strings.EqualFold(parsed.Host, host)
}
