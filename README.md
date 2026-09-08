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
npm --prefix frontend install
cp .env.example .env
make dev
```

The development defaults work without `.env`; copying the example makes them
easy to change. Values loaded by `make` use POSIX shell syntax and take
precedence over the parent environment. Set `ENV_FILE=/dev/null` to use only
exported variables. Production should use service-level environment settings,
absolute data paths, and a new `ADMIN_SECRET`.

Useful checks:

```sh
make test-backend
make check-frontend
make build
```

This bootstrap intentionally contains no posts, comments, authentication, or
database implementation. Those are separate backlog tasks.
