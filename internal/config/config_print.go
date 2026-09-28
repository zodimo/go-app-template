package config

import (
	"fmt"
	"io"

	"github.com/spf13/viper"
	"github.com/zodimo/go-app-template/internal/core"
)

var _ core.PrintableConfig = (*Config)(nil)

func (c *Config) Print(headingLabel string, output io.Writer, printOptions ...core.ConfigPrintOption) error {

	printOps := core.DefaultConfigPrintOptions()
	for _, opt := range printOptions {
		if opt != nil {
			opt(&printOps)
		}
	}

	if printOps.AssertValid {
		if !c.IsValid() {
			return fmt.Errorf("%s has config errors", headingLabel)
		}
	}

	rootIndentLevel := printOps.IndentLevel
	indentLevel := rootIndentLevel + 1

	if used := viper.ConfigFileUsed(); used != "" {
		fmt.Fprintf(output, "%sConfiguration from: %s\n", printOps.Indent(rootIndentLevel), used)
	} else {
		fmt.Fprintf(output, "%sConfiguration from: (no config file used)\n", printOps.Indent(rootIndentLevel))
	}
	fmt.Fprintf(output, "%sWorking directory: %s\n", printOps.Indent(indentLevel), c.WorkingDir)
	fmt.Fprintf(output, "%sDebug: %t\n", printOps.Indent(indentLevel), c.Debug)
	c.Data.Print("Application Data", output, append(printOptions, core.PrintConfigWithIndentLevel(indentLevel))...)
	c.Log.Print("Logging", output, append(printOptions, core.PrintConfigWithIndentLevel(indentLevel))...)
	c.Database.Print("Database", output, append(printOptions, core.PrintConfigWithIndentLevel(indentLevel))...)

	return nil

}
