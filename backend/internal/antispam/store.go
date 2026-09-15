package antispam

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	database "github.com/Levipanic/stereoblog-v2/backend/internal/storage/sqlite"
)

type Attempt struct {
	IPHash      string
	PostID      *int64
	Status      string
	Reason      string
	Content     string
	TextHash    string
	Fingerprint string
}

type Mute struct {
	Reason     string
	MutedUntil time.Time
	MuteCount  int
	RetryAfter time.Duration
}

func (s *Service) ConsumeChallenge(ctx context.Context, challenge VerifiedChallenge) (err error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire challenge connection: %w", err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("begin challenge consumption: %w", err)
	}
	defer func() {
		if err != nil {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	cutoff := database.FormatTime(s.now().Add(-2 * s.cfg.ChallengeTTL))
	if _, err = conn.ExecContext(ctx, "DELETE FROM comment_challenge_uses WHERE datetime(last_used_at) < datetime(?)", cutoff); err != nil {
		return fmt.Errorf("clean comment challenges: %w", err)
	}
	var used int
	err = conn.QueryRowContext(ctx, "SELECT used_count FROM comment_challenge_uses WHERE token_hash = ? AND post_id = ?", challenge.TokenHash, challenge.PostID).Scan(&used)
	if err == nil && used > 0 {
		return challengeError("challenge_replay")
	}
	now := database.FormatTime(s.now())
	if errors.Is(err, sql.ErrNoRows) {
		_, err = conn.ExecContext(ctx, `INSERT INTO comment_challenge_uses
			(token_hash, post_id, used_count, first_used_at, last_used_at) VALUES (?, ?, 1, ?, ?)`, challenge.TokenHash, challenge.PostID, now, now)
	} else if err == nil {
		_, err = conn.ExecContext(ctx, "UPDATE comment_challenge_uses SET used_count = 1, last_used_at = ? WHERE token_hash = ?", now, challenge.TokenHash)
	}
	if err != nil {
		return fmt.Errorf("consume comment challenge: %w", err)
	}
	if _, err = conn.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("commit challenge consumption: %w", err)
	}
	return nil
}

func (s *Service) AttemptStats(ctx context.Context, ipHash string, postID int64, textHash, fingerprint string) (Stats, error) {
	if err := s.cleanupAttempts(ctx); err != nil {
		return Stats{}, err
	}
	now := s.now()
	var last, duplicate sql.NullString
	var stats Stats
	err := s.db.QueryRowContext(ctx, `SELECT
		(SELECT created_at FROM comment_attempts WHERE ip_hash = ? ORDER BY datetime(created_at) DESC, id DESC LIMIT 1),
		(SELECT count(*) FROM comment_attempts WHERE ip_hash = ? AND datetime(created_at) >= datetime(?)),
		(SELECT count(*) FROM comment_attempts WHERE ip_hash = ? AND status IN ('rejected','muted') AND datetime(created_at) >= datetime(?)),
		(SELECT count(*) FROM comment_attempts WHERE post_id = ? AND status IN ('visible','pending') AND datetime(created_at) >= datetime(?)),
		(SELECT count(*) FROM comment_attempts WHERE status IN ('visible','pending') AND datetime(created_at) >= datetime(?)),
		(SELECT created_at FROM comment_attempts WHERE ip_hash = ? AND post_id = ? AND text_hash = ? AND status IN ('visible','pending') AND datetime(created_at) >= datetime(?) ORDER BY datetime(created_at) DESC, id DESC LIMIT 1),
		(SELECT count(*) FROM comment_attempts WHERE ip_hash = ? AND post_id = ? AND fingerprint = ? AND text_hash <> ? AND status IN ('visible','pending') AND datetime(created_at) >= datetime(?))`,
		ipHash,
		ipHash, database.FormatTime(now.Add(-s.cfg.BurstWindow)),
		ipHash, database.FormatTime(now.Add(-s.cfg.BurstWindow)),
		postID, database.FormatTime(now.Add(-s.cfg.PostRateLimitWindow)),
		database.FormatTime(now.Add(-s.cfg.GlobalRateLimitWindow)),
		ipHash, postID, textHash, database.FormatTime(now.Add(-s.cfg.DuplicateWindow)),
		ipHash, postID, fingerprint, textHash, database.FormatTime(now.Add(-s.cfg.DuplicateWindow)),
	).Scan(&last, &stats.IPRecentCount, &stats.IPRejectedCount, &stats.PostRecentCount, &stats.GlobalRecentCount, &duplicate, &stats.FingerprintRecentCount)
	if err != nil {
		return Stats{}, fmt.Errorf("query comment attempt stats: %w", err)
	}
	if last.Valid {
		value, err := database.ParseTime(last.String)
		if err != nil {
			return Stats{}, fmt.Errorf("invalid last comment attempt timestamp: %w", err)
		}
		stats.LastAttemptAt = &value
	}
	if duplicate.Valid {
		value, err := database.ParseTime(duplicate.String)
		if err != nil {
			return Stats{}, fmt.Errorf("invalid duplicate comment timestamp: %w", err)
		}
		stats.DuplicateAt = &value
	}
	return stats, nil
}

func (s *Service) RecordAttempt(ctx context.Context, attempt Attempt) error {
	if err := s.cleanupAttempts(ctx); err != nil {
		return err
	}
	return s.insertAttempt(ctx, s.db, attempt, s.now())
}

func (s *Service) insertAttempt(ctx context.Context, runner interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, attempt Attempt, now time.Time) error {
	status := attempt.Status
	if status == "" {
		status = Rejected
	}
	content := truncateUTF16(strings.TrimSpace(attempt.Content), s.cfg.AttemptContentMaxLength)
	_, err := runner.ExecContext(ctx, `INSERT INTO comment_attempts
		(ip_hash, post_id, status, reason, content, text_hash, fingerprint, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, attempt.IPHash, positiveID(attempt.PostID), status,
		nullString(attempt.Reason), nullString(content), nullString(attempt.TextHash), nullString(attempt.Fingerprint), database.FormatTime(now))
	if err != nil {
		return fmt.Errorf("record comment attempt: %w", err)
	}
	return nil
}

func (s *Service) RecordHoneypot(ctx context.Context, attempt Attempt) (*Mute, error) {
	attempt.Status, attempt.Reason = Rejected, "honeypot"
	if err := s.cleanupAttempts(ctx); err != nil {
		return nil, err
	}
	muted := false
	err := s.withImmediate(ctx, func(conn *sql.Conn) error {
		now := s.now()
		if err := s.insertAttempt(ctx, conn, attempt, now); err != nil {
			return err
		}
		var hits int
		if err := conn.QueryRowContext(ctx, `SELECT count(*) FROM comment_attempts
			WHERE ip_hash = ? AND reason = 'honeypot' AND datetime(created_at) >= datetime(?)`,
			attempt.IPHash, database.FormatTime(now.Add(-s.cfg.BurstWindow))).Scan(&hits); err != nil {
			return fmt.Errorf("count honeypot attempts: %w", err)
		}
		muted = hits >= s.cfg.HoneypotMuteThreshold
		if muted {
			return s.upsertMute(ctx, conn, attempt.IPHash, "honeypot", now)
		}
		return nil
	})
	if err != nil || !muted {
		return nil, err
	}
	return s.ActiveMute(ctx, attempt.IPHash)
}

func (s *Service) RecordRateRejection(ctx context.Context, attempt Attempt, rate RateDecision) (*Mute, error) {
	attempt.Status, attempt.Reason = Rejected, "rate:"+rate.Reason
	if err := s.cleanupAttempts(ctx); err != nil {
		return nil, err
	}
	muted := false
	err := s.withImmediate(ctx, func(conn *sql.Conn) error {
		now := s.now()
		if err := s.insertAttempt(ctx, conn, attempt, now); err != nil {
			return err
		}
		var rejected int
		if err := conn.QueryRowContext(ctx, `SELECT count(*) FROM comment_attempts
			WHERE ip_hash = ? AND status IN ('rejected','muted') AND datetime(created_at) >= datetime(?)`,
			attempt.IPHash, database.FormatTime(now.Add(-s.cfg.BurstWindow))).Scan(&rejected); err != nil {
			return fmt.Errorf("count rejected comment attempts: %w", err)
		}
		muted = rejected >= s.cfg.RejectedMuteThreshold
		if muted {
			return s.upsertMute(ctx, conn, attempt.IPHash, attempt.Reason, now)
		}
		return nil
	})
	if err != nil || !muted {
		return nil, err
	}
	return s.ActiveMute(ctx, attempt.IPHash)
}

func (s *Service) withImmediate(ctx context.Context, apply func(*sql.Conn) error) (err error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err = conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	if err = apply(conn); err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, "COMMIT")
	return err
}

func (s *Service) upsertMute(ctx context.Context, runner interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, ipHash, reason string, now time.Time) error {
	until := now.Add(max(time.Second, s.cfg.MuteDuration))
	_, err := runner.ExecContext(ctx, `INSERT INTO comment_mutes (ip_hash, reason, muted_until, mute_count, created_at)
		VALUES (?, ?, ?, 1, ?) ON CONFLICT(ip_hash) DO UPDATE SET reason = excluded.reason,
		muted_until = excluded.muted_until, mute_count = comment_mutes.mute_count + 1`,
		ipHash, reason, database.FormatTime(until), database.FormatTime(now))
	if err != nil {
		return fmt.Errorf("mute comment IP: %w", err)
	}
	return nil
}

func (s *Service) ActiveMute(ctx context.Context, ipHash string) (*Mute, error) {
	now := s.now()
	if _, err := s.db.ExecContext(ctx, "DELETE FROM comment_mutes WHERE datetime(muted_until) <= datetime(?)", database.FormatTime(now)); err != nil {
		return nil, fmt.Errorf("clean comment mutes: %w", err)
	}
	var mute Mute
	var reason sql.NullString
	var mutedUntil string
	err := s.db.QueryRowContext(ctx, `SELECT reason, muted_until, mute_count FROM comment_mutes
		WHERE ip_hash = ? AND datetime(muted_until) > datetime(?)`, ipHash, database.FormatTime(now)).Scan(&reason, &mutedUntil, &mute.MuteCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query comment mute: %w", err)
	}
	mute.Reason = "active_mute"
	if reason.Valid && reason.String != "" {
		mute.Reason = reason.String
	}
	mute.MutedUntil, err = database.ParseTime(mutedUntil)
	if err != nil {
		return nil, fmt.Errorf("invalid comment mute timestamp: %w", err)
	}
	mute.RetryAfter = max(time.Second, mute.MutedUntil.Sub(now))
	return &mute, nil
}

func (s *Service) cleanupAttempts(ctx context.Context) error {
	s.cleanupMu.Lock()
	defer s.cleanupMu.Unlock()
	now := s.now()
	if !s.lastAttemptCleanup.IsZero() && now.Before(s.lastAttemptCleanup.Add(time.Hour)) {
		return nil
	}
	retention := s.cfg.AttemptsTTL
	for _, window := range []time.Duration{s.cfg.BurstWindow, s.cfg.DuplicateWindow, s.cfg.PostRateLimitWindow, s.cfg.GlobalRateLimitWindow} {
		if window > retention {
			retention = window
		}
	}
	cutoff := database.FormatTime(now.Add(-retention))
	if _, err := s.db.ExecContext(ctx, "DELETE FROM comment_attempts WHERE datetime(created_at) < datetime(?)", cutoff); err != nil {
		return fmt.Errorf("clean comment attempts: %w", err)
	}
	s.lastAttemptCleanup = now
	return nil
}

func positiveID(value *int64) any {
	if value != nil && *value > 0 {
		return *value
	}
	return nil
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
