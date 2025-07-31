package parser

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseIntArg(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		hasError bool
	}{
		{
			name:     "valid positive integer",
			input:    "123",
			expected: 123,
			hasError: false,
		},
		{
			name:     "valid negative integer",
			input:    "-456",
			expected: -456,
			hasError: false,
		},
		{
			name:     "zero",
			input:    "0",
			expected: 0,
			hasError: false,
		},
		{
			name:     "invalid input - non-numeric",
			input:    "abc",
			expected: 0,
			hasError: true,
		},
		{
			name:     "invalid input - empty string",
			input:    "",
			expected: 0,
			hasError: true,
		},
		{
			name:     "invalid input - float",
			input:    "123.45",
			expected: 0,
			hasError: true,
		},
		{
			name:     "large integer",
			input:    "999999999",
			expected: 999999999,
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseIntArg(tt.input)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseDurationArg(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		hasError bool
	}{
		{
			name:     "seconds",
			input:    "30s",
			expected: 30 * time.Second,
			hasError: false,
		},
		{
			name:     "minutes",
			input:    "5m",
			expected: 5 * time.Minute,
			hasError: false,
		},
		{
			name:     "hours",
			input:    "2h",
			expected: 2 * time.Hour,
			hasError: false,
		},
		{
			name:     "milliseconds",
			input:    "500ms",
			expected: 500 * time.Millisecond,
			hasError: false,
		},
		{
			name:     "microseconds",
			input:    "100us",
			expected: 100 * time.Microsecond,
			hasError: false,
		},
		{
			name:     "nanoseconds",
			input:    "1000ns",
			expected: 1000 * time.Nanosecond,
			hasError: false,
		},
		{
			name:     "complex duration",
			input:    "1h30m45s",
			expected: 1*time.Hour + 30*time.Minute + 45*time.Second,
			hasError: false,
		},
		{
			name:     "zero duration",
			input:    "0s",
			expected: 0,
			hasError: false,
		},
		{
			name:     "invalid format",
			input:    "invalid",
			expected: 0,
			hasError: true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
			hasError: true,
		},
		{
			name:     "just number without unit",
			input:    "123",
			expected: 0,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDurationArg(tt.input)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestIsBuiltinType(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		expected bool
	}{
		// Built-in types
		{"bool", "bool", true},
		{"byte", "byte", true},
		{"complex64", "complex64", true},
		{"complex128", "complex128", true},
		{"error", "error", true},
		{"float32", "float32", true},
		{"float64", "float64", true},
		{"int", "int", true},
		{"int8", "int8", true},
		{"int16", "int16", true},
		{"int32", "int32", true},
		{"int64", "int64", true},
		{"rune", "rune", true},
		{"string", "string", true},
		{"uint", "uint", true},
		{"uint8", "uint8", true},
		{"uint16", "uint16", true},
		{"uint32", "uint32", true},
		{"uint64", "uint64", true},
		{"uintptr", "uintptr", true},

		// Non built-in types
		{"custom struct", "User", false},
		{"custom interface", "Reader", false},
		{"package qualified", "time.Time", false},
		{"pointer type", "*string", false},
		{"slice type", "[]int", false},
		{"map type", "map[string]int", false},
		{"channel type", "chan string", false},
		{"empty string", "", false},
		{"case sensitive", "String", false}, // should be "string"
		{"case sensitive", "INT", false},    // should be "int"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBuiltinType(tt.typeName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeTypeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple type",
			input:    "string",
			expected: "string", // Currently returns input as-is
		},
		{
			name:     "pointer type",
			input:    "*User",
			expected: "*User", // Currently returns input as-is
		},
		{
			name:     "package qualified",
			input:    "time.Time",
			expected: "time.Time", // Currently returns input as-is
		},
		{
			name:     "complex type",
			input:    "map[string]*User",
			expected: "map[string]*User", // Currently returns input as-is
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeTypeName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid identifier",
			input:    "validName",
			expected: "validName",
		},
		{
			name:     "valid identifier with underscore",
			input:    "valid_name",
			expected: "valid_name",
		},
		{
			name:     "valid identifier with numbers",
			input:    "name123",
			expected: "name123",
		},
		{
			name:     "starts with underscore",
			input:    "_privateName",
			expected: "_privateName",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "field",
		},
		{
			name:     "starts with number",
			input:    "123name",
			expected: "_23name",
		},
		{
			name:     "contains spaces",
			input:    "field name",
			expected: "field_name",
		},
		{
			name:     "contains hyphens",
			input:    "field-name",
			expected: "field_name",
		},
		{
			name:     "contains dots",
			input:    "field.name",
			expected: "field_name",
		},
		{
			name:     "contains special characters",
			input:    "field@#$%name",
			expected: "field____name",
		},
		{
			name:     "mixed invalid characters",
			input:    "field-name.test@123",
			expected: "field_name_test_123",
		},
		{
			name:     "all invalid characters",
			input:    "@#$%^&*()",
			expected: "_________",
		},
		{
			name:     "unicode characters",
			input:    "fieldñame",
			expected: "field_ame",
		},
		{
			name:     "starts with capital letter",
			input:    "FieldName",
			expected: "FieldName",
		},
		{
			name:     "starts with lowercase letter",
			input:    "fieldName",
			expected: "fieldName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func BenchmarkIsBuiltinType(b *testing.B) {
	testTypes := []string{"string", "int", "bool", "User", "time.Time", "*string"}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, typeName := range testTypes {
			isBuiltinType(typeName)
		}
	}
}

func BenchmarkSanitizeIdentifier(b *testing.B) {
	testNames := []string{
		"validName",
		"field-name",
		"field.name.test",
		"field@#$%name",
		"123invalid",
		"",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, name := range testNames {
			sanitizeIdentifier(name)
		}
	}
}

func TestParseIntArg_EdgeCases(t *testing.T) {
	// Test edge cases for integer parsing
	tests := []struct {
		name     string
		input    string
		hasError bool
	}{
		{"whitespace", " 123 ", true}, // strconv.Atoi doesn't handle whitespace
		{"leading zeros", "00123", false},
		{"plus sign", "+123", false},
		{"hex format", "0x123", true},   // strconv.Atoi doesn't handle hex
		{"octal format", "0123", false}, // This will be parsed as decimal 123
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseIntArg(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseDurationArg_EdgeCases(t *testing.T) {
	// Test edge cases for duration parsing
	tests := []struct {
		name     string
		input    string
		hasError bool
	}{
		{"negative duration", "-5s", false},
		{"fractional seconds", "1.5s", false},
		{"fractional milliseconds", "1.5ms", false},
		{"very small duration", "1ns", false},
		{"very large duration", "8760h", false}, // 1 year in hours
		{"whitespace", " 5s ", true},            // time.ParseDuration doesn't handle whitespace
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseDurationArg(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
