package logging

import (
	"fmt"
	"io"

	"github.com/zodimo/go-app-template/internal/core"
)

var _ core.PrintableConfig = (*LogConfig)(nil)

func (c *LogConfig) Print(headingLabel string, output io.Writer, printOptions ...core.ConfigPrintOption) error {
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
	fmt.Fprintf(output, "%s%s: %s\n", printOps.Indent(indentLevel), "Directory", c.Directory.UnwrapOr(""))
	fmt.Fprintf(output, "%s%s: %s\n", printOps.Indent(indentLevel), "Filename", c.Filename.UnwrapOr(""))
	fmt.Fprintf(output, "%s%s: %d\n", printOps.Indent(indentLevel), "MaxSize", c.MaxSize.UnwrapOr(0))
	fmt.Fprintf(output, "%s%s: %d\n", printOps.Indent(indentLevel), "MaxBackups", c.MaxBackups.UnwrapOr(0))
	fmt.Fprintf(output, "%s%s: %d\n", printOps.Indent(indentLevel), "MaxAge", c.MaxAge.UnwrapOr(0))
	fmt.Fprintf(output, "%s%s: %t\n", printOps.Indent(indentLevel), "Compress", c.Compress.UnwrapOr(false))
	fmt.Fprintf(output, "%s%s: %t\n", printOps.Indent(indentLevel), "Debug", c.Debug.UnwrapOr(false))

	return nil

}
