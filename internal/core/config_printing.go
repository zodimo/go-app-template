package core

import (
	"io"
	"strings"
)

type ConfigPrintOptions struct {
	IndentString string
	IndentLevel  int
	IndentFunc   func(i int) string
	Redacted     bool
	RedactFunc   func(s string) string
	AssertValid  bool
}

func DefaultConfigPrintOptions() ConfigPrintOptions {
	return ConfigPrintOptions{
		IndentString: "  ",
		IndentLevel:  0,
		Redacted:     true,
		AssertValid:  true,
	}
}

func (po *ConfigPrintOptions) Indent(level int) string {
	if po.IndentFunc == nil {
		defaultIndentFunc := func(i int) string {
			return strings.Repeat("  ", i)
		}
		return defaultIndentFunc(level)
	}
	return po.IndentFunc(level)
}

func (po *ConfigPrintOptions) Redact(input string) string {
	if po.Redacted {
		if po.RedactFunc == nil {
			defaultRedactFunc := MaskString
			return defaultRedactFunc(input)
		}
		return po.RedactFunc(input)
	}
	return input
}

func (po *ConfigPrintOptions) Copy(options ...ConfigPrintOption) *ConfigPrintOptions {
	localOptions := *po
	for _, opt := range options {
		if opt != nil {
			opt(&localOptions)
		}
	}
	return &localOptions
}

type ConfigPrintOption = func(o *ConfigPrintOptions)

func PrintConfigWithIndentString(indent string) ConfigPrintOption {
	return func(o *ConfigPrintOptions) {
		o.IndentString = indent
	}
}

func PrintConfigWithIndentLevel(level int) ConfigPrintOption {
	return func(o *ConfigPrintOptions) {
		o.IndentLevel = level
	}
}

func PrintConfigWithRedactedValues(redacted bool) ConfigPrintOption {
	return func(o *ConfigPrintOptions) {
		o.Redacted = redacted
	}
}

func PrintConfigWithRedactFunc(redact func(string) string) ConfigPrintOption {
	return func(o *ConfigPrintOptions) {
		o.RedactFunc = redact
	}
}

func PrintConfigWithAssertValid(validate bool) ConfigPrintOption {
	return func(o *ConfigPrintOptions) {
		o.AssertValid = validate
	}
}

type PrintableConfig interface {
	Print(headingLabel string, output io.Writer, printOptions ...ConfigPrintOption) error
}
