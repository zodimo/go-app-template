package core

type ConfigContext interface {
	GetEnvPrefix() string
	GetPrefix() string
	WithEnvPrefix(envPrefix string) ConfigContext
	WithPrefix(prefix string) ConfigContext
}

type ContextAwareConfig interface {
	WithConfigContext(configContext ConfigContext)
}
