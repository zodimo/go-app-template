## ADDED Requirements

### Requirement: Single-source application identity
The system SHALL maintain application identity (app name, config file base name, environment variable prefix, data directory name, and binary-trio prefix) in a single generated source of truth at `internal/identity/identity.go`. Consumers such as config resolution, logging defaults, database defaults, and filesystem locations MUST read identity from this source rather than embed hardcoded literals.

#### Scenario: Config reads AppName from identity
- **WHEN** the configuration subsystem initializes defaults
- **THEN** it reads the app name, config name, and data directory from `internal/identity` instead of hardcoding `go-cli` values

#### Scenario: Default log and DB filenames derive from identity
- **WHEN** default logging and database paths are computed
- **THEN** the log filename and database filename use the identity source's app/project names

#### Scenario: Environment prefix derives from identity
- **WHEN** viper configures environment variable mappings
- **THEN** the environment variable prefix (e.g. `MYAPP`) comes from the identity source

### Requirement: Wizard Karma identity is overridable
The identity source SHALL be writable by the `setup` command so that after cloning, the application identity can be changed to the consumer's values without modifying application logic. Changing identity MUST NOT require editing any file other than the generated identity source.

#### Scenario: Re-identify an app
- **WHEN** a consumer runs the `setup` command with new values
- **THEN** `internal/identity/identity.go` is regenerated and all dependent defaults reflect the new identity

#### Scenario: Default template identity
- **WHEN** the template is used as-is without running `setup`
- **THEN** the identity source yields the template defaults (`go-cli`, `GOCLI`, `.go-cli`)