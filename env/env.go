package env

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"fortio.org/log"
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

// CamelCaseToLowerSnakeCase converts a string from camelCase or CamelCase
// to lowe_snake_case. Handles cases like HTTPServer -> http_server.
// Good for JSON tags for instance.
func CamelCaseToLowerSnakeCase(s string) string {
	if s == "" {
		return ""
	}
	words := SplitByCase(s)
	// ToLower + Join by _
	return strings.ToLower(strings.Join(words, "_"))
}

// CamelCaseToLowerKebabCase converts a string from camelCase or CamelCase
// to lower-kebab-case. Handles cases like HTTPServer -> http-server.
// Good for command line flags for instance.
func CamelCaseToLowerKebabCase(s string) string {
	if s == "" {
		return ""
	}
	words := SplitByCase(s)
	// ToLower and join by -
	return strings.ToLower(strings.Join(words, "-"))
}

type KeyValue struct {
	Key   string
	Value string // Already quoted/escaped.
}

func (kv KeyValue) String() string {
	return fmt.Sprintf("%s=%s", kv.Key, kv.Value)
}

func ToShell(kvl []KeyValue) string {
	return ToShellWithPrefix("", kvl)
}

// This convert the key value pairs to bourne shell syntax (vs newer bash export FOO=bar).
func ToShellWithPrefix(prefix string, kvl []KeyValue) string {
	var sb strings.Builder
	keys := make([]string, 0, len(kvl))
	for _, kv := range kvl {
		sb.WriteString(prefix)
		sb.WriteString(kv.String())
		sb.WriteRune('\n')
		keys = append(keys, prefix+kv.Key)
	}
	sb.WriteString("export ")
	sb.WriteString(strings.Join(keys, " "))
	sb.WriteRune('\n')
	return sb.String()
}

func SerializeValue(value interface{}) string {
	switch v := value.(type) {
	case bool:
		res := "false"
		if v {
			res = "true"
		}
		return res
	case string:
		return strconv.Quote(v)
	default:
		return strconv.Quote(fmt.Sprint(value))
	}
}

// StructToEnvVars converts a struct to a map of environment variables.
// The struct can have a `env` tag on each field.
// The tag should be in the format `env:"ENV_VAR_NAME"`.
// The tag can also be `env:"-"` to exclude the field from the map.
// If the field is exportable and the tag is missing we'll use the field name
// converted to UPPER_SNAKE_CASE (using CamelCaseToUpperSnakeCase()) as the
// environment variable name.
func StructToEnvVars(s interface{}) []KeyValue {
	return structToEnvVars("", s)
}

func structToEnvVars(prefix string, s interface{}) []KeyValue {
	var envVars []KeyValue
	v := reflect.ValueOf(s)
	// if we're passed a pointer to a struct instead of the struct, let that work too
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		log.Errf("Unexpected kind %v, expected a struct", v.Kind())
		return envVars
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		fieldType := t.Field(i)
		tag := fieldType.Tag.Get("env")
		if tag == "-" {
			continue
		}
		if fieldType.Anonymous {
			// Recurse
			envVars = append(envVars, structToEnvVars("", v.Field(i).Interface())...)
			continue
		}
		if tag == "" {
			tag = CamelCaseToUpperSnakeCase(fieldType.Name)
		}
		fieldValue := v.Field(i)
		stringValue := ""
		switch fieldValue.Kind() { //nolint: exhaustive // we have default: for the other cases
		case reflect.Ptr:
			if !fieldValue.IsNil() {
				fieldValue = fieldValue.Elem()
				stringValue = SerializeValue(fieldValue.Interface())
			}
		case reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
			log.LogVf("Skipping field %s of type %v, not supported", fieldType.Name, fieldType.Type)
			continue
		case reflect.Struct:
			// Recurse with prefix
			envVars = append(envVars, structToEnvVars(tag+"_", fieldValue.Interface())...)
			continue
		default:
			value := fieldValue.Interface()
			stringValue = SerializeValue(value)
		}
		envVars = append(envVars, KeyValue{Key: prefix + tag, Value: stringValue})
	}
	return envVars
}
