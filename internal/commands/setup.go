package commands

import (
	"bufio"
	"errors"
	"fmt"
	"go/format"
	"os"
	osexec "os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zodimo/go-app-template/internal/identity"
)

// setupCmd re-identifies a cloned project. It prompts for an application name,
// environment prefix, data directory, binary-trio prefix, and whether to remove
// the template's agent/IDE scaffolding, then rewrites the identity-derived
// artifacts of the tree: internal/identity/identity.go, the three cmd/ entrypoint
// directories, the cobra Use strings, the panic-recovery tags, the Makefile
// BIN_TRIO line, and the identity lines in .gitignore. When consented, it also
// removes .opencode/, openspec/, *.code-workspace, AGENTS.md, and CLAUDE.md.
//
// The command is idempotent: re-running with the same values is a no-op, and
// re-running with new values re-identifies the tree consistently (because the
// plan is always computed from the live identity package).
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Re-identify the project interactively",
	Long: `Re-identify this project by rewriting its identity-derived artifacts.

Prompts for an application name, environment prefix, data directory, and
binary-trio prefix, then regenerates internal/identity/identity.go, renames the
command entrypoint directories, and rewrites cobra Use strings, panic tags, the
Makefile BIN_TRIO line, and .gitignore entries to match. It then offers,
defaulting to Yes, to remove the template's agent/IDE scaffolding
(.opencode/, openspec/, *.code-workspace, AGENTS.md, CLAUDE.md); removals run
after the identity rewrites and are skipped when the path is already absent or
when openspec/ appears to be the authoring repository. All inputs are validated
before any mutation, and the operation is idempotent.`,
	RunE: runSetup,
}

// setupDryRun controls whether the wizard only prints its plan. It is a local
// flag on setupCmd so it does not leak onto sibling commands or the config/
// migration roots.
var setupDryRun bool

func init() {
	setupCmd.Flags().BoolVar(&setupDryRun, "dry-run", false, "print planned changes without applying them")
}

// identityVals is the collected identity. ConfigName is derived (never prompted)
// as "<AppName>-config".
type identityVals struct {
	AppName    string
	ConfigName string
	EnvPrefix  string
	DataDir    string
	BinTrio    string
}

// newIdentityVals derives the target identity from the raw prompted fields,
// normalizing EnvPrefix to uppercase and deriving ConfigName.
func newIdentityVals(appName, envPrefix, dataDir, binTrio string) identityVals {
	return identityVals{
		AppName:    appName,
		ConfigName: appName + "-config",
		EnvPrefix:  strings.ToUpper(envPrefix),
		DataDir:    dataDir,
		BinTrio:    binTrio,
	}
}

// currentIdentity reads the live identity package for defaults and idempotency.
func currentIdentity() identityVals {
	return identityVals{
		AppName:    identity.AppName,
		ConfigName: identity.ConfigName,
		EnvPrefix:  identity.EnvPrefix,
		DataDir:    identity.DataDir,
		BinTrio:    identity.BinTrio,
	}
}

// sameValues reports whether next is identical to cur (fast idempotency path).
func sameValues(cur, next identityVals) bool {
	return cur.AppName == next.AppName &&
		cur.ConfigName == next.ConfigName &&
		cur.EnvPrefix == next.EnvPrefix &&
		cur.DataDir == next.DataDir &&
		cur.BinTrio == next.BinTrio
}

// identRe matches a clean identifier: starts with a letter, then letters/digits/
// underscores. It is used for EnvPrefix and BinTrio, which must form valid Go
// and environment identifiers and valid directory names (no hyphens, so every
// derived `<bin>-config`/`<bin>-migration` name is also a valid package dir).
var identRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// validateIdentity rejects inputs that would produce invalid Go code, invalid
// environment variables, or invalid directory names, before any mutation.
func validateIdentity(v identityVals) error {
	if strings.TrimSpace(v.AppName) == "" {
		return errors.New("app name must not be empty")
	}
	if strings.ContainsAny(v.AppName, " \t\r\n/\\") {
		return fmt.Errorf("invalid app name %q: must not contain whitespace or path separators", v.AppName)
	}

	if v.EnvPrefix == "" {
		return errors.New("env prefix must not be empty")
	}
	if !identRe.MatchString(v.EnvPrefix) {
		return fmt.Errorf("invalid env prefix %q: must match %q (letters, digits, underscore; no leading digit)", v.EnvPrefix, identRe.String())
	}

	if strings.TrimSpace(v.DataDir) == "" {
		return errors.New("data dir must not be empty")
	}
	if strings.ContainsAny(v.DataDir, " \t\r\n/\\") {
		return fmt.Errorf("invalid data dir %q: must not contain whitespace or path separators", v.DataDir)
	}

	if strings.TrimSpace(v.BinTrio) == "" {
		return errors.New("binary trio prefix must not be empty")
	}
	if !identRe.MatchString(v.BinTrio) {
		return fmt.Errorf("invalid binary trio prefix %q: must match %q (letters, digits, underscore; no leading digit; no hyphens)", v.BinTrio, identRe.String())
	}

	return nil
}

// changeKind enumerates the mutation shapes.
type changeKind string

const (
	kindRename  changeKind = "rename"
	kindWrite   changeKind = "write"
	kindRewrite changeKind = "rewrite"
	kindRemove  changeKind = "remove"
)

// change is a single planned mutation.
//
//   - rename:  From -> To (directory rename)
//   - write:   Path <- Content (full-file regeneration: identity.go, .gitignore)
//   - rewrite: Path: replace the literal Old with New (Use lines, panic tags)
//   - remove:  Path: delete the file or directory at Path (agent scaffolding)
type change struct {
	Kind          changeKind
	From          string // rename source
	To            string // rename destination
	Path          string // write/rewrite target
	Content       []byte // write content
	Old           string // rewrite literal to find
	New           string // rewrite literal to substitute
	Desc          string // human-readable description for --dry-run
	SkipIfMissing bool   // rewrite/remove: skip (no error) when the file/dir is absent
	ExpectUnique  bool   // rewrite: error unless the literal occurs exactly once
	Skip          bool   // remove: never execute (guarded); reported but left in place
}

// plan is the ordered set of changes.
type plan struct {
	Changes []change
}

func (p plan) empty() bool { return len(p.Changes) == 0 }

// identityPaths holds all identity-derived filesystem locations for a given
// identity value, relative to the workspace root. It is identity-driven so the
// same list works before and after a rename.
type identityPaths struct {
	IdentityFile  string
	CmdDir        string
	ConfigDir     string
	MigrationDir  string
	RootUseFile   string
	ConfigUseFile string
	DBUseFile     string
	CmdMain       string
	ConfigMain    string
	MigrationMain string
	Gitignore     string
	Makefile      string
}

func findPaths(root string, v identityVals) identityPaths {
	return identityPaths{
		IdentityFile:  filepath.Join(root, "internal", "identity", "identity.go"),
		CmdDir:        filepath.Join(root, "cmd", v.BinTrio),
		ConfigDir:     filepath.Join(root, "cmd", v.BinTrio+"-config"),
		MigrationDir:  filepath.Join(root, "cmd", v.BinTrio+"-migration"),
		RootUseFile:   filepath.Join(root, "internal", "commands", "root.go"),
		ConfigUseFile: filepath.Join(root, "internal", "config", "commands", "root.go"),
		DBUseFile:     filepath.Join(root, "internal", "database", "commands", "root.go"),
		CmdMain:       filepath.Join(root, "cmd", v.BinTrio, "main.go"),
		ConfigMain:    filepath.Join(root, "cmd", v.BinTrio+"-config", "main.go"),
		MigrationMain: filepath.Join(root, "cmd", v.BinTrio+"-migration", "main.go"),
		Gitignore:     filepath.Join(root, ".gitignore"),
		Makefile:      filepath.Join(root, "Makefile"),
	}
}

// agentArtifactSet is the fixed, root-relative list of agent/IDE scaffolding that
// belongs to the template repository, not to a downstream project. It is returned
// as literal paths (directories and files); `*.code-workspace` is the one glob and
// is expanded separately so a repo with no workspace file yields no entry.
var agentArtifactSet = []string{
	".opencode",
	"openspec",
	"AGENTS.md",
	"CLAUDE.md",
}

// agentArtifactPaths resolves the scaffolding removal set for root: the fixed
// literal paths plus every `*.code-workspace` match, returned workspace-rooted.
// It does not stat the paths; existence is checked when the plan is built so the
// list stays a pure function of root.
func agentArtifactPaths(root string) []string {
	paths := make([]string, 0, len(agentArtifactSet)+1)
	for _, rel := range agentArtifactSet {
		paths = append(paths, filepath.Join(root, rel))
	}
	if matches, err := filepath.Glob(filepath.Join(root, "*.code-workspace")); err == nil {
		paths = append(paths, matches...)
	}
	return paths
}

// openspecDir is the single agent artifact whose removal could destroy the
// authoring repository's own change history, so it is guarded separately.
func openspecDir(root string) string { return filepath.Join(root, "openspec") }

// shouldSkipOpenspec reports whether the openspec/ directory looks like the
// authoring repository (i.e. holds uncommitted tracked changes) and must not be
// removed. It shells out to git; when git is unavailable or the workspace is not
// a repository it returns false (skip nothing), leaving removal gated solely by
// the operator's consent.
func shouldSkipOpenspec(root string) bool {
	cmd := osexec.Command("git", "status", "--porcelain", "--", "openspec")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}

// resolveRemovals turns the operator's consent into the concrete list of
// removals to plan: only paths that currently exist are included (so a re-run
// over an already-clean tree plans nothing), and the guarded openspec/ target
// is marked Skip rather than dropped so it still surfaces in --dry-run. It
// performs the filesystem and git I/O so buildPlan can stay pure.
func resolveRemovals(root string, consent bool) []removal {
	if !consent {
		return nil
	}
	var removals []removal
	for _, p := range agentArtifactPaths(root) {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		removals = append(removals, removal{Path: p, Skip: p == openspecDir(root) && shouldSkipOpenspec(root)})
	}
	return removals
}

// buildIdentityContent renders the regenerated identity.go source for v,
// preserving the exact "GENERATED by `setup`" comment block and all five vars.
// The output is run through go/format so it is always gofmt-clean regardless of
// the chosen values (lengths vary, so manual comment alignment is not reliable).
func buildIdentityContent(v identityVals) ([]byte, error) {
	var b strings.Builder
	b.WriteString("// Package identity is the single source of truth for application identity.\n")
	b.WriteString("//\n")
	b.WriteString("// This file is GENERATED by the `setup` command. Do not edit by hand —\n")
	b.WriteString("// re-run `setup` to change identity. All config/logging/database defaults\n")
	b.WriteString("// and filesystem locations derive from the values below.\n")
	b.WriteString("package identity\n")
	b.WriteString("\n")
	b.WriteString("var (\n")
	fmt.Fprintf(&b, "\tAppName = %q // human/app name\n", v.AppName)
	fmt.Fprintf(&b, "\tConfigName = %q // config file base name\n", v.ConfigName)
	fmt.Fprintf(&b, "\tEnvPrefix = %q // environment variable prefix\n", v.EnvPrefix)
	fmt.Fprintf(&b, "\tDataDir = %q // runtime data directory\n", v.DataDir)
	fmt.Fprintf(&b, "\tBinTrio = %q // binary prefix: %s, %s-config, %s-migration\n", v.BinTrio, v.BinTrio, v.BinTrio, v.BinTrio)
	b.WriteString(")\n")

	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return nil, fmt.Errorf("formatting generated identity source: %w", err)
	}
	return formatted, nil
}

// gitignoreContent renders the four .gitignore lines, keeping .env and *.log
// verbatim and deriving the data-dir and config-name lines from v.
func gitignoreContent(v identityVals) []byte {
	var b strings.Builder
	b.WriteString(".env\n")
	b.WriteString("*.log\n")
	fmt.Fprintf(&b, "/%s\n", v.DataDir)
	fmt.Fprintf(&b, "/%s.json\n", v.ConfigName)
	return []byte(b.String())
}

// removal is a single agent/IDE scaffolding target considered for deletion. A
// target is resolved (and guarded) before buildPlan runs so the plan function
// itself stays pure.
//
//   - Path:  workspace-rooted path to remove
//   - Skip:  true when a guard forbids removing this target (e.g. openspec/ in
//     the authoring repo); a skipped target is still listed in --dry-run but is
//     never executed, so the operator can see what was deliberately left alone.
type removal struct {
	Path string
	Skip bool
}

// buildPlan computes the full, ordered set of changes to move from the current
// identity cur to the next identity next, rooted at root, and to remove the
// given agent/IDE scaffolding removals. It is a pure function (no I/O) and is
// idempotent by construction: anything already matching the target is skipped,
// so a same-value plan is empty and a new-value plan always derives targets
// from the live cur. Removals are appended last so a failure during
// re-identification never leaves the tree half-wiped.
func buildPlan(root string, cur, next identityVals, removals []removal) (plan, error) {
	var p plan

	curPaths := findPaths(root, cur)
	nextPaths := findPaths(root, next)

	// 1. Regenerate identity.go (content changes on any identity difference).
	if cur != next {
		content, err := buildIdentityContent(next)
		if err != nil {
			return plan{}, err
		}
		p.Changes = append(p.Changes, change{
			Kind:    kindWrite,
			Path:    nextPaths.IdentityFile,
			Content: content,
			Desc:    fmt.Sprintf("regenerate %s", nextPaths.IdentityFile),
		})
	}

	// 2. Rename the three entrypoint directories, and 3/4. rewrite Use strings
	// and panic tags — all only when the bin prefix changes.
	if cur.BinTrio != next.BinTrio {
		p.Changes = append(p.Changes, change{
			Kind: kindRename,
			From: curPaths.CmdDir,
			To:   nextPaths.CmdDir,
			Desc: fmt.Sprintf("rename %s → %s", curPaths.CmdDir, nextPaths.CmdDir),
		})
		p.Changes = append(p.Changes, change{
			Kind: kindRename,
			From: curPaths.ConfigDir,
			To:   nextPaths.ConfigDir,
			Desc: fmt.Sprintf("rename %s → %s", curPaths.ConfigDir, nextPaths.ConfigDir),
		})
		p.Changes = append(p.Changes, change{
			Kind: kindRename,
			From: curPaths.MigrationDir,
			To:   nextPaths.MigrationDir,
			Desc: fmt.Sprintf("rename %s → %s", curPaths.MigrationDir, nextPaths.MigrationDir),
		})

		// The config root uses `<bin>-config`; the migration root keeps the
		// plural `<bin>-migrations` convention (matching the existing
		// Use "cli-migrations" while its directory is "cli-migration").
		// We store old/new literals for deterministic in-place substitution.
		p.Changes = append(p.Changes, change{
			Kind: kindRewrite, Path: nextPaths.RootUseFile,
			Old: cur.BinTrio, New: next.BinTrio,
			Desc: fmt.Sprintf("rewrite Use %q → %q in %s", cur.BinTrio, next.BinTrio, nextPaths.RootUseFile),
		})
		p.Changes = append(p.Changes, change{
			Kind: kindRewrite, Path: nextPaths.ConfigUseFile,
			Old: cur.BinTrio + "-config", New: next.BinTrio + "-config",
			Desc: fmt.Sprintf("rewrite Use %q → %q in %s", cur.BinTrio+"-config", next.BinTrio+"-config", nextPaths.ConfigUseFile),
		})
		p.Changes = append(p.Changes, change{
			Kind: kindRewrite, Path: nextPaths.DBUseFile,
			Old: cur.BinTrio + "-migrations", New: next.BinTrio + "-migrations",
			Desc: fmt.Sprintf("rewrite Use %q → %q in %s", cur.BinTrio+"-migrations", next.BinTrio+"-migrations", nextPaths.DBUseFile),
		})

		// Panic-recovery tags.
		p.Changes = append(p.Changes, change{
			Kind: kindRewrite, Path: nextPaths.CmdMain,
			Old: cur.BinTrio + "-main", New: next.BinTrio + "-main",
			Desc: fmt.Sprintf("rewrite panic tag %q → %q in %s", cur.BinTrio+"-main", next.BinTrio+"-main", nextPaths.CmdMain),
		})
		p.Changes = append(p.Changes, change{
			Kind: kindRewrite, Path: nextPaths.ConfigMain,
			Old: cur.BinTrio + "-config-main", New: next.BinTrio + "-config-main",
			Desc: fmt.Sprintf("rewrite panic tag %q → %q in %s", cur.BinTrio+"-config-main", next.BinTrio+"-config-main", nextPaths.ConfigMain),
		})
		p.Changes = append(p.Changes, change{
			Kind: kindRewrite, Path: nextPaths.MigrationMain,
			Old: cur.BinTrio + "-migration-main", New: next.BinTrio + "-migration-main",
			Desc: fmt.Sprintf("rewrite panic tag %q → %q in %s", cur.BinTrio+"-migration-main", next.BinTrio+"-migration-main", nextPaths.MigrationMain),
		})

		// The root Makefile exposes a single setup-managed line that drives the
		// binary trio. SkipIfMissing keeps a deleted or reformatted Makefile from
		// failing setup (buildPlan stays pure; apply performs the I/O).
		p.Changes = append(p.Changes, change{
			Kind: kindRewrite, Path: nextPaths.Makefile,
			Old: "BIN_TRIO ?= " + cur.BinTrio, New: "BIN_TRIO ?= " + next.BinTrio,
			Desc:          fmt.Sprintf("rewrite BIN_TRIO in %s", nextPaths.Makefile),
			SkipIfMissing: true,
			ExpectUnique:  true,
		})
	}

	// 5. Rewrite .gitignore identity lines (full-file regenerate; deterministic
	// and keeps .env / *.log verbatim).
	if cur.DataDir != next.DataDir || cur.ConfigName != next.ConfigName {
		p.Changes = append(p.Changes, change{
			Kind:    kindWrite,
			Path:    nextPaths.Gitignore,
			Content: gitignoreContent(next),
			Desc:    fmt.Sprintf("rewrite identity lines in %s", nextPaths.Gitignore),
		})
	}

	// 6. Remove agent/IDE scaffolding, after every identity rewrite, so a
	// failure above cannot half-wipe the tree. Absent paths are a no-op at
	// apply time (SkipIfMissing); a guarded target is never executed.
	for _, r := range removals {
		desc := fmt.Sprintf("remove %s", r.Path)
		if r.Skip {
			desc = fmt.Sprintf("skip removing %s (appears to be the authoring repo)", r.Path)
		}
		p.Changes = append(p.Changes, change{
			Kind:          kindRemove,
			Path:          r.Path,
			Desc:          desc,
			SkipIfMissing: true,
			Skip:          r.Skip,
		})
	}

	return p, nil
}

// apply performs the plan's changes, or (when dryRun) prints them. It mutates
// nothing on dryRun and stops at the first error.
func apply(p plan, dryRun bool) error {
	if p.empty() {
		fmt.Fprintln(os.Stderr, "setup: nothing to do (identity already matches)")
		return nil
	}

	for _, c := range p.Changes {
		if dryRun {
			fmt.Fprintln(os.Stderr, "would: "+c.Desc)
			continue
		}
		switch c.Kind {
		case kindRename:
			if err := os.Rename(c.From, c.To); err != nil {
				return fmt.Errorf("rename %s → %s: %w", c.From, c.To, err)
			}
		case kindWrite:
			if err := os.WriteFile(c.Path, c.Content, 0o644); err != nil {
				return fmt.Errorf("write %s: %w", c.Path, err)
			}
		case kindRewrite:
			if c.SkipIfMissing {
				data, err := os.ReadFile(c.Path)
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				if err != nil {
					return fmt.Errorf("rewrite %s: %w", c.Path, err)
				}
				if !strings.Contains(string(data), c.Old) {
					continue
				}
			}
			if err := rewriteLiteral(c.Path, c.Old, c.New, c.ExpectUnique); err != nil {
				return fmt.Errorf("rewrite %s: %w", c.Path, err)
			}
		case kindRemove:
			if c.Skip {
				fmt.Fprintln(os.Stderr, "guarded: "+c.Desc)
				continue
			}
			if c.SkipIfMissing {
				if _, err := os.Stat(c.Path); errors.Is(err, os.ErrNotExist) {
					continue
				}
			}
			if err := os.RemoveAll(c.Path); err != nil {
				return fmt.Errorf("remove %s: %w", c.Path, err)
			}
		default:
			return fmt.Errorf("unknown change kind %q", c.Kind)
		}
		fmt.Fprintln(os.Stderr, "applied: "+c.Desc)
	}

	if !dryRun {
		fmt.Fprintln(os.Stderr, "setup: done. run `go build ./...` to rebuild the trio.")
	}
	return nil
}

// rewriteLiteral performs a single deterministic in-place replacement of old
// with new in path, failing if old is not present (so partial/unknown states
// are not silently ignored). When expectUnique is true it also fails unless old
// occurs exactly once, guarding against a blanket rewrite of an ambiguous
// literal.
func rewriteLiteral(path, old, new string, expectUnique bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	n := strings.Count(string(data), old)
	if n == 0 {
		return fmt.Errorf("literal %q not found in %s", old, path)
	}
	if expectUnique && n != 1 {
		return fmt.Errorf("literal %q occurs %d times in %s (want exactly 1)", old, n, path)
	}
	return os.WriteFile(path, []byte(strings.Replace(string(data), old, new, 1)), 0o644)
}

// prompt reads one trimmed line from r, falling back to def on an empty line or
// EOF (so the wizard is scriptable, e.g. `< /dev/null` accepts all defaults).
func prompt(r *bufio.Scanner, label, def string) string {
	fmt.Fprintf(os.Stderr, "%s [%s]: ", label, def)
	if !r.Scan() {
		return def
	}
	line := strings.TrimSpace(r.Text())
	if line == "" {
		return def
	}
	return line
}

// promptYesNo is the boolean sibling of prompt. It treats an empty line or EOF
// as the default value, so the wizard stays scriptable (`< /dev/null` accepts
// the default). Any answer beginning with y/Y is true, n/N is false; anything
// else falls back to def.
func promptYesNo(r *bufio.Scanner, label string, def bool) bool {
	defStr := "Y/n"
	if !def {
		defStr = "y/N"
	}
	fmt.Fprintf(os.Stderr, "%s [%s]: ", label, defStr)
	if !r.Scan() {
		return def
	}
	line := strings.ToLower(strings.TrimSpace(r.Text()))
	if line == "" {
		return def
	}
	switch line[0] {
	case 'y':
		return true
	case 'n':
		return false
	default:
		return def
	}
}

// runSetup orchestrates the wizard.
func runSetup(cmd *cobra.Command, args []string) error {
	root, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("determining working directory: %w", err)
	}

	cur := currentIdentity()

	scanner := bufio.NewScanner(os.Stdin)
	appName := prompt(scanner, "App name", cur.AppName)
	envPrefix := prompt(scanner, "Env prefix", cur.EnvPrefix)
	dataDir := prompt(scanner, "Data dir", cur.DataDir)
	binTrio := prompt(scanner, "Binary trio prefix", cur.BinTrio)
	removeAgents := promptYesNo(scanner, "Remove agent scaffolding (.opencode/, openspec/, *.code-workspace, AGENTS.md/CLAUDE.md)?", true)

	next := newIdentityVals(appName, envPrefix, dataDir, binTrio)

	if err := validateIdentity(next); err != nil {
		return err
	}

	removals := resolveRemovals(root, removeAgents)

	if sameValues(cur, next) && len(removals) == 0 {
		fmt.Fprintln(os.Stderr, "setup: identity already matches the requested values; nothing to do")
		return nil
	}

	p, err := buildPlan(root, cur, next, removals)
	if err != nil {
		return err
	}
	return apply(p, setupDryRun)
}
