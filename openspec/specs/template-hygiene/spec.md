# template-hygiene Specification

## Purpose
TBD - created by archiving change starter-template-scaffolding. Update Purpose after archive.
## Requirements
### Requirement: No machine-local or generated runtime artifacts are committed
The repository SHALL NOT contain committed machine-local runtime artifacts such as generated config files, local data directories, logs, or build output. Runtime state MUST be git-ignored and generated on first run via `config init` rather than shipped with the template.

#### Scenario: Generated config is not committed
- **WHEN** a consumer inspects the template source tree
- **THEN** no generated config file (e.g. `go-cli-config.json` or its renamed equivalent) is present in version control

#### Scenario: Runtime data and logs are git-ignored
- **WHEN** a consumer runs the app and it creates a data directory and log file
- **THEN** those artifacts are ignored by the repository's ignore rules and do not appear as untracked files eligible for commit

#### Scenario: Stale test artifacts are absent
- **WHEN** a consumer inspects the source tree
- **THEN** no leftover test panic log files are present in package directories

### Requirement: Config generated on first run
The application SHALL support generating its configuration file on demand (via a `config init` command) using the current identity and working directory, so that a fresh clone can produce a local config without shipping machine-local absolute paths.

#### Scenario: Init writes a local config from identity
- **WHEN** a consumer runs `<app>-config init` in a clone
- **THEN** it writes a config file using paths derived from the current working directory and the identity source, not pre-committed absolute paths

### Requirement: No residual hardcoded branding outside identity source
Aside from the template defaults in `internal/identity`, the source tree SHALL NOT contain hardcoded project-branding literals (`go-cli`, `GOCLI`, `go-cli-config`, `.go-cli`, `cli-migration`, `cli-config`) used as runtime values. Test fixtures MUST parameterize their expected environment values from the identity source rather than embed literals. The root `Makefile` MUST NOT hardcode the module path — it MUST be derived from the build environment (e.g. `go list -m`) — and MUST NOT repeat the binary-trio literals; the binary-trio prefix MUST be a single `setup`-managed `BIN_TRIO` value from which the individual binary names are derived.

#### Scenario: Test fixtures use identity-derived env prefix
- **WHEN** a config test asserts an environment-based value
- **THEN** it builds the expected environment variable name from the identity environment prefix (e.g. `identity.EnvPrefix`) so the test remains valid after re-identity

#### Scenario: Makefile derives the module path
- **WHEN** a consumer inspects the linker flags in the root `Makefile`
- **THEN** no literal module path appears; the `-X` targets derive the module from `go list -m` against the current `go.mod`

#### Scenario: Makefile has a single binary-trio source
- **WHEN** a consumer inspects the binary names referenced by the root `Makefile`
- **THEN** they derive from one `BIN_TRIO` value rather than repeated `cli` / `cli-config` / `cli-migration` literals

### Requirement: Agent and IDE scaffolding does not survive into a downstream project
A project cloned from this template SHALL NOT retain the source repository's agent or IDE scaffolding. The `setup` command MUST offer to remove the following from the clone, defaulting to Yes: the `.opencode/` directory, the `openspec/` directory, any `*.code-workspace` file, and the `AGENTS.md` and `CLAUDE.md` files. Removal MUST be idempotent (absent paths are a no-op), MUST be reported through the existing plan/apply mechanism (`setup --dry-run` prints it), and MUST be guarded so the authoring repository's own `openspec/` change history cannot be destroyed.

#### Scenario: Agent scaffolding is offered for removal
- **WHEN** a consumer runs `<app> setup` in a clone
- **THEN** the command offers to remove `.opencode/`, `openspec/`, any `*.code-workspace` file, `AGENTS.md`, and `CLAUDE.md`, defaulting to Yes

#### Scenario: Consented scaffolding is removed
- **WHEN** a consumer consents and the listed paths exist
- **THEN** the command removes `.opencode/`, `openspec/`, and any matching `*.code-workspace` file present in the workspace root

#### Scenario: Absent scaffolding is tolerated
- **WHEN** some or all of the listed paths do not exist in the clone
- **THEN** the command completes successfully and treats the missing paths as nothing to remove

#### Scenario: Removal is previewed by dry-run
- **WHEN** a consumer runs `<app> setup --dry-run`
- **THEN** the command prints each agent-artifact removal it would perform, alongside the identity rewrites, without deleting anything

#### Scenario: Authoring repository is guarded
- **WHEN** `setup` runs where `openspec/` holds uncommitted tracked changes
- **THEN** the command does not remove `openspec/` in that workspace
