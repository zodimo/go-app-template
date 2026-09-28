package core

import (
	"regexp"
	"strings"
)

var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`PASSWORD`),
	regexp.MustCompile(`SECRET`),
	regexp.MustCompile(`TOKEN`),
	regexp.MustCompile(`KEY`),
}

func RedactedValue(key, value string) string {
	if value == "" {
		return ""
	}
	upper := strings.ToUpper(key)
	for _, re := range sensitivePatterns {
		if re.MatchString(upper) {
			return MaskString(value)
		}
	}
	return value
}

func MaskString(s string) string {
	runes := []rune(s)
	if len(runes) <= 4 {
		return "****"
	}
	return string(runes[:2]) + "****" + string(runes[len(runes)-2:])
}
