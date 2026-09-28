package commands

import (
	"github.com/spf13/cobra"
	"github.com/zodimo/go-app-template/internal/config"
	"github.com/zodimo/go-app-template/internal/version"
)

// printVersion renders the build info to stdout. It is shared by every
// binary's version command so all three print identical output.
func printVersion() {
	version.PrintVersion(version.GetBuildInfo())
}

// newVersionCmd returns a fresh version command each call. Each binary owns
// its own instance so help output names the correct parent and no command
// pointer is shared across the three roots.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Print version information including build date and commit ID",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.Init(
				config.WithDefaultsOnly(true),
			)
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			printVersion()
			return nil
		},
	}
}
