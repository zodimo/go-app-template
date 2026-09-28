package commands

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/spf13/cobra"
	"github.com/zodimo/go-app-template/internal/config"
	"github.com/zodimo/go-app-template/internal/logging"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Check migration state",
	Long:  `Validate the migration state without applying any migrations. Checks for dirty state and version-ahead conditions.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Init()
		if err != nil {
			return err
		}

		stdErrLogger := logging.NewStdErrLogger("root")
		logging.SetDefaultLogger(stdErrLogger)

		stdErrLogger.Debug("migrate-validate")

		m, err := newMigrator(&cfg.Database)
		if err != nil {
			return err
		}
		defer closeMigrator(m)

		if err := validateState(m); err != nil {
			return err
		}

		v, dirty, err := m.Version()
		if err != nil {
			if errors.Is(err, migrate.ErrNilVersion) {
				fmt.Println("valid: no migrations applied yet")
				return nil
			}
			return fmt.Errorf("reading version: %w", err)
		}

		fmt.Printf("valid: version=%d dirty=%t\n", v, dirty)
		return nil
	},
}
