package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/zodimo/go-app-template/internal/core"
	"github.com/zodimo/go-app-template/internal/database"
	"github.com/zodimo/go-app-template/internal/identity"
	"github.com/zodimo/go-app-template/internal/logging"
	"github.com/zodimo/go-maybe"
)

// Application constants
const (
	// cli flags
	cwdFlag          = "cwd"
	debugFlag        = "debug"
	defaultsOnlyFlag = "defaultConfigOnly"
)

var (
	defaultSearchPaths = []string{
		"$HOME",
		"$XDG_CONFIG_HOME",
		"$HOME/.config",
		filepath.Join(fmt.Sprintf("/opt/%s", identity.AppName)),
	}

	// Global configuration instance
	cfg *Config
)

func Get() *Config {
	if cfg == nil {
		panic("Config is not initialized")
	}
	return cfg
}

// Reset clears the loaded configuration so that the next Init() or Load()
// will re-read from sources. Primarily useful for tests.
func Reset() {
	cfg = nil
}

func ConfigName() string {
	return identity.ConfigName
}

var _ core.CompliantConfig = (*Config)(nil)

// Config holds the application configuration
type Config struct {
	configContext core.ConfigContext `mapstructure:"-" json:"-"`
	WorkingDir    string             `mapstructure:"working_dir" json:"working_dir"`
	Debug         bool               `mapstructure:"debug" json:"debug"`
	Data          Data               `mapstructure:"data" json:"data"`
	Log           logging.LogConfig  `mapstructure:"log" json:"log"`
	Database      database.Config    `mapstructure:"database" json:"database"`
}

func validateConfig(c *Config, configContext core.ConfigContext) error {
	// Required fields must NOT have a non-zero default in setDefaultsFor()
	// (viper fills defaults before Unmarshal, masking presence).
	c.WithConfigContext(configContext)
	return errors.Join(c.Validate()...)
}

// SaveConfig saves configuration to a file
func SaveConfig(config *Config, configPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %q: %w", dir, err)
		}
	}

	// Marshal config to JSON
	b, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config to JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, b, 0644); err != nil {
		return fmt.Errorf("failed to write config to %q: %w", configPath, err)
	}

	return nil
}

// setDefaultsFor configures default values for configuration options.
func setDefaultsFor(workingDir string, debug bool) {

	viper.SetDefault("data.directory", maybe.Some(filepath.Join(workingDir, identity.DataDir)))

	// None default (NOT a value): registers the key so AutomaticEnv values flow
	// through viper.Unmarshal. Without any default/config presence the key is
	// invisible to Unmarshal (absent from AllKeys) and an env-only value would
	// never reach the struct. None keeps the field "missing" when no env or
	// config file provides a real value, so validateConfig() can still flag it
	// as required.
	viper.SetDefault("database.database", maybe.None[string]())
	//LOGGING
	viper.SetDefault("log.max_size", maybe.Some(50))
	viper.SetDefault("log.max_backups", maybe.Some(3))
	viper.SetDefault("log.max_age", maybe.Some(30))
	viper.SetDefault("log.compress", maybe.Some(true))

	viper.SetDefault("log.debug", maybe.Some(debug))
	viper.SetDefault("debug", debug)

	// Keys read by Init()/Load() that are only bound to flags on the main
	// root. Defaulting them keeps every read key resolvable on the config
	// and migration binaries, which never bind these flags.
	viper.SetDefault("cwd", "")
	viper.SetDefault("defaultConfigOnly", false)

}

// applyDefaultValuesTo sets default values for configuration fields that need processing.
func applyDefaultValuesTo(c *Config) {
	// Application level defaults for sub-package configs
	// sub-package level defaults
	// we need to account for both

	if c.Database.Params.IsNone() {
		c.Database.Params = maybe.Some(database.DefaultParams())
	}
	if c.Database.MigrationParams.IsNone() {
		c.Database.MigrationParams = maybe.Some(database.DefaultMigrationQueries())
	}

	if c.Log.Directory.IsNone() {
		c.Log.Directory = maybe.Some(filepath.Join(c.Data.Directory.UnwrapUnsafe(), "logs"))
	}
	if c.Log.Filename.IsNone() {
		c.Log.Filename = maybe.Some(fmt.Sprintf("%s.log", identity.AppName))
	}

	if c.Debug {
		c.Log.Debug = maybe.Some(true)
	}
}
