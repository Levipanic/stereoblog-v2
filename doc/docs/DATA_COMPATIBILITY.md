# Data compatibility and migration contract

## Primary requirement

A copy of the current v1 production `data/blog.db` plus its `uploads/` directory must be attachable to v2 and produce the same historical blog content without manual record editing or data loss.

This is a release-blocking requirement.

## What “no data loss” includes

Preserve at minimum:

- every post row and `id`;
- titles;
- `blocks_json` semantic content;
- preview media;
- post like counts;
- creation timestamps;
- every comment and `id`;
- post/comment relationships;
- comment parent relationships;
- optional names/content/timestamps;
- comment like counts;
- moderation status/reason;
- antispam tables/records unless an explicit retention migration is later approved;
- like-event records that are still present;
- admin session rows may expire naturally but must not break migration;
- all referenced `/uploads/...` files;
- filenames/URLs of existing media.

## Golden rule

**Never solve a schema mismatch by dropping/resetting the production database.**

The v1 `db.js` contains historical reset behavior. It must not be copied to v2.

## Migration strategy

### Existing v1 database

On first v2 startup against a v1 DB:

1. Open DB carefully and identify schema.
2. Verify required v1 tables/columns sufficiently to run supported migrations.
3. Create/verify a consistent pre-migration backup according to implementation task.
4. Create migration metadata table if absent.
5. Apply ordered migrations.
6. Run integrity/sanity checks.
7. Start serving only if migration succeeded.

### Fresh database

A fresh install should build the same final schema through migrations, not through a separate divergent schema builder.

## Additive schema evolution

Preferred launch migrations include additions such as:

- migration metadata;
- post slug column/index;
- optional future media metadata tables;
- optional draft/support tables if needed.

Avoid changing the fundamental meaning/type of existing v1 columns unless absolutely necessary.

## Slug migration

Existing posts need slugs without losing IDs.

Requirements:

- deterministic backfill;
- unique collision handling;
- non-empty for every existing post after migration;
- stable across repeated migration runs;
- title edit after migration does not automatically mutate slug;
- ID remains available for legacy link resolution.

Slug generation must support the actual production titles (including Cyrillic). Use a tested deterministic policy. The admin may edit slugs for new posts before first publish.

## `blocks_json` migration policy

Do **not** bulk rewrite all old block JSON merely to match a new editor format.

- v2 reader supports legacy blocks directly.
- v2 editor adapter can load legacy blocks.
- a post is only rewritten into richer v2-compatible representation when the owner explicitly edits/saves it.
- text-like rich blocks retain a plain `text` fallback.

This drastically reduces migration blast radius.

## Media compatibility

- Keep existing `uploads/` as-is.
- Existing `/uploads/foo.jpg` must continue resolving.
- Do not rename files for aesthetics.
- New derivative files should use distinct paths/names and never overwrite originals unintentionally.
- Missing historical files should be handled gracefully in UI and reported during compatibility audit, not treated as a reason to mutate database references.

## Production-backup test fixture

The owner intends to test against a backup of the live DB repeatedly.

Provide an easy documented command/workflow such as:

```text
copy test DB/uploads into runtime fixture
-> run migrations
-> run compatibility audit command/test
-> start v2
```

The audit should report useful counts without exposing private secrets:

- posts before/after;
- comments before/after;
- likes_count sums or representative invariants;
- parent references valid;
- number of media references;
- number of missing referenced media files;
- migration version;
- SQLite integrity check result.

## Compatibility fixtures

Create sanitized/constructed fixtures that model v1 schema and content edge cases in git tests. Do not commit the owner's real production DB.

Fixtures should cover:

- normal text post;
- headings/quote/divider;
- each media kind;
- spoiler media;
- preview_media present and null;
- nested comments;
- optional/empty name;
- pending/rejected comments;
- likes;
- Cyrillic title/content;
- timestamps;
- malformed/unknown block entry alongside valid entries (reader must fail safely).

The owner can additionally run tests against an uncommitted real backup.

## Database backup before migrations

A production-safe approach is preferred:

- use SQLite backup API/consistent snapshot rather than raw file copy while writes are active;
- name it clearly, e.g. `data/backups/pre-v2-migration-<timestamp>.db`;
- do not overwrite an existing backup;
- verify backup opens before proceeding where practical;
- avoid accumulating unbounded automatic backups forever; document cleanup policy rather than silently deleting recent safety backups.

Production startup writes verified snapshots to `<database directory>/backups/pre-v2-migration-<UTC timestamp>-<unique suffix>/blog.db` before pending migrations on an existing database. These snapshots are never overwritten or deleted automatically; after an external backup and restore check, the owner may remove obsolete local snapshots during routine maintenance.

If automatic backup cannot be made safely, startup should fail with a clear instruction rather than applying risky migrations blindly in production mode.

## Rollback philosophy

The preferred rollback during development is operational, not reverse migration:

- keep untouched source backup;
- test on copies;
- if v2 migration/test fails, discard the migrated copy and recreate it from backup.

Do not write complicated down-migrations just to appear sophisticated. Data safety is more important than theoretical reversibility.

## Legacy public URLs

Data compatibility includes link compatibility.

Request:

```text
/post.html?id=123
```

should:

1. resolve post ID 123;
2. find its canonical slug;
3. redirect to `/posts/<slug>`;
4. preserve optional `#comments` or relevant fragment behavior where feasible (fragment is browser-side and may require frontend handling).

Invalid/nonexistent IDs should return a proper not-found result, not redirect to home.

## LocalStorage compatibility

Because the new version will replace v1 on the same domain, preserve or explicitly migrate:

- language;
- theme;
- feed list/grid preference;
- reader progress by post ID;
- audio volume.

The obsolete smooth-scroll key can simply be ignored/removed by a one-time frontend cleanup.

## Release proof

Before production cutover, produce a compatibility report from a recent production backup that confirms:

- SQLite integrity passes;
- row counts expected/preserved;
- migrations complete;
- all representative posts render;
- historical comment trees render;
- historical media resolves (with any known missing files listed);
- legacy links redirect;
- old localStorage keys are honored/migrated;
- edit and save of one cloned old post does not corrupt content;
- backup/restore of migrated v2 data works.
