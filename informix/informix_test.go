package informix

import (
	"database/sql/driver"
	"testing"
	"time"
)

// Custom type that implements driver.Valuer for testing
type customValuer struct {
	value string
}

func (c customValuer) Value() (driver.Value, error) {
	return c.value, nil
}

func TestInterpolateQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		args     []interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "no placeholders",
			query:    "SELECT * FROM users",
			args:     nil,
			expected: "SELECT * FROM users",
			wantErr:  false,
		},
		{
			name:     "basic types",
			query:    "SELECT * FROM users WHERE id = $1 AND name = $2 AND active = $3",
			args:     []interface{}{123, "John", true},
			expected: "SELECT * FROM users WHERE id = 123 AND name = 'John' AND active = true",
			wantErr:  false,
		},
		{
			name:     "string with quotes",
			query:    "SELECT * FROM users WHERE name = $1",
			args:     []interface{}{"O'Connor"},
			expected: "SELECT * FROM users WHERE name = 'O''Connor'",
			wantErr:  false,
		},
		{
			name:     "null value",
			query:    "UPDATE users SET last_login = $1",
			args:     []interface{}{nil},
			expected: "UPDATE users SET last_login = NULL",
			wantErr:  false,
		},
		{
			name:     "byte slice",
			query:    "INSERT INTO documents (data) VALUES ($1)",
			args:     []interface{}{[]byte{0x1, 0x2, 0x3}},
			expected: "INSERT INTO documents (data) VALUES ('\\x010203')",
			wantErr:  false,
		},
		{
			name:     "string slice",
			query:    "SELECT * FROM users WHERE role = ANY($1)",
			args:     []interface{}{[]string{"admin", "user"}},
			expected: "SELECT * FROM users WHERE role = ANY(('admin','user'))",
			wantErr:  false,
		},
		{
			name:     "question mark placeholder",
			query:    "SELECT * FROM users WHERE id = ? AND name = ?",
			args:     []interface{}{123, "John"},
			expected: "SELECT * FROM users WHERE id = 123 AND name = 'John'",
			wantErr:  false,
		},
		{
			name:     "too few arguments",
			query:    "SELECT * FROM users WHERE id = $1 AND name = $2",
			args:     []interface{}{123},
			expected: "SELECT * FROM users WHERE id = 123 AND name = $2",
			wantErr:  false,
		},
		{
			name:     "too many arguments",
			query:    "SELECT * FROM users WHERE id = $1",
			args:     []interface{}{123, "extra"},
			expected: "",
			wantErr:  true,
		},
		{
			name:     "float values",
			query:    "INSERT INTO measurements (value1, value2) VALUES ($1, $2)",
			args:     []interface{}{float32(1.23), float64(4.56)},
			expected: "INSERT INTO measurements (value1, value2) VALUES (1.230000, 4.560000)",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InterpolateQuery(tt.query, tt.args...)

			if (err != nil) != tt.wantErr {
				t.Errorf("InterpolateQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.expected {
				t.Errorf("InterpolateQuery() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFormatArgument(t *testing.T) {
	timeValue := time.Date(2024, 2, 12, 15, 4, 5, 999999000, time.UTC)

	tests := []struct {
		name     string
		arg      interface{}
		expected string
	}{
		{
			name:     "nil value",
			arg:      nil,
			expected: "NULL",
		},
		{
			name:     "string",
			arg:      "hello",
			expected: "'hello'",
		},
		{
			name:     "int",
			arg:      42,
			expected: "42",
		},
		{
			name:     "bool true",
			arg:      true,
			expected: "true",
		},
		{
			name:     "bool false",
			arg:      false,
			expected: "false",
		},
		{
			name:     "float64",
			arg:      3.14,
			expected: "3.140000",
		},
		{
			name:     "time.Time",
			arg:      timeValue,
			expected: "'2024-02-12 15:04:05.999999'",
		},
		{
			name:     "[]byte",
			arg:      []byte{0x1, 0x2, 0x3},
			expected: "'\\x010203'",
		},
		{
			name:     "custom valuer",
			arg:      customValuer{value: "custom"},
			expected: "'custom'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatArgument(tt.arg)
			if got != tt.expected {
				t.Errorf("formatArgument() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEscapeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal string",
			input:    "hello",
			expected: "'hello'",
		},
		{
			name:     "string with single quote",
			input:    "O'Connor",
			expected: "'O''Connor'",
		},
		{
			name:     "string with multiple quotes",
			input:    "It's O'Connor's",
			expected: "'It''s O''Connor''s'",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "''",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeString(tt.input)
			if got != tt.expected {
				t.Errorf("escapeString() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFormatArray(t *testing.T) {
	tests := []struct {
		name     string
		input    []interface{}
		expected string
	}{
		{
			name:     "string array",
			input:    []interface{}{"a", "b", "c"},
			expected: "ARRAY['a','b','c']",
		},
		{
			name:     "mixed array",
			input:    []interface{}{1, "two", true},
			expected: "ARRAY[1,'two',true]",
		},
		{
			name:     "empty array",
			input:    []interface{}{},
			expected: "ARRAY[]",
		},
		{
			name:     "array with null",
			input:    []interface{}{nil, "value", nil},
			expected: "ARRAY[NULL,'value',NULL]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatArray(tt.input)
			if got != tt.expected {
				t.Errorf("formatArray() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Benchmark the main function
func BenchmarkInterpolateQuery(b *testing.B) {
	query := "SELECT * FROM users WHERE id = $1 AND name = $2 AND active = $3"
	args := []interface{}{123, "John", true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := InterpolateQuery(query, args...)
		if err != nil {
			b.Fatal(err)
		}
	}
}
