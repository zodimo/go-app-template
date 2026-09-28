package core

type ValidatableConfig interface {
	Validate() (errs []error)
	IsValid() bool
}
