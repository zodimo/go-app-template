# Contributing

Thanks for helping with `go-app-template`. This template is a public starting point, so contributions tend to fall into two buckets: improving the template itself, or turning it into your own project. Both are covered here.

## Template reuse

If you want to build your own CLI from this template, the flow is deliberate, and the split between `gonew` and `setup` matters. Do not skip Layer 1 and hand-edit imports, and do not skip Layer 2 and leave the default identity in place.

**Layer 1. module re-typing with `gonew`:**

```sh
gonew github.com/zodimo/go-app-template@<version> your.domain/myapp
```

`gonew` does an AST-aware copy. It rewrites the module path in `go.mod` (to `your.domain/myapp`) and rewrites all internal imports to match. It touches only `.go` files and `go.mod`. It deliberately leaves alone: `internal/identity/identity.go`, the `cmd/` directory names, cobra `use:` strings, and settings such as `.gitignore`.

**Layer 2. identity with `setup`:**

```sh
cd your.domain/myapp
myapp setup
```

The `setup` command is an interactive cobra wizard that completes the clone. It collects an AppName, environment prefix, data directory, and binary-trio prefix, then:

- regenerates `internal/identity/identity.go` from the collected values,
- renames `cmd/cli` → `cmd/myapp`, `cmd/cli-config` → `cmd/myapp-config`, `cmd/cli-migration` → `cmd/myapp-migration`,
- rewrites the cobra `use:` strings, panic tags, and related identity-dependent files.

It then offers, **defaulting to Yes**, to remove the template's agent/IDE scaffolding from the clone: `.opencode/`, `openspec/`, any `*.code-workspace` file, `AGENTS.md`, and `CLAUDE.md`. The removals run after the identity rewrites, are skipped for paths that are already absent, and leave `openspec/` in place when it looks like the authoring repository (uncommitted tracked changes). `setup --dry-run` prints the plan, including the removals, without deleting anything.

It is idempotent: running it again with the same values makes no changes, and running it with new values re-identifies consistently. Note that `gonew` (Layer 1) copies only `.go` files and `go.mod`, and `.opencode/` is git-ignored and untracked, so a `gonew`-created project usually has none of this scaffolding — the cleanup step is defensive and does real work only when the project was seeded another way.

**Then build and generate your config:**

```sh
go build ./...
myapp-config init   # generates the config file; never committed
```

## The identity single-source contract

`internal/identity/identity.go` is the **single source of app identity**. It holds `AppName`, `ConfigName`, `EnvPrefix`, `DataDir`, and `BinTrio`. This file is **generated** by the `setup` command. Do not edit it by hand; re-run `setup` to change identity.

`setup` owns the binary-trio prefix wherever it appears in the template. During re-identification it rewrites the Makefile's `BIN_TRIO ?= <value>` line, alongside `internal/identity/identity.go`, the `cmd/` directories, the cobra `use:` strings, the panic tags, and `.gitignore`. Do not hand-edit that line. The Makefile must not hardcode the module path either: it derives it from `go list -m` at make time, so `gonew` needs no Makefile follow-up.

Every config, logging, database, and filesystem default in the project reads from these values:

- the config file base name comes from `ConfigName`,
- the environment prefix comes from `EnvPrefix`,
- the default data directory derives from `DataDir`,
- the log filename derives from `AppName`,
- the default database name derives from `AppName` (`<AppName>-data.db`),
- config search paths include `/opt/<AppName>`.

**Rule for new code:** when you add code that references the app name, config name, env prefix, data directory, log file, or any filesystem location, you MUST read from `identity.*`. Never hardcode a literal. If a literal exists in the template, route it through `identity` instead. Keep the `setup` command as the regeneration path for these values.

## The three-binary split is deliberate

`cli`, `cli-config`, and `cli-migration` exist as separate entrypoints on purpose:

- `cli` is the application itself.
- `cli-config` manages configuration (init, validate, show).
- `cli-migration` manages the database and schema migrations (and is the CI/CD surface for schema changes).

Preserve this split when you extend the template. Do not merge config or migration tooling back into the main `cli` binary. Each binary stays a thin `cmd/` wrapper over shared internal packages.

## Development workflow

Run the checks locally before opening a PR:

```sh
make build     # go build ./...
make test      # go test ./...
make vet       # go vet ./...
```

There are also convenience targets that match the README surface:

```sh
make config-init   # go run ./cmd/cli-config init
make migrate       # go run ./cmd/cli-migration migrate
```

Keep the docs in `README.md` accurate: if you change a runtime flag, a config search path, a default file location, or a command name, update the README (and this file if the contribution contract changes) in the same change. Tests run through `go test ./...`; add tests for new packages or behaviors you introduce.