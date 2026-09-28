# Go App Template

This project is in active Developement

# GOAL 

A production-ready Go CLI starter template, authored by zodimo. It gives you a solid, opinionated foundation for a command-line application so you can skip the plumbing and get to the parts that matter for your own tool.

The template ships three binaries, configurable structured logging, a config layer that reads from files and environment, and SQLite-backed migrations. It is designed to be cloned, renamed, and re-identified for your own project using the clone flow below.

## About

`go-app-template` is a complete CLI skeleton built around three small binaries (the "binary trio"), each acting as a `cmd/` entrypoint into a shared set of internal packages:

- **`cli`** (`cmd/cli`). The main application.
  Root command `use: cli` with subcommands `run` and `version`.
- **`cli-config`** (`cmd/cli-config`). Configuration management.
  Root command `use: cli-config` with subcommands `init`, `validate`, `show`, and `version`. `cli-config init` generates the config file on first run.
- **`cli-migration`** (`cmd/cli-migration`). Database and migrations.
  Root command `use: cli-migrations` with subcommand `migrate` (and its own subcommands `create`, `down`, `force`, `goto`, `up`, `validate`, `version`) plus `version`.

The architecture keeps concerns separate:

- **Command routing** via `github.com/spf13/cobra`.
- **Configuration** via `github.com/spf13/viper`, layered from defaults, config files, and environment variables.
- **Logging** as structured JSON via the standard `log/slog`, with rotation through `gopkg.in/natefinch/lumberjack.v2`.
- **Database** as SQLite through `modernc.org/sqlite` (pure Go, no CGO), with schema migrations driven by `github.com/golang-migrate/migrate/v4`.

## Prerequisites

- **Go 1.27 or newer.**
- The external dependency `github.com/zodimo/go-maybe` (used for `maybe.Maybe[T]` config fields). It is a required direct dependency declared in `go.mod` and is NOT vendored, so you need module downloads (or a proxy) available when building.

## Building

Build all three binaries and every package:

```sh
make build
```

This runs `go build ./...`. You can also use the Go toolchain directly:

```sh
go build ./cmd/cli ./cmd/cli-config ./cmd/cli-migration
```

## Runtime Flags

These persistent flags live on the main `cli` root command and apply to all of its subcommands:

| Flag | Type | Description |
| --- | --- | --- |
| `--config` | string | cli config file |
| `--require-config` | bool | fail if no config file is found |
| `--debug` | bool | Debug (overrides config/env) |
| `--json` | bool | Output in JSON |
| `--cwd` | string | current working directory |

`--config` points at a specific config file. `--require-config` turns a missing config file into an error even when no explicit `--config` path is given, instead of silently falling back to defaults. `--debug` forces debug mode and overrides whatever the config or environment says. `--json` switches error output to JSON. `--cwd` lets you run the app from a different working directory, which the config layer honors when resolving defaults and search paths.

## Config Resolution

Configuration uses viper and resolves from a set of sources in order of precedence:

1. Flags (bound via cobra persistent flags).
2. Environment variables (prefix `GOCLI`, derived from the app identity's `EnvPrefix`).
3. A config file found in the search paths.
4. Built-in defaults.

Config file search paths include, in order:

- the current working directory,
- `$HOME`,
- `$XDG_CONFIG_HOME`,
- `$HOME/.config`,
- `/opt/<AppName>`.

Environment variable keys use the `GOCLI` prefix and convert `.` and `-` separators to `_`. For example, the config key `database.database` reads from `GOCLI_DATABASE_DATABASE`.

The config file is generated on first run with:

```sh
make config-init
# or
go run ./cmd/cli-config init
```

`cli-config init` writes a JSON config file (default name derived from identity, `<ConfigName>.json`) and does not overwrite an existing one. Additional management is available through `cli-config validate` (check an existing config) and `cli-config show` (display the effective config).

The `--config`, `--require-config`, and `--cwd` flags shape resolution: `--config` selects an explicit file (a missing file at an explicit path is always an error), `--require-config` opts into strict mode when relying on the search paths, and `--cwd` sets the working directory used when building default paths.

## Logger

Logging is written as structured JSON through the standard library's `log/slog` handler.

- **Rotation** is handled by `gopkg.in/natefinch/lumberjack.v2`, so logs roll over by size and age automatically instead of growing forever.
- **Panic recovery** catches panics and writes them to the panic log, so a crash leaves an actionable trace behind.
- **Debug mode** raises the log level to debug. It is forced by the `--debug` flag, which overrides the config and environment.

Default log settings live in the config under the `log` section (max size, max backups, max age, and compress all default to sane values), and the log file path derives from the app identity.

## Database

The template uses SQLite via `modernc.org/sqlite`, a pure-Go driver with no CGO dependency.

Migrations are managed with `github.com/golang-migrate/migrate/v4`, and the migration source directory is `internal/database/migrations/`. The default database file name derives from the app identity as `<AppName>-data.db` (for example, `go-cli-data.db`), placed under the data directory.

Drive migrations with the dedicated binary:

```sh
go run ./cmd/cli-migration migrate up      # apply all pending migrations
go run ./cmd/cli-migration migrate up <N>  # apply the next N
go run ./cmd/cli-migration migrate down <N># roll back the most recent N
go run ./cmd/cli-migration migrate create <name>  # generate new migration files
go run ./cmd/cli-migration migrate version # print current version + dirty flag
```

`up --dry-run` previews pending migrations. golang-migrate tracks applied versions in a `schema_migrations` table inside the target database, so re-running is idempotent, which makes `cli-migration` well suited to CI/CD.

## Version

Each binary exposes a `version` subcommand:

```sh
./cli version
```

It prints the app version, build date, and commit ID. These values come from `internal/version` and can be embedded at build time via ldflags, so a released binary carries the exact commit and build date it was produced from.

## Exit Codes

Every binary maps a command outcome to a deterministic process exit code at the `cmd/` entry point:

| Outcome | Exit code |
| --- | --- |
| Command returns no error | `0` |
| Command returns a plain error (unknown flag/command, missing required config, ...) | `1` |
| Command returns a non-zero `exitcode.ExitStatus` | that code |
| Command returns `exitcode.ExitStatusOK` (the zero value) | `0`, with nothing written to stderr |
| Panic recovered at the entry point | `2` |

The mapping lives in `commands.ExecuteCmd` (`internal/commands/root.go`); each `main` assigns its result to `exitCode` and exits non-zero only when it is non-zero. Panic recovery is a `defer` registered *after* (so it runs *before*) the exit-code `defer`, and it sets the code to `2`, so a crash can never be reported as success. `2` is deliberately distinct from the ordinary-error code `1`.

A command can request a specific exit code by returning `exitcode.NewExitStatus(n)`; `App.Run()` already returns an `exitcode.ExitStatus`, which the template teaches you to propagate directly.

```sh
./cli version                 # 0
./cli bogus                   # 1, one error line on stderr
./cli --config missing.json run  # 1, "config file required but not found"
```

Errors are written exactly once: `SilenceErrors` is set on all three roots so cobra stays quiet and `ExecuteCmd` is the single writer (plain text, or a single JSON document when `--json` is enabled on the main `cli` binary).

## Layout

A brief map of the source tree:

```
cmd/
  cli/              # main application entrypoint (run, version)
  cli-config/       # config management entrypoint (init, validate, show, version)
  cli-migration/    # database/migrations entrypoint (migrate ..., version)
internal/
  app/              # application runtime (Run)
  commands/         # cobra commands for the main cli binary
  config/           # viper config loading, search paths, defaults
  config/commands/  # cobra commands for the cli-config binary
  core/             # config context, validation, redaction helpers
  database/         # SQLite store and golang-migrate wiring
  database/commands/# cobra commands for the cli-migration binary
  identity/         # single source of app identity (generated)
  logging/          # slog + lumberjack setup, panic recovery
  version/          # version/build-info handling
```

## Clone Flow

To turn this template into your own application, follow two complementary layers. `gonew` rewrites Go code and the module path, then `setup` re-identifies the app.

**Layer 1. module re-typing with `gonew`:**

```sh
gonew github.com/zodimo/go-app-template@<version> your.domain/myapp
```

`gonew` performs an AST-aware copy: it rewrites the module path in `go.mod` (to `your.domain/myapp`) and rewrites all internal imports to match. It only touches `.go` files and `go.mod`. It leaves identity, the `cmd/` directory names, cobra command names, and settings like `.gitignore` alone.

**Layer 2. identity with `setup`:**

```sh
cd myapp
go run ./cmd/cli setup
```

The `setup` command (a cobra wizard) completes the clone by re-identifying the project. It collects an AppName, environment prefix, data directory, and binary-trio prefix, then:

- regenerates `internal/identity/identity.go`,
- renames `cmd/cli` → `cmd/myapp`, `cmd/cli-config` → `cmd/myapp-config`, `cmd/cli-migration` → `cmd/myapp-migration`,
- rewrites the cobra `use:` strings, panic tags, the Makefile's `BIN_TRIO` line, and related identity-dependent files.

**Then build and initialize:**

```sh
go build ./...
myapp-config init   # generates your config file (never shipped)
```

Because `gonew` stops at Go code and `setup` stops at identity, the two layers are complementary: `gonew` gives you correct module plumbing, and `setup` gives you an app whose name, config file, environment prefix, data directory, log file, and database name all consistently derive from your new identity.

Neither layer needs a manual Makefile edit. After `gonew` rewrites `go.mod`, the Makefile derives the module path from `go list -m` at make time, so the linker `-X` flags never point at a stale module. After `setup` renames the binary trio, it rewrites the Makefile's single `BIN_TRIO ?= ...` line, so `make build` keeps targeting the renamed `cmd/` directories.

See `CONTRIBUTING.md` for the same flow restated for contributors, plus the identity single-source contract.

## License

This repository is a public template. Add a license that fits your project before publishing a fork.