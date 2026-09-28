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
Aside from the template defaults in `internal/identity`, the source tree SHALL NOT contain hardcoded project-branding literals (`go-cli`, `GOCLI`, `go-cli-config`, `.go-cli`, `cli-migration`, `cli-config`) used as runtime values. Test fixtures MUST parameterize their expected environment values from the identity source rather than embed literals.

#### Scenario: Test fixtures use identity-derived env prefix
- **WHEN** a config test asserts an environment-based value
- **THEN** it builds the expected environment variable name from the identity environment prefix (e.g. `identity.EnvPrefix`) so the test remains valid after re-identity

