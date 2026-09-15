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
failed child command. `make e2e` is a reserved hook that fails until its backlog
task provides a real implementation.

The backend opens and migrates the configured SQLite database on startup. Never
point development commands at the only production copy. To audit an already
migrated disposable copy without changing it:

```sh
make audit AUDIT_ARGS='-db /absolute/path/to/blog.db -uploads /absolute/path/to/uploads'
```

The command exits non-zero for integrity, schema, relationship, content, or
missing-media failures. Posts, comments, authentication, and other product APIs
are still separate backlog tasks.
