## Context

The template's clone flow has three logical layers, and this change adds work to Layer 2:

- **Layer 1 — `gonew`** rewrites `.go` files and `go.mod` only. It does not carry `.opencode/`, `openspec/`, or `*.code-workspace` into the clone, and root `.opencode/` is additionally git-ignored and untracked.
- **Layer 2 — `setup`** (`internal/commands/setup.go`) runs inside the clone and is the *only* code that mutates a cloned tree. It rewrites `internal/identity/identity.go`, renames the three `cmd/cli*` directories, rewrites cobra `Use:` strings and panic tags, rewrites the Makefile's `BIN_TRIO` line, and regenerates `.gitignore`.
- **Layer 3 — hygiene** currently has no runtime mechanism: the `template-hygiene` spec states a policy ("no machine-local or generated runtime artifacts are committed") but nothing removes agent/IDE scaffolding.

The relevant machinery in `setup.go` is a pure `buildPlan(root, cur, next) → plan{[]change}` plus an `apply(plan, dryRun)` that performs the I/O. `change` has three kinds today (`kindRename`, `kindWrite`, `kindRewrite`) and two guard fields (`SkipIfMissing`, `ExpectUnique`). **There is no deletion primitive anywhere in the repo** (`os.Remove`/`os.RemoveAll` do not appear in `internal/`). Prompts use a stdlib `bufio.Scanner` helper (`prompt`) that returns a default on empty input or EOF, with no third-party TUI dependency.

Constraints carried forward from the two archived clone/hygiene changes:

- `buildPlan` MUST stay pure (no I/O); all mutation belongs in `apply`.
- Every change MUST carry a `Desc` so `setup --dry-run` can print the plan.
- `setup` MUST be idempotent and re-runnable.
- Absence of a convenience artifact MUST be tolerated, not fatal (the `D3` convention established for a missing Makefile).

## Goals / Non-Goals

**Goals:**
- Add a single, default-`Y` consent prompt to the `setup` wizard that removes the agent/IDE scaffolding that belongs to this repo and not to a downstream project: `.opencode/`, `openspec/`, `*.code-workspace`, `AGENTS.md`, `CLAUDE.md`.
- Introduce a first-class deletion primitive (`kindRemove`) that fits the existing plan/apply model, prints in `--dry-run`, and is idempotent via `SkipIfMissing`.
- Guarantee a destructive default cannot destroy this repo's own `openspec/` history by adding a repo-run guard.
- Keep all clone-time mutation inside `setup` (no Makefile target, no manual `rm`).

**Non-Goals:**
- Removing runtime/build artifacts — `bin/`, `.go-cli/`, `*.log`, and generated config JSON. `.gitignore` regeneration already covers runtime state, and no user selected `bin/`.
- Changing `gonew`'s behaviour or replacing it; the step is complementary and defensive.
- Adding a third-party TUI/prompt dependency (survey, bubbletea, promptui, huh).
- Refactoring `setup`'s identity handling or the `VERSION`/`CURRENT_VERSION` Makefile question (still out of scope from the earlier change).
- Making deletion opt-in-by-default (the chosen default is `Y`).

## Decisions

### D1 — Extend `setup` with a new `kindRemove`; do not add a separate command or Makefile target

Add `kindRemove` to the `changeKind` enum and a branch in `apply`:

```go
case kindRemove:
    if c.SkipIfMissing {
        if _, err := os.Stat(c.Path); errors.Is(err, os.ErrNotExist) {
            continue
        }
    }
    if err := os.RemoveAll(c.Path); err != nil {
        return fmt.Errorf("remove %s: %w", c.Path, err)
    }
```

`os.RemoveAll` handles both files and directories (it is not an error on a missing path, but the explicit `SkipIfMissing` stat keeps the "would: ..." dry-run and the "skipped" semantics honest). The path to remove is carried in `change.Path`; `From`/`To`/`Old`/`New`/`Content` stay unused.

**Alternatives considered:**
- *Separate `cli prune` command* — rejected: splits clone-time mutation across two commands and duplicates the plan/apply/prompt/reporting plumbing.
- *Makefile `clean-template` target* — rejected: contradicts the earlier decision to keep the Makefile a non-mutation convenience artifact.
- *`setup --clean` flag* — rejected for the primary path: the wizard is the documented, discoverable entry point; a flag can be added later without changing this design.

### D2 — One combined, default-Yes prompt; fold the answer into the plan

Add a boolean input collected in `runSetup` alongside the four identity prompts, using a sibling helper to `prompt` (the existing helper returns a string default; a `promptYesNo(r, label, def bool) bool` reads one line and treats empty/EOF as `def`):

```
Remove agent scaffolding (.opencode/, openspec/, *.code-workspace, AGENTS.md/CLAUDE.md)? [Y/n]:
```

Default `true`. Shelling `< /dev/null` therefore accepts the default (Yes), consistent with how the other prompts accept defaults.

**Why default Yes:** the target audience is a downstream consumer whose project should not carry the source repo's agent scaffolding. A default-No prompt would leave the scaffolding in place for anyone who just presses Enter, defeating the purpose. Safety is provided by the repo-run guard (D4) and `--dry-run` (D3), not by an inconvenient default.

**Alternatives considered:**
- *Two separate prompts* (one for `.opencode/`+agent files, one for `openspec/`) — rejected by the user in favour of a single combined consent.
- *Typed confirmation ("type yes")* — rejected by the user; default-Yes was chosen explicitly.

### D3 — Removals are planned last, reported in `--dry-run`, and idempotent

`buildPlan` appends removal changes **after** all identity changes (rename/write/rewrite), so if re-identification fails mid-apply the tree is not left half-wiped. Removal entries are only added when the operator consented; `SkipIfMissing: true` makes a re-run a no-op once the paths are gone. Because `apply` already prints `would: <Desc>` on dry-run and `applied: <Desc>` otherwise, removals are transparent with no extra reporting code.

The removal set is a fixed, explicit list resolved against `root`, not a glob walk:

| Target | Kind | Notes |
|---|---|---|
| `.opencode/` | directory | git-ignored + untracked; often already absent |
| `openspec/` | directory | tracked; the guard in D4 protects this repo |
| `*.code-workspace` | file glob | resolve via `filepath.Glob`; may match zero files |
| `AGENTS.md` | file | typically absent today |
| `CLAUDE.md` | file | typically absent today |

A small helper resolves the set for a given `root` and emits one `kindRemove` per existing path (plus the glob expansion). `identityPaths` is *not* the right home for these — it is identity-derived and computed for both `cur` and `next`; agent-artifact paths are identity-independent. A dedicated `agentArtifactPaths(root) []string` keeps that distinction clear.

**Alternatives considered:**
- *Walk the tree deleting anything matching `**/AGENTS.md`* — rejected: over-broad and surprising; a fixed root-level list is auditable.
- *Put paths in `identityPaths`* — rejected: they are not identity-derived; conflating the two would recompute them twice and muddy the struct's contract.

### D4 — Repo-run guard: refuse to remove `openspec/` inside the authoring repo

Deleting `openspec/` inside *this* repository would be self-destructive. Two cheap signals decide whether the tree is safe to clean:

1. The removal set is only offered when the workspace looks like a **cloned template**, and
2. the guard is scoped to the risky entry (`openspec/`) so that removing just `.opencode/`/agent files remains available.

Concrete rule (recorded here, refined during implementation):

- If `openspec/` contains any **git-tracked file with uncommitted modifications** (i.e. `git status --porcelain -- openspec/` is non-empty), treat the workspace as *the authoring repo* and **skip the `openspec/` removal** (with a clear stderr warning), while still allowing the other targets.
- If `git` is unavailable or the directory is not a repo, fall back to treating a present `openspec/` as removable but **require the operator's explicit consent** (the prompt answer).

This keeps the guard simple, offline-tolerant, and aligned with the existing `SkipIfMissing` "tolerate absence" convention.

**Alternatives considered:**
- *Marker file* (e.g. `.template`) — rejected: adds a new artifact and a new thing to maintain; git state is already present and accurate.
- *Refuse the whole cleanup when `openspec/` is dirty* — rejected: too coarse; it would block removing `.opencode/` in a repo that is legitimately being authored.

### D5 — Prompt/help text and docs stay in sync

The `setupCmd.Long` help, `README.md` "Clone Flow" (Layer 2), and `CONTRIBUTING.md` "Template reuse" each gain a sentence describing the cleanup prompt, its default, and the removal set. This mirrors the documentation obligation the two prior changes carried.

## Risks / Trade-offs

- [Destructive default (`Y`) deletes `openspec/` in a repo that wants to keep it] → Mitigation: repo-run guard (D4) blocks the authoring repo; `--dry-run` prints the full plan first; the consumer can answer `n`.
- [`os.RemoveAll` on a mistyped path] → Mitigation: paths are fixed literals joined to `root` (no user input feeds a path); the glob for `*.code-workspace` is anchored at `root` and uses `filepath.Glob` (no recursion).
- [Removal breaks builds if `openspec/` or agent files are imported by Go code] → Verified: no `.go` file imports them; `openspec/` and `.opencode/` are not Go packages. The three `cmd/` entrypoints import only `internal/*`.
- [Non-idempotency on re-run] → Mitigation: `SkipIfMissing` stat-guard makes a second run a no-op; plan is empty when nothing remains.
- [Guard misfires because `git` is missing in a minimal container] → Mitigation: documented fallback to explicit consent (D4), never a hard failure.
- [Ordering bug: removals executed before identity rewrites] → Mitigation: `buildPlan` appends removals last; a `setup_test.go` assertion pins the ordering.

## Migration Plan

Repo-internal change; no runtime consumers and no data migration.

1. Extend `internal/commands/setup.go`: add `kindRemove`, the `apply` branch + `SkipIfMissing` stat, `agentArtifactPaths(root)`, the `promptYesNo` helper, the consent input in `runSetup`, the removal entries in `buildPlan` (last), and the D4 guard.
2. Extend `internal/commands/setup_test.go`: removal plan shape, `--dry-run` reporting, idempotency, default-Yes, guard behaviour, deletions-last ordering.
3. Update `setupCmd.Long`, `README.md`, `CONTRIBUTING.md`.
4. Verify: `make build`, `make test`, `make vet`; run `setup --dry-run` in a scratch copy to confirm the printed plan; run `setup` with `Y` and with `n`.

Rollback: revert `setup.go`; identity handling is untouched and independent, so the command degrades to its current behaviour.

## Open Questions

- None blocking. The exact guard predicate (D4) is recorded as a rule and refined in implementation; it does not change the spec-level contract.
