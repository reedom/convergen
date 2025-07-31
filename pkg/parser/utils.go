package parser

import (
	"fmt"
	"strconv"
	"time"
)

// parseIntArg parses an integer argument from a string.
func parseIntArg(arg string) (int, error) {
	val, err := strconv.Atoi(arg)
	if err != nil {
		return 0, fmt.Errorf("failed to parse integer argument %q: %w", arg, err)
	}

	return val, nil
}

// parseDurationArg parses a duration argument from a string.
func parseDurationArg(arg string) (time.Duration, error) {
	duration, err := time.ParseDuration(arg)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration argument %q: %w", arg, err)
	}

	return duration, nil
}

// isBuiltinType checks if a type name represents a Go builtin type.
func isBuiltinType(typeName string) bool {
	builtins := map[string]bool{
		"bool":       true,
		"byte":       true,
		"complex64":  true,
		"complex128": true,
		"error":      true,
		"float32":    true,
		"float64":    true,
		"int":        true,
		"int8":       true,
		"int16":      true,
		"int32":      true,
		"int64":      true,
		"rune":       true,
		"string":     true,
		"uint":       true,
		"uint8":      true,
		"uint16":     true,
		"uint32":     true,
		"uint64":     true,
		"uintptr":    true,
	}

	return builtins[typeName]
}

// normalizeTypeName normalizes a type name for consistent comparison.
func normalizeTypeName(typeName string) string {
	// Remove package prefixes and pointer indicators
	// This is a simplified implementation
	return typeName
}

// sanitizeIdentifier ensures a string is a valid Go identifier.
func sanitizeIdentifier(name string) string {
	if name == "" {
		return "field"
	}

	// Simple sanitization - replace invalid characters with underscores
	result := make([]rune, 0, len(name))

	for i, r := range name {
		if i == 0 {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' {
				result = append(result, r)
			} else {
				result = append(result, '_')
			}
		} else {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
				result = append(result, r)
			} else {
				result = append(result, '_')
			}
		}
	}

	return string(result)
}
