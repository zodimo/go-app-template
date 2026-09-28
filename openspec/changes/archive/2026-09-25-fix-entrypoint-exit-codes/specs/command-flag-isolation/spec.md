## ADDED Requirements

### Requirement: Explicit config path is honored per binary

Each binary that exposes a `--config` flag SHALL load the file named by that flag, independently of the other binaries. A missing file at an explicit `--config` path MUST produce the "config file required" error and a non-zero exit code; it MUST NOT silently fall back to a discovered config file or to defaults.

#### Scenario: Explicit config file is selected

- **WHEN** a user passes `--config <path>` to `cli-config` or `cli-migration`
- **THEN** the configuration is loaded from that path and not from a discovered default-named file

#### Scenario: Missing explicit config file fails

- **WHEN** a user passes `--config` with a path that does not exist
- **THEN** the command fails with the config-file-required error and exits non-zero, without falling back

#### Scenario: Main binary retains explicit path behavior

- **WHEN** a user passes `--config <path>` to `cli`
- **THEN** the configuration is loaded from that path, preserving existing behavior

### Requirement: No shared process-global flag bindings

No viper key SHALL be bound from flags belonging to more than one command root, and every viper key that is read SHALL be bound to a flag or given a default. Flag values that select a configuration file or data directory MUST reach the loading code on the binary that declares the flag.

#### Scenario: Config flag binding is unique

- **WHEN** the three roots register their persistent flags
- **THEN** no flag key is bound by two different roots such that the last registration wins

#### Scenario: Data directory flag is honored

- **WHEN** a user passes `--data-dir` or `--config` to a binary that declares it
- **THEN** the value is observable in the loaded configuration for that binary

#### Scenario: Read keys are resolvable

- **WHEN** command code reads a viper key to decide behavior
- **THEN** that key is either bound by a flag on the same binary or given a default value

### Requirement: Independent version command per binary

Each binary SHALL present its own `version` command whose usage name and inherited flags correspond to that binary. A single command instance MUST NOT be registered into multiple command trees.

#### Scenario: Version usage matches the binary

- **WHEN** the `version` command help is requested from `cli-config` or `cli-migration`
- **THEN** the usage line names that binary, not another binary

#### Scenario: Version flags match the binary

- **WHEN** the `version` command help is requested from a binary
- **THEN** its inherited global flags are those of that binary's root

#### Scenario: Version command still prints build info

- **WHEN** `version` is run on any binary
- **THEN** the version, build date, and commit ID are printed and the process exits `0`
