package database

import (
	"fmt"
	"io"

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

	fmt.Fprintf(output, "%s%s\n", printOps.Indent(rootIndentLevel), headingLabel)
	fmt.Fprintf(output, "%s%s: %s\n", printOps.Indent(indentLevel), "Database", printOps.Redact(c.Database.UnwrapOr("")))
	fmt.Fprintf(output, "%s%s\n", printOps.Indent(indentLevel), "Params")

	params := c.Params.UnwrapOr(map[string]string{})
	if len(params) > 0 {
		for k, v := range params {
			fmt.Fprintf(output, "%s%s: %s\n", printOps.Indent(indentLevel+1), k, v)
		}
	} else {
		fmt.Fprintf(output, "%sNo Params\n", printOps.Indent(indentLevel+1))
	}

	fmt.Fprintf(output, "%s%s\n", printOps.Indent(indentLevel), "Migration Params")
	migrationParams := c.MigrationParams.UnwrapOr(map[string]string{})
	if len(migrationParams) > 0 {
		for k, v := range migrationParams {
			fmt.Fprintf(output, "%s%s: %s\n", printOps.Indent(indentLevel+1), k, v)
		}
	} else {
		fmt.Fprintf(output, "%sNo Migration Params\n", printOps.Indent(indentLevel+1))
	}

	return nil
}
