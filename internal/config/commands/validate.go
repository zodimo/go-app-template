package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zodimo/go-app-template/internal/config"
	"github.com/zodimo/go-app-template/internal/logging"
)

var ValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {

		logger := logging.GetLogger().NewLogger("config-validate-root")

		configFile := viper.GetString("config-config-file")
		logger.Debug("input args", "config-file", configFile)

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		dataDir := viper.GetString("config-data-dir")

		// Load and validate configuration
		_, err = config.Load(cwd, config.WithDataDirectory(dataDir), config.WithConfigFile(configFile))
		if err != nil {
			logger.Error("config.Load", "error", err)
			fmt.Printf("Configuration validation failed: %v\n", err)
			return err
		}

		if used := viper.ConfigFileUsed(); used != "" {
			fmt.Printf("Configuration is valid: %s\n", used)
		} else {
			fmt.Println("Configuration is valid (defaults, no config file used)")
		}

		return nil
	},
}
