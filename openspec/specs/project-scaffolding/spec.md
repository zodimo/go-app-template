# project-scaffolding Specification

## Purpose
TBD - created by archiving change starter-template-scaffolding. Update Purpose after archive.
## Requirements
### Requirement: Setup command rewrites cloned project identity
The application SHALL provide a `setup` cobra command that interactively collects application identity values (app name, environment prefix, data directory, binary-trio prefix) using standard-library input (no third-party TUI dependency) and rewrites the project to use them. The command MUST be idempotent and re-runnable so renaming later is non-destructive. The set of rewritten artifacts MUST include the binary-trio prefix line in the root `Makefile`, alongside `internal/identity/identity.go`, the `cmd/` entrypoint directories, cobra `Use:` strings, panic-recovery tags, and `.gitignore` identity lines.

#### Scenario: Initial setup after clone
- **WHEN** a consumer runs `<app> setup` and provides new identity values
- **THEN** the command writes `internal/identity/identity.go`, renames the binary directories, updates cobra `Use:` strings and panic-recovery tags, rewrites the `BIN_TRIO` line in the root `Makefile`, and reports the files it changed

#### Scenario: Setup is idempotent
- **WHEN** `setup` is run twice with the same values
- **THEN** the second run is a no-op and does not corrupt or duplicate the tree

#### Scenario: Setup dry-run
- **WHEN** a consumer runs `<app> setup --dry-run`
- **THEN** the command prints the planned changes, including the Makefile `BIN_TRIO` rewrite, without modifying any files

#### Scenario: Setup validation before mutation
- **WHEN** a consumer provides invalid identity input (empty app name, or a prefix producing invalid code identifiers)
- **THEN** the command rejects the input and does not modify any files

### Requirement: Rename binary trio
The `setup` command SHALL rename the three command entrypoint directories (`cmd/cli`, `cmd/cli-config`, `cmd/cli-migration`) and their associated cobra `Use:` names and panic-recovery tags to match the target binary-trio prefix. It MUST also update the single `BIN_TRIO` value in the root `Makefile` from which the build targets derive the binary names, so the Makefile continues to build the renamed trio. The split of the three binaries MUST be preserved.

#### Scenario: Rename directory and command name together
- **WHEN** a consumer sets a binary-trio prefix (e.g. `myapp`)
- **THEN** the `cmd/cli`, `cmd/cli-config`, and `cmd/cli-migration` directories and their cobra `Use:` strings and panic tags are renamed consistently to `myapp`, `myapp-config`, and `myapp-migration`

#### Scenario: Internal imports remain valid after rename
- **WHEN** the binary directories are renamed
- **THEN** internal package import paths remain intact because entrypoints import the same `internal/*` packages

#### Scenario: Makefile builds the renamed trio
- **WHEN** a consumer runs `make build` after `setup` renames the trio to `myapp`
- **THEN** the Makefile derives `myapp`, `myapp-config`, and `myapp-migration` from its single `BIN_TRIO` value and builds each from the renamed `cmd/` directories without manual edits
### Requirement: Documented clone flow using gonew
The template SHALL document a clone flow using `golang.org/x/tools/cmd/gonew` for module-layer rewriting (go.mod module path and internal imports), which MUST be presented as the step before running `setup`. The documented flow MUST state that the root `Makefile` requires no manual module-path edit after `gonew` because it derives the module path from the build environment at make time.

#### Scenario: Module path rewrite is delegated to gonew
- **WHEN** the README describes cloning a new project
- **THEN** it instructs the consumer to use `gonew <srcmod>@<version> <dstmod>` for module path and import rewriting before running `<app> setup`

#### Scenario: Makefile needs no module-path edit after gonew
- **WHEN** a consumer runs `gonew <srcmod>@<version> <dstmod>` and then `make build` without editing the Makefile
- **THEN** the build embeds version metadata under the new module path
