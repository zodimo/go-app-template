## Why

Exit-code handling at the three `cmd/` entry points is unreliable in ways that are silent and CI-hostile: a panic in any binary reports success (`exit 0`), every successful command prints a bogus `exit status 0` to stderr, and two of the three binaries ignore the explicit `--config` path while sharing one `version` command pointer across three trees. A starter template whose whole purpose is to be copied must have a trustworthy `os.Exit` contract at its entry points, because every downstream app inherits it.

## What Changes

- **Panic path returns a non-zero exit code.** Reorder the entry-point defers so `RecoverPanic` runs before the exit-code defer, and have recovery set a non-zero code (e.g. `2`) so a recovered panic can no longer be reported as success. **BREAKING**: apps that relied on panics exiting `0` (none should) will now exit non-zero.
- **Remove the spurious `exit status 0` on success.** Stop treating the `exitcode.ExitStatus` zero value (which is a non-nil `error` interface) as a failure: only print and set `exitCode = 1` for genuine errors, then map a typed `ExitStatus` to the process code. Wire the return path so a command can actually return a non-zero `ExitStatus`.
- **Honor `--config` on all three binaries.** De-collide the process-global viper key collision between the config and database roots (both bind `config-config-file`; the database root reads `root-config-file`), so `cli-config --config` and `cli-migration --config` select the file they claim to, and a missing explicit path fails with the documented "config file required" error instead of silently falling back.
- **Give each binary its own `version` command.** Stop registering one package-level `versionCmd` pointer in three command trees; each root gets its own command so `Use`, help text, and inherited flags are correct per binary.
- **Stop duplicating error output.** Set `SilenceErrors` consistently on the config and database roots (the main root already does), so cobra does not print `Error: ...` and then have `ExecuteCmd` print it again.
- **Fail on unknown subcommands under container commands.** A typo such as `cli-migration migrate bogus` currently prints help and exits `0`; make unknown child commands an error.
- **Write panic logs to the configured location.** Ensure the panic log directory is established for the entry points instead of falling back to the current working directory.
- **Suppress the empty config source line.** `show` prints `Configuration from: ` with a blank path when no config file was used.

## Capabilities

### New Capabilities
- `exit-code-handling`: The contract that all three CLI entry points map command outcomes (success, returned typed `ExitStatus`, ordinary error, recovered panic) to a deterministic process exit code, emit errors exactly once, and never report a crash as success. Covers the `ExecuteCmd` mapping, the `main()` defer ordering, and the panic-to-exit path.
- `command-flag-isolation`: The contract that each binary's command tree and flag bindings are self-contained: explicit `--config`/`--data-dir` flags select the file/value they name on the binary that declares them, no viper key is bound by two roots, and no cobra command pointer is shared across trees.

### Modified Capabilities
<!-- No existing capability's requirements change at the spec level; this change adds entry-point correctness that the existing specs did not cover. -->

## Impact

- **Code**: `cmd/cli/main.go`, `cmd/cli-config/main.go`, `cmd/cli-migration/main.go` (defer ordering, panic-to-exit), `internal/commands/root.go` (`ExecuteCmd` mapping, `SilenceErrors`), `internal/commands/version.go`, `internal/commands/config.go`, `internal/commands/database.go` (per-tree version commands), `internal/config/commands/root.go`, `internal/database/commands/root.go` (viper key de-collision, `SilenceErrors`), `internal/config/viper.go` / `internal/config/load.go` (config-file selection), `internal/logging/recover.go` (panic log location / exit signalling).
- **APIs**: `internal/exitcode` semantics become load-bearing again; the `ExitStatus`-as-command-error convention gains a working non-zero path. `ExecuteCmd`'s signature may carry the resolved exit code.
- **Dependencies**: none. Standard library plus the already-present cobra/viper.
- **Behavioral**: `**BREAKING**` for anyone depending on a panic exiting `0`; `cli-migration` and `cli-config` gain correct `--config` behavior and may begin failing where they previously silently fell back.
- **Tests**: new coverage for `ExecuteCmd` mapping (success, error, typed `ExitStatus`), panic-to-exit-code, `--config` selection per binary, no-duplicate stderr, and unknown-subcommand failure. Existing `internal/commands/root_test.go` only checks flag registration and is insufficient.
