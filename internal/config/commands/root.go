package commands

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zodimo/go-app-template/internal/config"
)

var RootConfigCmd = &cobra.Command{
	Use:   "cli-config",
	Short: "Configuration management",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Only the root-level build-info `version` and the `init` scaffolder
		// need no config; every other command loads it. A direct child of the
		// root has cmd.Parent() == cmd.Root().
		if cmd.Parent() == cmd.Root() && (cmd.Name() == "version" || cmd.Name() == "init") {
			return nil
		}
		configFile := viper.GetString("config-config-file")
		// show --no-validate must bypass field validation so an invalid config
		// can still be inspected. For other commands this key is unset -> false.
		disableValidation := viper.GetBool("no-validate")
		_, err := config.Init(
			config.WithConfigFile(configFile),
			config.WithDataDirectory(viper.GetString("config-data-dir")),
			config.WithDisableValidation(disableValidation),
		)
		return err
	},
}

func init() {

	RootConfigCmd.SilenceUsage = true
	RootConfigCmd.SilenceErrors = true
	RootConfigCmd.PersistentFlags().String("config", "", "config file")
	RootConfigCmd.PersistentFlags().String("data-dir", "", "Data directory")

	_ = viper.BindPFlag("config-config-file", RootConfigCmd.PersistentFlags().Lookup("config"))

	RootConfigCmd.AddCommand(InitCmd)
	RootConfigCmd.AddCommand(ValidateCmd)
	RootConfigCmd.AddCommand(ShowCmd)

	_ = viper.BindPFlag("config-data-dir", RootConfigCmd.PersistentFlags().Lookup("data-dir"))

}
