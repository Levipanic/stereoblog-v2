# Architecture — StereoDamage v2

## 1. High-level topology

```text
Browser
  |
  | HTTPS same origin
  v
Reverse proxy
  |-----------------------------|
  |                             |
  | /api/v1/*                   | everything else
  v                             v
Go / Gin API                Nuxt 4 / Nitro
  |                             |
  |                             | SSR fetches internal API
  v                             |
SQLite + uploads/ <-------------|
```

The browser should not need to know that there are two application processes.

## 2. Responsibilities

### Gin owns

- canonical API;
- SQLite access/migrations;
- posts/comments/likes data rules;
- slug resolution data;
- admin auth/session/CSRF;
- comment challenge/antispam/moderation;
- uploads and media metadata validation;
- backup generation;
- RSS may live here or Nuxt, but choose one clear owner during its task;
- health endpoints.

### Nuxt owns

- public feed and article rendering;
- public metadata/SEO presentation using backend data;
- infinite-scroll UX/state restoration;
- reader local state;
- public comments UI;
- sharing UI;
- image viewer;
- persistent client audio state/player;
- private admin UI/editor;
- RU/EN UI strings.

Nuxt server routes must not become a second business-logic backend. Thin proxy/adaptation is acceptable where Nuxt SSR/runtime needs it, but canonical data rules belong in Go.

## 3. Backend package shape

Keep it understandable. Example, not dogma:

```text
backend/
  cmd/server/main.go
  internal/
    config/
    http/
      router.go
      middleware/
      handlers/
    posts/
    comments/
    auth/
    antispam/
    media/
    backup/
    storage/
      sqlite/
  migrations/
```

Avoid a giant `main.go`/`server.go`, but also avoid dozens of one-function packages.

Domain packages may contain service + repository code together until separation is actually useful.

## 4. Frontend shape

Nuxt 4 conventional layout, e.g.:

```text
frontend/
  app/
    pages/
      index.vue
      posts/[slug].vue
      admin/...
    components/
      feed/
      post/
      comments/
      media/
      audio/
      admin/
    composables/
    utils/
    types/
    assets/css/
  public/
```

Do not recreate v1 as one giant composable or one giant page component.

## 5. API design

Version new routes under `/api/v1`.

Illustrative resource surface:

```text
GET    /api/v1/posts?cursor=...&limit=...
GET    /api/v1/posts/:slug
GET    /api/v1/posts/by-id/:id              # internal/legacy resolution if useful
POST   /api/v1/posts/:id/likes

GET    /api/v1/posts/:id/comments
GET    /api/v1/posts/:id/comments/challenge
POST   /api/v1/posts/:id/comments
POST   /api/v1/comments/:id/likes

POST   /api/v1/admin/login
POST   /api/v1/admin/logout
GET    /api/v1/admin/session
POST   /api/v1/admin/posts
PUT    /api/v1/admin/posts/:id
DELETE /api/v1/admin/posts/:id
POST   /api/v1/admin/uploads
GET    /api/v1/admin/moderation
...
GET    /api/v1/admin/backup
```

Exact paths can be refined during API tasks. Keep semantics consistent.

Do not preserve old v1 API paths merely for architectural nostalgia. Old **public URLs** must remain compatible; old internal API URLs only need compatibility if a backlog task proves a real external consumer.

## 6. Feed response design

Avoid v1's browser fan-out for comment previews.

Feed item should contain enough data for the card:

- id;
- slug;
- title;
- created_at;
- reading_minutes;
- preview text;
- preview media;
- likes count;
- visible comment count;
- small comment preview array;
- any stable fields required by reader progress mapping.

Use cursor pagination based on stable reverse chronology, e.g. `(created_at, id)` cursor. Encode cursor opaquely if convenient.

Do not include full `blocks_json` for every feed item.

## 7. Slugs

Add slug support without changing IDs.

Properties:

- unique;
- deterministic for migrated posts;
- generated for new posts;
- editable before first publish;
- not automatically changed when title changes later;
- resolver supports ID -> canonical slug for legacy redirects.

If a manual future slug change is supported, preserve old slugs through redirects rather than breaking links. This is post-launch unless explicitly included.

## 8. Rendering and caching

Public routes use Nuxt universal rendering by default.

Be conservative with caching because posts/comments/likes can change.

Possible strategy:

- article body can be efficiently fetched/SSR-rendered;
- comments/like counts can hydrate/revalidate separately if needed;
- do not introduce complex cache invalidation for launch;
- reverse proxy static caching is appropriate for hashed frontend assets and immutable media derivatives, not blindly for authenticated/admin/API responses.

Nuxt `/admin/**` may use `ssr: false` route rules if that materially simplifies editor libraries and avoids shipping them to server/public bundles.

## 9. SQLite

Use `database/sql`, parameterized SQL, explicit transactions.

Recommended runtime pragmas should be deliberate and tested, e.g.:

- `foreign_keys=ON` mandatory;
- consider WAL for concurrency/backup ergonomics, but validate behavior on the deployment filesystem;
- set a sensible busy timeout;
- avoid opening unbounded connections to SQLite.

Do not assume “Go is concurrent” means SQLite should have a huge connection pool.

## 10. Migrations

Implement a migration table and ordered SQL/Go migrations.

Rules:

- detect existing v1 schema safely;
- no destructive reset;
- additive changes first;
- migration version stored in DB;
- migration tests start from a v1 fixture;
- migration failures abort startup clearly;
- automatic pre-migration backup behavior is desirable for production safety and must be coordinated with `docs/DATA_COMPATIBILITY.md`.

## 11. Media

Canonical media files remain under local filesystem.

Keep old `/uploads/<filename>` URLs valid.

Do not rename/move old files during launch migration.

Future/launch-safe additions may include:

- stored media metadata table for new uploads;
- width/height for images;
- generated thumbnails/AVIF/WebP derivatives;
- EXIF stripping/orientation normalization where implemented safely.

But original upload must not be silently destroyed until backup/derivative behavior is proven.

## 12. Backup

Backup endpoint/service should create a portable archive with at least:

```text
manifest.json
data/blog.db
uploads/**
```

Use a consistent SQLite snapshot/backup mechanism rather than blindly copying a live DB file while it is being written.

Do not include `.env`/admin secret in browser-download backup by default.

Manifest can contain non-secret metadata such as:

- backup format version;
- created_at;
- app version/commit if available;
- database filename/schema version;
- media file count;
- optional checksums.

## 13. RSS

RSS is intentionally small/simple.

- generated automatically from published posts;
- canonical slug links;
- valid dates/title/description;
- no manual author maintenance;
- no external service.

Choose whether Gin or Nuxt owns generation based on which yields simpler canonical URLs/SSR deployment. Do not duplicate feed-generation logic.

## 14. Observability

Keep operations simple:

- structured-enough server logs to debug requests/errors;
- no analytics tracking of readers;
- health endpoint for process monitoring;
- startup logs show DB path, migration result/version, uploads path, listening address, production mode without printing secrets;
- systemd/journald is sufficient initially.

Do not add an external logging/APM service by default.
