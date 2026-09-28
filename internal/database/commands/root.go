package commands

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zodimo/go-app-template/internal/config"
)

var RootConfigCmd = &cobra.Command{
	Use:   "cli-migrations",
	Short: "Database management",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Only the bare `migrate` container and the root-level build-info
		// `version` need no config. `migrate version` is a distinct command
		// deeper in the tree and must load config so explicit --config/--data-dir
		// flags are honored for it (see internal/database/commands/version.go).
		// A direct child of the root has cmd.Parent() == cmd.Root().
		if cmd.Parent() == cmd.Root() && (cmd.Name() == "migrate" || cmd.Name() == "version") {
			return nil
		}
		configFile := viper.GetString("migration-config-file")
		dataDir := viper.GetString("migration-data-dir")
		_, err := config.Init(
			config.WithConfigFile(configFile),
			config.WithDataDirectory(dataDir),
		)
		return err
	},
}

func init() {

	RootConfigCmd.SilenceUsage = true
	RootConfigCmd.SilenceErrors = true
	RootConfigCmd.PersistentFlags().String("config", "", "config file")
	RootConfigCmd.PersistentFlags().String("data-dir", "", "Data directory")

	_ = viper.BindPFlag("migration-config-file", RootConfigCmd.PersistentFlags().Lookup("config"))
	_ = viper.BindPFlag("migration-data-dir", RootConfigCmd.PersistentFlags().Lookup("data-dir"))

	RootConfigCmd.AddCommand(migrateCmd)

}
