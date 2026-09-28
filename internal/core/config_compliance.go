package core

type CompliantConfig interface {
	PrintableConfig
	ValidatableConfig
	ContextAwareConfig
}
