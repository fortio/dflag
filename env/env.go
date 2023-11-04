// Package env provides conversion from structure to and from environment variables.
//
// It supports converting struct fields to environment variables using field tags,
// handling different data types, and transforming strings between different case
// conventions, which is useful for generating or parsing environment variables,
// JSON tags, or command line flags.
//
// The package defines several case conversion functions that aid in manipulating
// strings to fit conventional casing for various programming and configuration contexts.
// Additionally, it provides functions to serialize structs into slices of key-value pairs
// where the keys are derived from struct field names transformed to upper snake case by default,
// or specified explicitly via struct field tags.
//
// It also includes functionality to deserialize environment variables back into
// struct fields, handling pointers and nested structs appropriately, as well as providing
// shell-compatible output for environment variable definitions.
//
// The package leverages reflection to dynamically handle arbitrary struct types,
// and logs its operations and errors using the 'fortio.org/log' package.
package env

import (
	"fmt"
	"os"
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

// TODO: consider returning errors or at least counting issues.

// setInt sets an integer field from a string.
func setInt(fieldValue reflect.Value, val string) {
	intVal, err := strconv.ParseInt(val, 10, fieldValue.Type().Bits())
	if err != nil {
		log.Errf("Invalid integer value %q: %v", val, err)
		return
	}
	fieldValue.SetInt(intVal)
}

// setFloat sets a float field from a string.
func setFloat(fieldValue reflect.Value, val string) {
	floatVal, err := strconv.ParseFloat(val, fieldValue.Type().Bits())
	if err != nil {
		log.Errf("Invalid float value %q: %v", val, err)
		return
	}
	fieldValue.SetFloat(floatVal)
}

func setBool(fieldValue reflect.Value, val string) {
	boolVal, err := strconv.ParseBool(val)
	if err != nil {
		log.Errf("Invalid bool value %q: %v", val, err)
		return
	}
	fieldValue.SetBool(boolVal)
}

func setPointer(fieldValue reflect.Value) reflect.Value {
	// Ensure we have a pointer to work with, allocate if nil.
	if fieldValue.IsNil() {
		fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
	}
	// Get the element the pointer is pointing to.
	return fieldValue.Elem()
}

func checkEnv(envName, fieldName string, fieldValue reflect.Value) *string {
	val, found := os.LookupEnv(envName)
	if !found {
		log.LogVf("%q not set for %s", envName, fieldName)
		return nil
	}
	log.Infof("Found %s=%q to set %s", envName, val, fieldName)
	if !fieldValue.CanSet() {
		log.Errf("Can't set %s (found %s=%q)", fieldName, envName, val)
		return nil
	}
	return &val
}

func SetFromEnv(prefix string, s interface{}) {
	v := reflect.ValueOf(s)
	// if we're passed a pointer to a struct instead of the struct, let that work too
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		log.Errf("Unexpected kind %v, expected a struct", v.Kind())
		return
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		fieldType := t.Field(i)
		tag := fieldType.Tag.Get("env")
		if tag == "-" {
			continue
		}
		if tag == "" {
			tag = CamelCaseToUpperSnakeCase(fieldType.Name)
		}
		envName := prefix + tag
		fieldValue := v.Field(i)

		kind := fieldValue.Kind()

		if kind == reflect.Struct {
			// Recurse with prefix
			if fieldValue.CanAddr() { // Check if we can get the address
				SetFromEnv(envName+"_", fieldValue.Addr().Interface())
			} else {
				log.Errf("Cannot take the address of %s to recurse", fieldType.Name)
			}
			continue
		}

		val := checkEnv(envName, fieldType.Name, fieldValue)
		if val == nil {
			continue
		}
		envVal := *val

		// Handle pointer fields separately
		if kind == reflect.Ptr {
			kind = fieldValue.Type().Elem().Kind()
			fieldValue = setPointer(fieldValue)
		}

		switch kind { //nolint: exhaustive // we have default: for the other cases
		case reflect.String:
			fieldValue.SetString(envVal)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			setInt(fieldValue, envVal)
		case reflect.Float32, reflect.Float64:
			setFloat(fieldValue, envVal)
		case reflect.Bool:
			setBool(fieldValue, envVal)
		default:
			log.Warnf("Unsupported type %v to set from %s=%q", kind, envName, envVal)
		}
	}
}
