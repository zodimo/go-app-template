package commands

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/zodimo/go-app-template/internal/config"
	"github.com/zodimo/go-app-template/internal/logging"
)

var forceCmd = &cobra.Command{
	Use:   "force <version>",
	Short: "Force a specific version (recovery only)",
	Long:  `Mark a specific version as applied without running migrations. Use with caution.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Init()
		if err != nil {
			return err
		}

		stdErrLogger := logging.NewStdErrLogger("root")
		logging.SetDefaultLogger(stdErrLogger)

		stdErrLogger.Debug("migrate-force")

		m, err := newMigrator(&cfg.Database)
		if err != nil {
			return err
		}
		defer closeMigrator(m)

		v, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid version: %w", err)
		}
		return m.Force(v)
	},
}
