# StereoDamage v2

StereoDamage v2 is a Go/Nuxt rewrite of the single-author personal blog at
<https://github.com/Levipanic/blog_proj>. It preserves the existing SQLite
data, local media, anonymous comments, and visual identity without carrying
over the v1 Express/vanilla-JS architecture.

Project contracts and the implementation backlog live in [`doc/`](doc/).
The local `old_repo/` directory is an ignored v1 and production-snapshot
reference; never commit its contents or modify the only production copy.

## Requirements

- Go 1.27.1 (older Go installations can use `GOTOOLCHAIN=auto`)
- Node.js 24.20.0
- npm 11.19.0
- GNU Make

## Development

```sh
nvm use
make install-frontend
```

Frontend commands stop immediately if Node/npm do not match the pinned versions.

Start the processes in two terminals so either can be stopped and restarted
independently:

```sh
make dev-backend
make dev-frontend
```

Both use safe development defaults. Optionally copy `.env.example` to `.env`
to override them. Values loaded by the dev commands use POSIX shell syntax and
take precedence over the parent environment. Set `ENV_FILE=/dev/null` to use
only exported variables. Production should use service-level environment
settings, absolute data paths, and a new `ADMIN_SECRET`.

Useful checks:

```sh
make test
make check
make build
```

These commands run without `.env` or production secrets and stop on the first
failed child command. For browser checks, install Chromium once with
`npm --prefix frontend exec -- playwright install chromium`, then run `make e2e`.
The tests build and start disposable Go/Nuxt instances on ports 4010–4012, using
temporary SQLite/uploads rather than your local data. Screenshots/traces are in
`frontend/test-results/`. `npm --prefix frontend run test:ssr` checks production SSR.
Use `NUXT_E2E_DEV=1 make e2e` to exercise Nuxt's development API/media proxy as well.
On minimal Linux installations Chromium may also need system libraries:
`npm --prefix frontend exec -- playwright install-deps chromium` (requires administrator access).

The backend opens and migrates the configured SQLite database on startup. Never
point development commands at the only production copy. To audit an already
migrated disposable copy without changing it:

```sh
make audit AUDIT_ARGS='-db /absolute/path/to/blog.db -uploads /absolute/path/to/uploads'
```

The command exits non-zero for integrity, schema, relationship, content, or
missing/unsafe-media failures.

## Backend API status

The Go API implements public posts/feed, post/comment likes, anonymous comments
and antispam, admin sessions/CSRF, post CRUD, uploads, moderation, and full backups.
See [`doc/docs/BACKEND_API.md`](doc/docs/BACKEND_API.md) for endpoint contracts and
[`doc/docs/DEPLOYMENT.md`](doc/docs/DEPLOYMENT.md#admin-portable-backup) for restore steps.
Backend Phase 2 is implemented and awaiting the owner's phase-gate review;
the remaining Nuxt UI and deployment/release work is tracked in the backlog.

## Public frontend status

The homepage renders the first feed page through SSR, including text/media previews,
reading time, likes and comment teasers from one API response. RU/EN, light/dark and
list/grid settings reuse v1 preferences. Grid is two columns on desktop and one on phones.
Cursor loading uses IntersectionObserver plus a keyboard-accessible load-more fallback,
explicit retry after errors, duplicate suppression and an end state.

Nuxt keeps the feed alive during same-session navigation, preserving loaded cards,
cursor and card state. Browser Back and the post's Back button use native router
scroll restoration. Reload starts from fresh SSR data; feed HTML is never saved in localStorage.
The minimal `/posts/:slug` page currently displays the title and Back button for this flow.
Full article rendering and discussion are still V2-401/V2-406. Phase 3 awaits owner acceptance.
