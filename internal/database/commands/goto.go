package commands

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/zodimo/go-app-template/internal/config"
	"github.com/zodimo/go-app-template/internal/logging"
)

var gotoCmd = &cobra.Command{
	Use:   "goto <version>",
	Short: "Migrate to a specific version",
	Long:  `Migrate up or down to a specific version number. Use --dry-run to preview.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		cfg, err := config.Init()
		if err != nil {
			return err
		}

		stdErrLogger := logging.NewStdErrLogger("root")
		logging.SetDefaultLogger(stdErrLogger)

		stdErrLogger.Debug("migrate-goto")

		m, err := newMigrator(&cfg.Database)
		if err != nil {
			return err
		}
		defer closeMigrator(m)

		if dryRun {
			return previewMigrations(m, "goto", args)
		}

		if err := validateState(m); err != nil {
			return err
		}

		v, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid version: %w", err)
		}
		if v < 0 {
			return errors.New("version must be non-negative")
		}
		return wrapNoChange(m.Migrate(uint(v)), "")
	},
}
