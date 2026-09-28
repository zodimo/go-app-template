package database

import "github.com/zodimo/go-app-template/internal/core"

func (c *Config) WithConfigContext(configContext core.ConfigContext) {
	c.configContext = configContext
}
