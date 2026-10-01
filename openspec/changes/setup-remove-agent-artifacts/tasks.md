## 1. Deletion primitive in setup.go

- [ ] 1.1 Add `kindRemove changeKind = "remove"` to the `changeKind` enum in `internal/commands/setup.go`
- [ ] 1.2 Add a `case kindRemove:` branch to `apply`: honor `SkipIfMissing` with an `os.Stat` not-exist skip, otherwise `os.RemoveAll(c.Path)` and wrap errors as `remove %s: %w`
- [ ] 1.3 Confirm removals print via the existing `would:` (dry-run) / `applied:` reporting paths with no extra code
- [ ] 1.4 Add the `os` import usage (already imported) is unchanged; add nothing new

## 2. Agent-artifact path set and guard

- [ ] 2.1 Add `agentArtifactPaths(root string) []string` returning the fixed set: `root/.opencode`, `root/openspec`, each match of `filepath.Glob(root/*.code-workspace)`, `root/AGENTS.md`, `root/CLAUDE.md`
- [ ] 2.2 Add a guard helper (e.g. `shouldSkipOpenspec(root string) bool`) that returns true when `openspec/` contains uncommitted tracked changes (`git status --porcelain -- openspec/` non-empty)
- [ ] 2.3 Make the guard tolerant of missing `git` or a non-repo workspace: fall back to not skipping (removal still gated by consent), never error

## 3. Wizard prompt

- [ ] 3.1 Add `promptYesNo(r *bufio.Scanner, label string, def bool) bool` mirroring `prompt`'s default-on-empty/EOF behavior
- [ ] 3.2 In `runSetup`, after the four identity prompts, collect `removeAgents := promptYesNo(scanner, "Remove agent scaffolding (.opencode/, openspec/, *.code-workspace, AGENTS.md/CLAUDE.md)?", true)`
- [ ] 3.3 Pass `removeAgents` into `buildPlan` (extend its signature; keep it a pure function — the guard must be evaluated in `runSetup`/`apply`, not via I/O inside `buildPlan`)

## 4. Plan integration

- [ ] 4.1 In `buildPlan`, when consent is true, append one `kindRemove` change per resolved existing path, each with `Desc: "remove <path>"`, `SkipIfMissing: true`
- [ ] 4.2 Append all removal changes AFTER all identity changes so removals run last
- [ ] 4.3 Apply the `openspec/` guard: exclude the `openspec/` removal (and emit a warning at apply time) when the guard reports the authoring repo
- [ ] 4.4 Ensure a same-value + consent re-run produces an empty/short plan (idempotency) once artifacts are gone

## 5. Tests (internal/commands/setup_test.go)

- [ ] 5.1 Assert `buildPlan` includes the expected `kindRemove` entries when consent is true and paths exist
- [ ] 5.2 Assert removals are ordered after the rename/write/rewrite changes
- [ ] 5.3 Assert `apply` with `dryRun=true` deletes nothing and the plan is untouched
- [ ] 5.4 Assert idempotency: a second `apply` over already-removed paths is a no-op
- [ ] 5.5 Assert `SkipIfMissing` makes absent artifacts (e.g. no `.opencode/`) a non-error
- [ ] 5.6 Assert default-Yes parsing: empty line and EOF (`< /dev/null`) consent to removal
- [ ] 5.7 Assert explicit No performs no removals
- [ ] 5.8 Assert the `openspec/` guard skips removal when `openspec/` is dirty

## 6. Docs and help text

- [ ] 6.1 Update the `setupCmd.Long` help text in `internal/commands/setup.go` to describe the cleanup prompt and removal set
- [ ] 6.2 Update `README.md` "Clone Flow" (Layer 2) to mention the default-Yes cleanup prompt
- [ ] 6.3 Update `CONTRIBUTING.md` "Template reuse" with the same
- [ ] 6.4 Note in the docs that `gonew` does not carry untracked `.opencode/`, so cleanup is defensive/complementary

## 7. Verification

- [ ] 7.1 `make build`, `make test`, `make vet` all pass
- [ ] 7.2 Run `go run ./cmd/cli setup --dry-run` in a scratch copy and confirm the printed plan lists the expected removals
- [ ] 7.3 Run `go run ./cmd/cli setup` answering `Y` and confirm artifacts are removed and reported
- [ ] 7.4 Run `go run ./cmd/cli setup` answering `n` and confirm no artifacts are removed
- [ ] 7.5 Confirm the guard prevents `openspec/` deletion when run in the authoring repo with a dirty `openspec/`
