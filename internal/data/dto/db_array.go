package dto

import (
	"fmt"
	"regexp"
	"strings"
)

type DbStrArray []string

func (d *DbStrArray) Scan(val interface{}) error {
	switch v := val.(type) {
	case []byte:
		*d = parseArray(string(v))
		return nil
	case string:
		*d = parseArray(v)
		return nil
	default:
		return fmt.Errorf("unsupported type: %v", v)
	}
}

func parseArray(array string) []string {
	// unquoted array values must not contain: (" , \ { } whitespace NULL)
	// and must be at least one char
	unquotedChar := `[^",\\{}\s(NULL)]`
	unquotedValue := fmt.Sprintf("(%s)+", unquotedChar)

	// quoted array values are surrounded by double quotes, can be any
	// character except " or \, which must be backslash escaped:
	quotedChar := `[^"\\]|\\"|\\\\`
	quotedValue := fmt.Sprintf("\"(%s)*\"", quotedChar)

	// an array value may be either quoted or unquoted:
	arrayValue := fmt.Sprintf("(?P<value>(%s|%s))", unquotedValue, quotedValue)

	// Array values are separated with a comma IF there is more than one value:
	arrayExp := regexp.MustCompile(fmt.Sprintf("((%s)(,)?)", arrayValue))

	var valueIndex int

	results := make([]string, 0)
	matches := arrayExp.FindAllStringSubmatch(array, -1)
	for _, match := range matches {
		s := match[valueIndex]
		// the string _might_ be wrapped in quotes, so trim them:
		s = strings.Trim(s, "\"")
		results = append(results, s)
	}
	return results
}
