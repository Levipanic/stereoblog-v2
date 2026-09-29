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
