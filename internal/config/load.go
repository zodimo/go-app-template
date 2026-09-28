package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
	"github.com/zodimo/go-app-template/internal/identity"
	"github.com/zodimo/go-app-template/internal/logging"
	"github.com/zodimo/go-maybe"
)

var ErrConfigFileRequired = errors.New("config file required but not found")

type LoadOptions struct {
	dataDir           string
	debug             bool
	disableValidation bool
	loadDefaultsOnly  bool
	adHocConfigFile   string
	requireConfig     bool
}

type LoadOption func(*LoadOptions)

func defaultLoadOptions() LoadOptions {
	return LoadOptions{
		dataDir:           "",
		debug:             false,
		disableValidation: false,
		loadDefaultsOnly:  false,
		requireConfig:     false,
	}
}

func WithDataDirectory(dataDir string) LoadOption {
	return func(opts *LoadOptions) {
		opts.dataDir = dataDir
	}
}

func WithDebug(debug bool) LoadOption {
	return func(opts *LoadOptions) {
		opts.debug = debug
	}
}

func WithDisableValidation(disableValidation bool) LoadOption {
	return func(opts *LoadOptions) {
		opts.disableValidation = disableValidation
	}
}

func WithDefaultsOnly(loadDefaultsOnly bool) LoadOption {
	return func(opts *LoadOptions) {
		opts.loadDefaultsOnly = loadDefaultsOnly
	}
}

func WithConfigFile(configFile string) LoadOption {
	return func(opts *LoadOptions) {
		opts.adHocConfigFile = configFile
	}
}

// WithRequireConfig makes a missing config file fatal when no explicit --config path is given.
func WithRequireConfig(requireConfig bool) LoadOption {
	return func(opts *LoadOptions) {
		opts.requireConfig = requireConfig
	}
}

// Load initializes the configuration from environment variables and config files.
// If debug is true, debug mode is enabled and log level is set to debug.
// It returns an error if configuration loading fails.
func Load(workingDir string, options ...LoadOption) (*Config, error) {

	rootConfigContext := NewAppConfigContext(
		"",
		identity.EnvPrefix,
	)

	opts := defaultLoadOptions()
	for _, opt := range options {
		if opt != nil {
			opt(&opts)
		}
	}
	if cfg != nil {
		if opts.debug {
			fmt.Println("Configuration already loaded")
		}
		return cfg, nil
	}

	localCfg := &Config{
		WorkingDir: workingDir,
		Data: Data{
			Directory: maybe.Some(opts.dataDir),
		},
	}

	// init app config context
	localCfg.WithConfigContext(rootConfigContext)
	localCfg.Data.WithConfigContext(
		rootConfigContext.WithPrefix("data").
			WithEnvPrefix(strings.Join([]string{rootConfigContext.GetEnvPrefix(), "DATA"}, "_")),
	)
	localCfg.Log.WithConfigContext(
		rootConfigContext.WithPrefix("log").
			WithEnvPrefix(strings.Join([]string{rootConfigContext.GetEnvPrefix(), "LOG"}, "_")),
	)
	localCfg.Database.WithConfigContext(
		rootConfigContext.WithPrefix("database").
			WithEnvPrefix(strings.Join([]string{rootConfigContext.GetEnvPrefix(), "DATABASE"}, "_")),
	)

	configureViper(workingDir)
	setDefaultsFor(workingDir, opts.debug)

	// A --data-dir flag has the highest precedence: it must override the
	// file/default value for data.directory.
	if opts.dataDir != "" {
		viper.Set("data.directory", opts.dataDir)
	}

	// Allow missing config file  only when explicitly set by this option
	if !opts.loadDefaultsOnly {

		err := handleConfigPath(opts.adHocConfigFile)
		if err != nil {
			return nil, fmt.Errorf("failed to handle config path %q: %w", opts.adHocConfigFile, err)
		}

		// Read global config
		if err := viper.ReadInConfig(); err != nil {
			var notFound viper.ConfigFileNotFoundError
			if errors.As(err, &notFound) || errors.Is(err, os.ErrNotExist) {
				// An explicit --config path is always required (k3d rule): a typo'd
				// path must never silently fall back. --require-config opts into
				// strict mode when relying on the search paths.
				if opts.adHocConfigFile != "" || opts.requireConfig {
					searchPaths := GetConfigSearchPaths(workingDir, defaultSearchPaths).GetSearchPaths()
					return nil, fmt.Errorf("%w; searched: %s", ErrConfigFileRequired, strings.Join(searchPaths, ", "))
				}
				// No explicit path and no --require-config: silently fall back to
				// defaults + environment variables.
			} else {
				return nil, fmt.Errorf("failed to read config file (%s): %w", viper.ConfigFileUsed(), err)
			}
		}
	}

	// Apply configuration to the struct. Compose the existing default hooks
	// (StringToTimeDurationHookFunc, stringToWeakSliceHookFunc) with the Maybe
	// decode hook so maybe.Maybe[T] fields decode from file scalars and env strings.
	if err := viper.Unmarshal(localCfg, func(c *mapstructure.DecoderConfig) {
		c.DecodeHook = mapstructure.ComposeDecodeHookFunc(c.DecodeHook, maybeDecodeHook)
	}); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config into struct: %w", err)
	}

	applyDefaultValuesTo(localCfg)

	if err := logging.Setup(localCfg.Log); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to setup logging: %v\n", err)
	}

	if !opts.disableValidation && !opts.loadDefaultsOnly {
		// Validate configuration. Defaults-only loads (version, config init,
		// show --defaults-only) are informational and must not trip
		// required-field validation.
		if err := validateConfig(localCfg, rootConfigContext); err != nil {
			return nil, fmt.Errorf("config validation failed: %w", err)
		}
	}

	cfg = localCfg
	if logging.Initialized() {
		logging.GetLogger().Info("configuration loaded", "file", viper.ConfigFileUsed())
	}
	return cfg, nil
}
