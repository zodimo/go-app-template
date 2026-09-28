package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zodimo/go-app-template/internal/logging"
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Generate a new migration",
	Long:  `Create a new pair of sequenced migration files (.up.sql and .down.sql) using zero-padded 4-digit numbers (0001, 0002, etc.).`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		stdErrLogger := logging.NewStdErrLogger("root")
		logging.SetDefaultLogger(stdErrLogger)

		stdErrLogger.Debug("migrate-create")

		name := strings.Join(args, "_")
		name = strings.ReplaceAll(name, " ", "_")

		path := "internal/database/migrations"

		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("creating migrations directory: %w", err)
		}

		nextSeq, err := nextSequence(path)
		if err != nil {
			return fmt.Errorf("determining next sequence: %w", err)
		}

		seq := fmt.Sprintf("%04d", nextSeq)
		base := filepath.Join(path, fmt.Sprintf("%s_%s", seq, name))

		upFile := base + ".up.sql"
		downFile := base + ".down.sql"

		if err := os.WriteFile(upFile, []byte("-- "+name+"\n\n"), 0644); err != nil {
			return fmt.Errorf("creating %s: %w", upFile, err)
		}
		if err := os.WriteFile(downFile, []byte("-- revert "+name+"\n\n"), 0644); err != nil {
			return fmt.Errorf("creating %s: %w", downFile, err)
		}

		fmt.Printf("Created %s\n", upFile)
		fmt.Printf("Created %s\n", downFile)
		return nil
	},
}
