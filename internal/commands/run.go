package commands

import (
	"github.com/spf13/cobra"
	"github.com/zodimo/go-app-template/internal/app"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run App.Run",
	RunE: func(cmd *cobra.Command, args []string) error {
		run := &app.App{}
		return run.Run()

	},
}
