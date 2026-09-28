package commands

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/zodimo/go-app-template/internal/config"
	"github.com/zodimo/go-app-template/internal/logging"
)

var downCmd = &cobra.Command{
	Use:   "down <N>",
	Short: "Roll back migrations",
	Long:  `Roll back the most recent N migrations. Disabled in production.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Init()
		if err != nil {
			return err
		}

		stdErrLogger := logging.NewStdErrLogger("root")
		logging.SetDefaultLogger(stdErrLogger)

		stdErrLogger.Debug("migrate-down")

		m, err := newMigrator(&cfg.Database)

		if err != nil {
			return err
		}
		defer closeMigrator(m)

		if dryRun {
			return previewMigrations(m, "down", args)
		}

		if err := validateState(m); err != nil {
			return err
		}

		n, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid step count: %w", err)
		}
		if n <= 0 {
			return errors.New("step count must be positive")
		}
		return wrapNoChange(m.Steps(-n), "down")
	},
}
