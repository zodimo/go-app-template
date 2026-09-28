package commands

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/spf13/cobra"
	"github.com/zodimo/go-app-template/internal/config"
	"github.com/zodimo/go-app-template/internal/logging"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print current migration version",
	Long:  `Display the current migration version and dirty flag.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Init()
		if err != nil {
			return err
		}

		stdErrLogger := logging.NewStdErrLogger("root")
		logging.SetDefaultLogger(stdErrLogger)

		stdErrLogger.Debug("migrate-version")

		m, err := newMigrator(&cfg.Database)
		if err != nil {
			return err
		}
		defer closeMigrator(m)

		v, dirty, err := m.Version()
		if err != nil {
			if errors.Is(err, migrate.ErrNilVersion) {
				fmt.Println("no migrations applied yet")
				return nil
			}
			return err
		}
		fmt.Printf("version=%d dirty=%t\n", v, dirty)
		return nil
	},
}
