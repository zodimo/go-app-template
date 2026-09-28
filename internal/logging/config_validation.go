package logging

import (
	"fmt"

	"github.com/zodimo/go-app-template/internal/core"
)

var _ core.PrintableConfig = (*LogConfig)(nil)

func (c *LogConfig) Validate() (errs []error) {

	configPrefix := c.configContext.GetPrefix()
	envPrefix := c.configContext.GetEnvPrefix()

	if c.Directory.IsNone() {
		errs = append(errs, fmt.Errorf("%s.directory is required: set it in the config file or the %s_DIRECTORY environment variable", configPrefix, envPrefix))
	}

	if c.Filename.IsNone() {
		errs = append(errs, fmt.Errorf("%s.filename is required: set it in the config file or the %s_FILENAME environment variable", configPrefix, envPrefix))
	}

	if c.MaxSize.IsNone() {
		errs = append(errs, fmt.Errorf("%s.max_size is required: set it in the config file or the %s_MAX_SIZE environment variable", configPrefix, envPrefix))
	}

	if c.MaxBackups.IsNone() {
		errs = append(errs, fmt.Errorf("%s.max_backups is required: set it in the config file or the %s_MAX_BACKUPS environment variable", configPrefix, envPrefix))
	}

	if c.MaxAge.IsNone() {
		errs = append(errs, fmt.Errorf("%s.max_age is required: set it in the config file or the %s_MAX_AGE environment variable", configPrefix, envPrefix))
	}

	if c.Compress.IsNone() {
		errs = append(errs, fmt.Errorf("%s.compress is required: set it in the config file or the %s_COMPRESS environment variable", configPrefix, envPrefix))
	}

	if c.Debug.IsNone() {
		errs = append(errs, fmt.Errorf("%s.debug is required: set it in the config file or the %s_DEBUG environment variable", configPrefix, envPrefix))
	}

	return
}

func (c *LogConfig) IsValid() bool {
	if c != nil {
		validationErrors := c.Validate()
		return len(validationErrors) == 0
	}
	return false
}
