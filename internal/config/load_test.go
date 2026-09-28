package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zodimo/go-app-template/internal/core"
	"github.com/zodimo/go-app-template/internal/identity"
)

func TestLoad_ResetsCfgOnReadInConfigFailure(t *testing.T) {
	Reset()
	_, err := Load("/nonexistent/path", WithConfigFile("/does/not/exist.yaml"))
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
	if cfg != nil {
		t.Fatal("cfg should be nil after failed load")
	}
}

func TestLoad_CfgNilOnErrorReuse(t *testing.T) {
	Reset()
	// First call fails
	_, _ = Load("/nonexistent", WithConfigFile("/no/such/file.yaml"))
	// Second call should NOT return stale cfg
	_, err := Load("/nonexistent", WithConfigFile("/no/such/file.yaml"))
	if err == nil {
		t.Fatal("expected error on second call with missing file")
	}
}

func TestUpdateCfgFile_PreservesUnknownFields(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	_ = os.WriteFile(cfgPath, []byte(`{
        "debug": false,
        "custom_feature_flag": true,
        "_comment": "auto-generated"
    }`), 0644)

	Reset()
	t.Setenv(identity.EnvPrefix+"_DATABASE_DATABASE", "test-db-string")
	_, _ = Load(tmpDir, WithConfigFile(cfgPath))
	_ = UpdateCfgFile(func(c *Config) {
		c.Debug = true
	})

	out, _ := os.ReadFile(cfgPath)
	content := string(out)
	if !strings.Contains(content, "custom_feature_flag") {
		t.Fatal("custom_feature_flag was dropped")
	}
	if !strings.Contains(content, "_comment") {
		t.Fatal("_comment was dropped")
	}
}

func TestGetConfigSearchPaths_NoDuplicateCwd(t *testing.T) {
	paths := GetConfigSearchPaths("/my/cwd", []string{"$HOME", "$XDG_CONFIG_HOME"})
	searchPaths := paths.GetSearchPaths()

	cwdCount := 0
	for _, p := range searchPaths {
		if p == "/my/cwd" {
			cwdCount++
		}
	}
	if cwdCount != 1 {
		t.Fatalf("cwd appeared %d times, expected exactly 1", cwdCount)
	}
}

func TestRedactedValue_MatchesSensitiveKeys(t *testing.T) {
	cases := []struct {
		key, value, expected string
	}{
		{"db_password", "s3cret", "s3****et"},
		{"api_token", "abc123", "ab****23"},
		{"my_secret_key", "xyz", "****"},
		{"safe_key", "value", "va****ue"},
		{"empty_key", "", ""},
	}
	for _, tc := range cases {
		got := core.RedactedValue(tc.key, tc.value)
		if got != tc.expected {
			t.Errorf("RedactedValue(%q, %q) = %q, want %q", tc.key, tc.value, got, tc.expected)
		}
	}
}

func TestLoad_RequireConfig(t *testing.T) {
	Reset()
	// implicit-miss + WithRequireConfig(true) -> ErrConfigFileRequired
	_, err := Load("/nonexistent", WithRequireConfig(true))
	if err == nil || !errors.Is(err, ErrConfigFileRequired) {
		t.Fatalf("expected ErrConfigFileRequired, got %v", err)
	}

	Reset()
	// implicit-miss alone -> nil error + defaults applied. The required
	// password field is satisfied via env, proving env can stand in for
	// a missing config file.
	t.Setenv(identity.EnvPrefix+"_DATABASE_DATABASE", "test-database")
	c, err := Load("/nonexistent")
	if err != nil {
		t.Fatalf("expected nil error on implicit miss, got %v", err)
	}
	if c == nil {
		t.Fatal("expected config to be loaded with defaults")
	}
	Reset()
	// explicit-path miss -> error even without require
	_, err = Load("/nonexistent", WithConfigFile("/does/not/exist.yaml"))
	if err == nil {
		t.Fatal("expected error for explicit path miss")
	}
}

func TestValidateConfig_RequiredFields(t *testing.T) {
	// database missing -> validation error naming the field and its sources
	Reset()
	_, err := Load("/nonexistent")
	if err == nil {
		t.Fatal("expected validation error for missing database")
	}
	if !strings.Contains(err.Error(), "database.database") {
		t.Fatalf("validation error should name the missing field, got: %v", err)
	}
	if !strings.Contains(err.Error(), identity.EnvPrefix+"_DATABASE_DATABASE") {
		t.Fatalf("validation error should name the env source, got: %v", err)
	}

	// password provided via env satisfies the required field even with no config file
	Reset()
	t.Setenv(identity.EnvPrefix+"_DATABASE_DATABASE", "test-database")
	c, err := Load("/nonexistent")
	if err != nil {
		t.Fatalf("expected nil error with env-provided password, got %v", err)
	}
	if c == nil || c.Database.Database.UnwrapOr("") != "test-database" {
		t.Fatalf("expected password from env, got %+v", c)
	}

	// defaults-only loads skip required-field validation (version, config init)
	Reset()
	t.Setenv(identity.EnvPrefix+"_DATABASE_DATABASE", "")
	d, err := Load("/nonexistent", WithDefaultsOnly(true))
	if err != nil {
		t.Fatalf("expected nil error on defaults-only load, got %v", err)
	}
	if d == nil {
		t.Fatal("expected defaults-only config")
	}
}
