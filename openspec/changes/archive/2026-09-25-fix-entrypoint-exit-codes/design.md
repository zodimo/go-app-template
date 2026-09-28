## Context

Three `cmd/` entry points build on shared `internal/` packages. Each `main()` uses the same shape:

```go
func main() {
	var exitCode uint8
	defer logging.RecoverPanic("X-main", nil)          // registered FIRST
	defer func() { if exitCode != 0 { os.Exit(int(exitCode)) } }()  // registered SECOND
	exitCode = commands.ExecuteCmd(<Root>)
}
```

`ExecuteCmd` (internal/commands/root.go) runs `cmd.Execute()`, sets `exitCode = 1` on a non-nil error, prints the error (JSON if requested), then maps a typed `exitcode.ExitStatus` onto the code. `App.Run()` returns `exitcode.ExitStatus`, which implements `error`, so its zero value is a non-nil interface value.

Verified defects (reproduced against built binaries):

- **Panic exits 0.** Defers run LIFO, so `RecoverPanic` (registered first) runs last — after the exit defer. It swallows the panic; the exit defer then sees `exitCode == 0` and never calls `os.Exit`. A forced panic produced `EXIT=0`.
- **`exit status 0` on success.** `errors.AsType[exitcode.ExitStatus](ExitStatusOK)` succeeds while `err != nil` is true, so every successful `cli run` printed `exit status 0` to stderr then exited 0.
- **`--config` ignored on two binaries.** `internal/config/commands/root.go` and `internal/database/commands/root.go` both bind the viper key `config-config-file`; the database root's `PersistentPreRunE` reads `root-config-file`, which nothing binds on that binary. `cli-config --config <path>` and `cli-migration --config <path>` loaded a discovered default instead; a nonexistent explicit path did not raise the "config file required" error.
- **One `versionCmd` pointer in three trees.** `internal/commands/version.go` defines a single package-level command added to `RootCmd`, `ConfigCmd`, and `DatabaseCmd`. Its `parent` is whichever tree registered last, so `cli-config version --help` prints `Usage: cli version` with cli-only flags.
- **Doubled errors, unknown-child success, CWD panic logs, empty source path.** The config/database roots set only `SilenceUsage`, so cobra prints `Error: ...` and `ExecuteCmd` prints again; `migrate bogus` prints help and exits 0; `panicLogLocation` is never configured by the entry points, so panic logs land in the working directory; `show` prints `Configuration from: ` blank when no file was used.

The tree is otherwise disciplined (generated identity single-source, table of existing specs, passing tests), so this is a correctness repair, not a restructure.

## Goals / Non-Goals

**Goals:**
- Make every command outcome map to a deterministic process exit code across all three binaries.
- Ensure a recovered panic can never be reported as success.
- Eliminate duplicated error output and the spurious success-path message.
- Make `--config` and `--data-dir` truthful on the binary that declares them.
- Give each binary an independent `version` command and independent flag surface.
- Keep panic logs in the configured directory and keep container commands honest about unknown children.
- Add tests that actually exercise exit-code mapping and panic-to-exit behavior.

**Non-Goals:**
- Not collapsing the three binaries into one; the trio split is a deliberate, preserved pattern.
- Not replacing cobra or viper, and not introducing a new dependency.
- Not redesigning the config model, identity single-source, or migration tooling.
- Not changing application behavior in `internal/app` beyond preserving the typed-`ExitStatus` return convention.
- Not changing the documented clone setup flow or the `setup` wizard.

## Decisions

### D1 — Panic fix by defer order plus an explicit non-zero code

Reorder each `main()` so the exit-code defer is registered **first** (runs last, after recovery has set the code) and `RecoverPanic` is registered **second** (runs first on unwind). Give recovery a way to contribute a non-zero code. Chosen as confirmed with the user.

```
defer exitWithCode(&exitCode)             // registered FIRST  → runs LAST
defer logging.RecoverPanic(name, &exitCode) // registered SECOND → runs FIRST, sets 2 on panic
exitCode = commands.ExecuteCmd(root)
```

`RecoverPanic` gains an exit-code-out parameter (or equivalent) and sets `2` when it recovers. Alternative considered: have `RecoverPanic` call `os.Exit(2)` directly — rejected because it couples the logging package to process lifecycle and skips `main`'s deferred cleanup. Alternative: pass a pointer only through a new wrapper — rejected as more plumbing for the same result. The existing `cleanup func()` signature is extended, not replaced, so the two `recover_test.go` call sites can be migrated deliberately.

### D2 — Fix the error-interface test in `ExecuteCmd`

Stop branching on `err != nil` alone. Distinguish three cases in order: (1) `err == nil` → success; (2) `err` is an `exitcode.ExitStatus` → use its numeric value directly, including `0` (no error printing); (3) otherwise a genuine error → set `1`, print once.

```
switch {
case err == nil:
    // exit 0
default:
    if es, ok := errors.AsType[exitcode.ExitStatus](err); ok {
        exitCode = uint8(es)              // 0 → success, non-zero → that code
        if es == exitcode.ExitStatusOK { return }
    } else {
        exitCode = 1
    }
    // print err exactly once
}
```

This preserves the documented convention that a command may return `App.Run()`'s value directly, and it makes the non-zero typed path real. Alternative: have `App.Run()` return `error` and wrap — rejected because the template teaches the typed status return and the `run.go` command already returns it directly.

### D3 — Per-binary viper isolation for config selection

The root cause is process-global viper shared by three `init()` functions that bind the same key. Two viable shapes:

- **A. Per-binary binding keys.** Give the config and database roots distinct, correctly-read keys (`config-config-file` for the config root; a correctly named key for the database root that its `PersistentPreRunE` actually reads), and ensure the main root keeps `root-config-file`. Minimal change.
- **B. Independent viper instances per binary.** Construct a `viper.New()` in each binary's wiring and pass it down, removing the global. Larger change touching every `viper.Get*` call site.

Decision: **A** for this change, because the immediate defect is a mis-bound/mis-read key and the fix is provably local; B is recorded as a follow-up. Either way, the invariant "no viper key bound by two roots; every read key is bound or defaulted" is enforced by the new capability spec and tests. Also remove the dead duplicate binding in the database root and make its read key match its bind key.

### D4 — Independent `version` commands

Move from one shared `versionCmd` pointer to one command instance per root. The build-info printing body is shared as a plain function; the command values themselves are not shared. This fixes `Use`, inherited flags, and help text per binary without duplicating logic. Alternative: keep one command and clone it — rejected; cobra commands are stateful and pointer-sharing is exactly the bug.

### D5 — Consistent silence and error ownership

Set `SilenceUsage` and `SilenceErrors` on all three roots (the main root already sets both). `ExecuteCmd` remains the single writer of error text. This removes the doubled output seen on `cli-config`/`cli-migration` and keeps the main binary unchanged.

### D6 — Container commands fail on unknown children

Give subcommand-group commands (`migrate`, and any other pure container) an `Args`/run guard that returns an error for an unrecognised child while still printing help for the bare invocation. cobra reports unknown subcommands only when the parent has no `Run`; the container currently falls through to help with exit 0. The guard restores the error path.

### D7 — Panic log location via existing configuration

Ensure each entry point establishes the panic log directory from the loaded log configuration (using the existing `SetupPanicLogLocation`/`SetupPanicRecovery` seam) before commands run, so recovered panics log to the configured directory. When the directory is unavailable, still report to stderr and still exit non-zero. This keeps the logging package free of hardcoded paths and preserves the existing tests' use of `SetupPanicRecovery`.

### D8 — `show` output when no file was used

Guard the source line on a non-empty `viper.ConfigFileUsed()`; otherwise print an explicit "no config file used" form or omit the line. Purely presentational.

## Risks / Trade-offs

- **[Changing `RecoverPanic`'s signature breaks its two test call sites]** → Update `internal/logging/recover_test.go` in the same change; the behavior under test (nil logger, single stack trace, sanitized app name) is preserved.
- **[Panic exit code choice]** → Use `2`, the conventional panic/shell-misuse code, so it is distinguishable from ordinary error `1`. The spec only requires non-zero and distinct.
- **[`--config` starts failing where it silently fell back]** → This is intended and matches the documented "explicit path is always required" rule, but it is observable behavior change for `cli-config`/`cli-migration`; call it out as **BREAKING** in release notes.
- **[Per-binary key fix may only mask a deeper global-viper coupling]** → The spec's invariant plus a test that invokes each binary's `--config` independent of discovery contains the risk; the independent-viper-instance option (D3-B) stays documented as the follow-up if further collisions surface.
- **[Container-command Args guard could regress `--help`/bare invocation]** → Explicit scenarios require bare container invocation to still print help and exit 0.
- **[Tests that spawn real binaries need a build step]** → Prefer in-process tests of `ExecuteCmd` and recovery for the mapping logic; reserve built-binary smoke tests for flag isolation only if needed. Keep added test wall-time small.
- **[Changing `main()` affects all three binaries via copy-paste]** → Apply the identical shape to all three; a shared helper for the defer pair is acceptable but must not reintroduce shared command state.

## Migration Plan

1. Fix `ExecuteCmd` mapping (D2) and add in-process tests for success/error/typed-status — this is the load-bearing change.
2. Reorder entry-point defers and extend recovery to set a non-zero code (D1); migrate the two recovery tests.
3. De-collide viper keys and make the database/config roots read the keys they bind (D3); add per-binary `--config` tests.
4. Replace the shared `versionCmd` with per-root commands (D4); add usage/flag assertions.
5. Apply silence consistency (D5), container `Args` guard (D6), panic log location (D7), and the `show` source guard (D8).
6. Run `go build ./...`, `go vet ./...`, `go test ./...`, and a manual exit-code matrix over the built binaries.

Rollback: each step is independent and behavior-preserving except the intended fixes; revert by step if a regression appears. No data migration is involved.

## Post-verification addenda

### D9 — Config-skip must be scoped to direct root children

The database root's `PersistentPreRunE` originally skipped config whenever `cmd.Name() == "version"`. That also matched the `migrate version` child, so `cli-migration --config <path> migrate version` silently ignored the explicit path (and `--data-dir`), violating the `command-flag-isolation` requirement. The guard is now scoped with `cmd.Parent() == cmd.Root()`, so only the root-level build-info `version` and the `migrate` container skip config; every real migration command (including `migrate version`) loads it. The same scoping is applied to the config root (`version`, `init`). Note `cmd.Parent() == RootConfigCmd` cannot be used: it creates an initialization cycle because the closure is part of the root's own initializer.

### D10 — Logger access is nil-safe

`GetLogger()` could return a nil `*Logger` when `logging.Setup` failed (for example an uncreatable log/data directory). Because `Load` swallows the setup error with a warning and continues, any later `GetLogger().NewLogger(...)` panicked with a nil-pointer dereference — a regression exposed once `--data-dir` genuinely reached `logging.Setup`. `GetLogger`, `Logger.NewLogger`, and `Logger.With` now return a discard `noopLogger` when the logger or its embedded `*slog.Logger` is nil, so a logging-setup failure degrades to silent logging instead of a crash while the command still runs.
