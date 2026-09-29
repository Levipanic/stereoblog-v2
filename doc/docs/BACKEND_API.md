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
