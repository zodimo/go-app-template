package config

import "github.com/zodimo/go-app-template/internal/core"

var _ core.ConfigContext = (*appConfigContext)(nil)

type appConfigContext struct {
	prefix    string
	envPrefix string
}

func (ac *appConfigContext) GetEnvPrefix() string {
	return ac.envPrefix
}
func (ac *appConfigContext) GetPrefix() string {
	return ac.prefix
}

func NewAppConfigContext(
	prefix string,
	envPrefix string,
) core.ConfigContext {
	return &appConfigContext{
		prefix:    prefix,
		envPrefix: envPrefix,
	}
}

func (ac *appConfigContext) WithPrefix(prefix string) core.ConfigContext {
	return &appConfigContext{
		prefix:    prefix,
		envPrefix: ac.envPrefix,
	}
}
func (ac *appConfigContext) WithEnvPrefix(envPrefix string) core.ConfigContext {
	return &appConfigContext{
		prefix:    ac.prefix,
		envPrefix: envPrefix,
	}
}

func (c *Config) WithConfigContext(configContext core.ConfigContext) {
	c.configContext = configContext
}
