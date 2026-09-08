# Backend agent rules — Go/Gin/SQLite

This file extends `doc/AGENTS.md` for work under `backend/`.

## 1. Backend mission

The Go backend exists to make v1 behavior safer, clearer, faster, and easier to evolve while preserving the owner's production SQLite/media data.

It is not a microservice platform. It is one small API process for one personal site.

## 2. Required reading by topic

Before DB/migrations:

- `docs/V1_REFERENCE.md`
- `docs/DATA_COMPATIBILITY.md`

Before posts/editor model:

- `docs/CONTENT_FORMAT.md`
- v1 `backend/server.js` block normalization/validation

Before auth/security/uploads/comments:

- `docs/SECURITY.md`
- relevant v1 `backend/server.js` implementation

Before backup/deploy:

- `docs/DEPLOYMENT.md`

## 3. Go style

- Use idiomatic Go and standard library first.
- Keep functions/packages focused but avoid interface/abstraction proliferation.
- Errors should carry useful context internally; HTTP responses should be safe and stable.
- Pass `context.Context` through DB/service operations where appropriate.
- Use `time.Time` internally; define one deliberate conversion policy for v1 SQLite UTC strings.
- Parameterize all SQL.
- Prefer explicit structs over `map[string]any` for API contracts.
- JSON field naming should be stable and documented through types/tests.
- Run `gofmt` and normal Go test/vet tooling.

## 4. Gin usage

Gin is the HTTP layer, not the domain layer.

Handlers should mainly:

1. parse/validate transport input;
2. call a focused service/repository operation;
3. map result/error to HTTP response.

Do not put 500 lines of antispam or SQL directly inside a route closure.

Use middleware for genuinely cross-cutting HTTP concerns such as:

- request IDs/logging if added;
- security headers when app-owned;
- trusted client IP derivation;
- rate limiting where appropriate;
- admin session loading;
- recovery/error handling.

Do not create middleware just to hide ordinary function calls.

## 5. SQLite rules

### Driver

Prefer a maintained CGO-free `database/sql` SQLite driver for deployment simplicity unless a task identifies a real blocker. Do not add an ORM.

### Connections

SQLite is not a network DB.

- keep pool settings conservative;
- `foreign_keys=ON` is mandatory;
- configure busy timeout;
- consider WAL after compatibility testing;
- do not set a huge open-connection count because Go can create goroutines.

### Queries

- Keep SQL near the owning repository/package.
- Use transactions for multi-step state changes that must be atomic (e.g. like event + count increment, comment insertion + attempt record where required).
- Preserve cascade semantics and validate parent comment belongs to same post.

## 6. Migrations: zero tolerance for destructive reset

Never port v1's drop-on-schema-mismatch code.

Migration rules:

- ordered migration versions;
- final schema produced through same migrations for fresh and v1 DB;
- safe detection of v1 schema;
- additive by default;
- abort with actionable error on unsupported/corrupt schema;
- never automatically delete the user's content to recover;
- tests from v1 fixture;
- pre-migration backup behavior per backlog task.

Any `DROP TABLE`, mass `DELETE`, or data rewrite in a migration requires explicit task text and owner approval.

## 7. Content model

Canonical `blocks_json` stays StereoDamage-owned.

Backend must support legacy blocks exactly enough to render/edit old posts.

For rich v2 text blocks:

- validate supported inline content/marks;
- produce/store plain `text` fallback;
- never trust arbitrary HTML;
- retain old media block shape/paths;
- do not bulk-convert old rows.

Avoid a clever generic AST. Define the small schema the product actually supports.

## 8. API conventions

New API namespace: `/api/v1`.

Use consistent JSON errors, e.g. a small shape such as:

```json
{
  "error": {
    "code": "post_not_found",
    "message": "Post not found."
  }
}
```

Exact shape should be established in bootstrap/API task and then kept stable.

For validation errors, include safe field/detail information useful to admin UI.

Do not leak SQL/errors/paths to public responses.

## 9. Pagination

Feed uses cursor pagination, not offset pages.

Cursor must be based on stable reverse chronology and handle equal timestamps using ID tie-break.

Response should make infinite scrolling simple and should contain feed comment previews/counts without client-side request fan-out.

## 10. Slugs

- ID remains permanent identity for data relationships/progress/legacy links.
- Slug is public canonical route identity.
- Unique collision handling deterministic.
- Existing titles can be Cyrillic.
- New title edits do not automatically rotate an already-published slug.
- API can resolve legacy ID -> slug.

Do not make slug the DB primary key.

## 11. Likes

Preserve v1 semantics unless the task explicitly adjusts details:

- hashed client IP;
- cooldown;
- rate limit;
- atomic event/count update;
- no reader account.

Do not expose IP hashes publicly.

## 12. Comments/antispam

Port behavior methodically; do not “simplify while rewriting”.

Keep separate concepts understandable:

- challenge generation/verification;
- text normalization/fingerprint;
- frequency/history stats;
- moderation scoring;
- mute decision;
- attempt recording;
- comment persistence;
- admin moderation.

A suspicious human comment should be able to become `pending`; antispam is not only accept/reject.

Use tests copied from observed v1 edge cases where practical.

## 13. Admin auth

Maintain a simple single-secret model, but implement it rigorously.

- env secret required in production;
- no default secret allowed in production;
- login rate limit;
- random session token;
- hashed DB storage;
- expiry;
- secure HttpOnly cookie;
- CSRF for writes;
- logout invalidation;
- do not expose raw session to frontend JS.

Do not add roles/users tables for hypothetical authors.

## 14. Uploads

Preserve existing `/uploads/...` URL compatibility.

For new uploads:

- generated safe names;
- bounded size;
- server-side type validation;
- no traversal;
- stream rather than buffering whole large files where possible;
- cleanup partial/failed files;
- return structured media info usable by editor adapter.

Do not require S3.

If adding thumbnails/optimization:

- keep original;
- bound resource usage;
- make derivative generation failure non-destructive;
- do not rewrite old media references unnecessarily.

## 15. Backup

Full backup must be streamable and consistent.

Do not `os.ReadFile` the whole database/uploads archive into RAM.

Use safe temporary files/streaming zip and SQLite backup/snapshot logic.

Backup endpoint must be admin-only and must not include secrets.

Add restore/audit documentation/tests as part of the task.

## 16. RSS

Keep it boring:

- standard valid XML;
- canonical URLs;
- derived automatically from posts;
- no extra service.

## 17. Logging

Log useful operational context, not personal secrets.

Good:

- startup/migration version;
- route/status/latency at reasonable verbosity;
- error category;
- backup start/finish metadata without archive secrets.

Bad:

- admin secret;
- raw session/CSRF tokens;
- raw user IP unless explicitly needed and privacy-reviewed;
- full comment body in normal request logs.

## 18. Backend definition of done additions

For backend tasks:

- Go tests pass;
- migration tests if schema touched;
- `go vet`/lint configured checks pass;
- error behavior tested;
- env changes reflected in `.env.example`;
- no production data path is hardcoded;
- no resource leak (files/rows/bodies closed);
- behavior behind reverse proxy considered for IP-sensitive code.
