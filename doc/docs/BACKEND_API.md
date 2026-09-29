# Backend API additions

All endpoints use `/api/v1` and the standard `{ "error": { "code", "message" } }` error envelope.

## Comment likes

`POST /comments/:id/likes` returns `{ "success": true, "comment_id": 10, "likes": 3 }`.
Only visible comments can be liked. Post/comment likes share the per-IP minute limiter;
per-item cooldown and event retention use existing `LIKE_*` settings. A 429 includes `Retry-After`.

## Admin sessions

- `POST /admin/login`, JSON `{ "secret": "..." }`: returns authenticated session metadata and `csrf_token`.
- `GET /admin/session`: returns `{ "authenticated": false }` or authenticated metadata and `csrf_token`.
- `POST /admin/logout`: requires session and `X-CSRF-Token`; revokes the session and clears the cookie.

The random session token is sent only in the `admin_session` HttpOnly, SameSite=Lax,
Path=/ cookie (Secure in production). SQLite stores only its salted SHA-256 hash,
compatible with v1 session storage. CSRF is session-bound HMAC, compatible with v1.
All admin responses are `Cache-Control: no-store`. Admin writes require the cookie,
CSRF header and non-cross-site origin signals; login also rejects cross-site requests.
Login accepts only bounded JSON and has the v1 15-minute per-IP rate window.
Existing `ADMIN_*` environment settings configure secret, lifetime, salts and limits.
Production configuration refuses the default secret. No schema migration is needed.

## Admin posts

- `GET /admin/posts?limit=10&cursor=...`: existing cursor feed contract.
- `GET /admin/posts/:id`: editable stored blocks and explicit `preview_media` (null means automatic).
- `POST /admin/posts`: JSON `{ "title", "blocks", "slug"?, "preview_media"? }`, returns 201 with saved post.
- `PUT /admin/posts/:id`: title and blocks required. Omitted preview preserves its previous value;
  explicit null resets it to automatic selection. Returns saved post.
- `DELETE /admin/posts/:id`: deletes post and cascaded comments/like events, retaining upload files.

Canonical validation regenerates plain fallback text. Unknown legacy blocks are returned only to
authenticated editing clients, and saves reject unsupported blocks instead of silently dropping them.
Title limit is 160 UTF-16 characters, matching v1. Slugs may be selected at creation;
automatic collisions receive numeric suffixes, explicit collisions return 409 `slug_conflict`.
Published slugs are immutable (409 `slug_immutable`); changing titles preserves links.
Creation uses existing `ADMIN_POST_RATE_LIMIT_*` settings. All writes require session + CSRF.

## Uploads

`POST /admin/uploads` accepts exactly one multipart `file`, with session + CSRF.
Returns 201 `{ "url", "original_name", "stored_name", "media_kind", "width"?, "height"? }`.
`UPLOAD_MAX_SIZE` bounds file size; multipart overhead is also bounded. Files stream to a hidden
temporary file, are inspected server-side, then atomically renamed to a random immutable filename.
Failures remove partial files. PNG/JPEG/GIF include dimensions; WebP/video/audio types are identified
using the already-installed MIME detector, not the browser's Content-Type. Known media extensions
must match detected contents. Other files are download-only attachments. Empty files and path-like
names are rejected. No image re-encoding or media optimization is performed.

New SVG and detected HTML uploads are rejected. Historical SVG remains available with
`Content-Security-Policy: sandbox; default-src 'none'; style-src 'unsafe-inline'`, which disables scripts
and external resources without modifying stored files. This may prevent external references in old SVGs.
`GET/HEAD /uploads/*` serves media with nosniff, range support and no directory listing.
Generic files are attachments; hidden paths and symlinks are rejected. Route `/uploads/*` to Go in
production to retain these protections (a future proxy implementation must reproduce the policy).

## Moderation

- `GET /admin/moderation`: `{ "pending_comments": [], "attempts": [], "mutes": [] }`.
  Each list is bounded by `COMMENT_ADMIN_LIST_LIMIT` (1–100), newest first with ID tie-break.
  Mutes include active records only; GET does not mutate moderation history.
- `POST /admin/comments/:id/approve`: makes the comment visible and clears its moderation reason.
- `POST /admin/comments/:id/reject`: hides the comment with `admin_rejected` reason.
- `DELETE /admin/comments/:id`: atomically deletes the entire reply subtree and its like events.
- `DELETE /admin/comment-mutes/:id`: removes the mute.

All routes require authentication and all writes require CSRF. Attempts/mutes expose only
the first 12 characters of the salted IP hash, matching v1. Missing records return 404.
Comment bodies and reasons remain text; future admin UI must render them as text.

## Portable backup

`POST /admin/backup` requires session + CSRF and downloads `application/zip` with:

```text
manifest.json
data/blog.db
uploads/**
```

Generation is limited to one request per minute for the owner and one active
generation/download at a time. A 429 includes `Retry-After`. The handler extends
its write deadline to 15 minutes; normal API requests retain their short timeout.
The database is a verified SQLite `VACUUM INTO` snapshot, never a raw live DB copy.
The snapshot passes the compatibility audit before archival. Missing or unsafe referenced
media fails the backup rather than reporting an incomplete archive as successful.
The audit also rejects hidden paths and symlinks, matching the media-serving policy.

Files stream into a private temporary ZIP on disk, then to the response. Media is
stored without recompression. Memory does not grow with file byte size (ZIP metadata
still scales with the number of files). Temporary resources are removed on success,
generation failure or disconnected download. `TMPDIR` controls temporary disk placement.
The manifest records format/schema versions, UTC creation time, post/comment/media
counts and the database SHA-256; ZIP CRCs cover each entry.

Only the SQLite snapshot and regular upload files are included. Hidden files/directories
(including `.env` and in-progress uploads) are excluded; symlinks and special files fail
the backup. The DB retains hashed session/moderation records, but configuration secrets
are not included. Treat the archive as private.

Consistency relies on the existing immutable-media policy: upload completes before its
URL can be saved in a post, and post deletion does not remove media. The DB snapshot is
taken first, then media copied; concurrent uploads can add harmless unreferenced files.
Do not externally overwrite/delete upload files during generation. No service shutdown
is needed. Restore instructions are in `DEPLOYMENT.md`.
