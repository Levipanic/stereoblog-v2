# Product brief — StereoDamage v2

## Product sentence

StereoDamage is a single-author personal long-form blog with first-class media and anonymous comments, built to feel like a small piece of the old personal web while providing modern speed, mobile ergonomics, sharing, and authoring quality.

## Why v2 exists

v1 already proves the concept and contains many useful features. Its limitations are mostly accumulated implementation and UX friction:

- Express backend is concentrated in a very large server file.
- Public vanilla JS is large and imperative.
- Multipage navigation makes persistent media state fragile.
- Feed rendering performs avoidable client request fan-out for comment previews.
- Post sharing uses legacy query-string URLs and lacks first-class social metadata/share UX.
- Long reads are presented within UI structures that can feel more like a large feed card than a dedicated reading surface.
- Admin authoring exposes the internal block mechanics too directly and makes media insertion a multi-step workflow.
- Responsive behavior exists but was not designed mobile-first.

v2 is a product-quality rewrite, not a feature explosion.

## Audience

Primary public audience:

- friends, acquaintances, and people receiving a shared link;
- readers who may browse the timeline after reading one post;
- readers on phones and lower-powered devices;
- people who want to leave a quick comment without creating an account.

Primary private user:

- exactly one author/administrator.

## Core public jobs

A reader should be able to:

1. open a shared long-read link and see meaningful content immediately;
2. understand whose personal site this is;
3. read comfortably for a long session;
4. view/play attached media without fighting the page;
5. start audio and keep it playing while navigating the site;
6. see a small amount of social activity under feed posts;
7. jump directly to discussion when desired;
8. comment anonymously or optionally add a name;
9. like posts/comments;
10. share/copy a clean canonical URL;
11. browse older posts through a long infinite chronological feed;
12. return from a post to the exact previous feed position.

## Core author jobs

The owner should be able to:

1. log into a private admin route securely;
2. start writing immediately;
3. format text using familiar rich-text controls without memorizing Markdown;
4. insert images/video/audio/files at the current writing position;
5. publish comfortably from a phone;
6. preview exactly how the post will render publicly;
7. autosave and resume unfinished work;
8. edit existing v1 and v2 posts;
9. choose/adjust preview media and slug when needed;
10. moderate pending/spam comments;
11. download a complete portable backup of database + uploads;
12. recover the site from that backup without cloud storage.

## Product principles

### Personal, not platform-like

The page should feel authored. Do not add generic community machinery.

### Content before chrome

Navigation and controls support the writing and media rather than competing with them.

### Long-form first

The design assumes posts can be substantial. Reading comfort matters more than optimizing for one-line posts.

### Frictionless participation

No reader account. Commenting should feel closer to sending a simple message than filling out a registration form.

### Fast enough to disappear

Loading states should be rare and short. Avoid interface decisions that remind the reader they are operating a heavy web app.

### Indie-web character

A little “webby”, personal, compact, forum-like, or old-fashioned is desirable if usability remains strong. Sterile product-design sameness is not.

### Ownership and portability

Posts live in SQLite and media files live on disk. The owner can back everything up and move it to another cheap VPS.

## Explicit non-goals for launch

Do not build:

- reader registration/login;
- multiple authors;
- profiles/follows;
- notifications;
- comment media attachments;
- recommendations/algorithmic feed;
- search;
- archive section;
- tags/categories as a required navigation system;
- analytics/ads;
- paid cloud dependencies;
- S3 requirement;
- realtime chat;
- PWA/offline mode unless later explicitly requested;
- a generic CMS/plugin ecosystem.

## Success criteria

v2 is successful when the owner can replace v1 in one cutover and:

- connect a production-data copy with no lost posts/comments/likes/media;
- preserve old shared links;
- publish a media-rich long read from desktop and phone with substantially less friction;
- read it comfortably on phone and desktop;
- share it with correct preview metadata;
- navigate feed/post/feed without losing place;
- keep audio playing across route navigation;
- leave anonymous comments easily;
- download and validate a full backup;
- run the whole system cheaply on the existing class of tiny VPS.
