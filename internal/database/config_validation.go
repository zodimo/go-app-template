package database

import (
	"fmt"

	"github.com/zodimo/go-app-template/internal/core"
)

var _ core.PrintableConfig = (*Config)(nil)

func (c *Config) Validate() (errs []error) {
	if c.Database.IsNone() {
		errs = append(errs, fmt.Errorf("%s.database is required: set it in the config file or the %s_DATABASE environment variable", c.configContext.GetPrefix(), c.configContext.GetEnvPrefix()))
	}

	if c.Params.IsNone() {
		errs = append(errs, fmt.Errorf("%s.params is required: set it in the config file or the %s_PARAMS environment variable", c.configContext.GetPrefix(), c.configContext.GetEnvPrefix()))
	}

	if c.MigrationParams.IsNone() {
		errs = append(errs, fmt.Errorf("%s.migration_params is required: set it in the config file or the %s_MIGRATION_PARAMS environment variable", c.configContext.GetPrefix(), c.configContext.GetEnvPrefix()))
	}

	return
}

func (c *Config) IsValid() bool {
	if c != nil {
		validationErrors := c.Validate()
		return len(validationErrors) == 0
	}
	return false
}
