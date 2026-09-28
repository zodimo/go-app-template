## MODIFIED Requirements

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
