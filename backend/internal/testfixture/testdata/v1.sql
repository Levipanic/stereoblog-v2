PRAGMA foreign_keys = ON;

BEGIN;

CREATE TABLE posts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  blocks_json TEXT NOT NULL,
  likes_count INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  preview_media TEXT DEFAULT NULL
);

CREATE TABLE comments (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  post_id INTEGER NOT NULL,
  parent_id INTEGER,
  name TEXT,
  content TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  likes_count INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'visible',
  moderation_reason TEXT,
  text_hash TEXT,
  text_fingerprint TEXT,
  FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
  FOREIGN KEY (parent_id) REFERENCES comments(id) ON DELETE CASCADE
);

CREATE INDEX idx_comments_post_parent ON comments(post_id, parent_id, id);
CREATE INDEX idx_comments_status_created ON comments(status, created_at);

CREATE TABLE like_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  post_id INTEGER NOT NULL,
  ip_hash TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
);

CREATE INDEX idx_like_events_post_ip_created ON like_events(post_id, ip_hash, created_at);

CREATE TABLE comment_like_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  comment_id INTEGER NOT NULL,
  ip_hash TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY (comment_id) REFERENCES comments(id) ON DELETE CASCADE
);

CREATE INDEX idx_comment_like_events_comment_ip_created ON comment_like_events(comment_id, ip_hash, created_at);

CREATE TABLE comment_attempts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ip_hash TEXT NOT NULL,
  post_id INTEGER,
  status TEXT NOT NULL,
  reason TEXT,
  content TEXT,
  text_hash TEXT,
  fingerprint TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_comment_attempts_ip_time ON comment_attempts(ip_hash, created_at);
CREATE INDEX idx_comment_attempts_post_time ON comment_attempts(post_id, created_at);
CREATE INDEX idx_comment_attempts_ip_post_hash_time ON comment_attempts(ip_hash, post_id, text_hash, created_at);
CREATE INDEX idx_comment_attempts_ip_status_time ON comment_attempts(ip_hash, status, created_at);

CREATE TABLE comment_mutes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ip_hash TEXT NOT NULL UNIQUE,
  reason TEXT,
  muted_until TEXT NOT NULL,
  mute_count INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_comment_mutes_until ON comment_mutes(muted_until);

CREATE TABLE comment_challenge_uses (
  token_hash TEXT PRIMARY KEY,
  post_id INTEGER NOT NULL,
  used_count INTEGER NOT NULL DEFAULT 0,
  first_used_at TEXT NOT NULL DEFAULT (datetime('now')),
  last_used_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE admin_sessions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  token_hash TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  expires_at TEXT NOT NULL
);

CREATE INDEX idx_admin_sessions_expires_at ON admin_sessions(expires_at);

INSERT INTO posts (id, title, blocks_json, likes_count, created_at, preview_media) VALUES
  (1, 'Тестовая публикация', '[{"type":"paragraph","text":"Привет из синтетической фикстуры."},{"type":"heading","level":2,"text":"Раздел"},{"type":"quote","text":"Цитата"},{"type":"divider"},{"type":"media","mediaKind":"image","src":"/uploads/fixture-image.jpg","spoiler":true,"name":"image.jpg","alt":"Описание","caption":"Подпись"},{"type":"media","mediaKind":"gif","src":"/uploads/fixture-animation.gif","spoiler":false,"name":"animation.gif","alt":"GIF","caption":""},{"type":"media","mediaKind":"video","src":"/uploads/fixture-video.mp4","spoiler":false,"name":"video.mp4","alt":"","caption":"Видео"},{"type":"media","mediaKind":"audio","src":"/uploads/fixture-audio.mp3","spoiler":false,"name":"audio.mp3","alt":"","caption":"Аудио"},{"type":"media","mediaKind":"file","src":"/uploads/fixture-file.zip","spoiler":false,"name":"archive.zip","alt":"","caption":"Файл"}]', 7, '2024-01-02 03:04:05', '{"src":"/uploads/fixture-image.jpg","mediaKind":"image","alt":"Описание","caption":"Подпись"}'),
  (2, 'English fixture post', '[{"type":"paragraph","text":"Deterministic English content."},{"type":"heading","level":9,"text":"Invalid legacy heading"},{"type":"future-block","payload":{"keep":true}}]', 0, '2024-01-02 03:04:05', NULL);

INSERT INTO comments (id, post_id, parent_id, name, content, created_at, likes_count, status, moderation_reason, text_hash, text_fingerprint) VALUES
  (10, 1, NULL, NULL, 'Анонимный корневой комментарий', '2024-01-02 04:00:00', 2, 'visible', NULL, 'hash-visible', 'fingerprint-visible'),
  (11, 1, 10, 'Reader', 'Nested reply', '2024-01-02 04:01:00', 0, 'visible', NULL, 'hash-reply', 'fingerprint-reply'),
  (12, 1, NULL, '', 'Pending comment', '2024-01-02 04:02:00', 0, 'pending', 'score', 'hash-pending', 'fingerprint-pending'),
  (13, 2, NULL, 'Spammer', 'Rejected comment', '2024-01-02 04:03:00', 0, 'rejected', 'admin_rejected', 'hash-rejected', 'fingerprint-rejected');

INSERT INTO like_events (id, post_id, ip_hash, created_at) VALUES
  (20, 1, 'fixture-post-like-ip-hash', '2024-01-02 05:00:00');

INSERT INTO comment_like_events (id, comment_id, ip_hash, created_at) VALUES
  (30, 10, 'fixture-comment-like-ip-hash', '2024-01-02 05:01:00');

INSERT INTO comment_attempts (id, ip_hash, post_id, status, reason, content, text_hash, fingerprint, created_at) VALUES
  (40, 'fixture-attempt-ip-hash', 1, 'visible', NULL, 'Accepted attempt', 'attempt-visible', 'attempt-visible-fingerprint', '2024-01-02 04:00:00'),
  (41, 'fixture-attempt-ip-hash', 1, 'pending', 'score', 'Pending attempt', 'attempt-pending', 'attempt-pending-fingerprint', '2024-01-02 04:02:00'),
  (42, 'fixture-muted-ip-hash', NULL, 'muted', 'honeypot', 'Muted attempt', 'attempt-muted', 'attempt-muted-fingerprint', '2024-01-02 04:04:00');

INSERT INTO comment_mutes (id, ip_hash, reason, muted_until, mute_count, created_at) VALUES
  (50, 'fixture-muted-ip-hash', 'honeypot', '2024-01-03 04:04:00', 2, '2024-01-02 04:04:00');

INSERT INTO comment_challenge_uses (token_hash, post_id, used_count, first_used_at, last_used_at) VALUES
  ('fixture-challenge-hash', 1, 2, '2024-01-02 03:55:00', '2024-01-02 03:56:00');

INSERT INTO admin_sessions (id, token_hash, created_at, expires_at) VALUES
  (60, 'fixture-session-hash', '2024-01-02 02:00:00', '2024-01-02 14:00:00');

COMMIT;
