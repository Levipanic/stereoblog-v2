# Content format and rich-text authoring contract

## Goal

The author should experience a simple rich-text writing surface. The database should continue to store StereoDamage-owned structured blocks compatible with v1 data.

Product shorthand:

> **Rich-text authoring, block-based storage.**

The author should not need to think in JSON blocks or memorize Markdown syntax during normal posting.

## Canonical storage

`posts.blocks_json` remains the canonical post body for v2 launch.

Old v1 block arrays must render directly without a bulk conversion step.

Do not store arbitrary HTML as canonical content.
Do not store editor-library JSON as the only canonical content.

The editor is an adapter over the StereoDamage document model.

```text
StereoDamage blocks -> editor adapter -> editor state
editor state -> adapter/validator -> StereoDamage blocks
```

## Legacy block compatibility

v1 blocks remain valid:

- paragraph
- heading (levels 1–3 in legacy data)
- quote
- divider
- media (image/gif/video/audio/file + spoiler/name/alt/caption)

The v2 parser must be tolerant of legacy optional fields and must not destroy unknown JSON in the database merely because the public renderer cannot display it. Read paths may normalize for output; write paths should only rewrite content when the author explicitly saves that post.

## Rich text inside text blocks

For v2, paragraph/heading/quote blocks may gain a richer inline representation while keeping a plain-text fallback.

Conceptual example:

```json
{
  "type": "paragraph",
  "text": "I wrote a very important thing.",
  "content": [
    { "type": "text", "text": "I wrote a " },
    {
      "type": "text",
      "text": "very important",
      "marks": [{ "type": "bold" }]
    },
    { "type": "text", "text": " thing." }
  ]
}
```

The exact inline schema may be refined during the editor task, but it must obey these rules:

1. `text` remains a semantic plain-text fallback for text-like blocks.
2. The fallback is generated from rich content whenever saved.
3. v2 prefers rich content when valid and falls back to `text` when not.
4. Existing v1 blocks containing only `text` edit and render normally.
5. Rich content is sanitized/validated server-side; browser output is never trusted HTML.
6. Marks/nodes outside the supported set are rejected or safely normalized, not blindly stored/rendered.

Keeping `text` means an emergency rollback to v1 can still show the textual substance of rich v2 paragraph/heading/quote blocks even if it cannot reproduce inline formatting.

## Initial rich-text feature set

Launch editor should prioritize common long-read needs and avoid becoming Google Docs.

Required or high-priority:

- paragraphs;
- bold;
- italic;
- link;
- inline code;
- headings;
- quote;
- divider;
- image;
- GIF;
- video;
- audio;
- file attachment;
- media caption/alt text;
- media spoiler;
- undo/redo through the editor engine;
- sensible paste behavior.

Lists are useful but must not force a destructive or awkward content-model change. Implement them only when the storage adapter has a clean StereoDamage-owned representation and legacy fallback strategy. They are not worth blocking the core editor.

Explicitly out of scope unless later requested:

- arbitrary fonts;
- font sizes;
- text colors/highlights as a design tool;
- arbitrary alignment;
- tables;
- embedded scripts/iframes;
- arbitrary HTML;
- complex nested document structures;
- collaboration/multi-user cursors.

## Editor engine

Preferred direction: use a mature Vue-friendly rich-text engine (Tiptap/ProseMirror ecosystem is a strong candidate) instead of building contenteditable behavior from scratch.

However:

- editor packages must be lazy/client-only and must not enter public-route bundles;
- the editor's internal JSON is not the database contract;
- implement explicit conversion functions with tests;
- do not import a giant extension bundle merely for convenience;
- use only extensions required by the agreed feature set.

If the preferred editor has a material blocker, the agent should document the blocker before switching engines.

## Authoring UX

Normal writing should behave like a familiar text editor:

- click/tap editor and type;
- Enter makes the next paragraph naturally;
- selection shows compact formatting controls (or a mobile-appropriate toolbar);
- links can be added without Markdown syntax;
- `+` / insert control in an empty/current block opens media/structural inserts;
- paste plain text cleanly;
- paste a URL over selected text to create a link when feasible;
- desktop image paste/drop may upload directly;
- mobile “Photo” uses the native file/photo picker;
- uploaded media appears at the intended insertion point immediately;
- block ordering is manageable on touch without requiring HTML5 drag-and-drop.

The block structure should be mostly invisible during ordinary writing.

## Preview

Preview must use the same post renderer/components as the public article whenever practical.

Do not maintain a second “approximately similar” renderer in the admin that diverges over time.

Preview should show:

- title;
- blocks;
- selected/derived preview media where relevant;
- dark/light theme compatibility;
- realistic article width and media treatment.

## Drafts

Autosave is mandatory for the new editor experience.

Minimum behavior:

- local draft recovery works even before server-side draft persistence is introduced;
- editing an existing post does not accidentally publish partial autosave changes;
- explicit Publish/Update remains the commit point for public content;
- closing/reopening admin should recover an unfinished new post;
- autosave status should be quiet but visible (“Saved” / error when needed).

Do not make the author manually press “save draft” after every writing session.

## Slugs and editor behavior

- Title remains required for v2 launch because v1 schema requires it and the site is long-form-first.
- A slug is generated from title for new posts.
- Author may edit the slug before first publish.
- Once published, changing the title must not automatically change the slug.
- Existing posts receive deterministic migrated slugs without changing IDs.
- Do not expose slug controls prominently during normal writing; keep them in a compact post-settings area.

## Media insertion contract

The old workflow “upload file -> then add last upload as media block” must disappear.

Target:

```text
place cursor / insert point
-> choose Photo/Video/Audio/File
-> choose file
-> upload progress
-> media appears in document at that point
```

The upload result should carry enough metadata to construct the existing media block shape.

## Validation

Validation happens at multiple levels:

- editor prevents obviously unsupported operations;
- frontend adapter validates before request;
- Gin validates all canonical blocks/inline marks independently;
- renderer never trusts arbitrary markup.

Server validation is authoritative.

## Round-trip tests

Maintain fixtures proving:

1. v1 block JSON -> editor document -> save -> equivalent readable content;
2. rich v2 paragraph -> save -> load -> formatting preserved;
3. plain fallback text matches rich content semantic text;
4. unsupported inline markup is rejected/normalized safely;
5. media/spoiler metadata survives editing;
6. editing a legacy post does not silently remove untouched blocks;
7. public renderer and admin preview agree on representative fixtures.
