# UX specification — StereoDamage v2

This document describes intended behavior and experience. It is not a mandate for a pixel-perfect redesign. Preserve v1 identity and iterate conservatively.

## 1. Global experience

### Desired feeling

- personal site, not product landing page;
- old Twitter circa 2012 and old personal web as influences, not a literal clone;
- slightly forum-like / “hidden corner of the internet” character;
- compact but readable;
- modern ergonomics without generic modern visual clichés;
- almost no perceived loading.

### Navigation

Public navigation should remain minimal.

The site primarily has:

- timeline `/`;
- article `/posts/:slug`;
- lightweight settings/theme/language controls;
- share actions;
- RSS endpoint (not necessarily a large nav item).

Do not add search/archive/tags navigation in v2 launch.

The admin route must not be advertised to unauthenticated public readers.

## 2. Feed/timeline

### Purpose

The feed is a single reverse-chronological stream of long-form posts. It is not a social recommendation feed.

### Preserve

- chronological ordering;
- list view;
- grid view (user explicitly wants existing features retained except smooth-scroll setting);
- preview text;
- preview/cover media;
- likes;
- reading time/progress status;
- a small number of visible comment previews;
- link into full post/discussion.

### Infinite scroll

Implement intentionally:

- first page SSR renders immediately;
- subsequent pages load via cursor-based API and IntersectionObserver/sentinel;
- loading must not duplicate/skip posts when new content appears;
- there is a clear but unobtrusive end state;
- failure offers a retry rather than silently stopping;
- route navigation to a post and Back must restore the previous loaded feed items and scroll position;
- direct reload at `/` does not need to recreate an arbitrarily deep historical session, but same-session navigation should feel reliable;
- infinite feed must not cause runaway DOM/resource usage in normal browsing. If windowing is later necessary, introduce only after measurement.

### Post click targets

- Title/main post area opens canonical `/posts/:slug`.
- Comment/reply count opens `/posts/:slug#comments`.
- The comments anchor flow must wait for/render the section correctly and land at a useful position even for very long posts.
- Likes should not accidentally trigger navigation.

### Comment previews

Show roughly the current spirit: a small teaser of discussion beneath the post.

- Desktop can show about 2 visible comments.
- Mobile can show about 1 if space is tight.
- Do not fetch comment previews with a separate browser request for every feed post.
- Prefer an API response that includes the needed preview comments/counts efficiently.
- Preview comments are invitations to discussion, not mini full threads.

## 3. Article/long-read page

### Reading mode

Opening a post should visually shift from browsing to reading.

Keep the site identity/header compact, but reduce chrome around the article.

The article body should not feel like a giant card nested inside multiple panels. Use page hierarchy/spacing/typography rather than decorative containers wherever possible.

### Typography

Optimize for 10–30 minute reading sessions:

- comfortable line length;
- sufficient line height;
- clear heading hierarchy;
- readable contrast in both themes;
- no tiny desktop-derived body text on mobile;
- avoid excessive width on large screens;
- allow media to use wider widths than text when appropriate.

### Media widths

Support visually coherent cases such as:

- normal-width media aligned to text;
- wider landscape images when useful;
- full available width on phone without awkward padding;
- vertical images constrained sensibly;
- captions directly associated with media.

Do not invent complex editorial layout controls for v2 launch.

### TOC

Preserve the v1 concept, but keep it quiet.

- only show when enough headings/length justify it;
- desktop may use a side/sticky TOC if it does not crowd reading;
- mobile should use a compact disclosure/sheet/action, not a permanently large panel;
- active section indication is useful;
- anchors should be stable enough for in-page navigation.

### Reader progress

Preserve local reading progress.

- continue-reading prompt should be subtle;
- progress/status in feed should be informative but not visually dominant;
- completion threshold can remain near v1 behavior unless measured otherwise;
- preserve v1 progress localStorage by post ID where possible;
- smooth-scroll preference is removed; browser/default motion should respect reduced-motion preferences.

## 4. Sharing and URLs

Sharing is a first-class use case.

### Canonical URLs

New canonical format:

```text
/posts/why-old-web-was-better
```

Legacy:

```text
/post.html?id=123
```

must redirect to canonical slug URL when post 123 exists.

Use permanent redirect only after behavior is stable; during development/test, status may be selected deliberately. Production should preserve SEO/link equity appropriately.

### Share UI

- Mobile: use Web Share API when available.
- Desktop/fallback: Copy link.
- Keep share menu small; do not add a row of social-network-specific buttons.
- Show brief copy success feedback.

### Metadata

Every public post response must have server-rendered:

- `<title>`;
- description derived from content;
- canonical URL;
- Open Graph title/description/url/type;
- appropriate image when available;
- Twitter card-compatible metadata;
- author/site name as appropriate.

Primary OG image strategy:

1. selected/preview image if suitable;
2. otherwise default site OG asset;
3. custom generated text-based OG cards can be added as a later sharing polish task if they do not add client runtime weight.

## 5. Comments

### Philosophy

Borrow the low-friction participation principle of anonymous boards without copying imageboard visual/semantic conventions.

Do not add post numbers, tripcodes, `>>123` syntax, or comment media for launch.

### Composer

- Anonymous/default name is the easiest path.
- Name field is optional.
- Optionally remembering the name locally is acceptable if simple and transparent.
- Primary composer should be easy to reach near the start of the discussion; reader should not have to scroll through hundreds of comments to find the form.
- Replying to a comment should set clear reply context without spawning a huge nested form UI.
- On mobile, composer controls must be thumb-friendly and keyboard behavior must not break layout.

### Thread display

- Preserve parent/child replies.
- Keep nesting visually understandable but prevent deep indentation from crushing mobile width.
- Use a maximum visual indentation and/or flatten presentation depth while preserving relationships.
- Comment text is plain/safe text.
- Like/reply actions are compact.
- Pending moderation response should tell the user the comment was accepted for review rather than pretending it failed.

### Anchor flow

`/posts/:slug#comments` should land at the discussion header/composer, not somewhere arbitrary after layout shift.

## 6. Media UX

### Images/GIF

- Responsive sizing.
- Offscreen lazy loading.
- Correct aspect-ratio reservation when dimensions are known to reduce layout shift.
- Tap/click can open a lightweight viewer/lightbox for images.
- Viewer supports escape/backdrop close on desktop and sane touch behavior on mobile.
- Do not ship a huge gallery framework unless needed.

### Video

- Native controls are acceptable unless there is a concrete design reason to wrap them.
- Avoid autoplay.
- Preload conservatively.

### Audio

Persistent audio is special.

Desired lifecycle:

```text
play inline inside article
-> continue reading
-> navigate through Nuxt
-> audio keeps playing
-> compact global player remains available
-> user pauses/stops/closes explicitly
```

Rules:

- only one global active track at a time for launch;
- inline controls and persistent player reflect the same playback state;
- route changes must not spawn duplicate `<audio>` elements;
- user gesture/autoplay browser rules are respected;
- volume preference is local and can reuse/migrate v1 key;
- waveform is optional progressive enhancement; playback must work if waveform generation fails;
- do not decode huge audio files eagerly just to draw a waveform;
- closing the mini player stops/releases playback deliberately.

## 7. Settings

Preserve:

- theme;
- RU/EN UI;
- list/grid feed view;
- reset reader preferences/progress if still useful.

Remove:

- smooth-scroll setting.

Settings should remain secondary and compact. Do not turn them into an application settings dashboard.

## 8. Admin UX

Admin is a first-class product for the owner but invisible to normal readers.

### Login

Simple screen:

- one secret/password input;
- sign in;
- optional “remember this device” only if session semantics are explicitly implemented safely;
- no unnecessary OAuth/email provider infrastructure.

### Admin home

Prioritize:

1. New post / resume draft.
2. Existing posts/edit.
3. Moderation when pending items exist.
4. Backup.

Do not make antispam logs dominate daily authoring.

### Editor

See `docs/CONTENT_FORMAT.md`.

Desired feeling: a lightweight writing app, not a block-database form.

Mobile authoring is required:

- title input comfortable;
- editor usable with phone keyboard;
- formatting controls do not cover text;
- media picker works with native photo/file picker;
- reorder/edit media without desktop-only drag behavior;
- preview and publish are reachable without awkward scrolling.

### Preview/publish

- Preview must use real public renderer.
- Publish/update action should clearly indicate success/failure.
- Prevent accidental double publish.
- Destructive delete requires deliberate confirmation.
- Editing a published post must not unexpectedly change its canonical slug.

### Backup

Admin should provide a simple “Download full backup” action producing DB + uploads + manifest. It must not include secrets by default.

## 9. Accessibility baseline

The style can be unusual; interaction should still be robust.

- semantic headings;
- real buttons/links;
- visible focus;
- keyboard-operable dialogs/viewer;
- `alt` handling for images;
- labels for form controls;
- sufficient contrast;
- `prefers-reduced-motion` respected;
- no information available only through hover;
- touch targets large enough on mobile.

## 10. Performance experience/budgets

Exact numbers should be measured on release hardware, but design to these targets:

- SSR public HTML should contain the actual first-view content, not a loading shell.
- Keep public initial JS deliberately small; editor/admin code must be route-split out.
- Avoid large UI frameworks.
- No browser N+1 comment-preview fanout.
- First feed API/render should not need all historical posts.
- Offscreen media should not all download eagerly.
- Aim for good Core Web Vitals on representative mobile throttling; investigate any obvious LCP/CLS/INP regression before release.
- A page should remain usable if waveform, localStorage, or optional enhancement fails.

Do not chase a vanity Lighthouse 100 at the expense of correct UX, but do not accept avoidable heaviness.
