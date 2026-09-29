package admin

import (
	"context"
	"database/sql"
	"errors"
)

var ErrNotFound = errors.New("moderation record not found")

type Moderation struct{ db *sql.DB }

func NewModeration(db *sql.DB) *Moderation { return &Moderation{db} }

type PendingComment struct {
	ID        int64   `json:"id"`
	PostID    int64   `json:"post_id"`
	ParentID  *int64  `json:"parent_id"`
	Name      *string `json:"name"`
	Content   string  `json:"content"`
	Reason    *string `json:"moderation_reason"`
	CreatedAt string  `json:"created_at"`
}
type Attempt struct {
	ID          int64   `json:"id"`
	PostID      *int64  `json:"post_id"`
	Status      string  `json:"status"`
	Reason      *string `json:"reason"`
	IPHashShort string  `json:"ip_hash_short"`
	Content     *string `json:"content"`
	CreatedAt   string  `json:"created_at"`
}
type Mute struct {
	ID          int64   `json:"id"`
	IPHashShort string  `json:"ip_hash_short"`
	Reason      *string `json:"reason"`
	MutedUntil  string  `json:"muted_until"`
	MuteCount   int     `json:"mute_count"`
	CreatedAt   string  `json:"created_at"`
}
type Overview struct {
	Pending  []PendingComment `json:"pending_comments"`
	Attempts []Attempt        `json:"attempts"`
	Mutes    []Mute           `json:"mutes"`
}

func (m *Moderation) Overview(ctx context.Context, limit int) (Overview, error) {
	o := Overview{Pending: []PendingComment{}, Attempts: []Attempt{}, Mutes: []Mute{}}
	limit = max(1, min(100, limit))
	rows, err := m.db.QueryContext(ctx, `SELECT id,post_id,parent_id,name,content,moderation_reason,created_at FROM comments WHERE status='pending' ORDER BY datetime(created_at) DESC,id DESC LIMIT ?`, limit)
	if err != nil {
		return o, err
	}
	for rows.Next() {
		var p PendingComment
		if err := rows.Scan(&p.ID, &p.PostID, &p.ParentID, &p.Name, &p.Content, &p.Reason, &p.CreatedAt); err != nil {
			rows.Close()
			return o, err
		}
		o.Pending = append(o.Pending, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return o, err
	}
	rows, err = m.db.QueryContext(ctx, `SELECT id,post_id,status,reason,substr(ip_hash,1,12),content,created_at FROM comment_attempts ORDER BY datetime(created_at) DESC,id DESC LIMIT ?`, limit)
	if err != nil {
		return o, err
	}
	for rows.Next() {
		var a Attempt
		if err := rows.Scan(&a.ID, &a.PostID, &a.Status, &a.Reason, &a.IPHashShort, &a.Content, &a.CreatedAt); err != nil {
			rows.Close()
			return o, err
		}
		o.Attempts = append(o.Attempts, a)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return o, err
	}
	rows, err = m.db.QueryContext(ctx, `SELECT id,substr(ip_hash,1,12),reason,muted_until,mute_count,created_at FROM comment_mutes WHERE datetime(muted_until)>datetime('now') ORDER BY datetime(muted_until) DESC,id DESC LIMIT ?`, limit)
	if err != nil {
		return o, err
	}
	defer rows.Close()
	for rows.Next() {
		var m Mute
		if err := rows.Scan(&m.ID, &m.IPHashShort, &m.Reason, &m.MutedUntil, &m.MuteCount, &m.CreatedAt); err != nil {
			return o, err
		}
		o.Mutes = append(o.Mutes, m)
	}
	return o, rows.Err()
}
func (m *Moderation) SetStatus(ctx context.Context, id int64, approve bool) error {
	status := "rejected"
	var reason any = "admin_rejected"
	if approve {
		status = "visible"
		reason = nil
	}
	result, err := m.db.ExecContext(ctx, "UPDATE comments SET status=?,moderation_reason=? WHERE id=?", status, reason, id)
	return affected(result, err)
}
func (m *Moderation) DeleteComment(ctx context.Context, id int64) (int64, error) {
	// Legacy databases may lack a parent_id foreign key; explicitly delete the subtree.
	// UNION also terminates safely if old data contains a parent cycle.
	var postID int64
	err := m.db.QueryRowContext(ctx, `WITH RECURSIVE tree(id) AS (
		SELECT id FROM comments WHERE id = ?
		UNION SELECT comments.id FROM comments JOIN tree ON comments.parent_id = tree.id
	) DELETE FROM comments WHERE id IN (SELECT id FROM tree) RETURNING post_id`, id).Scan(&postID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return postID, err
}
func (m *Moderation) Unmute(ctx context.Context, id int64) error {
	result, err := m.db.ExecContext(ctx, "DELETE FROM comment_mutes WHERE id=?", id)
	return affected(result, err)
}
func affected(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return ErrNotFound
	}
	return err
}
