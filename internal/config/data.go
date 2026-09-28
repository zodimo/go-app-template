package config

import (
	"fmt"
	"io"
	"strings"

	"github.com/zodimo/go-app-template/internal/core"
	"github.com/zodimo/go-maybe"
)

var _ core.CompliantConfig = (*Data)(nil)

// Data defines storage configuration.
type Data struct {
	configContext core.ConfigContext  `mapstructure:"-" json:"-"`
	Directory     maybe.Maybe[string] `mapstructure:"directory" json:"directory,omitempty"`
}

func (d *Data) Print(headingLabel string, output io.Writer, printOptions ...core.ConfigPrintOption) error {
	if !d.IsValid() {
		return fmt.Errorf("%s has config errors", headingLabel)
	}

	printOps := core.DefaultConfigPrintOptions()
	for _, opt := range printOptions {
		if opt != nil {
			opt(&printOps)
		}
	}

	rootIndentLevel := printOps.IndentLevel
	indentLevel := rootIndentLevel + 1

	fmt.Fprintf(output, "%s%s\n", printOps.Indent(rootIndentLevel), headingLabel)
	fmt.Fprintf(output, "%s%s: %s\n", printOps.Indent(indentLevel), "Directory", d.Directory.UnwrapUnsafe())
	return nil

}

func (d *Data) Validate() (errs []error) {
	configPrefix := d.configContext.GetPrefix()
	envPrefix := d.configContext.GetEnvPrefix()
	if d.Directory.IsNone() {
		configEnvPrefix := strings.ToUpper(configPrefix)
		errs = append(errs, fmt.Errorf("%s.database is required: set it in the config file or the %s%s_DIRECTORY environment variable", configPrefix, envPrefix, configEnvPrefix))
	}

	return
}

func (d *Data) IsValid() bool {
	validationErrors := d.Validate()
	return len(validationErrors) == 0
}

func (d *Data) WithConfigContext(configContext core.ConfigContext) {
	d.configContext = configContext
}
