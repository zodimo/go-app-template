package logging

import "github.com/zodimo/go-app-template/internal/core"

func (c *LogConfig) WithConfigContext(configContext core.ConfigContext) {
	c.configContext = configContext
}
