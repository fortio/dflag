package env

import (
	"strings"
	"unicode"
)

// Split strings into words, using CamelCase/camelCase/CAMELCase rules.
func SplitByCase(input string) []string {
	if input == "" {
		return nil
	}
	var words []string
	var buffer strings.Builder
	runes := []rune(input)

	for i := 0; i < len(runes); i++ {
		first := (i == 0)
		last := (i == len(runes)-1)
		if !first && unicode.IsUpper(runes[i]) {
			if !last && unicode.IsLower(runes[i+1]) || unicode.IsLower(runes[i-1]) {
				words = append(words, buffer.String())
				buffer.Reset()
			}
		}
		buffer.WriteRune(runes[i])
	}
	words = append(words, buffer.String())
	return words
}

// CamelCaseToUpperSnakeCase converts a string from camelCase or CamelCase
// to UPPER_SNAKE_CASE. Handles cases like HTTPServer -> HTTP_SERVER and
// httpServer -> HTTP_SERVER. Good for environment variables.
func CamelCaseToUpperSnakeCase(s string) string {
	if s == "" {
		return ""
	}
	words := SplitByCase(s)
	// ToUpper + Join by _
	return strings.ToUpper(strings.Join(words, "_"))
}

// CamelCaseToLowerKebabCase converts a string from camelCase or CamelCase
// to lower-kebab-case. Handles cases like HTTPServer -> http-server.
// Good for JSON tags and command line flags.
func CamelCaseToLowerKebabCase(s string) string {
	if s == "" {
		return ""
	}
	words := SplitByCase(s)
	// ToLower and join by -
	return strings.ToLower(strings.Join(words, "-"))
}

/*
func StructToEnvVars(s interface{}) map[string]string {
	m := make(map[string]string)
	// use reflection to get the fields of the struct
	st := reflect.TypeOf(s)
	fields := st.Fields()
	// for each field, get the tag
	for _, field := range fields {
		// get the tag
		tag := field.Tag

	return m
}
*/
