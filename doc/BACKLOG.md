# StereoDamage v2 — implementation backlog

This is the execution order for coding agents.

## How to use this backlog

- Work on **one task at a time**.
- Read `doc/AGENTS.md`, relevant agent files under `doc/backend/` or `doc/frontend/`, and linked docs before implementation. Documentation paths below are relative to `doc/`.
- Do not begin a task whose dependencies are unfinished.
- Acceptance criteria are contractual.
- “Do not” sections exist specifically to stop agent scope creep.
- The owner should test/review at each phase gate before moving on.

Task status notation:

- `[ ]` TODO
- `[-]` in progress
- `[x]` accepted by owner

## Roadmap at a glance

| Phase | Purpose | Gate before continuing |
|---|---|---|
| 0 | Clean Go/Nuxt repo + dev workflow | Both processes/builds/SSR work and project still feels intentionally small |
| 1 | SQLite/v1 migration safety | Real production backup copy migrates/audits with no unexplained loss |
| 2 | Go API feature parity | Posts/comments/likes/admin/uploads/moderation/backup work through tested API |
| 3 | Public feed | SSR feed + grid + infinite scroll + back restoration feel good |
| 4 | Reading/media/discussion/sharing | A shared long read is clearly better than v1 on phone/desktop |
| 5 | Private admin/editor | Owner can comfortably publish rich media posts from phone and desktop |
| 6 | Lightweight polish | RSS/media/a11y improvements add value without heaviness |
| 7 | Hardening/deploy | Production-data rehearsal + release checklist pass before cutover |

The detailed order inside phases matters. In particular, do not skip Phase 1.

---

# Phase 0 — clean v2 foundation

Goal: create a boring, understandable monorepo that can run Go + Nuxt locally and later on a tiny VPS. No feature porting yet.

## [x] V2-001 — Bootstrap the v2 monorepo

**Depends on:** none

**Read:** `AGENTS.md`, `docs/ARCHITECTURE.md`

### Goal

Create the actual v2 project skeleton without copying v1 runtime architecture.

### Scope

- Initialize `backend/` Go module using current agreed stable Go 1.27.x.
- Add Gin at current agreed 1.12.x/stable version.
- Initialize `frontend/` as Nuxt 4 + TypeScript.
- Preserve `doc/AGENTS.md`, `doc/BACKLOG.md`, docs, and nested agent files.
- Add sensible `.gitignore` for Node/Go builds, `.env`, SQLite runtime DBs, backups, and uploads while keeping placeholders where needed.
- Add root `Makefile` or similarly tiny command surface for common actions.
- Add a root README describing v2 mission, local startup, and v1 reference link.
- Pin Node/package manager expectations in a conventional way.

### Acceptance criteria

- `backend` compiles.
- `frontend` installs and builds the default minimal app.
- Root commands exist for at least: dev/backend test/frontend check/build.
- No v1 `server.js` or giant vanilla frontend is copied as new implementation.
- Runtime `data/` and `uploads/` contents cannot be accidentally committed.

### Tests/checks

- Go build/test.
- Nuxt typecheck/build.
- Verify ignored runtime files with git status.

### Do not

- Do not implement posts/comments/auth yet.
- Do not install a UI component framework.
- Do not add Docker yet.
- Do not create speculative abstractions.

---

## [x] V2-002 — Unified runtime configuration and `.env.example`

**Depends on:** V2-001

**Read:** `docs/ARCHITECTURE.md`, `docs/SECURITY.md`, v1 `.env.example`

### Goal

Define clear config for two local processes while preserving meaningful v1 tunables.

### Scope

Backend config should cover at least:

- listen address/port;
- environment/production mode;
- DB path;
- uploads path;
- trusted proxy configuration;
- admin secret;
- session lifetime/salts as appropriate;
- upload max size;
- v1 comment/like rate-limit and antispam tunables that will later be ported.

Frontend/Nuxt runtime config should cover:

- internal SSR API base URL;
- browser API base (normally same-origin `/api/v1`);
- public site URL/name/handle where metadata needs it.

Create a safe root `.env.example` or clearly documented split env files.

### Acceptance criteria

- Config is parsed/validated on startup.
- Production mode refuses an obviously default admin secret once auth exists; for now establish validation plumbing.
- Startup never logs secret values.
- Runtime paths are configurable and default to developer-friendly local paths.
- v1 tunables that matter are not lost without an explicit later decision.

### Tests/checks

- Config unit tests for defaults/invalid numeric values/booleans.
- Build with example config.

### Do not

- Do not require a cloud secret manager.
- Do not duplicate the same config in many modules.

---

## [x] V2-003 — Minimal Gin server, health, error contract

**Depends on:** V2-001, V2-002

**Read:** `backend/AGENTS.md`, `docs/ARCHITECTURE.md`, `docs/SECURITY.md`

### Goal

Create the backend HTTP foundation that later tasks can extend.

### Scope

- Gin router setup.
- Production-safe recovery/error handling.
- Standard JSON error shape.
- `/api/v1/health` endpoint.
- Basic security headers where app-owned.
- Trusted proxy/client-IP helper foundation, without prematurely implementing all rate limiters.
- Graceful shutdown.
- Structured/simple request logging without secrets.

### Acceptance criteria

- Server starts/stops cleanly.
- Health endpoint returns predictable non-sensitive output.
- Unknown API route returns JSON 404 rather than HTML stack/error.
- Panic test path (test-only) demonstrates recovery behavior.
- Production errors do not leak stack/filesystem paths.

### Tests/checks

- HTTP integration tests.
- Go test/vet.

### Do not

- Do not build a custom logging framework.
- Do not add external APM.

---

## [x] V2-004 — Minimal Nuxt shell and dev API proxy

**Depends on:** V2-001, V2-002, V2-003

**Read:** `frontend/AGENTS.md`, `docs/UX_SPEC.md`, `docs/ARCHITECTURE.md`

### Goal

Make Nuxt render a minimal StereoDamage shell and communicate with Gin in dev/SSR without CORS complexity.

### Scope

- Universal SSR remains enabled for public routes.
- Minimal `/` page proves server-side fetch to Go health or a temporary diagnostic endpoint.
- Configure local browser API proxy/same-origin behavior.
- Establish global CSS entry and basic v1-inspired color/theme variables without redesigning screens.
- Create placeholder `/admin` route configured in the intended rendering mode (client-only if chosen).

### Acceptance criteria

- View-source/server response for `/` contains rendered content.
- Browser requests can call `/api/v1/...` same-origin in dev.
- Nuxt SSR can reach Go through internal base URL.
- No public Admin link is shown.
- No UI framework added.

### Tests/checks

- Nuxt build/typecheck.
- Simple SSR integration/smoke.

### Do not

- Do not implement feed UI yet.
- Do not make `/` a client-only page.

---

## [x] V2-005 — Root developer workflow commands

**Depends on:** V2-003, V2-004

**Read:** `AGENT_WORKFLOW.md`, `docs/TESTING.md`

### Goal

Make common agent/owner workflows one-command simple.

### Scope

Provide/document commands such as:

- start both dev processes;
- test backend;
- typecheck/test frontend;
- run all non-E2E checks;
- build production backend/frontend;
- later-compatible hooks for E2E and compatibility audit.

Use a small script/Makefile approach, not a process-orchestration platform.

### Acceptance criteria

- Owner can start local v2 with one documented command or two obvious commands.
- Root test/build commands fail when a child check fails.
- Commands work without production secrets.

### Do not

- Do not add Docker solely for local orchestration.

### Phase 0 gate

Owner verifies: repo is clean, both processes run, SSR works, commands are understandable, visual skeleton has not already become a generic template.

---

# Phase 1 — v1 data compatibility before feature rewrite

Goal: prove Go can safely open and evolve v1 data before building the new product around it.

## [x] V2-101 — Create sanitized v1 compatibility fixtures

**Depends on:** V2-003

**Read:** `docs/V1_REFERENCE.md`, `docs/DATA_COMPATIBILITY.md`, v1 `backend/db.js`

### Goal

Represent real v1 schema/content edge cases in tests without committing production data.

### Scope

Create test fixture builder/SQL containing:

- all v1 tables/columns/index-relevant relationships;
- posts with paragraph/heading/quote/divider;
- each media kind and spoiler;
- explicit/null preview media;
- Cyrillic/English titles;
- likes;
- nested comments;
- pending/rejected comments;
- moderation/attempt/mute/session rows;
- representative timestamps.

### Acceptance criteria

- Fixture is structurally compatible with current v1 schema.
- Tests can create isolated temporary DB copies deterministically.
- No real production content/secrets are committed.

### Tests/checks

- Fixture integrity/foreign-key checks.

### Do not

- Do not simplify fixture schema to only `posts`/`comments`.

---

## [x] V2-102 — SQLite connection layer and safe startup inspection

**Depends on:** V2-101

**Read:** `backend/AGENTS.md`, `docs/DATA_COMPATIBILITY.md`

### Goal

Open v1/fresh SQLite safely with deliberate pragmas/pool settings.

### Scope

- `database/sql` SQLite connection.
- `foreign_keys=ON` enforcement.
- sensible busy timeout/pool settings.
- evaluate/enable WAL only with tests and clear rationale.
- schema inspection helpers that identify v1/fresh/unsupported DB without destructive actions.
- clear startup error for unsupported/corrupt schema.

### Acceptance criteria

- Opens v1 fixture without modifying user rows.
- Opens fresh DB path.
- Never drops tables on mismatch.
- Integration test proves connection settings apply.

### Do not

- Do not implement full migrations in this task.
- Do not use an ORM.

---

## [x] V2-103 — Ordered migration runner + pre-migration safety backup

**Depends on:** V2-102

**Read:** `docs/DATA_COMPATIBILITY.md`, `docs/DEPLOYMENT.md`

### Goal

Create a trustworthy migration mechanism before adding v2 columns.

### Scope

- migration metadata table;
- ordered versioned migrations;
- same path for fresh and legacy DB to final schema;
- production-mode pre-migration consistent DB backup policy;
- migration lock/transaction semantics suitable for one-process startup;
- clear migration logging without data/secret leakage.

### Acceptance criteria

- v1 fixture migrates without row-count/content loss.
- fresh DB migrates to same schema version.
- rerunning startup does not reapply completed migrations.
- simulated failed migration does not silently continue serving.
- safety backup is created/verified according to documented policy before schema-changing migration in production mode.

### Tests/checks

- integration tests for fresh, v1, repeated, and failed migration.
- SQLite integrity check.

### Do not

- Do not add destructive down-migrations.
- Do not raw-copy a live DB as the only “backup” strategy if writes can make it inconsistent.

---

## [x] V2-104 — Add and backfill stable post slugs

**Depends on:** V2-103

**Read:** `docs/DATA_COMPATIBILITY.md`, `docs/CONTENT_FORMAT.md`

### Goal

Add canonical slug identity while preserving numeric IDs and all old data.

### Scope

- additive `posts.slug` migration (or equivalent minimal schema).
- deterministic slug generation supporting RU/EN titles.
- collision handling.
- backfill every existing post.
- uniqueness enforcement after/backed by safe migration logic.
- slug generation helper for future new posts.

### Acceptance criteria

- Every v1 fixture post gets a non-empty unique slug.
- IDs/titles/blocks/timestamps unchanged.
- Repeated migration is stable.
- Cyrillic titles have deterministic sensible output according to documented policy.
- Slug does not depend on current locale UI.

### Tests/checks

- collision cases, punctuation, Cyrillic, empty/degenerate title handling.

### Do not

- Do not use slug as primary key.
- Do not mutate old post titles.

---

## [x] V2-105 — Legacy block parser + v2 rich inline schema validation

**Depends on:** V2-102

**Read:** `docs/CONTENT_FORMAT.md`, v1 `normalizeBlock`/validation code

### Goal

Establish canonical Go types and validation for old and future rich text without bulk migration.

### Scope

- Go block types for paragraph/heading/quote/divider/media.
- Parse old `blocks_json` safely.
- Preserve `text` behavior.
- Define minimal supported rich inline node/mark schema for bold/italic/link/inline-code.
- Generate plain `text` fallback from rich content.
- Validate safe link schemes.
- Media source compatibility `/uploads/...`.
- Helpers for preview text/reading text/time/media derivation as needed.

### Acceptance criteria

- All fixture legacy posts parse.
- Invalid blocks fail safely according to public/admin context without panics.
- Rich inline -> plain fallback deterministic.
- Arbitrary HTML/javascript URL cannot enter canonical rich content.
- JSON serialization remains explicit/stable.

### Tests/checks

- fixture tests + malicious/invalid inline cases.

### Do not

- Do not introduce arbitrary HTML.
- Do not require rewriting old DB rows.

---

## [x] V2-106 — Compatibility audit command

**Depends on:** V2-103, V2-104, V2-105

**Read:** `docs/DATA_COMPATIBILITY.md`

### Goal

Give owner/agent a safe way to test a copied production DB and uploads repeatedly.

### Scope

A command/script should report:

- integrity check;
- migration version;
- post/comment counts;
- key relationship/foreign-key issues;
- number of post media references;
- missing referenced upload files;
- slug uniqueness/completeness;
- parseability of blocks;
- useful summary without changing content.

### Acceptance criteria

- Runs against fixture and uncommitted external DB path.
- Read-only audit mode does not mutate database.
- Exits non-zero on integrity/critical compatibility failures.
- Output is understandable to owner.

### Do not

- Do not “repair” failures automatically.

### Phase 1 gate

Owner runs migration + audit on an actual production backup copy and manually opens DB/row counts. Do not proceed to large feature work until this is trustworthy.

---

# Phase 2 — Go backend feature parity and improved API

Goal: replace v1 backend behavior with tested Go endpoints while improving API shape for Nuxt.

## [x] V2-201 — Post repositories and public feed API

**Depends on:** V2-104, V2-105

**Read:** `docs/ARCHITECTURE.md`, v1 `/posts` behavior

### Goal

Serve efficient feed summaries for Nuxt infinite scroll.

### Scope

- post repository queries;
- reverse chronological cursor pagination by timestamp + ID tie-break;
- feed response with id, slug, title, created_at, likes, reading minutes, preview text/media;
- visible comment count and small visible comment preview array included efficiently;
- stable opaque/simple cursor contract.

### Acceptance criteria

- First and subsequent pages do not duplicate/skip fixture posts.
- Equal timestamps handled correctly.
- Comment previews require no per-post browser/API fanout.
- Pending/rejected comments excluded from public counts/previews.
- Response excludes full post blocks.

### Tests/checks

- integration pagination tests;
- query behavior with nested/pending comments.

### Do not

- Do not implement search/archive/filter taxonomy.

---

## [x] V2-202 — Public post-by-slug and legacy ID resolution API

**Depends on:** V2-201

### Goal

Serve full long-read data by canonical slug and provide numeric-ID resolution for legacy links/progress.

### Scope

- `GET` full post by slug;
- safe parsed blocks;
- reading time/preview metadata;
- endpoint/helper to resolve numeric ID to canonical slug or fetch by ID for internal redirect use.

### Acceptance criteria

- Legacy fixture posts return correct blocks.
- Missing slug/ID gives typed 404.
- Unknown block corruption cannot crash server.
- ID remains in public post response for local reader progress/comments.

### Do not

- Do not add old `/posts/:id` path as canonical public route unless needed strictly as API compatibility helper.

---

## [x] V2-203 — Post likes parity

**Depends on:** V2-201

**Read:** v1 post-like implementation/config

### Goal

Port anonymous likes with hashed-IP cooldown/rate behavior safely.

### Scope

- trusted client IP -> salted hash;
- configurable cooldown/rate limit;
- atomic like event/count write;
- cleanup/retention behavior comparable to v1;
- API response for updated count/error.

### Acceptance criteria

- valid like increments exactly once;
- too-fast repeat rejected as intended;
- concurrent-ish test cannot desync event/count through obvious race;
- raw IP not persisted/exposed where v1 did not require it.

---

## [ ] V2-204 — Public comment read API

**Depends on:** V2-202

### Goal

Return visible comments/replies in a shape convenient for the new UI.

### Scope

- comments by post ID/slug-associated ID;
- visible-only;
- stable ordering;
- parent IDs and likes;
- optional pagination strategy if needed for very large threads, but do not overbuild.

### Acceptance criteria

- Nested fixture thread relationships preserved.
- Pending/rejected excluded.
- Optional name preserved/null handled.
- API is straightforward for frontend tree rendering.

### Do not

- Do not add imageboard numbering/media.

---

## [ ] V2-205 — Comment challenge and antispam parity port

**Depends on:** V2-204

**Read:** v1 comment challenge/scoring/mutes/attempt code, `docs/SECURITY.md`

### Goal

Port the existing invisible antispam protections before enabling comment writes.

### Scope

Port/organize:

- challenge HMAC/token parsing/TTL/clock skew;
- dynamic honeypot;
- replay/use tracking;
- text normalization/hash/fingerprint;
- cooldown/burst/duplicate checks;
- URL/repetition/random-text/diversity heuristics;
- source origin/referer signals;
- moderation score/reasons;
- attempt recording/TTL cleanup;
- mute thresholds/expiry.

Keep configuration mapping compatible where practical.

### Acceptance criteria

- Representative v1 accept/pending/reject/mute paths have tests.
- Valid human-like fixture comment is not blocked by default.
- Suspicious comments can become pending.
- Invalid/replayed/expired challenge behaves safely.
- No CAPTCHA added.
- Logic is split into understandable functions/packages rather than one route handler.

### Do not

- Do not redesign thresholds casually during port.
- Do not remove attempt/moderation tables to simplify code.

---

## [ ] V2-206 — Comment create/reply endpoint

**Depends on:** V2-205

### Goal

Enable anonymous/named comments and replies through the ported antispam pipeline.

### Scope

- validate post, optional parent same post, name/content lengths;
- apply challenge/antispam;
- save visible/pending as appropriate;
- record attempt/state atomically where required;
- response distinguishes visible vs pending.

### Acceptance criteria

- anonymous comment works;
- named comment works;
- reply works;
- cannot reply to comment from another post;
- pending response is distinct and frontend-friendly;
- malformed/oversized JSON safely rejected.

---

## [ ] V2-207 — Comment likes parity

**Depends on:** V2-204

### Goal

Port anonymous comment likes with v1-style protections.

### Acceptance criteria

- only visible existing comments can be liked;
- cooldown/rate limit works;
- count update is atomic;
- raw IP not exposed.

---

## [ ] V2-208 — Admin authentication/session/CSRF

**Depends on:** V2-103, V2-003

**Read:** v1 admin auth code, `docs/SECURITY.md`

### Goal

Establish secure single-owner admin auth before any v2 write endpoints.

### Scope

- login with env secret;
- timing-safe validation;
- login limiter;
- random session;
- hashed session storage in existing-compatible `admin_sessions` table;
- secure cookie settings;
- session status endpoint returning safe data + CSRF token mechanism as needed;
- CSRF enforcement middleware for writes;
- logout/expiry cleanup;
- production default-secret refusal.

### Acceptance criteria

- unauthorized writes impossible on protected test route.
- valid login creates session cookie without exposing token to JS body.
- invalid/expired session rejected.
- missing/wrong CSRF rejected for writes.
- cookie flags correct by environment.
- login/logout/session tests pass.

### Do not

- Do not add users/roles/OAuth.

---

## [ ] V2-209 — Admin post CRUD using canonical block validation

**Depends on:** V2-208, V2-105, V2-202

### Goal

Provide create/edit/delete endpoints for one author, including slug behavior.

### Scope

- create post with required title, blocks, preview media, generated/editable pre-publish slug;
- update existing post while preserving slug unless explicitly changed under allowed rules;
- delete with cascade behavior;
- list/fetch endpoints useful for admin edit selection;
- server canonical rich validation.

### Acceptance criteria

- create legacy-compatible/plain and rich text posts.
- update old fixture post safely.
- title change does not auto-change established slug.
- duplicate slug handled with clear error or deterministic generation.
- delete cascades comments/events according to schema.
- all writes require auth + CSRF.

### Do not

- Do not add multi-author/publish workflow roles.
- Do not add scheduling unless requested later.

---

## [ ] V2-210 — Local filesystem upload API

**Depends on:** V2-208

**Read:** v1 upload code, `docs/SECURITY.md`

### Goal

Provide safe media upload that the new editor can call directly at insertion time.

### Scope

- multipart single-file upload;
- generated safe name;
- preserve `/uploads/...` URL convention;
- size/type validation;
- media-kind detection;
- cleanup failed/partial upload;
- response includes URL/original name/stored name/kind and image dimensions if cheaply available.

### Acceptance criteria

- supported image/GIF/video/audio/generic file cases work according to policy.
- empty/oversized/traversal/mismatched unsafe cases rejected.
- existing uploads untouched.
- auth+CSRF required.

### Special decision

Handle SVG deliberately. Preserve safe access to historical SVGs but do not blindly accept new active SVG uploads without a safe policy.

### Do not

- Do not add S3.
- Do not require image optimization service.

---

## [ ] V2-211 — Admin moderation API parity

**Depends on:** V2-208, V2-205, V2-206

### Goal

Port the useful moderation controls from v1.

### Scope

- pending comments list;
- recent attempts summary;
- active mutes;
- approve/reject;
- delete comment subtree;
- unmute;
- safe short IP hash display only.

### Acceptance criteria

- v1 fixture moderation rows visible appropriately.
- approve makes comment public.
- reject hides it.
- delete subtree semantics preserved.
- no raw IP exposure.
- auth+CSRF correct.

---

## [ ] V2-212 — Portable full backup service/API

**Depends on:** V2-208, V2-102, V2-210

**Read:** `docs/DATA_COMPATIBILITY.md`, `docs/SECURITY.md`, `docs/DEPLOYMENT.md`

### Goal

One admin action downloads a restorable archive of SQLite + uploads without stopping the site.

### Scope

Archive format:

```text
manifest.json
data/blog.db
uploads/**
```

- consistent SQLite snapshot/backup API;
- stream ZIP/temporary file safely;
- manifest version/timestamp/schema/media count/checks as useful;
- cleanup temporary resources;
- auth/rate protection;
- exclude `.env` and secrets.

### Acceptance criteria

- backup DB passes integrity check when extracted.
- representative fixture/media archive restores to a disposable v2 runtime.
- upload files byte-match originals.
- unauthorized request blocked.
- backup does not require stopping server.
- memory use is bounded for archive size.

### Do not

- Do not archive entire source tree/node_modules.
- Do not include `.env`.

### Phase 2 gate

Backend parity can be exercised through API tests against migrated v1 fixture/production copy. Owner verifies migration, read APIs, admin auth, one create/edit, comments, uploads, moderation, backup.

---

# Phase 3 — public Nuxt timeline foundation

Goal: replace the public MPA with SSR Nuxt while preserving StereoDamage identity and making feed navigation genuinely better.

## [ ] V2-301 — Typed frontend API client and shared public data types

**Depends on:** V2-201, V2-202, V2-204

**Read:** `frontend/AGENTS.md`, `docs/ARCHITECTURE.md`

### Goal

Create a small typed boundary between Nuxt and Gin.

### Scope

- typed response models;
- server/client base URL handling;
- consistent error mapping;
- request cancellation where useful;
- no duplicate ad-hoc fetch helpers per page.

### Acceptance criteria

- works during SSR and browser navigation.
- no public domain hardcoding.
- errors can distinguish 404/rate/pending/etc. where UI needs it.

### Do not

- Do not create a generic enterprise SDK.

---

## [ ] V2-302 — Global visual foundation and v1 settings compatibility

**Depends on:** V2-004

**Read:** v1 `styles.css`, v1 `theme.js`/i18n, `docs/UX_SPEC.md`

### Goal

Establish the polished-but-recognizable StereoDamage shell before implementing all screens.

### Scope

- theme variables/light-dark behavior;
- migrate/reuse `stereoDamageTheme`;
- RU/EN lightweight UI foundation reusing `stereoDamageLanguage`;
- preserve list/grid preference key;
- remove/ignore smooth-scroll preference;
- compact top-level site chrome/profile identity as specified, without giant redesign;
- typography/spacing primitives.

### Acceptance criteria

- v1 theme/language/list-grid preferences continue on same-domain localStorage fixture.
- smooth-scroll setting/control absent.
- dark/light no flash severe enough to be distracting.
- public header has no unauthenticated Admin link.
- mobile header is intentional, not desktop controls wrapped awkwardly.

### Visual owner check

Owner must approve overall vibe before many pages are built.

### Do not

- Do not switch to generic SaaS aesthetic.
- Do not remove grid mode.

---

## [ ] V2-303 — SSR feed/list view

**Depends on:** V2-301, V2-302

**Read:** `docs/UX_SPEC.md`, v1 feed renderer

### Goal

Render the first feed page as immediate, content-rich SSR HTML.

### Scope

- feed post component;
- title/date/reading time/progress placeholder integration;
- preview text/media;
- likes count/action UI wiring may remain incremental if V2-203 API ready;
- visible comment previews/count;
- clean main/list visual consistent with v1;
- loading/error/empty states.

### Acceptance criteria

- raw server HTML contains first post titles/text.
- no per-card comment API calls.
- posts render stable without layout collapse.
- comment count target reserved for later `#comments` routing.
- mobile/desktop visually usable.

### Do not

- Do not implement infinite scroll in same task.

---

## [ ] V2-304 — Grid feed mode parity

**Depends on:** V2-303

### Goal

Preserve v1 grid mode with v2 media/responsive polish.

### Scope

- list/grid toggle in compact settings/control.
- persist v1-compatible key.
- two-column or suitable desktop grid.
- mobile behavior avoids cramped forced grid.

### Acceptance criteria

- preference persists.
- grid cards retain all essential click/actions without broken hierarchy.
- no duplicate fetch/state architecture for the two modes.

---

## [ ] V2-305 — Infinite scroll with cursor loading

**Depends on:** V2-303

### Goal

Turn feed into a long seamless timeline without sacrificing reliability.

### Scope

- observer/sentinel loading;
- cursor state;
- loading/retry/end states;
- deduplicate items/requests;
- keep first page SSR;
- no `page=2` offset assumptions.

### Acceptance criteria

- scrolling loads all fixture pages exactly once.
- temporary API failure can retry.
- end state stops requests.
- fast observer triggers do not issue overlapping duplicate fetches.
- keyboard/non-observer fallback remains reasonable (e.g. load-more action if IntersectionObserver absent).

---

## [ ] V2-306 — Feed state and scroll restoration across post navigation

**Depends on:** V2-305

### Goal

Make infinite scroll pleasant: Back returns exactly where reader left off.

### Scope

- preserve loaded items/cursor/scroll position for same-session internal navigation.
- integrate with Nuxt router history/scroll behavior carefully.
- direct navigation/reload remains sane.

### Acceptance criteria

E2E:

1. load multiple feed batches;
2. scroll to old post;
3. open post;
4. press browser/app Back;
5. previous batches remain available or are restored without visible reset;
6. reader returns near exact original post position.

### Do not

- Do not save unbounded historical feed HTML in localStorage.
- Do not add a heavyweight state system solely for this if Nuxt state/composables suffice.

### Phase 3 gate

Owner browses the v2 timeline on phone/desktop for several pages and confirms it still feels like StereoDamage and navigation feels materially better than v1.

---

# Phase 4 — long-read, media, discussion, sharing UX

Goal: make opening and reading a shared post the strongest public experience.

## [ ] V2-401 — Canonical `/posts/:slug` SSR article route

**Depends on:** V2-202, V2-301, V2-302

### Goal

Render migrated v1 long reads at canonical slug URLs with a true reading-oriented layout.

### Scope

- full structured block renderer;
- article title/date/reading time;
- paragraph/heading/quote/divider/media/spoiler;
- rich inline marks;
- reading-width typography;
- sensible wider media behavior;
- not-found state.

### Acceptance criteria

- raw SSR HTML contains title/body text.
- all legacy fixture block/media types render.
- no arbitrary `v-html` for content.
- long post comfortable at mobile/desktop widths.
- visual layout reduces unnecessary nested-card/panel feeling without radically changing brand.

---

## [ ] V2-402 — Reader progress + continue-reading compatibility

**Depends on:** V2-401, V2-303

### Goal

Preserve v1 reader-state value with quieter UX.

### Scope

- reuse/migrate `stereoDamageReadingProgress` keyed by numeric post ID;
- track progress/scroll position efficiently;
- completed threshold;
- subtle continue prompt;
- feed status/progress integration;
- storage failure safe.

### Acceptance criteria

- synthetic v1 localStorage entry is recognized.
- progress updates are throttled/debounced, not every scroll event synchronous write.
- completion behaves consistently.
- continue action returns to sensible reading location after layout is stable.

---

## [ ] V2-403 — Long-read TOC and heading anchors

**Depends on:** V2-401

### Goal

Preserve/improve v1 TOC without adding chrome to short posts.

### Scope

- stable heading IDs;
- only show TOC above a sensible heading/length threshold;
- desktop sticky/side treatment if layout allows;
- compact mobile disclosure;
- active section via observer.

### Acceptance criteria

- short post has no unnecessary TOC.
- long fixture post navigation works.
- mobile TOC does not cover reading area.
- anchor navigation respects sticky header offset.

---

## [ ] V2-404 — Image/media responsiveness and lightweight image viewer

**Depends on:** V2-401

### Goal

Make images feel first-class without a heavy gallery dependency.

### Scope

- lazy offscreen images;
- aspect ratio/dimensions where known;
- captions;
- click/tap image viewer;
- keyboard Escape/focus behavior;
- mobile viewport/touch close behavior;
- graceful missing-media fallback.

### Acceptance criteria

- long image post does not eagerly download every offscreen image unnecessarily.
- viewer works keyboard/mouse/touch.
- no severe background scroll/focus bug.
- GIF/video/audio/file behavior unaffected.

---

## [ ] V2-405 — Global persistent audio engine and mini-player

**Depends on:** V2-401, V2-306

**Read:** v1 audio code, `docs/UX_SPEC.md`

### Goal

Rebuild the beloved but brittle v1 audio concept around Nuxt's persistent app lifecycle.

### Scope

- one global playback controller/store;
- one active audio element/track;
- inline track controls bind to global state;
- route navigation preserves playback;
- mini-player appears when active/progress meaningful;
- play/pause/seek/volume/close;
- preserve/migrate audio volume preference;
- optional waveform progressive enhancement implemented efficiently or deferred behind a clean interface.

### Acceptance criteria

E2E:

- start audio inside article;
- navigate to feed/another post via Nuxt;
- playback continues;
- mini-player controls same audio;
- returning to original track reflects state;
- close stops/releases it;
- no duplicate simultaneous players.

Waveform failure must not affect audio playback.

### Do not

- Do not make a Spotify clone.
- Do not decode every feed audio item on page load.

---

## [ ] V2-406 — Public comments UI + low-friction composer

**Depends on:** V2-204, V2-206, V2-207, V2-401

### Goal

Make discussion easy without copying imageboard UI.

### Scope

- discussion section near article end;
- composer easy to reach at discussion start;
- anonymous default;
- optional name;
- reply context;
- nested thread rendering with capped visual indentation;
- comment likes;
- challenge/honeypot integration invisible to normal user;
- pending/posted/errors feedback.

### Acceptance criteria

- anonymous/named/reply flows pass E2E.
- mobile nested comments retain usable width.
- replying does not create unwieldy full forms at every nesting point.
- no media/comment numbering/tripcodes.
- plain comment text cannot render HTML.

---

## [ ] V2-407 — Feed comment-preview click -> reliable `#comments`

**Depends on:** V2-303, V2-406

### Goal

Reader can choose “read post” vs “join discussion” from feed.

### Scope

- comment count/preview action routes to `/posts/:slug#comments`;
- wait for proper article/comments layout before final positioning;
- work on direct URL and client navigation.

### Acceptance criteria

- from feed, click replies lands at discussion/composer rather than top.
- direct browser open with hash works.
- long post/media loading does not leave final anchor wildly wrong.

---

## [ ] V2-408 — Sharing UI + canonical SEO/Open Graph metadata

**Depends on:** V2-401

**Read:** `docs/UX_SPEC.md`

### Goal

Make “send this long read to someone” a polished first-class flow.

### Scope

- canonical URL;
- SSR title/description/OG/Twitter metadata;
- preview image if selected/suitable, default site image otherwise;
- native Web Share on supported mobile;
- copy-link fallback;
- compact feedback.

### Acceptance criteria

- metadata exists in server HTML without client JS.
- copy/share uses canonical slug URL.
- no network-specific SDK/buttons.
- post with and without image gets valid share image metadata strategy.

---

## [ ] V2-409 — Legacy `/post.html?id=` redirect compatibility

**Depends on:** V2-202, V2-401

### Goal

Never break old social links already shared by owner.

### Scope

- Nuxt/Nitro route or reverse-proxy-compatible handler for `/post.html?id=<id>`;
- resolve ID through backend;
- redirect to canonical slug URL;
- invalid/missing ID proper error/not found.

### Acceptance criteria

- fixture old link redirects correctly.
- nonexistent ID does not redirect home.
- query garbage handled safely.
- production redirect status/canonical behavior documented.

---

## [ ] V2-410 — Public UX/i18n/settings polish pass

**Depends on:** V2-402 through V2-409

### Goal

Bring public experience to cohesive feature parity before admin rewrite dominates attention.

### Scope

- all new public strings RU/EN;
- theme/list-grid/settings UI polish;
- reader reset behavior if retained;
- loading/error states;
- focus/reduced-motion basics;
- ensure smooth-scroll option completely removed;
- verify no public admin link.

### Acceptance criteria

- representative public E2E passes in RU and EN UI.
- dark/light checked.
- mobile no obvious overflow/overlap.

### Phase 4 gate

Owner shares a v2 test post link to phone/desktop, reads it, plays media, comments, returns to feed, and compares feel directly with v1. Public side should already feel worth the rewrite.

---

# Phase 5 — private admin and rich long-form publishing

Goal: transform authoring from “manage JSON blocks/uploads” into a lightweight writing experience, including mobile.

## [ ] V2-501 — Private admin login/shell

**Depends on:** V2-208, V2-302

### Goal

Create a clean private admin entry that is secure but not annoying for the single owner.

### Scope

- `/admin` login view;
- session check;
- secret input;
- logout;
- authenticated admin shell/navigation;
- no public link;
- session/CSRF client helper.

Admin home prioritizes New/Resume Post, Posts, Moderation, Backup.

### Acceptance criteria

- unauthenticated admin APIs remain inaccessible even if UI manipulated.
- successful login enters shell.
- refresh preserves valid session via cookie.
- logout returns to login.
- mobile shell usable.

---

## [ ] V2-502 — StereoDamage editor adapter and rich-text engine spike

**Depends on:** V2-105, V2-501

**Read:** `docs/CONTENT_FORMAT.md`

### Goal

Prove the editor can feel like rich text while canonical storage stays StereoDamage blocks.

### Scope

- integrate preferred mature Vue editor engine client-only (Tiptap/ProseMirror candidate) with minimal required extensions;
- explicit `blocks -> editor document` adapter;
- explicit `editor document -> blocks` adapter;
- paragraphs, bold, italic, links, inline code, headings, quote, divider representation;
- tests using legacy and rich fixtures;
- ensure editor package is absent from public initial route bundle.

### Acceptance criteria

- legacy block fixture loads as editable document.
- formatting round-trip preserves content.
- save adapter generates plain `text` fallbacks.
- unsupported editor nodes cannot silently become canonical arbitrary HTML.
- public bundle inspection shows editor code route-split.

### Do not

- Do not build custom contenteditable engine.
- Do not yet build full media upload/admin management screen.

---

## [ ] V2-503 — New-post writing surface

**Depends on:** V2-502

### Goal

Make starting/writing a long post feel natural and low-friction.

### Scope

- title;
- large writing surface;
- compact formatting toolbar/context controls;
- insertion control (`+` or equivalent) for heading/quote/divider/media placeholders;
- undo/redo through editor;
- post settings area for slug/preview options kept secondary;
- clean mobile keyboard/toolbar layout.

### Acceptance criteria

- author can type several paragraphs/headings without interacting with “block cards”.
- formatting does not require Markdown syntax.
- toolbar does not cover text on mobile.
- title remains required but editor does not nag until publish/validation.

---

## [ ] V2-504 — Direct media insertion at cursor/block position

**Depends on:** V2-210, V2-503

### Goal

Eliminate v1's separate upload-then-insert workflow.

### Scope

- insert Photo/GIF/Video/Audio/File from editor;
- native file picker on phone;
- upload progress/error;
- media block inserted at intended position;
- caption/alt/name/spoiler controls;
- desktop paste/drop for images if straightforward;
- touch-friendly reorder/edit without requiring HTML5 drag-and-drop.

### Acceptance criteria

- choose media -> upload -> appears at insertion point in one flow.
- failed upload does not leave corrupt phantom block.
- editing media metadata updates canonical block.
- mobile photo picker path manually/E2E-smoke tested.

### Do not

- Do not introduce media-library/search system unless later needed.

---

## [ ] V2-505 — Autosave and draft recovery

**Depends on:** V2-503

### Goal

Author should not fear losing a writing session.

### Scope

- local autosave for new unpublished draft at minimum;
- quiet saved/error state;
- recover after refresh/reopen;
- distinguish editing published post from unpublished local draft;
- avoid public write until explicit publish/update.

Optionally introduce server draft persistence only if clearly justified and scoped; it is not required merely to satisfy autosave.

### Acceptance criteria

- write draft, refresh, recover title/content.
- corrupted local draft fails safely.
- autosave does not spam backend/public updates.
- clear/reset draft is deliberate.

---

## [ ] V2-506 — Exact public-renderer preview

**Depends on:** V2-401, V2-503, V2-504

### Goal

Preview what will actually be published without maintaining a second fake renderer.

### Scope

- preview mode/dialog/route using shared post-renderer components;
- current unsaved editor state converted through canonical adapter;
- realistic article layout/theme/media.

### Acceptance criteria

- representative rich/media draft preview matches final published renderer structurally/visually.
- preview cannot accidentally publish.
- mobile preview easy to exit.

---

## [ ] V2-507 — Publish new post workflow

**Depends on:** V2-209, V2-505, V2-506

### Goal

Turn a draft into a canonical post confidently.

### Scope

- validate title/content;
- generate/show optional slug setting;
- preview media selection/auto strategy;
- authenticated+CSRF create request;
- prevent double submit;
- success opens canonical post/admin success state and clears matching draft safely.

### Acceptance criteria

- create a real rich/media post and public page renders it.
- slug canonical URL stable.
- network error does not lose local draft.
- double click cannot create duplicate posts.

---

## [ ] V2-508 — Existing post list/edit/update workflow

**Depends on:** V2-209, V2-502, V2-506

### Goal

Edit old migrated v1 posts and new v2 posts through the same friendly editor.

### Scope

- compact chronological admin post list;
- open post in editor;
- legacy blocks adapter;
- update;
- delete with explicit confirmation;
- published slug remains stable on title edits;
- preview media preserved/adjustable.

### Acceptance criteria

- edit fixture legacy post text and save without losing untouched media/blocks.
- rich formatting added to old paragraph survives reload.
- title edit does not change slug automatically.
- deletion is not one accidental tap on mobile.

---

## [ ] V2-509 — Admin moderation UI

**Depends on:** V2-211, V2-501

### Goal

Port moderation functionality without letting it dominate daily posting UX.

### Scope

- pending comments badge/section;
- approve/reject/delete;
- recent attempts and mutes in secondary view;
- unmute;
- no raw IP.

### Acceptance criteria

- all backend moderation actions usable on phone/desktop.
- pending comment can be reviewed with post/context link where practical.
- errors are clear.

---

## [ ] V2-510 — Admin full-backup UI

**Depends on:** V2-212, V2-501

### Goal

Owner can download a full portable backup without shell commands.

### Scope

- clear Backup section/action;
- explain contents: DB + uploads, no secrets;
- initiate authenticated download;
- progress/wait UX suitable for a small VPS;
- prevent accidental repeated parallel backups.

### Acceptance criteria

- downloaded archive passes V2-212 restore test.
- mobile browser can initiate download where platform allows.
- auth expiry/error handled.

---

## [ ] V2-511 — Mobile-first admin polish pass

**Depends on:** V2-503 through V2-510

### Goal

Make posting from a phone a genuinely good path, not merely technically responsive.

### Scope

Test/polish:

- phone keyboard and viewport;
- title/editor toolbar;
- media insertion;
- block/media reorder/edit;
- preview;
- publish/update/delete;
- moderation;
- backup;
- safe-area/fixed controls.

### Acceptance criteria

Owner can create a realistic post with photo/media from real phone without switching to desktop.

No critical action is hidden behind hover or tiny desktop controls.

### Phase 5 gate

Owner personally writes and publishes at least one disposable long-form test post from desktop and one meaningful draft/post from phone. If the editor still feels like “managing blocks”, phase is not accepted.

---

# Phase 6 — low-cost polish: RSS, OG enhancement, media/UX quality

Goal: add small high-value features and finish the “10x better” experience without feature creep.

## [ ] V2-601 — Automatic RSS feed

**Depends on:** V2-202, V2-408

### Goal

Expose a zero-maintenance RSS feed consistent with indie-web values.

### Scope

- `/feed.xml` or explicitly selected path;
- published posts reverse chronological;
- canonical slug URLs;
- title/date/description and valid content strategy;
- correct content type;
- automatic inclusion of new posts.

### Acceptance criteria

- validates with a standard RSS parser/test.
- no manual feed update step.
- no external service.

---

## [ ] V2-602 — Optional generated OG card enhancement

**Depends on:** V2-408

### Goal

Provide a good share image even for text-heavy posts without a suitable cover.

### Scope

Evaluate a server/build-time solution that does **not** add meaningful client runtime weight.

Generated card can include site name/title/simple visual identity.

### Acceptance criteria

- fallback generated/default card works for no-image post.
- generation failure falls back to static default, never blocks article.
- no huge public client dependency.

### Do not

- Do not make this block launch if a simple static default OG image already works well.

---

## [ ] V2-603 — Media optimization pass within local-storage budget

**Depends on:** V2-404, V2-504

### Goal

Improve real mobile loading without introducing paid storage/CDN.

### Scope

Measure before deciding. Potential safe work:

- image dimensions metadata;
- optional thumbnails/responsive derivatives for new uploads;
- WebP/AVIF derivative generation if resource-safe;
- correct lazy/preload behavior;
- EXIF orientation/metadata cleanup if desired;
- original file retention.

### Acceptance criteria

- representative image-heavy post transfers less unnecessary data or has clearly improved layout/loading.
- original files remain recoverable.
- tiny VPS memory/CPU bounded.
- old uploads continue unchanged.

### Do not

- Do not require S3/CDN.
- Do not generate every format/size “because modern”.

---

## [ ] V2-604 — Accessibility and interaction polish

**Depends on:** Phase 4 + Phase 5 public/admin UX

### Goal

Ensure retro/indie style does not imply brittle interactions.

### Scope

- semantic heading structure;
- focus visibility;
- labels;
- keyboard image viewer/dialog/editor basics;
- reduced motion;
- contrast review;
- touch targets;
- no hover-only essential controls.

### Acceptance criteria

- keyboard smoke through public share/comments/viewer/admin basics.
- automated accessibility checks have no obvious critical violations in key pages.
- mobile touch targets/overlap manually checked.

---

# Phase 7 — performance, security, operations, production rehearsal

Goal: prove the rewrite is fast, secure enough, portable, and actually deployable on the cheap VPS.

## [ ] V2-701 — Public performance audit and dependency diet

**Depends on:** Phase 4, V2-601 optional

**Read:** `docs/UX_SPEC.md`, `docs/TESTING.md`

### Goal

Validate “instant-feeling” instead of assuming Go/Nuxt guarantees it.

### Scope

- inspect Nuxt route chunks/bundle;
- prove editor/admin packages absent from public initial bundle;
- inspect feed request count;
- measure SSR/LCP/CLS/INP representative mobile throttling;
- identify eager media/audio work;
- remove obviously unnecessary dependencies/work.

### Acceptance criteria

- no per-feed-card comment request fanout.
- no editor code in public initial route bundle.
- SSR first content visible without JS.
- no obvious media-induced CLS/eager-download regression.
- findings and fixes documented.

### Do not

- Do not sacrifice functionality/accessibility for a vanity score.

---

## [ ] V2-702 — Tiny-VPS resource test

**Depends on:** V2-701, V2-212

### Goal

Ensure production topology fits realistic resource limits.

### Scope

Run locally constrained or on staging VPS-like environment:

- Go idle/request memory;
- Nuxt idle/SSR memory;
- backup generation;
- image derivative generation if enabled;
- representative concurrent comments/feed reads;
- restart behavior.

### Acceptance criteria

- no unbounded memory growth in normal browsing.
- backup does not load entire archive into RAM.
- one Nuxt + one Go process is practical for target VPS.
- expensive derivative jobs bounded or disabled.

---

## [ ] V2-703 — Security hardening review

**Depends on:** V2-208 through V2-212, Phase 5

**Read:** `docs/SECURITY.md`

### Goal

Review implemented security as a system before production.

### Scope

- admin auth/cookies/CSRF;
- route hiding vs true auth;
- rate limits;
- proxy IP trust;
- uploads/SVG policy;
- rich content link/XSS handling;
- comments/antispam;
- backup auth/temp files;
- security headers;
- dependency vulnerability checks;
- production error/log secret leakage.

### Acceptance criteria

- security test cases in `docs/TESTING.md` pass.
- no known default secret allowed in production.
- unauthenticated admin APIs all rejected.
- client IP behind actual proxy verified.
- critical dependency advisories resolved or explicitly documented.

---

## [ ] V2-704 — Native systemd production service files and deploy docs

**Depends on:** V2-702

**Read:** `docs/DEPLOYMENT.md`

### Goal

Replace PM2-only v1 operations with a simple two-service native deployment.

### Scope

- example/real `stereodamage-api.service`;
- `stereodamage-web.service`;
- environment/config path strategy;
- service user/workdir/restart behavior;
- build/deploy script or concise documented commands;
- journal commands;
- persistent data path protections.

### Acceptance criteria

- both services start, stop, restart, survive boot simulation/documented enablement.
- bind only intended interfaces.
- deployment does not overwrite data/uploads.

### Do not

- Do not require Docker.

---

## [ ] V2-705 — Reverse proxy/TLS routing configuration

**Depends on:** V2-704

### Goal

Document/implement production same-origin routing with existing proxy technology.

### Scope

- `/api/v1/*` -> Go;
- `/uploads/*` -> chosen safe handler;
- all public/admin Nuxt routes -> Nuxt;
- HTTPS;
- forwarded client IP correctly configured;
- static asset caching appropriate;
- no accidental caching of admin/authenticated responses.

### Acceptance criteria

- same-origin browser API works without production CORS.
- SSR Nuxt internal API works.
- real client IP rate-limit test behind proxy passes.
- legacy redirect route works through proxy.

---

## [ ] V2-706 — End-to-end backup restore rehearsal

**Depends on:** V2-212, V2-510, V2-704

### Goal

Prove backups are not decorative.

### Scope

- generate admin backup from a populated v2 test instance;
- extract into clean disposable runtime;
- start services against restored data;
- audit DB/media;
- manually open representative post/comments/audio.

### Acceptance criteria

- restored instance starts and matches source data counts/content.
- no `.env` expected in backup; fresh runtime config can be supplied separately.
- procedure documented in simple owner language.

---

## [ ] V2-707 — Full production-data migration rehearsal

**Depends on:** V2-106, all core backend/public/admin phases, V2-706

### Goal

Run near-final v2 against a recent copy of actual production DB/uploads exactly as cutover will.

### Scope

- make fresh source backup copy;
- run migration;
- run compatibility audit;
- run full automated suite;
- manually inspect representative oldest/newest/media-heavy/comment-heavy posts;
- edit/save one copy-only legacy post;
- test legacy links;
- generate/restore v2 backup;
- record issues.

### Acceptance criteria

All `docs/DATA_COMPATIBILITY.md` release proof points pass.

No unexplained row loss, broken media references, or unreadable posts.

### Do not

- Do not use the only production DB.

---

## [ ] V2-708 — Final E2E and real-device release pass

**Depends on:** V2-707, V2-703, V2-705

### Goal

Complete `docs/RELEASE_CHECKLIST.md` on the real target environment/real phone as much as possible.

### Scope

- desktop + phone;
- feed infinite/back restoration;
- long read/TOC/progress;
- comments/replies;
- share preview;
- all media/audio navigation;
- admin mobile publishing;
- moderation;
- backup;
- RU/EN + themes;
- logs/performance.

### Acceptance criteria

Release checklist has no critical unchecked item; any deliberately deferred non-critical item documented.

---

## [ ] V2-709 — Production cutover and rollback-ready release

**Depends on:** V2-708

### Goal

Replace v1 in one controlled release with a known rollback path.

### Scope

- tag/release final v2 code;
- fresh external v1 backup;
- short maintenance/final sync plan;
- deploy Go/Nuxt/systemd/proxy;
- migrate final DB;
- smoke test;
- verify existing shared legacy URL;
- retain untouched v1 release/final backup.

### Acceptance criteria

- public domain serves v2.
- old link redirects.
- new canonical post works/share metadata present.
- comments/media/admin work.
- system services healthy.
- rollback assets remain available.

### Phase 7 / project gate

v2 is accepted when it is not merely feature-complete but materially better in:

- perceived speed;
- mobile reading;
- long-read presentation;
- sharing;
- comments reachability/usability;
- media/audio continuity;
- mobile/desktop authoring;
- backup/recovery confidence;
- code maintainability;

while still unmistakably feeling like the same personal StereoDamage site.

---

# Post-launch parking lot — explicitly NOT part of v2 launch unless owner promotes an item

These ideas may be useful later, but agents must not pull them into current tasks opportunistically.

- search;
- archive/year/month browser;
- tags/categories;
- multilingual post variants;
- slug history/manual post-publish slug changes with redirects;
- richer list nodes if not included in launch content format;
- media library/orphan cleanup UI;
- server-persisted drafts/version history;
- scheduled publishing;
- PWA/offline;
- S3-compatible optional storage;
- Docker Compose alternative deployment;
- advanced generated OG themes;
- feed virtualization for extremely deep sessions if measurement proves necessary;
- more audio playlist behavior;
- comments realtime updates;
- additional author accounts (currently a product non-goal).
