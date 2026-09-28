// Command cli-migration is a thin wrapper around golang-migrate that drives the
// SQL migration files under internal/database/migrations. Use this in CI/CD as the
// single source of truth for schema changes — running it is idempotent
// because golang-migrate tracks applied versions in a `schema_migrations`
// table inside the target database.
//
// Usage:
//
//	go run ./cmd/cli-migration migrate up               # apply all pending migrations
//	go run ./cmd/cli-migration migrate up <N>           # apply the next N migrations
//	go run ./cmd/cli-migration migrate up --dry-run     # preview pending migrations
//	go run ./cmd/cli-migration migrate down <N>         # roll back the most recent N
//	go run ./cmd/cli-migration migrate goto <V>         # migrate to version V
//	go run ./cmd/cli-migration migrate version          # print current version + dirty flag
//	go run ./cmd/cli-migration migrate force <V>        # force the recorded version (recovery only)
//	go run ./cmd/cli-migration migrate create <name>    # generate new migration files
//	go run ./cmd/cli-migration migrate validate         # check state without applying
package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/spf13/cobra"
	"github.com/zodimo/go-app-template/internal/database"
)

var (
	dryRun bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Database migration tool",
	Long:  `A CLI wrapper around golang-migrate for managing database schema changes.`,
	// Container command: bare `migrate` prints help; an unknown child is
	// rejected by the Args validator before RunE runs.
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	Args: cobra.NoArgs,
}

func init() {
	migrateCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Preview migrations without executing")

	migrateCmd.AddCommand(upCmd)
	migrateCmd.AddCommand(downCmd)
	migrateCmd.AddCommand(gotoCmd)
	migrateCmd.AddCommand(versionCmd)
	migrateCmd.AddCommand(forceCmd)
	migrateCmd.AddCommand(createCmd)
	migrateCmd.AddCommand(validateCmd)
}

func newMigrator(config *database.Config) (*migrate.Migrate, error) {
	if !config.IsValid() {
		return nil, fmt.Errorf("Config has errors")
	}

	dbURL := config.DbUrlForMigration()
	path := "internal/database/migrations"

	dir := filepath.Dir(config.Database.UnwrapUnsafe())
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("NewStore: failed to create directory: %w", err)
		}
	}

	m, err := migrate.New("file://"+path, dbURL)
	if err != nil {
		return nil, fmt.Errorf("opening migration source: %w", err)
	}

	m.LockTimeout = 30 * time.Second

	return m, nil
}

func closeMigrator(m *migrate.Migrate) {
	if m == nil {
		return
	}
	if srcErr, dbErr := m.Close(); srcErr != nil || dbErr != nil {
		fmt.Fprintf(os.Stderr, "migrate: close: source=%v db=%v\n", srcErr, dbErr)
	}
}

func validateState(m *migrate.Migrate) error {
	v, dirty, err := m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			return nil
		}
		return fmt.Errorf("checking migration state: %w", err)
	}
	if dirty {
		return fmt.Errorf("database is dirty at version %d — manual intervention required (use 'force %d' only if you know what you're doing)", v, v)
	}
	return nil
}

func wrapNoChange(err error, direction string) error {
	if direction == "up" {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("no pending migrations")
			return nil
		}
		if err == nil {
			fmt.Println("up migrations done")
		}
	}
	if direction == "down" {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("no migrations to rollback")
			return nil
		}
		if err == nil {
			fmt.Println("down migrations done")
		}
	}
	return err
}

func nextSequence(path string) (int, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return 1, nil
	}

	maxSeq := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".up.sql") && !strings.HasSuffix(name, ".down.sql") {
			continue
		}
		parts := strings.SplitN(name, "_", 2)
		if len(parts) < 2 {
			continue
		}
		seq, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		if seq > maxSeq && seq < 10000 {
			maxSeq = seq
		}
	}

	return maxSeq + 1, nil
}

func previewMigrations(m *migrate.Migrate, direction string, args []string) error {
	currentVersion, dirty, err := m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			currentVersion = 0
		} else {
			return fmt.Errorf("reading version: %w", err)
		}
	}
	if dirty {
		return fmt.Errorf("database is dirty at version %d", currentVersion)
	}

	path := "internal/database/migrations"

	migrations, err := listMigrations(path)
	if err != nil {
		return err
	}

	switch direction {
	case "up":
		var pending []migrationFile
		for _, mig := range migrations {
			if mig.version > currentVersion {
				pending = append(pending, mig)
			}
		}
		if len(pending) == 0 {
			fmt.Println("No pending migrations.")
			return nil
		}

		if len(args) > 0 {
			n, _ := strconv.Atoi(args[0])
			if n > 0 && n < len(pending) {
				pending = pending[:n]
			}
		}

		fmt.Printf("Current version: %d\n", currentVersion)
		fmt.Printf("Would apply %d migration(s):\n", len(pending))
		for _, mig := range pending {
			fmt.Printf("  → %s\n", mig.name)
		}

	case "down":
		if currentVersion == 0 {
			fmt.Println("No migrations applied.")
			return nil
		}

		n, _ := strconv.Atoi(args[0])
		var rollback []migrationFile
		for i := len(migrations) - 1; i >= 0; i-- {
			if migrations[i].version <= currentVersion && len(rollback) < n {
				rollback = append(rollback, migrations[i])
			}
		}

		fmt.Printf("Current version: %d\n", currentVersion)
		fmt.Printf("Would roll back %d migration(s):\n", len(rollback))
		for _, mig := range rollback {
			fmt.Printf("  → %s (down)\n", mig.name)
		}

	case "goto":
		target, _ := strconv.Atoi(args[0])
		fmt.Printf("Current version: %d\n", currentVersion)
		fmt.Printf("Target version: %d\n", target)

		if uint(target) == currentVersion {
			fmt.Println("Already at target version.")
			return nil
		}

		if uint(target) > currentVersion {
			var pending []migrationFile
			for _, mig := range migrations {
				if mig.version > currentVersion && mig.version <= uint(target) {
					pending = append(pending, mig)
				}
			}
			fmt.Printf("Would apply %d migration(s):\n", len(pending))
			for _, mig := range pending {
				fmt.Printf("  → %s\n", mig.name)
			}
		} else {
			var rollback []migrationFile
			for i := len(migrations) - 1; i >= 0; i-- {
				if migrations[i].version > uint(target) && migrations[i].version <= currentVersion {
					rollback = append(rollback, migrations[i])
				}
			}
			fmt.Printf("Would roll back %d migration(s):\n", len(rollback))
			for _, mig := range rollback {
				fmt.Printf("  → %s (down)\n", mig.name)
			}
		}
	}

	return nil
}

type migrationFile struct {
	version uint
	name    string
}

func listMigrations(path string) ([]migrationFile, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("reading migrations directory: %w", err)
	}

	var migrations []migrationFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}

		base := strings.TrimSuffix(entry.Name(), ".up.sql")
		parts := strings.SplitN(base, "_", 2)
		if len(parts) < 2 {
			continue
		}

		version, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			continue
		}

		migrations = append(migrations, migrationFile{
			version: uint(version),
			name:    entry.Name(),
		})
	}

	for i := 0; i < len(migrations); i++ {
		for j := i + 1; j < len(migrations); j++ {
			if migrations[j].version < migrations[i].version {
				migrations[i], migrations[j] = migrations[j], migrations[i]
			}
		}
	}

	return migrations, nil
}
