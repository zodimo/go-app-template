package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/zodimo/go-app-template/internal/commands"
	"github.com/zodimo/go-app-template/internal/logging"
)

func init() {
	cobra.EnableTraverseRunHooks = true
}

func main() {
	var exitCode uint8
	defer func() {
		if exitCode != 0 {
			os.Exit(int(exitCode))
		}
	}()
	defer logging.RecoverPanic("cli-config-main", &exitCode, nil)

	exitCode = commands.ExecuteCmd(commands.ConfigCmd)
}
