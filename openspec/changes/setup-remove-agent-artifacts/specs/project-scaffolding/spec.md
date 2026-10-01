## MODIFIED Requirements

### Requirement: Setup command rewrites cloned project identity
The application SHALL provide a `setup` cobra command that interactively collects application identity values (app name, environment prefix, data directory, binary-trio prefix) using standard-library input (no third-party TUI dependency) and rewrites the project to use them. The command MUST be idempotent and re-runnable so renaming later is non-destructive. The set of rewritten artifacts MUST include the binary-trio prefix line in the root `Makefile`, alongside `internal/identity/identity.go`, the `cmd/` entrypoint directories, cobra `Use:` strings, panic-recovery tags, and `.gitignore` identity lines. In addition, `setup` SHALL offer a single default-Yes consent prompt to remove agent/IDE scaffolding from the clone, and MUST apply that removal only when the operator consents, executing all removals after the identity rewrites so a failure during re-identification never leaves the tree partially wiped.

#### Scenario: Initial setup after clone
- **WHEN** a consumer runs `<app> setup` and provides new identity values
- **THEN** the command writes `internal/identity/identity.go`, renames the binary directories, updates cobra `Use:` strings and panic-recovery tags, rewrites the `BIN_TRIO` line in the root `Makefile`, and reports the files it changed

#### Scenario: Setup is idempotent
- **WHEN** `setup` is run twice with the same values
- **THEN** the second run is a no-op and does not corrupt or duplicate the tree

#### Scenario: Setup dry-run
- **WHEN** a consumer runs `<app> setup --dry-run`
- **THEN** the command prints the planned changes, including the Makefile `BIN_TRIO` rewrite and every agent-artifact removal it would perform, without modifying any files

#### Scenario: Setup validation before mutation
- **WHEN** a consumer provides invalid identity input (empty app name, or a prefix producing invalid code identifiers)
- **THEN** the command rejects the input and does not modify any files

#### Scenario: Setup offers agent-artifact cleanup with a default-Yes prompt
- **WHEN** a consumer runs `<app> setup` and reaches the cleanup prompt
- **THEN** the wizard presents a single consent prompt listing the agent/IDE artifacts it can remove (`.opencode/`, `openspec/`, `*.code-workspace`, `AGENTS.md`, `CLAUDE.md`) and defaults to Yes on an empty line or EOF

#### Scenario: Setup removes consented agent artifacts last
- **WHEN** a consumer consents to cleanup and runs `<app> setup`
- **THEN** the command removes the listed artifacts that exist and reports each removal, and it performs those removals after all identity rewrites have succeeded

#### Scenario: Setup tolerates already-absent agent artifacts
- **WHEN** a consumer consents to cleanup in a clone where some listed artifacts are absent (for example `.opencode/` in a `gonew`-created project)
- **THEN** removing an absent path is a no-op and `setup` completes without error

#### Scenario: Setup cleanup is idempotent
- **WHEN** `setup` is run twice with the same values and consent to remove agent artifacts
- **THEN** the second run makes no further changes because the artifacts are already gone

#### Scenario: Setup declines cleanup
- **WHEN** a consumer answers No to the cleanup prompt
- **THEN** the command performs the identity rewrites and removes no agent/IDE artifacts

#### Scenario: Setup guards the authoring repository
- **WHEN** `<app> setup` runs in a workspace whose `openspec/` contains uncommitted tracked changes (i.e. the authoring repository)
- **THEN** the command skips removal of `openspec/`, warns on stderr, and still permits removal of the remaining agent/IDE artifacts
