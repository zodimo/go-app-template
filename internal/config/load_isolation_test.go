package config

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/zodimo/go-app-template/internal/core"
	"github.com/zodimo/go-app-template/internal/identity"
	"github.com/zodimo/go-maybe"
)

// TestLoad_ExplicitMissingConfigFileRequired is the 3.4 regression test: an
// explicit --config path that does not exist must fail with ErrConfigFileRequired
// and never silently fall back to a discovered file or defaults.
func TestLoad_ExplicitMissingConfigFileRequired(t *testing.T) {
	Reset()
	missing := filepath.Join(t.TempDir(), "missing.json")
	_, err := Load("/nonexistent", WithConfigFile(missing))
	if err == nil {
		t.Fatal("expected error for missing explicit config path")
	}
	if !errors.Is(err, ErrConfigFileRequired) {
		t.Fatalf("expected ErrConfigFileRequired, got %v", err)
	}
}

// TestLoad_DataDirectoryFlagWins is the 3.3 regression test: a --data-dir flag
// must be observable in the loaded config as Data.Directory.
func TestLoad_DataDirectoryFlagWins(t *testing.T) {
	Reset()
	c, err := Load("/nonexistent", WithDataDirectory("/custom/data"), WithDisableValidation(true))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := c.Data.Directory.UnwrapUnsafe(); got != "/custom/data" {
		t.Fatalf("Data.Directory = %q, want /custom/data", got)
	}
}

// TestLoad_ExplicitConfigWinsOverDiscovered asserts that an explicit --config
// path selects that file even when a discovered default-named file exists.
func TestLoad_ExplicitConfigWinsOverDiscovered(t *testing.T) {
	Reset()
	dir := t.TempDir()

	discovered := filepath.Join(dir, identity.ConfigName+".json")
	if err := os.WriteFile(discovered, []byte(`{"debug": false, "database": {"database": "discovered-db"}}`), 0644); err != nil {
		t.Fatalf("write discovered file: %v", err)
	}

	explicit := filepath.Join(dir, "explicit.json")
	if err := os.WriteFile(explicit, []byte(`{"debug": true, "database": {"database": "explicit-db"}}`), 0644); err != nil {
		t.Fatalf("write explicit file: %v", err)
	}

	cfg, err := Load(dir, WithConfigFile(explicit), WithDisableValidation(true))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.Debug {
		t.Fatal("expected debug=true from the explicit file")
	}
	if got := cfg.Database.Database.UnwrapUnsafe(); got != "explicit-db" {
		t.Fatalf("expected explicit-db from the explicit file, got %q", got)
	}
}

// TestConfigPrint_NoConfigFileUsed guards the source line: when no config file
// was used, Print must not emit a trailing blank path.
func TestConfigPrint_NoConfigFileUsed(t *testing.T) {
	viper.Reset() // ensure ConfigFileUsed() == ""
	data := Data{Directory: maybe.Some("/tmp/data")}
	data.WithConfigContext(NewAppConfigContext("data", identity.EnvPrefix+"_DATA"))
	cfg := &Config{
		WorkingDir: "/tmp",
		Data:       data,
	}
	var buf bytes.Buffer
	if err := cfg.Print("App", &buf, core.PrintConfigWithAssertValid(false)); err != nil {
		t.Fatalf("Print: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "Configuration from: \n") {
		t.Fatalf("output contains a blank source line: %q", out)
	}
	if !strings.Contains(out, "Configuration from: (no config file used)") {
		t.Fatalf("output missing explicit no-config marker: %q", out)
	}
}
