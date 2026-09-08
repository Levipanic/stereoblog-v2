# StereoDamage v2 — Agent Operating Contract

This file is the primary operating contract for coding agents working on StereoDamage v2.
Read it completely before changing code. Then read the relevant nested agent file under `doc/backend/` or `doc/frontend/` and the exact task in `doc/BACKLOG.md`.

Documentation paths in this pack are relative to `doc/` unless stated otherwise.

## 1. Project mission

StereoDamage v2 is a complete rewrite of the author's existing personal blog while preserving its identity, data, behavior, and low-cost self-hosted nature.

The goal is **not** to build a generic blogging platform, CMS, social network, or SaaS product.

The goal is:

> A single-author, long-form personal blog with images and media, anonymous low-friction comments, old-web / circa-2012 Twitter / indie-web character, excellent mobile UX, excellent authoring UX, and an extremely fast perceived experience.

The rewrite exists to make the old features dramatically better, remove accumulated implementation friction, and create a maintainable foundation without losing the personality of the original site.

## 2. Reference implementation

The v1 project is the behavioral and data-compatibility reference:

- Repository: `https://github.com/Levipanic/blog_proj`
- Treat current v1 `main` as the source of truth when a parity question is not explicitly answered by these docs.
- v1 is **not** an architecture template. Do not mechanically port its Express/vanilla-JS structure.
- Do not “clean up” or reinterpret existing behavior if doing so risks data loss or silently removes a feature.

Important v1 source files:

- `backend/db.js` — current SQLite schema and startup schema behavior.
- `backend/server.js` — API behavior, validation, antispam, admin sessions, uploads, likes, comments.
- `frontend/app.js` — public feed, post renderer, comments, reader state, TOC, audio behavior.
- `frontend/admin.js` / `admin.html` — current authoring and moderation behavior.
- `frontend/i18n.js` — current RU/EN UI strings and feature surface.
- `frontend/styles.css` — visual identity reference.

See `docs/V1_REFERENCE.md` for a compact map.

## 3. Non-negotiable product truths

These are project invariants. Do not change them as part of ordinary implementation work.

1. **Single author.** Only the owner can create, edit, delete, or publish posts.
2. **No reader accounts.** Readers never register or log in.
3. **Comments are anonymous-first.** A name is optional. Media in comments is out of scope.
4. **Long-form posts are the core content.** Do not introduce separate “note”, “tweet”, or short-post product types in v2.
5. **Media is first-class.** Images, GIFs, video, audio, files, captions, preview media, and spoilers from v1 must remain supported unless a backlog task explicitly changes behavior.
6. **Persistent audio is a product feature.** Audio may start inside a post and continue while navigating the Nuxt app until the reader stops/closes it.
7. **The public site must feel extremely fast.** Perceived speed is part of the design, not a benchmark-only concern.
8. **Mobile is first-class.** Reading, comments, sharing, and admin publishing must all be comfortable on a phone.
9. **The existing visual identity is preserved.** Polish and improve; do not replace it with generic modern app aesthetics.
10. **SQLite stays.** Do not introduce PostgreSQL or another database without an explicit owner decision.
11. **Media stays on the local filesystem.** Do not introduce S3/object storage as a requirement.
12. **Existing production data must survive.** A copy of the v1 production database and `uploads/` directory must be usable by v2 without manual data editing or loss.
13. **Old shared post links must keep working.** Legacy `/post.html?id=<id>` URLs must redirect to the canonical slug URL.
14. **No analytics, ads, recommendation algorithms, reader profiles, notification system, search, archive section, or taxonomy in the v2 launch scope.**
15. **RU/EN UI stays.** Post content itself is not translated automatically and does not need locale-specific copies.
16. **Smooth-scroll setting is intentionally removed.** Other existing reader features remain unless explicitly changed.
17. **Infinite scroll is desired.** It must be implemented without breaking back-navigation or scroll restoration.
18. **The admin surface is private.** The public UI must not advertise or link to `/admin` for unauthenticated readers.

## 4. Target stack

Use a conventional stack that coding agents and future maintainers can reason about quickly.

### Backend

- Go 1.27.x (pin an explicit supported patch version in project/tooling files at bootstrap).
- Gin 1.12.x unless a later stable compatible version is deliberately selected during bootstrap.
- `database/sql` with SQLite; no ORM.
- Prefer a CGO-free SQLite driver for easy deployment to the tiny VPS unless a concrete compatibility or correctness blocker is demonstrated.
- Explicit SQL migrations.
- Filesystem media storage.

### Frontend

- Nuxt 4.x + Vue 3 + TypeScript.
- Universal/SSR rendering for public pages.
- `/admin/**` may be client-heavy/client-rendered where useful.
- Native Nuxt routing and data fetching.
- Custom project CSS/components; no large generic component framework by default.
- Keep client dependencies small and justified.

Nuxt is intentionally chosen so first loads can arrive as useful HTML while navigation after hydration remains SPA-like. This is also what enables persistent audio across route changes.

## 5. Intended production topology

Keep deployment boring and cheap.

```text
Internet
  |
reverse proxy / TLS (existing nginx/caddy or equivalent)
  |
  +-- /api/*      -> Go/Gin on localhost
  +-- /uploads/*  -> Go or reverse-proxy static serving, whichever is safer/simpler
  +-- everything else -> Nuxt node server on localhost
```

Default recommendation for the tiny Linux VPS:

- systemd service for the Go binary;
- systemd service for the Nuxt/Nitro node server;
- existing reverse proxy if already present; otherwise a small conventional reverse proxy;
- Docker is optional, not a launch requirement.

Do not add orchestration, queues, Redis, Kubernetes, service discovery, or cloud-only infrastructure.

## 6. Repository shape

Target a simple monorepo:

```text
/
  backend/
    cmd/server/
    internal/
    migrations/
  frontend/
    app/
    public/
  doc/
    AGENTS.md
    BACKLOG.md
    AGENT_WORKFLOW.md
    backend/AGENTS.md
    frontend/AGENTS.md
    docs/
  data/          # runtime, gitignored except placeholder
  uploads/       # runtime, gitignored except placeholder
  scripts/       # only useful operational/dev scripts
```

Exact internal package folders can evolve, but keep boundaries obvious and shallow. Do not create abstractions only to satisfy a diagram.

## 7. Architecture principles

### 7.1 Simple boundaries, not enterprise ceremony

Backend code should generally flow:

```text
HTTP handler -> domain/service logic -> repository/SQL
```

This is a guideline, not an excuse for one interface per function. Introduce interfaces when they provide a real seam for testing or replacement.

Frontend code should generally flow:

```text
page -> focused components/composables -> typed API client / local state
```

Do not create a global state store for data that naturally belongs to one page/component. Persistent cross-route audio and a small amount of app-level reader state are legitimate shared state.

### 7.2 API-first but same-origin in production

Public and admin data comes from Gin through a versioned API, preferably `/api/v1/...`.

The browser should use same-origin URLs in production. Avoid CORS complexity unless local development genuinely needs it; proxy API calls in Nuxt dev instead.

### 7.3 Public SSR, app-like navigation

- Feed and post routes must render useful HTML on first request.
- SEO/share metadata must be correct without client JS.
- After hydration, internal navigation should use Nuxt routing, preserve app state, and avoid full page reloads.
- `/admin` can prioritize interaction over SEO and can be client-only if that simplifies the editor.

### 7.4 Content ownership

The canonical content format belongs to StereoDamage, not to a rich-text editor library.

Never store arbitrary editor-generated HTML as canonical post content.
Never make Tiptap/ProseMirror (or another editor) JSON the only source of truth.

See `docs/CONTENT_FORMAT.md`.

## 8. Data compatibility is a release gate

The production v1 SQLite database is precious user data.

Rules:

- Never drop a v1 table automatically.
- Never delete or rewrite existing rows merely because a schema check fails.
- Migrations must be explicit, ordered, additive where practical, idempotent at the migration-runner level, and tested against a real production backup copy.
- Preserve v1 post IDs, comments, parent relationships, likes, timestamps, moderation records, and media paths.
- Preserve `/uploads/...` references.
- Before the first schema-changing migration on an unknown existing database, create a consistent pre-migration database backup or require/prove one exists according to the migration task.
- The owner will repeatedly test v2 on a copy of the current production DB. Make that workflow easy.

A successful migration means more than “the process starts”: representative old posts, comments, likes, media, admin data, and timestamps must render and behave correctly.

See `docs/DATA_COMPATIBILITY.md`.

## 9. Visual design guardrails

The desired vibe is approximately:

> old Twitter around 2012 + indie web + personal site + slightly secret-forum feeling, with modern polish and excellent mobile behavior.

Modernization must be conservative.

Do:

- preserve the recognizable StereoDamage palette, typography character, density, and web-like feel;
- improve spacing, hierarchy, touch targets, responsive media, focus states, reader typography, and motion restraint;
- keep the site content-forward;
- prefer thin separators and clear structure over nested decorative containers where appropriate;
- keep controls compact and obvious.

Do **not** autonomously introduce:

- generic SaaS/dashboard design;
- glassmorphism/backdrop-blur as a theme;
- giant landing-page hero sections;
- excessive rounded cards;
- gradient-heavy redesigns;
- animation-heavy transitions;
- a Material/Vuetify/Tailwind-component-library look;
- visual changes whose main justification is “more modern”.

If a task requires a meaningful visual departure from v1 and it is not specified in `docs/UX_SPEC.md`, stop and ask the owner instead of inventing a new identity.

## 10. Performance rules

“Fast” means the reader should rarely perceive waiting.

Every dependency and client-side feature must justify its weight.

Prefer:

- SSR HTML for public routes;
- small components and native browser APIs;
- lazy hydration/interaction where useful;
- responsive/lazy-loaded offscreen media;
- cursor-based feed pagination;
- API responses shaped for the screen to avoid N+1 browser requests;
- request cancellation/deduplication where navigation can race;
- stable layout to avoid content jumps;
- route state/scroll restoration for infinite feed navigation.

Avoid:

- fetching one comment-preview request per feed card;
- shipping editor code to public routes;
- globally loading heavy media/editor libraries when they are only needed on one route;
- large UI libraries for a small custom interface;
- decorative JS animation libraries without a concrete UX need.

Performance budgets are documented in `docs/UX_SPEC.md` and should be measured near release, not guessed.

## 11. Security rules

The admin plane is critical because a compromise gives write access to the entire blog.

At minimum preserve or improve v1's principles:

- admin secret/password comes from environment, never source control;
- constant-time secret comparison where relevant;
- server-issued random session tokens;
- store only hashed session tokens in SQLite;
- HttpOnly cookie;
- Secure cookie in production;
- SameSite protection;
- CSRF protection for admin writes;
- login rate limiting;
- admin API authorization on every write, regardless of whether `/admin` is hidden;
- uploads validated server-side by size/type, never trusted from the browser;
- comments rendered as text/sanitized structured content, never raw untrusted HTML;
- IP addresses are not exposed to admin UI as raw values;
- antispam must fail safely.

“Hiding `/admin`” is not authentication. Keep the route unadvertised, but secure it as though its URL is public knowledge.

See `docs/SECURITY.md`.

## 12. Agent working procedure

For every backlog task:

1. Read this file.
2. Read the relevant nested `doc/backend/AGENTS.md` and/or `doc/frontend/AGENTS.md`.
3. Read the exact task in `doc/BACKLOG.md`, including dependencies, acceptance criteria, tests, and “do not”.
4. Read any docs explicitly linked from that task.
5. If parity is involved, inspect the relevant v1 source before coding. Do not rely on memory.
6. Inspect current v2 code and tests before editing.
7. Implement **only the requested task plus the smallest supporting changes necessary**.
8. Add/update tests that prove the acceptance criteria.
9. Run the relevant checks locally.
10. Summarize:
   - what changed;
   - files changed;
   - tests/checks run and their result;
   - any migration or operational impact;
   - any intentional deviation from the task (should be rare).
11. Do not mark a backlog task complete if acceptance criteria are not met.

## 13. No-overengineering rule

The owner is intentionally building a low-cost personal site.

Do not add or design for hypothetical needs such as:

- multiple authors;
- organizations/roles/permissions matrices;
- millions of users;
- distributed deployment;
- event buses;
- generic plugin systems;
- storage-provider abstraction before there is a second provider;
- database abstraction before there is a second database;
- CQRS, DDD ceremony, microservices;
- elaborate design-token platforms;
- generic CMS capabilities unrelated to this blog.

If a simple local solution satisfies the current product and can be changed later without data loss, prefer it.

## 14. When the agent should ask instead of decide

Do not interrupt for ordinary implementation choices.

Ask the owner **before** making a change that:

- changes a product invariant in section 3;
- deletes or semantically rewrites existing production data;
- changes the visual identity substantially;
- adds paid infrastructure or an external service;
- introduces reader accounts or additional authors;
- changes the canonical post model in a way that makes old content unreadable;
- removes an existing v1 feature not explicitly approved for removal;
- introduces a large runtime dependency or UI framework;
- changes deployment topology substantially;
- weakens admin security or antispam behavior;
- requires a trade-off where either choice materially affects future product behavior and the backlog does not resolve it.

For smaller engineering decisions, choose the simplest conventional option and document it in the task summary.

## 15. Testing philosophy

Tests exist to prevent regressions, especially during an agent-driven rewrite. The goal is not arbitrary coverage percentage.

Prioritize:

- DB migration/compatibility tests;
- content parsing/rendering fixtures from v1;
- admin auth and CSRF;
- comment validation/antispam edge cases;
- CRUD and slug/legacy URL behavior;
- backup correctness;
- editor adapter round-trips;
- critical browser flows: feed -> post -> back, post -> comments anchor, comment submit, login -> edit -> preview -> publish, persistent audio navigation;
- mobile viewport E2E smoke tests.

See `docs/TESTING.md`.

## 16. Definition of done for any task

A task is done only when:

- its acceptance criteria are satisfied;
- relevant tests pass;
- no unrelated feature was removed;
- no new warnings/errors are introduced in normal browser/server logs;
- any new config is documented in `.env.example` or deployment docs;
- any schema change has a migration and compatibility test;
- any user-visible text is wired through RU/EN i18n unless explicitly exempted;
- mobile behavior was considered for public/admin UI changes;
- accessibility basics are maintained: semantic controls, keyboard access where applicable, labels, focus visibility, alt/caption behavior;
- the change remains compatible with the low-resource VPS goal.

## 17. Scope discipline

`doc/BACKLOG.md` is the implementation order and scope contract.

Do not silently “finish future tasks early” if that expands the current change significantly. Small foundations that are directly necessary are fine; large future features should remain future tasks.

If you discover a bug or missing requirement outside the active task:

- note it in the task summary;
- add a proposed backlog item only if the owner asks;
- do not opportunistically refactor large unrelated areas.

The desired workflow is deliberate, task-by-task progress that the owner can inspect and test after every step.
