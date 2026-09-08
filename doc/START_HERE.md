# Start here — StereoDamage v2

If you just created the v2 repo and want to start coding with an agent, do this:

## 1. Put this whole pack under `doc/`

Expected important files:

```text
doc/AGENTS.md
doc/BACKLOG.md
doc/AGENT_WORKFLOW.md
doc/backend/AGENTS.md
doc/frontend/AGENTS.md
doc/docs/*
```

Do not cherry-pick only `BACKLOG.md`; the nested context files are what stop an agent from forgetting product/data constraints.

## 2. Give the coding agent this first prompt

```text
Read doc/AGENTS.md completely.
Read doc/AGENT_WORKFLOW.md and the Phase 0 introduction in doc/BACKLOG.md.
Do not implement anything yet.
Summarize the project mission, non-negotiable constraints, target architecture, data-compatibility rules, and the task workflow in your own words.
Then tell me what task V2-001 will do and what it explicitly must not do.
```

Check that the summary matches what you want. This is a cheap way to catch an agent that ignored the files before it starts writing code.

## 3. Start task V2-001

Then use:

```text
Implement only V2-001 from doc/BACKLOG.md.
Read every file that V2-001 tells you to read first.
Follow doc/AGENTS.md and the relevant nested agent files under doc/.
Run the required checks.
At the end, give me a compact review report: changed files, acceptance criteria, commands/checks, and what I should manually verify.
Do not start V2-002.
```

## 4. Repeat one task at a time

The phase order is intentional:

```text
Phase 0  project skeleton
Phase 1  prove old SQLite compatibility
Phase 2  Go backend parity
Phase 3  Nuxt feed/infinite navigation
Phase 4  long-read/media/comments/sharing
Phase 5  new mobile-friendly admin/editor
Phase 6  lightweight polish/RSS/media
Phase 7  performance/security/deploy/cutover
```

Do not skip Phase 1 and build the pretty frontend first. The production database is the most valuable thing in this project and compatibility should be proven early.

## 5. At every phase gate

Stop coding and personally click through the result.

You mainly judge things the agent cannot reliably judge for you:

- does it still feel like StereoDamage;
- does it feel instant;
- is it actually good on your phone;
- is reading a long post comfortable;
- is posting from admin less annoying;
- did any old data/media disappear;
- did the agent add unnecessary architecture/dependencies.

## 6. Production DB rule

Never hand a new migration the only production database.

Use:

```text
production -> backup -> disposable copy -> v2 migration/tests
```

The backlog eventually creates an audit command specifically for this workflow.

## 7. If an agent starts overengineering

Point it back to these sections of `doc/AGENTS.md`:

- Non-negotiable product truths
- No-overengineering rule
- Visual design guardrails
- When the agent should ask instead of decide

A useful correction prompt:

```text
This is a personal single-author blog on a tiny VPS. Re-read the no-overengineering and product-invariant sections in doc/AGENTS.md. Simplify the current solution to the minimum conventional implementation needed for the active backlog task. Do not remove required behavior.
```

## 8. v1 reference

The old project remains the behavioral/data reference:

`https://github.com/Levipanic/blog_proj`

The agent should inspect it whenever a parity task says so. It should **not** copy its architecture mechanically.
