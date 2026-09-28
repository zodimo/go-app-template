package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zodimo/go-app-template/internal/config"
	"github.com/zodimo/go-app-template/internal/core"
	"github.com/zodimo/go-app-template/internal/logging"
)

var ShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show configuration (sensitive values hidden)",
	RunE: func(cmd *cobra.Command, args []string) error {

		logger := logging.GetLogger().NewLogger("config-show-root")

		configFile := viper.GetString("config-config-file")

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		dataDir := viper.GetString("config-data-dir")

		noValidate := viper.GetBool("no-validate")
		showEnv := viper.GetBool("show-env")
		defaultsOnly := viper.GetBool("defaults-only")
		logger.Debug("input args", "config-file", configFile, "no-validate", noValidate, "show-env", showEnv, "defaults-only", defaultsOnly)

		var cfg *config.Config

		loadOptions := []config.LoadOption{
			config.WithDataDirectory(dataDir),
			config.WithConfigFile(configFile),
		}

		if defaultsOnly {
			loadOptions = append(loadOptions, config.WithDefaultsOnly(defaultsOnly))
		}

		if noValidate {
			loadOptions = append(loadOptions, config.WithDisableValidation(noValidate))
		}

		// Load configuration
		cfg, err = config.Load(cwd, loadOptions...)
		if err != nil {
			logger.Error("config.Load", "error", err)
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		replacer := strings.NewReplacer(".", "_", "-", "_")
		viper.SetEnvKeyReplacer(replacer)

		if showEnv {
			keys := viper.AllKeys()
			sort.Strings(keys)
			for _, key := range keys {
				// Convert key to env format
				envKey := strings.ToUpper(replacer.Replace(key))
				if viper.GetEnvPrefix() != "" {
					envKey = strings.ToUpper(viper.GetEnvPrefix()) + "_" + envKey
				}

				val := viper.Get(key)
				if stringVal, ok := val.(string); ok {
					fmt.Printf("%s=%v\n", envKey, core.RedactedValue(key, stringVal))
				} else {
					fmt.Printf("%s=%v\n", envKey, val)
				}
			}
		} else {
			// Display configuration (hiding sensitive values)
			err := cfg.Print("Aplication Config", cmd.OutOrStdout(), core.PrintConfigWithRedactedValues(true), core.PrintConfigWithAssertValid(!noValidate))
			if err != nil {
				logger.Error("cfg.Print", "error", err)
				return fmt.Errorf("failed to print configuration: %w", err)
			}
		}

		return nil
	},
}

func init() {
	//disable validation
	ShowCmd.Flags().BoolP("no-validate", "n", false, "Disable validation")
	ShowCmd.Flags().BoolP("show-env", "e", false, "Show environment variables")
	ShowCmd.Flags().BoolP("defaults-only", "d", false, "Load defaults only")
	_ = viper.BindPFlag("no-validate", ShowCmd.Flags().Lookup("no-validate"))
	_ = viper.BindPFlag("show-env", ShowCmd.Flags().Lookup("show-env"))
	_ = viper.BindPFlag("defaults-only", ShowCmd.Flags().Lookup("defaults-only"))

}
