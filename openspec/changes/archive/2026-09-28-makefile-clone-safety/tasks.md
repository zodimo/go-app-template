## 1. Makefile: derive module path and fix defects

- [x] 1.1 Add `MODULE := $(shell go list -m 2>/dev/null)` near the version variables in `Makefile`.
- [x] 1.2 Replace the three hardcoded `github.com/zodimo/go-app-template/internal/version.*` `-X` flags in `LDFLAGS_CONTENT` with `$(MODULE)/internal/version.Version|CommitID|BuildDate`.
- [x] 1.3 Remove the duplicated `@mkdir -p $(BUILD_DIR)/$(BINARY)-v$(CLEAN_VERSION)` line in the `build` target.
- [x] 1.4 Fix the migration copy in `build` so it copies `$(BUILD_DIR)/$(MIGRATION_BINARY)` into the versioned directory (currently copies `$(CONFIG_BINARY)`).
- [x] 1.5 Add `version` and `validate-version` to the `.PHONY` list.

## 2. Makefile: single BIN_TRIO source

- [x] 2.1 Introduce `BIN_TRIO ?= cli` and derive `BINARY = $(BIN_TRIO)`, `CONFIG_BINARY = $(BIN_TRIO)-config`, `MIGRATION_BINARY = $(BIN_TRIO)-migration`.
- [x] 2.2 Route every `./cmd/...` reference — in `build`, `config-init`, and `migrate` — through `$(BINARY)` / `$(CONFIG_BINARY)` / `$(MIGRATION_BINARY)`.
- [x] 2.3 Add a short comment documenting that `MODULE` is derived from `go.mod` and that `BIN_TRIO` is the single `setup`-managed line.

## 3. setup: rewrite the Makefile BIN_TRIO line

- [x] 3.1 Add a `Makefile string` field to `identityPaths` and set it in `findPaths` to `filepath.Join(root, "Makefile")`.
- [x] 3.2 In `buildPlan`, inside the `if cur.BinTrio != next.BinTrio` block, append a `kindRewrite` change with `Path: nextPaths.Makefile`, `Old: "BIN_TRIO ?= " + cur.BinTrio`, `New: "BIN_TRIO ?= " + next.BinTrio`, and a `--dry-run` `Desc`.
- [x] 3.3 Add a `SkipIfMissing bool` field to `change` and honor it in `apply`'s `kindRewrite` branch (skip when the target file does not exist) so a deleted Makefile does not fail `setup`.
- [x] 3.4 Set `SkipIfMissing: true` on the Makefile rewrite change so the plan stays pure (`buildPlan` does no I/O).
- [x] 3.5 Confirm the Makefile rewrite is printed by `setup --dry-run`.

## 4. Tests

- [x] 4.1 Extend `internal/commands/setup_test.go` with a fixture `Makefile` containing `BIN_TRIO ?= cli` plus `cli-config` / `cli-migration` references; assert only the `BIN_TRIO` line becomes `BIN_TRIO ?= myapp`.
- [x] 4.2 Assert idempotency: a same-value `buildPlan` yields no Makefile change (empty plan path).
- [x] 4.3 Assert `--dry-run` lists the Makefile rewrite and leaves the file byte-identical.
- [x] 4.4 Assert a missing `Makefile` is skipped without error.

## 5. Documentation

- [x] 5.1 Update `README.md` clone flow: the root `Makefile` needs no module-path edit after `gonew` (module derived via `go list -m`) and derives binary names from `setup`'s `BIN_TRIO`.
- [x] 5.2 Update `CONTRIBUTING.md` identity single-source contract to list the Makefile `BIN_TRIO` line as a `setup`-managed artifact.

## 6. Verification

- [x] 6.1 Run `make build`, `make test`, `make vet` — all pass.
- [x] 6.2 Confirm `make config-init` and `make migrate` resolve their `./cmd/...` paths through the derived variables.
- [x] 6.3 Simulate a module rename on a scratch copy (change the `go.mod` module path) and confirm `make build` embeds version metadata under the new module without editing the Makefile.
- [x] 6.4 Run `setup` to rename the trio on a scratch copy and confirm `make build` succeeds with no Makefile edits.
