package commands

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	resetConfigViper(t)

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
	if got := viper.GetString("config-config-file"); got != "/tmp/cfg.json" {
		t.Fatalf("config-config-file = %q, want /tmp/cfg.json", got)
	}

	if err := RootConfigCmd.PersistentFlags().Set("data-dir", "/tmp/data"); err != nil {
		t.Fatalf("set --data-dir: %v", err)
	}
	if got := viper.GetString("config-data-dir"); got != "/tmp/data" {
		t.Fatalf("config-data-dir = %q, want /tmp/data", got)
	}

	// The migration root's keys must not be bound by the config root.
	if got := viper.GetString("migration-config-file"); got != "" {
		t.Fatalf("migration-config-file unexpectedly bound by config root: %q", got)
	}
	if got := viper.GetString("migration-data-dir"); got != "" {
		t.Fatalf("migration-data-dir unexpectedly bound by config root: %q", got)
	}
	// No root binds data.directory from a flag anymore.
	if got := viper.GetString("data.directory"); got != "" {
		t.Fatalf("data.directory should not be flag-bound, got %q", got)
	}
}

func TestRootConfigCmd_HonorsExplicitConfigFile(t *testing.T) {
	resetConfigViper(t)

	tmpDir := t.TempDir()
	viper.AddConfigPath(tmpDir)

	discovered := filepath.Join(tmpDir, identity.ConfigName+".json")
	discoveredContent := fmt.Sprintf(`{"data":{"directory":%q},"database":{"database":%q}}`, tmpDir, filepath.Join(tmpDir, "discovered.db"))
	if err := os.WriteFile(discovered, []byte(discoveredContent), 0o644); err != nil {
		t.Fatalf("write discovered file: %v", err)
	}

	explicit := filepath.Join(tmpDir, "explicit.json")
	explicitContent := fmt.Sprintf(`{"data":{"directory":%q},"database":{"database":%q}}`, tmpDir, filepath.Join(tmpDir, "explicit.sqlite"))
	if err := os.WriteFile(explicit, []byte(explicitContent), 0o644); err != nil {
		t.Fatalf("write explicit file: %v", err)
	}

	explicitAbs, err := filepath.Abs(explicit)
	if err != nil {
		t.Fatalf("abs explicit: %v", err)
	}

	RootConfigCmd.SetArgs([]string{"--config", explicit, "show", "--no-validate"})
	if err := RootConfigCmd.Execute(); err != nil {
		t.Fatalf("show with explicit config: %v", err)
	}
	if got := filepath.Clean(viper.ConfigFileUsed()); got != filepath.Clean(explicitAbs) {
		t.Fatalf("ConfigFileUsed = %q, want %q", got, explicitAbs)
	}
}

func TestRootConfigCmd_MissingExplicitConfigFails(t *testing.T) {
	resetConfigViper(t)

	missing := filepath.Join(t.TempDir(), "missing.json")

	RootConfigCmd.SetArgs([]string{"--config", missing, "validate"})
	err := RootConfigCmd.Execute()
	if !errors.Is(err, config.ErrConfigFileRequired) {
		t.Fatalf("expected ErrConfigFileRequired for missing explicit --config, got %v", err)
	}
}

func TestRootConfigCmd_SilenceErrors_EmitsOnce(t *testing.T) {
	if !RootConfigCmd.SilenceErrors {
		t.Fatal("SilenceErrors should be true")
	}
	if !RootConfigCmd.SilenceUsage {
		t.Fatal("SilenceUsage should be true")
	}

	var buf bytes.Buffer
	RootConfigCmd.SetOut(&buf)
	RootConfigCmd.SetErr(&buf)

	RootConfigCmd.SetArgs([]string{"bogus"})
	err := RootConfigCmd.Execute()
	if err == nil {
		t.Fatal("expected non-nil error for unknown command 'bogus'")
	}
	if bytes.Contains(buf.Bytes(), []byte("Error:")) {
		t.Fatalf("SilenceErrors should suppress cobra's error emission, got %q", buf.String())
	}

	t.Cleanup(func() {
		RootConfigCmd.SetArgs(nil)
		RootConfigCmd.SetOut(nil)
		RootConfigCmd.SetErr(nil)
	})
}

// resetConfigViper restores a clean viper + config state for tests that drive the
// real config command tree. viper.Reset() drops the flag bindings made in package
// init(), so the config flags are re-bound explicitly (re-running init() is not
// possible and would double-register commands).
func resetConfigViper(t *testing.T) {
	t.Helper()
	viper.Reset()
	_ = viper.BindPFlag("config-config-file", RootConfigCmd.PersistentFlags().Lookup("config"))
	_ = viper.BindPFlag("config-data-dir", RootConfigCmd.PersistentFlags().Lookup("data-dir"))
	noValidateFlag := ShowCmd.Flags().Lookup("no-validate")
	_ = viper.BindPFlag("no-validate", noValidateFlag)
	_ = noValidateFlag.Value.Set("false")
	config.Reset()
	t.Cleanup(func() {
		RootConfigCmd.SetArgs(nil)
		config.Reset()
		viper.Reset()
	})
}
