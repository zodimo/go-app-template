package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

func Init(opts ...LoadOption) (*Config, error) {
	// Register defaults for the keys read directly below. Load() also sets
	// these, but it runs after these reads, so on the config/migration
	// binaries (which never bind cwd/debug flags) the defaults must exist
	// here for every read key to be resolvable at read time.
	ensureInitDefaults()
	cwd := viper.GetString(cwdFlag)
	debug := viper.GetBool(debugFlag)
	defaultsOnly := viper.GetBool(defaultsOnlyFlag)

	loadOpts := []LoadOption{}
	if debug {
		loadOpts = append(loadOpts, WithDebug(debug))
	}
	if defaultsOnly {
		loadOpts = append(loadOpts, WithDefaultsOnly(defaultsOnly))
	}

	if cwd != "" {
		if err := os.Chdir(cwd); err != nil {
			return nil, fmt.Errorf("failed to change directory to %q: %w", cwd, err)
		}
	} else {
		c, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %w", err)
		}
		cwd = c
	}
	return Load(cwd, append(loadOpts, opts...)...)
}

// ensureInitDefaults makes the keys Init reads directly resolvable even when
// no command root has bound them and Load has not yet run.
func ensureInitDefaults() {
	viper.SetDefault(cwdFlag, "")
	viper.SetDefault(debugFlag, false)
	viper.SetDefault(defaultsOnlyFlag, false)
}
