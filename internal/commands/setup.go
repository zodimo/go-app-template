package commands

import (
	"bufio"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zodimo/go-app-template/internal/identity"
)

// setupCmd re-identifies a cloned project. It prompts for an application name,
// environment prefix, data directory, and binary-trio prefix, then rewrites the
// identity-derived artifacts of the tree: internal/identity/identity.go, the three
// cmd/ entrypoint directories, the cobra Use strings, the panic-recovery tags,
// and the identity lines in .gitignore.
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
command entrypoint directories, and rewrites cobra Use strings, panic tags, and
.gitignore entries to match. All inputs are validated before any mutation, and
the operation is idempotent.`,
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

// changeKind enumerates the three mutation shapes.
type changeKind string

const (
	kindRename  changeKind = "rename"
	kindWrite   changeKind = "write"
	kindRewrite changeKind = "rewrite"
)

// change is a single planned mutation.
//
//   - rename:  From -> To (directory rename)
//   - write:   Path <- Content (full-file regeneration: identity.go, .gitignore)
//   - rewrite: Path: replace the literal Old with New (Use lines, panic tags)
type change struct {
	Kind          changeKind
	From          string // rename source
	To            string // rename destination
	Path          string // write/rewrite target
	Content       []byte // write content
	Old           string // rewrite literal to find
	New           string // rewrite literal to substitute
	Desc          string // human-readable description for --dry-run
	SkipIfMissing bool   // rewrite: skip (no error) when the file or literal is absent
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

// buildPlan computes the full, ordered set of changes to move from the current
// identity cur to the next identity next, rooted at root. It is a pure function
// (no I/O) and is idempotent by construction: anything already matching the
// target is skipped, so a same-value plan is empty and a new-value plan always
// derives targets from the live cur.
func buildPlan(root string, cur, next identityVals) (plan, error) {
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
			if err := rewriteLiteral(c.Path, c.Old, c.New); err != nil {
				return fmt.Errorf("rewrite %s: %w", c.Path, err)
			}
		default:
			return fmt.Errorf("unknown change kind %q", c.Kind)
		}
	}

	if !dryRun {
		fmt.Fprintln(os.Stderr, "setup: done. run `go build ./...` to rebuild the trio.")
	}
	return nil
}

// rewriteLiteral performs a single deterministic in-place replacement of old
// with new in path, failing if old is not present (so partial/unknown states
// are not silently ignored).
func rewriteLiteral(path, old, new string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !strings.Contains(string(data), old) {
		return fmt.Errorf("literal %q not found in %s", old, path)
	}
	replaced := strings.Replace(string(data), old, new, 1)
	return os.WriteFile(path, []byte(replaced), 0o644)
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

	next := newIdentityVals(appName, envPrefix, dataDir, binTrio)

	if err := validateIdentity(next); err != nil {
		return err
	}

	if sameValues(cur, next) {
		fmt.Fprintln(os.Stderr, "setup: identity already matches the requested values; nothing to do")
		return nil
	}

	p, err := buildPlan(root, cur, next)
	if err != nil {
		return err
	}
	return apply(p, setupDryRun)
}
