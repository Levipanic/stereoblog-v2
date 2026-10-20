# AGENTS.md

StereoDamage v2: a single-author blog. Go/Gin + SQLite API in `backend/`; Nuxt 4 SSR app in `frontend/`.

## Read before coding

- `doc/AGENTS.md` is the binding product, data-compatibility, security, and scope contract.
- Backend work: also read `doc/backend/AGENTS.md`. Frontend work: also read `doc/frontend/AGENTS.md`.
- Work on one `doc/BACKLOG.md` task at a time (IDs look like `V2-503`); do not start later tasks. Acceptance criteria and "Do not" sections are contractual.
- Topic contracts live in `doc/docs/`: `ARCHITECTURE.md`, `TESTING.md`, `DATA_COMPATIBILITY.md`, `SECURITY.md`, `CONTENT_FORMAT.md`, `DEPLOYMENT.md`.

## Commands (from repo root)

```sh
nvm use
make install-frontend   # npm ci; exact Node 24.20.0 / npm 11.19.0 enforced
make dev-backend        # Go API :8080; opens and migrates the DB at startup
make dev-frontend       # Nuxt :3000; proxies /api/v1 and /uploads to the API
make test               # default target: Go tests, then Nuxt typecheck + node:test
make check              # make test + go vet (there is no separate linter)
make build
make e2e                # builds Go + Nuxt, starts a disposable stack on ports 4010-4012
make audit AUDIT_ARGS='-db /abs/blog.db -uploads /abs/uploads'
```

- Run `nvm use` before any frontend command; Makefile targets reject mismatched Node/npm, direct npm scripts bypass that guard.
- `.env` is optional. The Makefile sources it with `set -a`, so it overrides exported values; use `ENV_FILE=/dev/null` to ignore it.
- Focused checks: `cd backend && go test ./internal/httpapi -run TestName -count=1`; `cd frontend && node --test --test-name-pattern 'name' test/feed.test.ts`; `npm --prefix frontend exec -- playwright test test/browser/feed.spec.ts --project=mobile`.
- First browser run needs Chromium: `npm --prefix frontend exec -- playwright install chromium`.

## Repo-specific gotchas

- Backend migrations live in `backend/internal/storage/sqlite/migrate.go` (there is no `backend/migrations/` despite older docs). Add ordered entries to the Go migration slice; never drop or reset v1 data.
- Startup migrates automatically and snapshots an existing DB before migrations in production mode. Only point it at a disposable copy; `make audit` is read-only.
- `frontend/test/browser/server.ts` seeds SQLite with raw `INSERT`s and starts real Go/Nuxt processes. A schema change can require updating it too.
- `make e2e` uses ports 4010-4012 and temporary DB/uploads, never `data/` or `uploads/`. `NUXT_E2E_DEV=1 make e2e` also exercises Nuxt's dev proxy.
- Go owns canonical API/data/auth/uploads/backup logic. Nitro is presentation plus thin routes such as `frontend/server/routes/post.html.get.ts`; do not add business logic there.
- `/admin/**` is `ssr: false`, and Tiptap loads only after login. Never link `/admin` from public UI or ship editor code in the public bundle.
- Canonical post content is StereoDamage `blocks_json`; never store editor HTML or ProseMirror JSON as the source of truth.
- Preserve v1 IDs, `/uploads/...` paths, legacy `/post.html?id=<id>` redirects, localStorage keys, and RU/EN UI.
- A local `old_repo/` reference may be present and is ignored. `data/`, `uploads/`, `.env*`, and backups are runtime/secret: never commit or overwrite the only production copy. `blog-backup-*.tar.gz` is ignored; still do not use `git add -A` blindly around runtime files.

## Verification

- Run the checks required by the active backlog task; `make check` is the broad default.
- User-visible text needs RU/EN. UI changes need mobile behavior, keyboard/focus basics, and both light/dark themes checked.
