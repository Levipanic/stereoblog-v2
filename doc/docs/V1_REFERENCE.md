# v1 reference map

This is a compact reference for agents. When exact behavior matters, inspect the live v1 source at `https://github.com/Levipanic/blog_proj` instead of relying only on this summary.

## Current v1 architecture

- Node.js + Express.
- `better-sqlite3`.
- Vanilla HTML/CSS/JS MPA.
- Local `data/blog.db`.
- Local `uploads/` directory.
- Public feed, standalone post page, admin page.
- RU/EN UI via custom frontend i18n.

## Current SQLite schema

From v1 `backend/db.js`.

### `posts`

- `id INTEGER PRIMARY KEY AUTOINCREMENT`
- `title TEXT NOT NULL`
- `blocks_json TEXT NOT NULL`
- `likes_count INTEGER NOT NULL DEFAULT 0`
- `created_at TEXT NOT NULL DEFAULT (datetime('now'))`
- `preview_media TEXT DEFAULT NULL`

### `comments`

- `id INTEGER PRIMARY KEY AUTOINCREMENT`
- `post_id INTEGER NOT NULL`
- `parent_id INTEGER`
- `name TEXT`
- `content TEXT NOT NULL`
- `created_at TEXT NOT NULL DEFAULT (datetime('now'))`
- `likes_count INTEGER NOT NULL DEFAULT 0`
- `status TEXT NOT NULL DEFAULT 'visible'`
- `moderation_reason TEXT`
- `text_hash TEXT`
- `text_fingerprint TEXT`
- FK post -> posts, cascade delete
- FK parent -> comments, cascade delete

### `like_events`

Tracks post like cooldown by hashed IP.

- `id`
- `post_id`
- `ip_hash`
- `created_at`

### `comment_like_events`

- `id`
- `comment_id`
- `ip_hash`
- `created_at`

### `comment_attempts`

Antispam audit/state:

- `id`
- `ip_hash`
- `post_id`
- `status`
- `reason`
- `content`
- `text_hash`
- `fingerprint`
- `created_at`

### `comment_mutes`

- `id`
- `ip_hash UNIQUE`
- `reason`
- `muted_until`
- `mute_count`
- `created_at`

### `comment_challenge_uses`

- `token_hash PRIMARY KEY`
- `post_id`
- `used_count`
- `first_used_at`
- `last_used_at`

### `admin_sessions`

- `id`
- `token_hash UNIQUE`
- `created_at`
- `expires_at`

## Important warning about v1 startup behavior

v1 contains an old schema-compatibility branch that can drop `like_events`, `comments`, and `posts` if the `posts` table lacks required historical columns. **Do not port this behavior.** v2 migrations must never “fix” unknown production schema by dropping data.

## v1 content blocks

Canonical post content is a JSON array in `posts.blocks_json`.

Supported shapes include:

```json
{ "type": "paragraph", "text": "..." }
```

```json
{ "type": "heading", "level": 1, "text": "..." }
```

```json
{ "type": "quote", "text": "..." }
```

```json
{ "type": "divider" }
```

```json
{
  "type": "media",
  "mediaKind": "image|gif|video|audio|file",
  "src": "/uploads/...",
  "spoiler": false,
  "name": "optional",
  "alt": "optional",
  "caption": "optional"
}
```

Server normalization ignores unknown block shapes. Preserve all readable legacy blocks in v2.

## v1 post behavior

- Title is required.
- Up to 60 blocks by default.
- Text length and media metadata have configurable limits.
- Preview media may be explicit or derived from blocks.
- Reading time is derived from block text.
- Feed returns page-based post summaries.
- Standalone post returns full blocks.
- Posts have likes with cooldown/rate limiting based on hashed IP.

## v1 comments behavior

- Reader name optional; UI fallback is anonymous/anon.
- Plain-text comments only.
- Parent-child replies.
- Comment likes.
- Public API only exposes visible comments.
- Antispam has challenge token + dynamic honeypot, burst/cooldown/duplicate checks, text heuristics/fingerprints, moderation score, pending/rejected states, mute records, and attempt logging.
- Suspicious comments can be accepted as `pending` rather than silently destroyed.
- Admin can inspect antispam state, approve/reject pending comments, delete comments, and unmute.

Do not simplify antispam during the rewrite until parity is proven.

## v1 admin auth behavior

- Secret from `ADMIN_SECRET`.
- Login rate limit.
- Random server-created session token.
- Hashed token stored in `admin_sessions`.
- HttpOnly cookie; Secure in production; SameSite=Lax.
- Admin write requests require a CSRF token derived from the session.
- Session cleanup by expiry.

This is a good baseline to preserve/improve.

## v1 uploads

- Stored under local `uploads/`.
- Randomized filename containing timestamp/random bytes + safe extension.
- Default max size 25 MB.
- Media kind inferred from extension.
- Existing posts reference `/uploads/<file>` paths.

v2 must preserve those paths.

## v1 reader/localStorage keys

Preserve these keys or migrate them deliberately so users on the same domain do not unnecessarily lose preferences/progress:

- `stereoDamageLanguage`
- `stereoDamageTheme`
- `stereoDamageFeedView`
- `stereoDamageReadingProgress`
- `stereodamage_audio_volume_v1`
- `stereodamage_audio_session_v1` (session continuity can be redesigned; preserve only if sensible)

Intentionally obsolete:

- `stereoDamageSmoothScroll` — v2 removes the smooth-scroll setting.

Current reading-progress entries are keyed by post ID and contain approximately:

- `opened_at`
- `updated_at`
- `progress`
- `scroll_y`
- `completed`

Preserving ID-based progress is useful even after canonical URLs become slug-based.

## v1 public UX/features to preserve or improve

- chronological feed;
- list/grid feed modes;
- title + preview text + preview media;
- a small number of comment previews beneath feed posts;
- post/comment likes;
- reading time;
- read/started/completed local reader state;
- continue-reading position;
- post TOC when headings justify it;
- light/dark theme;
- RU/EN UI;
- media/spoilers;
- custom audio/waveform/mini-player concept;
- comments/replies;
- admin moderation.

## Known v1 UX/technical pain points to improve

- Public frontend is a large imperative JS file.
- Admin frontend is a large imperative JS file.
- Feed may issue comment-preview requests per visible post.
- Legacy URL is `/post.html?id=<id>`.
- Share metadata is not first-class server-rendered post metadata.
- Multipage transitions make persistent audio brittle.
- Admin upload and block insertion are separate steps.
- Admin exposes block mechanics/raw JSON too directly for normal authoring.
- Mobile admin authoring is not a first-class flow.
- Reader/post layout can contain too many “panel/card within panel” surfaces for long reading.
