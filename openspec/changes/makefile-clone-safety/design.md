## Context

The template's clone flow has two layers, and the root `Makefile` participates in neither:

- `gonew` rewrites only `.go` files and `go.mod` (verified in the archived scaffolding design: `fixGo` + `fixGoMod`).
- `setup` (`internal/commands/setup.go`) rewrites `internal/identity/identity.go`, renames the three `cmd/cli*` directories, rewrites cobra `Use:` strings and panic-recovery tags, and regenerates the identity lines in `.gitignore`. It does **not** touch the Makefile.

The Makefile hardcodes two families of names:

1. **Module/import path** — `github.com/zodimo/go-app-template/internal/version.*` in `LDFLAGS_CONTENT` (three `-X` flags). After `gonew`, these point at a module that no longer exists and version embedding silently no-ops.
2. **Binary trio** — `BINARY=cli`, `CONFIG_BINARY=cli-config`, `MIGRATION_BINARY=cli-migration`, plus hardcoded `./cmd/cli-config` / `./cmd/cli-migration` in the `config-init` and `migrate` targets. After `setup` renames the `cmd/` directories, `make build` targets directories that no longer exist.

Neither failure is caught until a consumer clones and builds. This design closes the gap without adding a new subsystem, by reusing the two mechanisms the template already trusts: `go list` for module data and the `setup` change plan for identity data.

## Goals / Non-Goals

**Goals:**
- `make build` (and `config-init`, `migrate`) work after `gonew` with zero Makefile edits.
- `make build` works after `setup` renames the binary trio with zero Makefile edits.
- The Makefile stops being a hardcoded-branding site, satisfying `template-hygiene`.
- Fix the four pre-existing Makefile defects found during review.

**Non-Goals:**
- Collapsing or changing the three-binary split.
- Making the Makefile non-editable / fully generated (hand-tuned targets must remain).
- Changing the Makefile's target surface or build outputs.
- Reconciling the pre-existing `VERSION` vs `CURRENT_VERSION` dual version source (noted, out of scope).
- Adding any module dependency; `go list` is the Go toolchain.

## Decisions

### D1 — Derive the module path at make time (`go list -m`)

```makefile
MODULE := $(shell go list -m 2>/dev/null)

LDFLAGS_CONTENT = -s -w \
    -X $(MODULE)/internal/version.Version=v$(CLEAN_VERSION) \
    -X $(MODULE)/internal/version.CommitID=$(COMMIT_ID) \
    -X $(MODULE)/internal/version.BuildDate=$(BUILD_DATE)
```

`go list -m` with no arguments prints the main module path straight from `go.mod`; it does not resolve the dependency graph, so it is offline-safe and cheap. After `gonew` rewrites `go.mod`, the next `make build` targets the new module automatically. If there is no module context, `go list -m` fails and the build fails with a clear toolchain error rather than silently embedding version fields under a stale path.

**Alternatives considered:**
- *A post-`gonew` manual edit* — rejected: reintroduces the manual step the template exists to eliminate.
- *A `MODULE ?=` literal that `setup` also rewrites* — rejected: `setup` runs after `gonew` but module path ownership belongs to `gonew`; making the Makefile self-derive keeps the two layers cleanly separated (module = build env, identity = `setup`).
- *Read the module from `go.mod` with `sed`/`awk`* — rejected: brittle parsing when the toolchain already exposes the value.

### D2 — Collapse the binary trio to one `BIN_TRIO` variable, rewritten by `setup`

```makefile
BIN_TRIO        ?= cli
BINARY           = $(BIN_TRIO)
CONFIG_BINARY    = $(BIN_TRIO)-config
MIGRATION_BINARY = $(BIN_TRIO)-migration
```

Every target references `./cmd/$(BINARY)`, `./cmd/$(CONFIG_BINARY)`, `./cmd/$(MIGRATION_BINARY)`, including the currently hardcoded `config-init` and `migrate` targets. The variable name mirrors `identity.BinTrio`, making the correspondence obvious.

`setup` gains one rewrite target: `BIN_TRIO ?= <cur.BinTrio>` → `BIN_TRIO ?= <next.BinTrio>`. This threads through the existing machinery in `internal/commands/setup.go`:

- add a `Makefile` path to the `identityPaths` struct (alongside `IdentityFile`, `Gitignore`, `RootUseFile`, …),
- add one `buildChange{kind: changeRewrite, Path: Makefile, Old: "BIN_TRIO ?= " + cur.BinTrio, New: "BIN_TRIO ?= " + next.BinTrio}` in `buildChanges`,
- the existing `rewriteLiteral` (first-occurrence replacement), `--dry-run` reporting, and idempotency all apply unchanged.

Anchoring on the **full line** (not the bare token `cli`) is deliberate: `rewriteLiteral` replaces the first occurrence of a literal, and the Makefile also contains `cli-config` and `cli-migration`, so a token-level replacement would be ambiguous. Full-line anchoring is unique and makes re-running `setup` with identical values compute `Old == New` (no-op).

**Alternatives considered:**
- *Fully dynamic read of `identity.BinTrio` at make time* — would require a `go run` helper or a generated `.mk`, adding machinery and a second source of truth for a starter template.
- *`setup` regenerates the entire Makefile* (like `.gitignore`) — rejected: loses hand-tuned targets and the "readable template" property.
- *Leave the Makefile to a documented manual step* — rejected: contradicts the one-step rename contract.

### D3 — `setup` is tolerant of a missing Makefile

If the root `Makefile` is absent (a consumer deleted it), `setup` MUST skip the Makefile rewrite rather than fail. The Makefile is a convenience artifact, not identity state; its absence should not block re-identification.

### D4 — Bundle the pre-existing Makefile defect fixes

Surfaced during review, fixed in the same change because they touch the same lines:

- L49–50 duplicate `mkdir -p $(BUILD_DIR)/$(BINARY)-v$(CLEAN_VERSION)` → single call.
- L56 `cp $(BUILD_DIR)/$(CONFIG_BINARY) .../$(MIGRATION_BINARY)` → source must be `$(BUILD_DIR)/$(MIGRATION_BINARY)`.
- L68 / L72 hardcoded `./cmd/cli-config` / `./cmd/cli-migration` → `$(CONFIG_BINARY)` / `$(MIGRATION_BINARY)`.
- L1 `.PHONY` missing `version` and `validate-version`.

## Risks / Trade-offs

- [First-occurrence replacement corrupts Makefile] → Mitigation: anchor `Old`/`New` on the full `BIN_TRIO ?= <value>` line; assert exactly one match before writing; covered by a `setup_test.go` case.
- [`go list -m` adds a toolchain call at parse time] → Negligible; `make` already shells out for git/date. Value is offline and deterministic. Mitigation: `2>/dev/null` and rely on the subsequent `go build` to surface a genuine no-module error.
- [Setup run before `gonew` or outside a module leaves `MODULE` empty] → `go list -m` and `go build` share the same module context, so `make build` fails loudly rather than embedding a wrong path.
- [Idempotency drift if the Makefile is hand-edited to a different `BIN_TRIO` formatting] → `setup` matches the literal line it generated; if a consumer reformats it, the rewrite target simply does not match and no change is made (no corruption). Documented in CONTRIBUTING as a `setup`-managed line.
- [Scope creep into `VERSION` vs `CURRENT_VERSION`] → Explicitly out of scope; the existing `build`/tag behavior is preserved unchanged.

## Migration Plan

This is a repo-internal change with no runtime consumers of the Makefile, and the Makefile is not yet committed (staged only), so there is no external migration.

1. Rewrite `Makefile`: add `MODULE`, add `BIN_TRIO`/derived vars, route `./cmd/...` through the vars, apply the four defect fixes.
2. Extend `internal/commands/setup.go`: add the Makefile path and the single rewrite target; skip when absent.
3. Extend `internal/commands/setup_test.go`: assert the Makefile rewrite, `--dry-run` reporting, and idempotency.
4. Update `README.md` / `CONTRIBUTING.md` clone-flow wording.
5. Verify: `make build`, `make test`, `make vet`; then simulate `gonew` (module rename) and `setup` (trio rename) and re-run `make build`.

Rollback: revert the Makefile and the `setup.go` target; identity handling is unchanged and independent.
