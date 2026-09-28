package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/zodimo/go-app-template/internal/config"
	"github.com/zodimo/go-app-template/internal/identity"
)

func TestRootConfigCmd_SilencesErrorsAndUsage(t *testing.T) {
	if !RootConfigCmd.SilenceErrors {
		t.Error("SilenceErrors should be true (single error emission)")
	}
	if !RootConfigCmd.SilenceUsage {
		t.Error("SilenceUsage should be true")
	}
}

func TestRootConfigCmd_BindsDistinctKeys(t *testing.T) {
	// viper.Reset() in the other tests drops the flag bindings made in
	// package init(), so re-establish them before asserting. This makes the
	// test order-independent.
	resetMigrationViper(t)

	// Save the flag values (the viper keys read here are bound directly to these
	// flags) so this test does not leak paths into later tests.
	configFlag := RootConfigCmd.PersistentFlags().Lookup("config")
	dataFlag := RootConfigCmd.PersistentFlags().Lookup("data-dir")
	prevConfig := configFlag.Value.String()
	prevDataDir := dataFlag.Value.String()
	t.Cleanup(func() {
		_ = configFlag.Value.Set(prevConfig)
		_ = dataFlag.Value.Set(prevDataDir)
	})

	if err := RootConfigCmd.PersistentFlags().Set("config", "/tmp/cfg.json"); err != nil {
		t.Fatalf("set --config: %v", err)
	}
	if got := viper.GetString("migration-config-file"); got != "/tmp/cfg.json" {
		t.Fatalf("migration-config-file = %q, want /tmp/cfg.json", got)
	}

	if err := RootConfigCmd.PersistentFlags().Set("data-dir", "/tmp/data"); err != nil {
		t.Fatalf("set --data-dir: %v", err)
	}
	if got := viper.GetString("migration-data-dir"); got != "/tmp/data" {
		t.Fatalf("migration-data-dir = %q, want /tmp/data", got)
	}

	// The config root's keys must not be bound by the migration root.
	if got := viper.GetString("config-config-file"); got != "" {
		t.Fatalf("config-config-file unexpectedly bound by migration root: %q", got)
	}
	if got := viper.GetString("config-data-dir"); got != "" {
		t.Fatalf("config-data-dir unexpectedly bound by migration root: %q", got)
	}
	// No root binds data.directory from a flag anymore.
	if got := viper.GetString("data.directory"); got != "" {
		t.Fatalf("data.directory should not be flag-bound, got %q", got)
	}
}

func TestMigrateCmd_RejectsUnknownChild(t *testing.T) {
	if migrateCmd.Args == nil {
		t.Fatal("migrateCmd.Args should be set so unknown children are rejected")
	}
}

func TestMigrateContainer_Behavior(t *testing.T) {
	// Exercise the real migrateCmd through its parent root: Execute() on a child
	// delegates to the root, which owns the args.
	RootConfigCmd.SetArgs([]string{"migrate", "bogus"})
	if err := RootConfigCmd.Execute(); err == nil {
		t.Fatal("expected error for unknown child 'bogus'")
	}

	RootConfigCmd.SetArgs([]string{"migrate"})
	if err := RootConfigCmd.Execute(); err != nil {
		t.Fatalf("bare migrate container should not error, got %v", err)
	}

	t.Cleanup(func() { RootConfigCmd.SetArgs(nil) })
}

func TestMigrateVersion_HonorsConfigFile(t *testing.T) {
	resetMigrationViper(t)

	tmpDir := t.TempDir()
	viper.AddConfigPath(tmpDir)

	discovered := filepath.Join(tmpDir, identity.ConfigName+".json")
	content := fmt.Sprintf(`{"data":{"directory":%q},"database":{"database":%q}}`, tmpDir, filepath.Join(tmpDir, "discovered.db"))
	if err := os.WriteFile(discovered, []byte(content), 0o644); err != nil {
		t.Fatalf("write discovered file: %v", err)
	}

	missing := filepath.Join(tmpDir, "missing.json")

	RootConfigCmd.SetArgs([]string{"--config", missing, "migrate", "version"})
	err := RootConfigCmd.Execute()
	if !errors.Is(err, config.ErrConfigFileRequired) {
		t.Fatalf("expected ErrConfigFileRequired for explicit missing --config, got %v", err)
	}
}

func TestMigrateVersion_ExplicitConfigBeatsDiscovered(t *testing.T) {
	resetMigrationViper(t)

	tmpDir := t.TempDir()
	viper.AddConfigPath(tmpDir)

	discovered := filepath.Join(tmpDir, identity.ConfigName+".json")
	if err := os.WriteFile(discovered, []byte(`{"database": {}}`), 0o644); err != nil {
		t.Fatalf("write discovered file: %v", err)
	}

	explicit := filepath.Join(tmpDir, "explicit.json")
	content := fmt.Sprintf(`{"data":{"directory":%q},"database":{"database":%q}}`, tmpDir, filepath.Join(tmpDir, "explicit.sqlite"))
	if err := os.WriteFile(explicit, []byte(content), 0o644); err != nil {
		t.Fatalf("write explicit file: %v", err)
	}

	RootConfigCmd.SetArgs([]string{"--config", explicit, "migrate", "version"})
	err := RootConfigCmd.Execute()
	if errors.Is(err, config.ErrConfigFileRequired) {
		t.Fatalf("explicit --config must be honored; got ErrConfigFileRequired: %v", err)
	}
	if err != nil && strings.Contains(err.Error(), "database.database is required") {
		t.Fatalf("explicit config must beat the invalid discovered file; got %v", err)
	}
}

// resetMigrationViper restores a clean viper + config state for tests that drive
// the real migration command tree. viper.Reset() drops the flag bindings made in
// package init(), so the two migration flags are re-bound explicitly (re-running
// init() is not possible and would double-register commands).
func resetMigrationViper(t *testing.T) {
	t.Helper()
	viper.Reset()
	_ = viper.BindPFlag("migration-config-file", RootConfigCmd.PersistentFlags().Lookup("config"))
	_ = viper.BindPFlag("migration-data-dir", RootConfigCmd.PersistentFlags().Lookup("data-dir"))
	config.Reset()
	t.Cleanup(func() {
		RootConfigCmd.SetArgs(nil)
		config.Reset()
		viper.Reset()
	})
}
