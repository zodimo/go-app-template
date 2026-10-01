## Why

The template ships agent/IDE scaffolding that belongs to *this* repo, not to a downstream project: `.opencode/` (opencode agent config, skills, plugins), `openspec/` (the spec-driven-development corpus), the machine-local `go-app-template.code-workspace`, and any `AGENTS.md`/`CLAUDE.md` agent-instruction files. A consumer who seeds their project by copying the tree (or whenever `gonew`'s `.go`+`go.mod`-only behaviour does not apply) inherits all of it, then has to manually hunt down and delete files they may not recognise. The clone flow currently has no step that offers to remove this scaffolding, so the "clean start" the template promises is not enforced anywhere.

The `setup` command is already the single mechanism that mutates a cloned tree (identity rewrite, plan/apply, `--dry-run`, idempotent). Adding a consent prompt there — rather than a Makefile target or a manual `rm` — keeps all clone-time mutation in one auditable place and lets the removal reuse the existing plan/apply machinery. The Cleanup prompt defaults to **Yes**, because a downstream project almost never wants the source repo's agent scaffolding, while `--dry-run` and the reported plan keep the action transparent and reversible-in-spirit (nothing is written until the plan is applied).

## What Changes

- **Add a Cleanup prompt to the `setup` wizard.** One combined, default-`Y` consent prompt — e.g. `Remove agent scaffolding (.opencode/, openspec/, *.code-workspace, AGENTS.md/CLAUDE.md)? [Y/n]` — collected alongside the four identity prompts and folded into the same `buildPlan`/`apply` flow.
- **Introduce the first deletion primitive.** A new `kindRemove` change kind, executed in `apply` via `os.RemoveAll` for directories and `os.Remove` for files. Every removal carries a `Desc` so `setup --dry-run` prints exactly what would be deleted.
- **Guard removals with `SkipIfMissing`.** Removal is a no-op when a path is already absent, so `setup` stays idempotent and safe to re-run. `.opencode/` in particular is git-ignored and untracked, so in many clones it simply will not exist.
- **Order removals last.** The removal changes are appended after the identity renames/writes, so a failure during re-identification never half-wipes the tree.
- **Add a repo-run guard.** Refuse to remove `openspec/` (and, defensively, the whole removal set) unless the workspace looks like a cloned template — specifically, unless `openspec/` is present *and* no uncommitted tracked changes exist in it — so running cleanup inside the authoring repo cannot destroy its own change history. (Exact guard recorded in design.md.)
- **Extend the removal set to a single, explicit list** rooted at the workspace: `.opencode/`, `openspec/`, files matching `*.code-workspace`, `AGENTS.md`, `CLAUDE.md`. `bin/`, `.go-cli/`, `*.log`, and generated config JSON stay out of scope — `.gitignore` regeneration already handles runtime state.

No breaking change to built binaries; `setup`'s existing prompt sequence gains one prompt and its plan gains removal entries.

## Capabilities

### New Capabilities
<!-- None: this extends existing clone/hygiene contracts. -->

### Modified Capabilities
- `project-scaffolding`: The `setup` command's described behaviour and its enumerated set of mutated artifacts gain the agent-artifact removal step (prompt + `kindRemove`), including the default-Yes consent and the repo-run guard.
- `template-hygiene`: A new requirement states that agent/IDE scaffolding (`.opencode/`, `openspec/`, `*.code-workspace`, `AGENTS.md`/`CLAUDE.md`) MUST NOT survive into a downstream project, and that `setup` MUST offer a default-Yes removal of it, executed idempotently and last.

## Impact

- **Code**: `internal/commands/setup.go` (new `kindRemove` kind + apply branch, `identityPaths` additions for the agent-artifact paths, `buildPlan` removal entries, `runSetup` prompt, repo-run guard), `internal/commands/setup_test.go` (removal plan, `--dry-run` reporting, idempotency, guard, default-Yes).
- **Docs**: `README.md` "Clone Flow" (Layer 2 gains the cleanup prompt), `CONTRIBUTING.md` "Template reuse" (same), and the `setup` command's `Long` help text.
- **Specs**: delta specs for `project-scaffolding` and `template-hygiene`.
- **Dependencies**: none added — stdlib `os` + existing `bufio` prompt helper.
- **Risk**: a destructive default (`Y`) on a path list that includes `openspec/`; mitigated by the repo-run guard, deletions-last ordering, `SkipIfMissing`, and full `--dry-run` transparency.
