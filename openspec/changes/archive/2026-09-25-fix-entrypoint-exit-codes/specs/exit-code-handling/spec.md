## ADDED Requirements

### Requirement: Deterministic exit code mapping

Each CLI entry point SHALL map every command outcome to a deterministic process exit code: a successful command MUST exit `0`; an ordinary returned error MUST exit `1`; a command that returns a non-zero `exitcode.ExitStatus` MUST exit that code; and a returned `ExitStatus` of `0` MUST be treated as success, not as an error. The zero value of `ExitStatus` MUST NOT cause a non-zero exit code.

#### Scenario: Successful command exits zero

- **WHEN** a command completes without returning an error
- **THEN** the process exits with code `0`

#### Scenario: Ordinary error exits one

- **WHEN** a command returns a plain error (for example an unknown flag or a missing required config)
- **THEN** the process exits with code `1`

#### Scenario: Zero ExitStatus is success

- **WHEN** a command returns `exitcode.ExitStatusOK` (the zero value) as its error result
- **THEN** the process exits with code `0` and no error text is written to stderr

#### Scenario: Non-zero ExitStatus is propagated

- **WHEN** a command returns a non-zero `exitcode.ExitStatus`
- **THEN** the process exits with that exact numeric code

### Requirement: Errors are emitted exactly once

When a command fails, the CLI SHALL write the error to stderr exactly once. The command root's own error printing MUST be suppressed so that the central error handler is the single writer, regardless of which binary is invoked.

#### Scenario: Single error line on unknown command

- **WHEN** an unknown command is given to any of the three binaries
- **THEN** the error message appears exactly once on stderr and the process exits non-zero

#### Scenario: JSON error mode emits one document

- **WHEN** a command fails and the JSON output mode is enabled
- **THEN** exactly one JSON error document is written to stderr

### Requirement: Recovered panic exits non-zero

A panic recovered at an entry point SHALL cause the process to exit with a non-zero code. Panic recovery MUST run before the exit-code decision so that the recovering handler can set the exit code, and a recovered panic MUST NOT be reported to the shell as success.

#### Scenario: Panic yields non-zero exit

- **WHEN** a panic is raised while executing a command and is recovered by the entry-point handler
- **THEN** a panic report is written to stderr and the process exits with a non-zero code

#### Scenario: Panic exit code is distinguishable

- **WHEN** a panic is recovered
- **THEN** the exit code is non-zero and distinct from the ordinary error code of `1`

### Requirement: Panic logs are written to the configured location

When a panic is recovered, the panic log file SHALL be written to the application's configured log directory. It MUST NOT be written to the process current working directory as a side effect of the log location never being configured.

#### Scenario: Panic log uses configured directory

- **WHEN** a panic is recovered after logging has been configured
- **THEN** the panic log file is created inside the configured log directory

#### Scenario: Panic still reports when log dir is unavailable

- **WHEN** the configured log directory cannot be created
- **THEN** the panic is still reported to stderr and the process still exits non-zero

### Requirement: Container commands reject unknown subcommands

A command that groups subcommands and has no run behaviour of its own SHALL return an error and a non-zero exit code when given an unrecognised subcommand, instead of printing help and reporting success.

#### Scenario: Unknown subcommand fails

- **WHEN** a user invokes a container command (for example `migrate`) with an unknown child name
- **THEN** an error is reported and the process exits non-zero

#### Scenario: Bare container command shows help

- **WHEN** a container command is invoked with no child
- **THEN** its help is printed and the process exits `0`

### Requirement: Absent config source is not reported as a path

When no configuration file was used, configuration display SHALL NOT print an empty source path. It SHALL either omit the source line or state explicitly that no config file was used.

#### Scenario: No config file used

- **WHEN** configuration is displayed and no config file was loaded
- **THEN** the output does not contain a trailing empty source path
