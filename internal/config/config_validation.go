package config

import (
	"fmt"
	"strings"
)

func (c *Config) Validate() (errs []error) {

	configPrefix := c.configContext.GetPrefix()
	envPrefix := c.configContext.GetEnvPrefix()

	if c.WorkingDir == "" {
		configEnvPrefix := strings.ToUpper(configPrefix)
		errs = append(errs, fmt.Errorf("%s.working_dir is required: set it in the config file or the %s%s_WORKING DIR environment variable", configPrefix, envPrefix, configEnvPrefix))
	}

	errs = append(errs, c.Data.Validate()...)
	errs = append(errs, c.Log.Validate()...)
	errs = append(errs, c.Database.Validate()...)

	return
}

func (c *Config) IsValid() bool {
	if c != nil {
		validationErrors := c.Validate()
		return len(validationErrors) == 0
	}
	return false
}
