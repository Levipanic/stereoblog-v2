# How to run StereoDamage v2 with a coding agent

This file is for the project owner. The goal is to make the rewrite easy to supervise without keeping architecture details in your head.

## Before starting

Create a new v2 repository. It may be created from a GitHub fork for convenience, but treat it as a **new project**, not a continuation of the v1 architecture.

Recommended practical setup:

1. Create the v2 repo.
2. Copy this agent pack into it.
3. Keep `https://github.com/Levipanic/blog_proj` available as the v1 reference.
4. Put a **copy**, never the only copy, of the production `data/blog.db` and `uploads/` somewhere local for migration testing.
5. Never give an agent the real production `.env` or secret unless the environment genuinely requires it.

## The one-task loop

Use one backlog task at a time.

Suggested prompt:

```text
Read doc/AGENTS.md completely.
Then read the relevant nested agent files and docs linked by task <TASK-ID> in doc/BACKLOG.md.
Inspect the current code and the v1 reference where the task requires parity.
Implement only task <TASK-ID>.
Run all tests/checks required by that task.
Do not start later backlog tasks.
At the end, summarize changes, files, checks, and anything I should manually test.
```

Then manually inspect/test what matters and either:

- accept it and move to the next task; or
- tell the agent exactly what failed and keep working on the same task.

## Good owner prompts after implementation

For review:

```text
Review your implementation of <TASK-ID> against every acceptance criterion in doc/BACKLOG.md. Do not change code yet. Tell me which criteria are proven, how they are proven, and which are not.
```

For bug fixing:

```text
We are still on <TASK-ID>. This behavior is wrong: <what you observed>.
Find the cause, fix it without expanding scope, rerun the task checks, and summarize the fix.
```

For production-DB compatibility:

```text
Use this database only as a test fixture/copy. Never destructively reset it.
Run the compatibility checks required by <TASK-ID> and report row counts/invariants before and after migration.
```

## What to personally verify often

You do not need to inspect every implementation detail. Repeatedly check the things that are hardest for an agent to judge:

- Does the site still feel like StereoDamage rather than a generic Nuxt template?
- Is it fast on your actual phone and VPS-like hardware/network?
- Are long reads pleasant to read for 10–30 minutes?
- Does back-navigation return to the right point in the infinite feed?
- Is commenting effortless without an account?
- Does the admin editor make you want to publish rather than postpone publishing?
- Can old production posts/comments/media be opened correctly?
- Do old links redirect correctly?
- Does audio survive Nuxt route navigation cleanly?
- Does a backup actually restore on a clean copy?

## Suggested git rhythm

Keep changes easy to undo.

- One branch or clearly bounded commit series per backlog task.
- Do not mix multiple large backlog tasks in one commit.
- Tag known-good milestones, especially before database migrations and before production cutover.
- Keep production data out of git.

A useful commit message pattern:

```text
v2/TASK-ID: short description
```

## Production data safety

Never test a new migration for the first time on the only production DB.

Use this sequence:

```text
production backup
  -> local/test copy
  -> run v2 migration
  -> compatibility checks
  -> manual browsing/editing test
  -> restore-test backup
  -> only later production cutover
```

If an agent proposes `DROP TABLE`, destructive data conversion, or “reset the DB because schema differs”, reject it unless a specifically approved migration task explicitly requires it.

## When to let the agent make a decision

Let the agent decide ordinary coding details: naming, small helpers, exact test structure, minor component decomposition.

Ask for discussion first when it proposes:

- a new paid/external service;
- a large UI/framework dependency;
- a significant redesign;
- a new content model incompatible with old posts;
- a destructive migration;
- a different database/storage system;
- reader accounts/multi-author behavior;
- major deployment complexity.

## Recommended milestone checkpoints

Do a hands-on checkpoint at the end of every phase in `doc/BACKLOG.md`, not only at the final release.

Especially important checkpoints:

1. v1 database opens safely under Go.
2. public feed/post parity works.
3. new Nuxt UX works on phone.
4. editor can create/edit a real long read.
5. persistent audio survives navigation.
6. backup can restore a disposable installation.
7. full production-data rehearsal passes.
8. final cutover rehearsal passes.

## Definition of “v2 is worth shipping”

Do not ship merely because feature parity is reached.

v2 should feel like a genuine upgrade:

- all important v1 functionality is present;
- mobile reading and publishing are much better;
- first loads and navigation feel immediate;
- long reads are better presented;
- comments are easier to reach and use;
- sharing is first-class;
- the admin editor is substantially easier than v1;
- data migration/backup/restore are trustworthy;
- the visual identity still feels like the same personal site.
