## 1. Exit-code mapping in ExecuteCmd

- [x] 1.1 In `internal/commands/root.go`, rewrite `ExecuteCmd` (currently L32-54) so it decides in order: `err == nil` → success; `err` is an `exitcode.ExitStatus` → use its numeric value directly (including `0`, which MUST NOT print or set `1`); otherwise a genuine error → set `exitCode = 1` and print once. Remove the current unconditional `exitCode = 1` that fires before the `errors.AsType` rewrite.
- [x] 1.2 Keep the `--json` single-document error output (`viper.GetBool("json")`) as the sole writer when a genuine error occurs; ensure the typed-zero path writes nothing.
- [x] 1.3 Confirm `errors.AsType[exitcode.ExitStatus]` compatibility with a value-typed `ExitStatus` that also satisfies `error`, and that a returned `ExitStatusOK` is detected as success rather than an error.
- [x] 1.4 Add `internal/exitcode/error_test.go` covering `ExitStatus.Error()`, `NewExitStatus(n)` round-trip, and `errors.As`/`errors.AsType` extraction for zero and non-zero values.
- [x] 1.5 Add `ExecuteCmd` tests (new `internal/commands/execute_test.go`, package `commands`) using throwaway `*cobra.Command` values with `SetArgs` and `SetOut`/`SetErr` buffers — do NOT call `os.Exit`. Cover: success returns `0`; ordinary error returns `1`; `exitcode.NewExitStatus(3)` returns `3`; `exitcode.ExitStatusOK` returns `0` and writes nothing to stderr (the `exit status 0` bug).

## 2. Panic path exits non-zero

- [x] 2.1 In `internal/logging/recover.go`, extend `RecoverPanic` (L25) to accept an exit-code contribution (e.g. a `*uint8` out-parameter or equivalent) and set it to `2` when a panic is recovered, while preserving the existing stderr report, panic-log file write, optional `cleanup func()`, and the previously-verified single-stack-trace behavior.
- [x] 2.2 In `cmd/cli/main.go`, `cmd/cli-config/main.go`, and `cmd/cli-migration/main.go`, reorder so the exit-code defer is registered FIRST (runs last, after recovery sets the code) and `logging.RecoverPanic(...)` is registered SECOND (runs first on unwind). Keep the existing `exitCode = commands.ExecuteCmd(<Root>)` assignment and `os.Exit` guard.
- [x] 2.3 Ensure a recovered panic always produces a non-zero exit and that the code is distinguishable from the ordinary-error code `1` (use `2`).
- [x] 2.4 Extend `internal/logging/recover_test.go` (`TestRecoverPanic_NilLogger`, `TestRecoverPanic_SingleStackTrace`) for the new signature, and add a test asserting the exit-code contribution is non-zero after a recovered panic; assert captured stderr content through the existing `os.Pipe` pattern.
- [x] 2.5 Add an entry-point smoke check (built binary or an in-process equivalent) that a panic-inducing failure exits non-zero; document the exact command in the task evidence.

## 3. Config-flag isolation

- [x] 3.1 De-collide viper key `config-config-file`: it is bound in both `internal/config/commands/root.go:36` and `internal/database/commands/root.go:32`. Give each root a distinct key and make each binary read the key it binds.
- [x] 3.2 Fix `internal/database/commands/root.go`: its `PersistentPreRunE` (L12) reads `root-config-file` and `require-config`, but the root binds `config-config-file` and `data-dir`. Make the migration root read the keys its own flags bind (or bind the keys it reads), so `cli-migration --config <path>` is honored.
- [x] 3.3 De-collide viper key `data.directory`, bound in both `internal/config/commands/root.go:42` and `internal/database/commands/root.go:33` (and defaulted at `internal/config/config.go:101`). Ensure the `--data-dir` value from the declaring root reaches `config.Load` for that binary.
- [x] 3.4 Ensure an explicit `--config` path that does not exist yields the documented `ErrConfigFileRequired` ("config file required but not found") and a non-zero exit on `cli-config` and `cli-migration`, not a silent fallback to a discovered file or defaults.
- [x] 3.5 Resolve read-without-bind keys: `defaultConfigOnly` (`internal/config/init.go`) is never bound or defaulted; and `cwd`/`debug` are bound only on `RootCmd`. Defaults are now registered at the start of `config.Init` (`ensureInitDefaults`) so every key read by `Init` is resolvable at read time, before `Load` runs.
- [x] 3.6 Add per-binary config-selection tests (`internal/config/commands/root_test.go`, `internal/database/commands/root_test.go`) using `t.TempDir()` and two distinctly-valued config files, asserting the explicit `--config` file wins and that a missing explicit path fails. The lower-level equivalents remain in `internal/config/load_isolation_test.go`.

## 4. Independent version commands

- [x] 4.1 Stop sharing the single package-level `versionCmd` from `internal/commands/version.go` across trees. Extract the build-info printing body into a shared plain function and register a distinct `*cobra.Command` per root.
- [x] 4.2 Update `internal/commands/config.go:10` and `internal/commands/database.go:10` so the config and database roots use their own `version` commands, leaving `RootCmd` (`internal/commands/root.go:85`) with the main one, and confirm `cli-migrations migrate version` (from `internal/database/commands/version.go`) is unaffected.
- [x] 4.3 Add tests asserting `cli-config version --help` and `cli-migration version --help` print their own usage name and own inherited global flags, and that `version` still prints version/build date/commit ID and exits `0`.

## 5. Error output, container commands, panic log, and show output

- [x] 5.1 Set `SilenceErrors = true` on the config root (`internal/config/commands/root.go`, currently only `SilenceUsage` at L32) and the database root (`internal/database/commands/root.go`, L28), matching `RootCmd` (`internal/commands/root.go:57-58`), so errors are printed exactly once.
- [x] 5.2 Add an `Args`/run guard to `migrateCmd` (`internal/database/commands/migrate.go:40`) so an unknown child (`migrate bogus`) returns an error and exits non-zero, while a bare `migrate` still prints help and exits `0`. Check other pure container commands for the same gap.
- [x] 5.3 Ensure the panic log location is established from the loaded log configuration before commands run (using the existing `SetupPanicLogLocation`/`SetupPanicRecovery` seam in `internal/logging/recover.go`), so recovered panic logs land in the configured log directory and not the process CWD. When the directory is unavailable, still report to stderr and still exit non-zero.
- [x] 5.4 Guard the config source line (implemented in `internal/config/config_print.go` `Config.Print`, which `show` calls) so an empty `viper.ConfigFileUsed()` does not print `Configuration from: ` with a blank path. `show` instead prints `Configuration from: (no config file used)`. Add a scenario-covering assertion.
- [x] 5.5 Add tests for single-error emission on the config/database binaries, unknown-subcommand failure and bare-container help, and the panic-log-directory behavior (extend `internal/logging/recover_test.go`).

## 6. Verification

- [x] 6.1 Update `internal/commands/setup_test.go` fixture strings that embed `defer logging.RecoverPanic("<tag>", nil)` (around L258-260 and L339-341) so they match the new entry-point defer shape and recovery signature.
- [x] 6.2 Run `go build ./...` and `go vet ./...`; both must pass.
- [x] 6.3 Run `go test ./...` (or `make test`); all packages must pass, with the new exit-code, panic, flag-isolation, and version tests included.
- [x] 6.4 Build all three binaries and run a manual exit-code matrix asserting: `version` → 0; unknown command/flag → 1; invalid config → 1; successful `run` → 0 with no `exit status 0` on stderr; recovered panic → 2; `cli-config --config <missing>` → 1 with the required-config error; `cli-migration --json migrate version` and `migrate bogus` → non-zero where applicable.
- [x] 6.5 Confirm each binary's `version --help` names that binary and lists only its own global flags.
- [x] 6.6 Verify no panic log file appears in the process CWD after the forced-panic check, and that the log appears under the configured log directory instead.

## 7. Post-verification fixes

- [x] 7.1 Make `logging.GetLogger()`/`Logger.NewLogger`/`Logger.With` nil-safe (`noopLogger`) so an entry point whose logging setup failed (for example an uncreatable log/data directory) cannot panic with a nil-pointer dereference.
- [x] 7.2 Fix the database root's config-skip guard (`cmd.Parent() == cmd.Root()`) so `cli-migration migrate version` honors explicit `--config`/`--data-dir` and reports `ErrConfigFileRequired` for a missing explicit path, matching the other migration subcommands.
- [x] 7.3 Apply the same root-child guard to the config root's config-skip check.
- [x] 7.4 Add in-process regression tests: nil-safe logging (`internal/logging/logger_test.go`), `migrate version` config selection and real `migrateCmd` unknown-child/bare behavior (`internal/database/commands/root_test.go`), per-binary `version --help` naming (`internal/commands/version_test.go`), config selection + single-error emission (`internal/config/commands/root_test.go`), and entry-point defer order (`internal/commands/entrypoint_test.go`). Each critical test was confirmed to fail against the pre-fix source.
- [x] 7.5 Document the exit-code contract in the README and close the documentation open task.
