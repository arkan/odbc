package informix

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// InterpolateQuery takes a SQL query with placeholders and arguments,
// and returns a safe SQL string with properly escaped and formatted values.
func InterpolateQuery(query string, args ...interface{}) (string, error) {
	if len(args) == 0 {
		return query, nil
	}

	// Handle different placeholder styles ($1, $2) or (?)
	placeholder := regexp.MustCompile(`\$\d+|\?`)
	argPosition := 0

	interpolated := placeholder.ReplaceAllStringFunc(query, func(match string) string {
		if argPosition >= len(args) {
			return match // Not enough arguments provided
		}

		// Get the current argument
		arg := args[argPosition]
		argPosition++

		return formatArgument(arg)
	})

	if argPosition < len(args) {
		return "", fmt.Errorf("too many arguments provided: expected %d, got %d", argPosition, len(args))
	}

	return interpolated, nil
}

// formatArgument converts a Go value to its SQL string representation
func formatArgument(arg interface{}) string {
	if arg == nil {
		return "NULL"
	}

	// Handle values that implement driver.Valuer
	if valuer, ok := arg.(driver.Valuer); ok {
		val, err := valuer.Value()
		if err != nil {
			return "NULL"
		}
		arg = val
	}

	switch v := arg.(type) {
	case bool:
		return strconv.FormatBool(v)

	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)

	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)

	case float32, float64:
		return fmt.Sprintf("%f", v)

	case string:
		return escapeString(v)

	case []byte:
		return formatBytes(v)

	case time.Time:
		return fmt.Sprintf("'%s'", v.Format("2006-01-02 15:04:05.999999"))

	case []interface{}:
		return formatArray(v)
	}

	// Handle slices of basic types
	rv := reflect.ValueOf(arg)
	if rv.Kind() == reflect.Slice {
		values := make([]string, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			values[i] = formatArgument(rv.Index(i).Interface())
		}
		return fmt.Sprintf("(%s)", strings.Join(values, ","))
	}

	// Default to string representation
	return escapeString(fmt.Sprintf("%v", arg))
}

// escapeString properly escapes a string for SQL
func escapeString(s string) string {
	// Replace any single quotes with two single quotes (SQL escape sequence)
	escaped := strings.ReplaceAll(s, "'", "''")
	// Wrap in single quotes
	return fmt.Sprintf("'%s'", escaped)
}

// formatBytes formats a byte slice as a hex string
func formatBytes(b []byte) string {
	return fmt.Sprintf("'\\x%x'", b)
}

// formatArray formats a slice as a SQL array string
func formatArray(arr []interface{}) string {
	elements := make([]string, len(arr))
	for i, v := range arr {
		elements[i] = formatArgument(v)
	}
	return fmt.Sprintf("ARRAY[%s]", strings.Join(elements, ","))
}
