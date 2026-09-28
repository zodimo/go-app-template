package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zodimo/go-app-template/internal/config"
	"github.com/zodimo/go-app-template/internal/identity"
	"github.com/zodimo/go-app-template/internal/logging"
	"github.com/zodimo/go-maybe"
)

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new configuration file",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		dataDir := viper.GetString("config-data-dir")
		_, err := config.Init(
			config.WithDefaultsOnly(true),
			config.WithDataDirectory(dataDir),
		)
		return err
	},
	RunE: func(cmd *cobra.Command, args []string) error {

		logger := logging.GetLogger().NewLogger("config-init-root")

		configFile := viper.GetString("config-config-file")
		logger.Debug("input args", "config-file", configFile)
		if configFile == "" {
			configFile = fmt.Sprintf("%s.json", config.ConfigName())
		}

		// Check if file already exists
		if _, err := os.Stat(configFile); err == nil {
			fmt.Printf("Configuration file already exists: %s\n", configFile)
			return fmt.Errorf("file already exists")
		}

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}

		dataDir := viper.GetString("config-data-dir")

		// Create example config
		cfg, err := config.Load(cwd, config.WithDataDirectory(dataDir), config.WithDisableValidation(true), config.WithDefaultsOnly(true))
		if err != nil {
			logger.Error("config.Load", "error", err)
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Create directory if needed
		dir := filepath.Dir(configFile)
		if dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		}

		if cfg.Database.Database.IsNone() {
			defaultDbName := fmt.Sprintf("%s-data.db", identity.AppName)
			cfg.Database.Database = maybe.Some(filepath.Join(cfg.Data.Directory.UnwrapUnsafe(), "db", defaultDbName))
		}

		// Save configuration
		if err := config.SaveConfig(cfg, configFile); err != nil {
			logger.Error("config.SaveConfig", "error", err)
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		fmt.Printf("Configuration file created: %s\n", configFile)
		fmt.Println("Please edit the configuration file with your specific settings")

		return nil
	},
}
