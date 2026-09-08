# Frontend agent rules — Nuxt 4/Vue/TypeScript

This file extends `doc/AGENTS.md` for work under `frontend/`.

## 1. Frontend mission

Build a public site that feels like the same StereoDamage but dramatically more polished, mobile-native, shareable, and fast; and build a private editor that makes publishing long-form media posts easy.

Do not turn the rewrite into a template-driven “modern dashboard” redesign.

## 2. Required reading

Always read:

- `docs/PRODUCT.md`
- `docs/UX_SPEC.md`

Editor work:

- `docs/CONTENT_FORMAT.md`

Compatibility/state work:

- `docs/V1_REFERENCE.md`
- `docs/DATA_COMPATIBILITY.md`

Admin/auth UI work:

- `docs/SECURITY.md`

## 3. Nuxt rendering strategy

Public routes are SSR/universal by default.

Requirements:

- `/` HTML contains initial feed content;
- `/posts/:slug` HTML contains article content and metadata;
- do not show a client-only “Loading…” shell as the main public first render;
- internal Nuxt navigation remains SPA-like after hydration;
- admin can be configured client-only if editor dependencies/UX justify it.

Do not put backend business logic into Nitro merely because Nuxt has server routes.

## 4. TypeScript

- Use real typed API response models.
- Avoid `any` as a shortcut around API/content types.
- Keep generated/central API types simple; do not add a codegen platform unless there is a clear payoff.
- Runtime validation is still required where untrusted JSON enters critical content/editor paths.

## 5. Components and state

Prefer focused components.

Legitimate shared/global state:

- persistent audio playback;
- perhaps current theme/language;
- feed navigation cache/scroll restoration if Nuxt built-ins are insufficient.

Do not put every API response into Pinia. Add Pinia only if it clearly simplifies cross-route state; Nuxt `useState`/composables may be enough.

## 6. CSS/design

Preserve v1 personality.

Use project-owned CSS. A utility CSS tool is not automatically forbidden, but do not introduce one merely because it is common. A full component library is strongly discouraged.

Visual guardrails:

- conservative polish;
- compact old-web character;
- familiar StereoDamage colors/contrast/density;
- avoid glass/gradient/huge-radius SaaS look;
- avoid enormous whitespace/hero typography;
- avoid unnecessary motion;
- use CSS for visual behavior before JS animation.

If unsure, compare to v1 `styles.css` and `docs/UX_SPEC.md`.

## 7. Mobile-first

For every new public/admin component, design narrow viewport behavior first or at least concurrently.

Check:

- touch target size;
- no horizontal overflow;
- keyboard/viewport behavior for forms/editor;
- fixed audio player overlap;
- deep comments;
- media widths;
- dialogs/sheets;
- sticky headers/TOC;
- safe-area insets where fixed bottom UI exists.

Do not “fix mobile later” after a desktop implementation is already structurally committed.

## 8. Feed

- SSR first page.
- Infinite scroll after hydration.
- Cursor API.
- Preserve list/grid user preference from v1 localStorage.
- Comment previews come in feed data; no per-card API fanout.
- Keep loaded feed/scroll state when navigating into a post and back.
- Prevent accidental duplicate loads during fast observer triggers.
- Handle retry/end states quietly.

## 9. Long-read renderer

Build renderer from canonical structured blocks.

- Same renderer/components should power public article and admin preview.
- Rich inline marks render through controlled Vue components, not arbitrary `v-html`.
- Text column optimized for reading.
- Media can be wider where spec allows.
- TOC only when justified.
- Reader progress persists by post ID.
- Preserve/migrate v1 localStorage keys.

## 10. Comments

- plain text;
- optional name;
- anonymous default;
- replies without imageboard-specific numbering syntax;
- cap visual indentation on mobile;
- accessible composer near discussion start;
- comment count click from feed routes to `#comments` reliably;
- challenge/honeypot fields should remain invisible/non-disruptive to normal users.

## 11. Media

### Images

- responsive;
- lazy offscreen;
- alt/caption;
- lightweight viewer;
- avoid CLS where dimensions available.

### Video

Native controls are okay. Do not build a custom player without need.

### Audio

Implement one global audio playback controller/store outside page component lifetimes.

- route navigation must not destroy active playback;
- inline and mini controls bind to same state;
- only one track active;
- no duplicate player instances;
- restore volume preference;
- waveform enhancement must not block playback;
- editor/admin audio preview must not accidentally hijack public persistent state unless intentionally designed.

## 12. Share/meta

Use Nuxt SEO/head facilities so metadata is present in SSR HTML.

- canonical URL;
- OG/Twitter fields;
- preview media/default image;
- description from plain post text;
- native Web Share when supported;
- copy-link fallback.

Do not make social-specific JS SDK integrations.

## 13. i18n

Keep RU/EN UI.

- Reuse `stereoDamageLanguage` or migrate it once.
- All normal user-visible strings should be translated.
- Post content remains exactly authored; do not introduce translation workflow.
- Avoid locale-prefixed URLs for launch unless explicitly requested.

A lightweight Nuxt i18n solution is acceptable, but do not create routing complexity for content language.

## 14. Reader settings/localStorage

Preserve/migrate:

- `stereoDamageLanguage`;
- `stereoDamageTheme`;
- `stereoDamageFeedView`;
- `stereoDamageReadingProgress`;
- audio volume key.

Remove/ignore:

- `stereoDamageSmoothScroll`.

LocalStorage failures must not break rendering.

## 15. Admin visibility

Do not put an unauthenticated Admin link in public header/footer.

The owner can type `/admin` manually/bookmark it.

Authenticated admin conveniences inside admin pages are fine. Avoid leaking moderation/admin state into public initial data.

## 16. Editor

The editor must feel like writing, not manipulating blocks.

Preferred mature editor engine may be used client-only, but:

- lazy-load on admin/editor route;
- no editor package in public initial bundle;
- explicit StereoDamage block adapter;
- no canonical arbitrary HTML;
- mobile toolbar/input behavior tested;
- media upload inserts at cursor/intended block position;
- autosave quiet/reliable;
- preview uses public renderer.

Do not implement a rich-text engine from scratch unless explicitly approved.

## 17. Accessibility

Use semantic HTML even with retro styling.

- links for navigation, buttons for actions;
- labels;
- focus styles;
- keyboard viewer/dialog handling;
- Escape where conventional;
- alt text;
- reduced motion;
- no hover-only controls.

## 18. Frontend performance

- route-split admin/editor;
- avoid client hydration for static decorative components where unnecessary;
- lazy load below-fold media;
- avoid giant icon packs; inline/small icons only as needed;
- no blanket animation library;
- inspect public bundle before release;
- abort stale fetches when route/data changes can race;
- avoid watchers/computed loops doing heavy work on scroll;
- throttle/debounce reader-progress persistence sensibly.

## 19. Frontend definition of done additions

For UI tasks:

- SSR behavior checked where relevant;
- TypeScript checks pass;
- unit/component tests pass;
- critical E2E updated/pass;
- narrow mobile viewport manually/automatically checked;
- dark/light checked;
- RU/EN checked for new text;
- keyboard/focus basics checked;
- browser console has no new normal-flow errors;
- visual change remains consistent with StereoDamage identity.
