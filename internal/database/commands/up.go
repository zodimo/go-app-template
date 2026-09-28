package commands

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/zodimo/go-app-template/internal/config"
	"github.com/zodimo/go-app-template/internal/logging"
)

var upCmd = &cobra.Command{
	Use:   "up [N]",
	Short: "Apply pending migrations",
	Long:  `Apply all pending migrations, or the next N if specified. Use --dry-run to preview.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Init()
		if err != nil {
			return err
		}

		stdErrLogger := logging.NewStdErrLogger("root")
		logging.SetDefaultLogger(stdErrLogger)

		stdErrLogger.Debug("migrate-up")

		m, err := newMigrator(&cfg.Database)
		if err != nil {
			return err
		}
		defer closeMigrator(m)

		if dryRun {
			return previewMigrations(m, "up", args)
		}

		if err := validateState(m); err != nil {
			return err
		}

		if len(args) > 0 {
			n, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid step count: %w", err)
			}
			return wrapNoChange(m.Steps(n), "up")
		}
		return wrapNoChange(m.Up(), "up")
	},
}
