package admin

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/Levipanic/stereoblog-v2/backend/internal/config"
	"github.com/Levipanic/stereoblog-v2/backend/internal/posts"
	database "github.com/Levipanic/stereoblog-v2/backend/internal/storage/sqlite"
)

type Sessions struct {
	db  *sql.DB
	cfg config.Admin
}
type Session struct {
	Hash          string `json:"-"`
	Authenticated bool   `json:"authenticated"`
	CreatedAt     string `json:"created_at,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`
	CSRF          string `json:"csrf_token,omitempty"`
}

func NewSessions(db *sql.DB, cfg config.Admin) *Sessions { return &Sessions{db, cfg} }

func Equal(a, b string) bool {
	x, y := sha256.Sum256([]byte(a)), sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(x[:], y[:]) == 1
}

func (s *Sessions) ValidSecret(secret string) bool {
	return s.cfg.Secret != "" && Equal(secret, s.cfg.Secret)
}
func (s *Sessions) csrf(hash string) string {
	h := hmac.New(sha256.New, []byte(s.cfg.SessionHashSalt))
	h.Write([]byte("csrf:" + hash))
	return hex.EncodeToString(h.Sum(nil))
}
func (s *Sessions) Create(ctx context.Context) (string, Session, error) {
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", Session{}, err
	}
	token := hex.EncodeToString(random[:])
	hash := posts.HashIP(s.cfg.SessionHashSalt, token)
	now := time.Now().UTC()
	session := Session{hash, true, database.FormatTime(now), database.FormatTime(now.Add(s.cfg.SessionTTL)), s.csrf(hash)}
	if _, err := s.db.ExecContext(ctx, "DELETE FROM admin_sessions WHERE datetime(expires_at) <= datetime(?)", database.FormatTime(now)); err != nil {
		return "", Session{}, err
	}
	_, err := s.db.ExecContext(ctx, "INSERT INTO admin_sessions (token_hash, created_at, expires_at) VALUES (?, ?, ?)", hash, session.CreatedAt, session.ExpiresAt)
	return token, session, err
}
func (s *Sessions) Get(ctx context.Context, token string) (Session, error) {
	if len(token) != 64 {
		return Session{}, nil
	}
	if _, err := hex.DecodeString(token); err != nil {
		return Session{}, nil
	}
	hash := posts.HashIP(s.cfg.SessionHashSalt, token)
	var session Session
	err := s.db.QueryRowContext(ctx, "SELECT created_at, expires_at FROM admin_sessions WHERE token_hash = ?", hash).Scan(&session.CreatedAt, &session.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, nil
	}
	if err != nil {
		return Session{}, err
	}
	created, e1 := database.ParseTime(session.CreatedAt)
	expires, e2 := database.ParseTime(session.ExpiresAt)
	now := time.Now().UTC()
	if e1 != nil || e2 != nil || !expires.After(now) || created.After(now.Add(s.cfg.SessionClockSkew)) {
		return Session{}, s.Delete(ctx, hash)
	}
	session.Hash, session.Authenticated, session.CSRF = hash, true, s.csrf(hash)
	return session, nil
}
func (s *Sessions) Delete(ctx context.Context, hash string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM admin_sessions WHERE token_hash = ?", hash)
	return err
}
