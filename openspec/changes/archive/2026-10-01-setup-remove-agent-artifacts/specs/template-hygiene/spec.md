## ADDED Requirements

### Requirement: Agent and IDE scaffolding does not survive into a downstream project
A project cloned from this template SHALL NOT retain the source repository's agent or IDE scaffolding. The `setup` command MUST offer to remove the following from the clone, defaulting to Yes: the `.opencode/` directory, the `openspec/` directory, any `*.code-workspace` file, and the `AGENTS.md` and `CLAUDE.md` files. Removal MUST be idempotent (absent paths are a no-op), MUST be reported through the existing plan/apply mechanism (`setup --dry-run` prints it), and MUST be guarded so the authoring repository's own `openspec/` change history cannot be destroyed.

#### Scenario: Agent scaffolding is offered for removal
- **WHEN** a consumer runs `<app> setup` in a clone
- **THEN** the command offers to remove `.opencode/`, `openspec/`, any `*.code-workspace` file, `AGENTS.md`, and `CLAUDE.md`, defaulting to Yes

#### Scenario: Consented scaffolding is removed
- **WHEN** a consumer consents and the listed paths exist
- **THEN** the command removes `.opencode/`, `openspec/`, and any matching `*.code-workspace` file present in the workspace root

#### Scenario: Absent scaffolding is tolerated
- **WHEN** some or all of the listed paths do not exist in the clone
- **THEN** the command completes successfully and treats the missing paths as nothing to remove

#### Scenario: Removal is previewed by dry-run
- **WHEN** a consumer runs `<app> setup --dry-run`
- **THEN** the command prints each agent-artifact removal it would perform, alongside the identity rewrites, without deleting anything

#### Scenario: Authoring repository is guarded
- **WHEN** `setup` runs where `openspec/` holds uncommitted tracked changes
- **THEN** the command does not remove `openspec/` in that workspace
