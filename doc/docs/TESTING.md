# Testing strategy

## Purpose

Tests exist to make an agent-driven full rewrite safe. Prefer high-value regression tests over chasing a coverage percentage.

## Test layers

### Backend unit/domain tests

Use for deterministic logic:

- block validation/normalization;
- rich-inline validation/plain-text fallback generation;
- slug generation/collision logic;
- reading-time/preview derivation if backend-owned;
- antispam token/challenge/scoring helpers;
- IP/session token hashing helpers;
- backup manifest helpers.

### Backend integration tests

Use temporary SQLite DB/filesystem fixtures.

Cover:

- migrations from v1 fixture;
- fresh schema;
- post CRUD;
- cursor pagination;
- slug lookup;
- legacy ID resolution;
- comments/replies/visibility;
- likes/cooldown;
- admin login/session/CSRF;
- moderation;
- upload validation/path safety;
- backup archive contents and DB consistency.

### Frontend unit/component tests

Keep focused:

- content renderer fixtures;
- editor adapters;
- i18n behavior;
- reader-progress storage migration/helper logic;
- feed state restoration logic where separable;
- audio store/player state transitions;
- share fallback behavior.

### Browser E2E

Use a modern browser test runner (Playwright is a reasonable default) for critical user flows.

Required launch flows:

1. Feed SSR content visible -> infinite scroll loads more.
2. Feed -> open post -> Back returns to same feed region.
3. Feed comment count -> post opens at comments.
4. Direct `/posts/:slug` renders post.
5. Legacy `/post.html?id=...` redirects.
6. Theme/language/list-grid setting persists.
7. Reader progress persists/continue behavior works.
8. Submit anonymous comment.
9. Submit named comment.
10. Reply to comment.
11. Pending moderation response path.
12. Like post/comment basic flow.
13. Start audio in post -> navigate route -> audio continues -> close stops.
14. Admin unauthenticated cannot call write API.
15. Admin login -> editor opens.
16. Create rich-text/media draft -> preview -> publish -> public render matches.
17. Edit a migrated legacy post -> save -> content remains intact.
18. Mobile viewport authoring smoke test.
19. Backup download as admin; unauthorized backup blocked.

## Mobile testing

At least test representative narrow viewports, but real-device manual checks are still important.

Automated checks should catch:

- horizontal overflow;
- hidden/unreachable actions;
- editor toolbar obscuring text;
- comment nesting collapse;
- fixed audio player overlapping essential controls;
- dialogs/viewer not fitting viewport.

## Production DB rehearsal

Real production DB/media copies are local/manual fixtures, not committed.

Provide a compatibility command or checklist that verifies:

- `PRAGMA integrity_check`;
- migration version;
- row counts;
- foreign-key integrity;
- media-reference existence report;
- representative API reads;
- no accidental destructive mutation.

Run this repeatedly during migration work and before release.

## Snapshot/golden fixtures

For the content renderer, prefer explicit structured fixtures and semantic assertions over giant brittle HTML snapshots.

Useful fixtures:

- legacy all-block-types post;
- rich inline formatting;
- multiple media types;
- missing media fallback;
- headings/TOC;
- Cyrillic/English content;
- long title/text edge cases.

## Performance checks

Near public UX completion and release:

- build bundle report/route chunks;
- verify editor dependencies are absent from public initial bundle;
- inspect network requests for feed to confirm no N+1 comment preview fanout;
- test representative mobile throttling in Lighthouse/WebPageTest/browser devtools;
- check image/video/audio loading strategy;
- measure on actual VPS after staging deploy if possible.

Treat obvious regressions as bugs even if functional tests pass.

## Security checks

At least automated/integration coverage for:

- wrong admin secret;
- brute-force limiter behavior where deterministic test setup permits;
- expired session;
- missing/invalid CSRF;
- unauthenticated admin endpoints;
- malicious rich text URL scheme;
- path traversal upload filename attempts;
- oversized/invalid upload;
- comment challenge invalid/reused/expired cases;
- backup endpoint auth.

## Test commands

Bootstrap tasks must define simple root commands, ideally something like:

```text
make test
make test-backend
make test-frontend
make e2e
make lint
make build
```

Exact tooling can differ, but the owner should not need to remember six nested package commands.

## Definition of tested

A task summary must list commands actually run and results. “Tests should pass” is not a test result.
