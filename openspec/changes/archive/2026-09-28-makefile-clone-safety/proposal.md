## Why

The template's clone flow is `gonew` (rewrites `go.mod` and `.go` imports) followed by `setup` (rewrites identity-derived files). Neither tool touches the root `Makefile`, so a freshly cloned project builds a stale module path via `-ldflags -X` (version embedding silently no-ops) and references `./cmd/cli*` directories that `setup` has already renamed (build fails). The Makefile is the largest remaining hardcoded-branding location outside `internal/identity`, so the template's central promise — clone and re-identify in one step — is broken at the build layer.

## What Changes

- **Derive the module path at make time.** Replace the three hardcoded `github.com/zodimo/go-app-template/internal/version.*` `-X` flags in `LDFLAGS_CONTENT` with a `MODULE := $(shell go list -m 2>/dev/null)` value so the Makefile survives `gonew` without edits.
- **Collapse the binary trio to a single rewritable variable.** Introduce `BIN_TRIO ?= cli` and derive `BINARY`, `CONFIG_BINARY`, `MIGRATION_BINARY` from it; route every `./cmd/...` reference (including the currently hardcoded `config-init` and `migrate` targets) through these variables.
- **Teach `setup` to rewrite the Makefile.** Add one rewrite target for the `BIN_TRIO ?= <cur.BinTrio>` line so re-identification updates the Makefile deterministically and idempotently, surfaced in `setup --dry-run`.
- **Fix pre-existing Makefile bugs** surfaced during review: duplicate `mkdir -p` in `build`, the migration copy that copies `$(CONFIG_BINARY)` but names it `$(MIGRATION_BINARY)`, hardcoded `./cmd/cli-config` / `./cmd/cli-migration` paths, and missing `version` / `validate-version` entries in `.PHONY`.
- **Document the guarantee**: README/CONTRIBUTING clone flow states the Makefile adapts automatically (module via `go list -m`, binary trio via `setup`); no manual edit step.
- **BREAKING** none for consumers of the built binaries; the Makefile's external targets and outputs are unchanged.

## Capabilities

### New Capabilities
<!-- No new capability: this tightens existing scaffolding and hygiene contracts. -->

### Modified Capabilities
- `project-scaffolding`: The `setup` command's set of rewritten artifacts gains the Makefile (`BIN_TRIO` line), and the documented clone flow must state that the Makefile module path is derived dynamically and therefore requires no post-`gonew` edit.
- `template-hygiene`: The "no residual hardcoded branding outside identity source" requirement extends to the root `Makefile` — it MUST NOT hardcode the module path or the binary-trio literals; the module path is derived from `go.mod` and the binary-trio prefix is the single `setup`-managed `BIN_TRIO` value.

## Impact

- **Code**: `Makefile` (module path dynamic, single `BIN_TRIO`, bug fixes), `.gitignore` (ignore `/bin`, the Makefile build output directory), `internal/commands/setup.go` (new change target + Makefile path in `identityPaths`/change plan), `internal/commands/setup_test.go` (assert Makefile rewrite and idempotency).
- **Docs**: `README.md` (clone flow), `CONTRIBUTING.md` (identity single-source contract note).
- **Specs**: delta specs for `project-scaffolding` and `template-hygiene`.
- **Dependencies**: none added. `go list -m` is the Go toolchain.
- **Risk**: `setup`'s `rewriteLiteral` replaces the first occurrence of a literal; the Makefile target must anchor on the full `BIN_TRIO ?= <value>` line to stay safe and idempotent (no substring collisions with `cli-config` / `cli-migration`).
