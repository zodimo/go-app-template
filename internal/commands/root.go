package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zodimo/go-app-template/internal/config"
	"github.com/zodimo/go-app-template/internal/exitcode"
)

var RootCmd = &cobra.Command{
	Use: "cli",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// dont load config for version command
		if cmd.Name() != "version" && cmd.Name() != "setup" {
			configFile := viper.GetString("root-config-file")
			requireConfig := viper.GetBool("require-config")
			_, err := config.Init(
				config.WithConfigFile(configFile),
				config.WithRequireConfig(requireConfig),
			)
			return err
		}
		return nil
	},
}

func ExecuteCmd(cmd *cobra.Command) (exitCode uint8) {
	err := cmd.Execute()
	if err == nil {
		return 0
	}

	if es, ok := errors.AsType[exitcode.ExitStatus](err); ok {
		exitCode = uint8(es)
		if es == exitcode.ExitStatusOK {
			return
		}
	} else {
		exitCode = 1
	}

	if viper.GetBool("json") {
		errJSON := map[string]string{"error": err.Error()}
		if out, marshalErr := json.MarshalIndent(errJSON, "", "  "); marshalErr == nil {
			fmt.Fprintln(os.Stderr, string(out))
		} else {
			// Fallback if marshal fails
			fmt.Fprintf(os.Stderr, "{\"error\": \"%s\"}\n", err.Error())
		}
	} else {
		fmt.Fprintln(os.Stderr, err)
	}

	return
}

func init() {
	RootCmd.SilenceUsage = true
	RootCmd.SilenceErrors = true

	RootCmd.PersistentFlags().String("config", "", "cli config file")
	if err := viper.BindPFlag("root-config-file", RootCmd.PersistentFlags().Lookup("config")); err != nil {
		panic(fmt.Sprintf("failed to bind --config flag: %v", err))
	}

	RootCmd.PersistentFlags().Bool("require-config", false, "fail if no config file is found")
	if err := viper.BindPFlag("require-config", RootCmd.PersistentFlags().Lookup("require-config")); err != nil {
		panic(fmt.Sprintf("failed to bind --require-config flag: %v", err))
	}
	RootCmd.PersistentFlags().Bool("debug", false, "Debug (overrides config/env)")
	if err := viper.BindPFlag("debug", RootCmd.PersistentFlags().Lookup("debug")); err != nil {
		panic(fmt.Sprintf("failed to bind --debug flag: %v", err))
	}

	// --json is deliberately bound only on the main `cli` root. ExecuteCmd is
	// shared, but only the main binary can enable JSON error mode; on the
	// config/migration binaries this key is unbound and reads as false.
	RootCmd.PersistentFlags().Bool("json", false, "Output in JSON (wip)")
	if err := viper.BindPFlag("json", RootCmd.PersistentFlags().Lookup("json")); err != nil {
		panic(fmt.Sprintf("failed to bind --json flag: %v", err))
	}

	RootCmd.PersistentFlags().String("cwd", "", "current working directory")
	if err := viper.BindPFlag("cwd", RootCmd.PersistentFlags().Lookup("cwd")); err != nil {
		panic(fmt.Sprintf("failed to bind --cwd flag: %v", err))
	}

	RootCmd.AddCommand(runCmd)
	RootCmd.AddCommand(newVersionCmd())
	RootCmd.AddCommand(setupCmd)

}
