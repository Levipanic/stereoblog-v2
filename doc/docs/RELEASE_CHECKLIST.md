# v2 production release checklist

Do not treat this as relevant only on launch day; use it to expose unfinished work during the final phase.

## Data safety

- [ ] Fresh production v1 backup exists outside the VPS.
- [ ] Backup restore has been rehearsed on a disposable location.
- [ ] Recent production DB copy passes v2 migration rehearsal.
- [ ] Pre/post migration row-count/integrity report reviewed.
- [ ] No known missing media files unexplained.
- [ ] v2 admin backup archive has been downloaded and restore-tested.
- [ ] Migration never uses destructive reset behavior.

## Public parity

- [ ] All representative old posts render.
- [ ] Headings/quotes/dividers render.
- [ ] Images/GIF/video/audio/files render.
- [ ] Spoilers render.
- [ ] Preview media works.
- [ ] Likes work.
- [ ] Comments and nested replies render.
- [ ] Comment likes work.
- [ ] Pending moderation behavior works.
- [ ] RU/EN UI works.
- [ ] Theme works.
- [ ] List/grid preference works.
- [ ] Reader progress works.
- [ ] Smooth-scroll setting is gone without breaking scrolling.

## New UX

- [ ] Feed first page arrives as useful SSR HTML.
- [ ] Infinite scroll is stable and retryable.
- [ ] Feed -> post -> Back restores loaded feed and position.
- [ ] Comment preview click opens `#comments` correctly.
- [ ] Long-read typography checked on real phone and desktop.
- [ ] TOC behavior appropriate on long post/mobile.
- [ ] Image viewer/mobile media works.
- [ ] Share uses native share/copy fallback.
- [ ] Canonical metadata verified by viewing raw/server-rendered HTML.
- [ ] OG preview checked with at least one external debugger/client if practical.
- [ ] Legacy `/post.html?id=` links redirect to slugs.
- [ ] 404 behavior correct for missing post/slug/id.

## Audio

- [ ] Inline play works.
- [ ] Route navigation does not stop playback.
- [ ] Only one active global track.
- [ ] Mini player state stays in sync.
- [ ] Seek/pause/volume work.
- [ ] Close stops/releases playback.
- [ ] Waveform failure does not break playback.
- [ ] Mobile fixed player does not cover critical UI.

## Admin/security

- [ ] `/admin` absent from unauthenticated public navigation.
- [ ] Wrong secret rejected.
- [ ] Default/insecure production secret rejected at startup.
- [ ] Session cookie secure settings verified over HTTPS.
- [ ] CSRF required for admin writes.
- [ ] Unauthorized post/upload/moderation/backup APIs rejected.
- [ ] Login rate limit active.
- [ ] Session expiry/logout tested.
- [ ] No secret/session token logged.

## Editor

- [ ] New long post can be written naturally without manipulating JSON.
- [ ] Bold/italic/link/inline-code formatting works.
- [ ] Heading/quote/divider works.
- [ ] Image/video/audio/file insertion occurs at intended position.
- [ ] Caption/alt/spoiler behavior preserved.
- [ ] Autosave/resume works.
- [ ] Preview matches public renderer.
- [ ] Existing v1 post can be edited and saved safely.
- [ ] Title edit does not unexpectedly change published slug.
- [ ] Mobile publishing checked on real phone.
- [ ] Double publish/update prevented.
- [ ] Delete confirmation clear.

## Comments/antispam

- [ ] Anonymous comment works.
- [ ] Optional name works.
- [ ] Reply works on mobile/desktop.
- [ ] Deep nesting remains readable on mobile.
- [ ] Comment composer easy to reach.
- [ ] Challenge/honeypot works.
- [ ] Burst/cooldown/duplicate defenses work.
- [ ] Suspicious comment pending path works.
- [ ] Admin approve/reject/unmute works.
- [ ] Raw IP is not shown.

## Performance

- [ ] Public initial bundle reviewed.
- [ ] Editor code not in public initial bundle.
- [ ] No per-card comment-preview browser N+1.
- [ ] Offscreen images/media are lazy/conservative.
- [ ] No obvious layout shift from media.
- [ ] Representative mobile performance checked.
- [ ] Representative tiny-VPS CPU/RAM behavior checked.
- [ ] Browser console clean in normal flows.
- [ ] Server logs clean in normal flows.

## RSS

- [ ] `/feed.xml` (or selected canonical route) valid.
- [ ] New posts appear automatically.
- [ ] Links use canonical slugs.
- [ ] No manual update step required.

## Deployment

- [ ] Go service unit installed/tested.
- [ ] Nuxt service unit installed/tested.
- [ ] Both restart on boot/failure as expected.
- [ ] Reverse proxy routes API/uploads/Nuxt correctly.
- [ ] Client IP derivation behind proxy verified.
- [ ] TLS works.
- [ ] Persistent data paths survive deploy.
- [ ] Health/smoke checks documented.
- [ ] Rollback commands documented and rehearsed.

## Cutover

- [ ] Announce/choose short maintenance window if needed.
- [ ] Stop v1 writes/process.
- [ ] Take final DB/uploads backup/sync.
- [ ] Deploy v2 release.
- [ ] Run migrations.
- [ ] Start API + web services.
- [ ] Smoke homepage/post/comments/admin/media.
- [ ] Verify old shared link.
- [ ] Verify external HTTPS domain.
- [ ] Keep v1 release + final backup untouched until v2 is proven stable.
