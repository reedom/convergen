// Package model provides the data structures used to represent the generated code.
package model

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// FunctionsBlock represents a group of functions.
type FunctionsBlock struct {
	Marker    string      // Marker is a special comment marker for indicating a specific section of functions.
	Functions []*Function // Functions is the list of functions.
}

// Function represents a function.
type Function struct {
	Comments           []string     // Comments is the list of comment lines before the function definition.
	Name               string       // Name is the function name.
	Receiver           string       // Receiver is the receiver specification, if any (e.g., "c" or "*UserService").
	ReceiverVar        string       // ReceiverVar is the receiver variable name (e.g., "s" for receiver type "Service").
	ReceiverType       string       // ReceiverType is the receiver type (e.g., "*UserService").
	Src                Var          // Src is the source variable.
	Dst                Var          // Dst is the destination variable.
	AdditionalArgs     []Var        // AdditionalArgs is the additional arguments variables.
	RetError           bool         // RetError indicates whether the function returns an error.
	DstVarStyle        DstVarStyle  // DstVarStyle is the style of the destination variable declaration.
	Assignments        []Assignment // Assignments is the list of assignments in the function body.
	PreProcess         *Manipulator // PreProcess is the function that is applied before the assignments.
	PostProcess        *Manipulator // PostProcess is the function that is applied after the assignments.
	OutputStyle        OutputStyle  // OutputStyle is the determined output style for code generation.
	FallbackReason     string       // FallbackReason explains why struct literal couldn't be used (if applicable).
	CanUseStructLit    bool         // CanUseStructLit indicates whether struct literal output is compatible.
	ForceStructLiteral bool         // ForceStructLiteral forces struct literal generation (from :struct-literal annotation).
	NoStructLiteral    bool         // NoStructLiteral disables struct literal generation (from :no-struct-literal annotation).
}

// HasReceiver returns true if this function has a receiver method.
func (f *Function) HasReceiver() bool {
	return f.Receiver != ""
}

// IsReceiverMethod returns true if this function should be generated as a method (not a standalone function).
// For backward compatibility, it returns true if there's a receiver even without ReceiverType being set.
func (f *Function) IsReceiverMethod() bool {
	if !f.HasReceiver() {
		return false
	}

	// If the new fields are populated, use them
	if f.ReceiverType != "" {
		return true
	}

	// Backward compatibility: if Receiver is set but ReceiverType is not,
	// assume it's a receiver method (old behavior)
	return true
}

// ParseReceiverSpec parses a receiver specification and populates ReceiverVar and ReceiverType.
// Handles both simple identifiers ("c") and type specifications ("*UserService").
func (f *Function) ParseReceiverSpec() {
	if f.Receiver == "" {
		f.ReceiverVar = ""
		f.ReceiverType = ""
		return
	}

	// Check if it's a simple identifier (backward compatibility)
	if isSimpleIdentifier(f.Receiver) {
		// Use as variable name, determine type from source
		f.ReceiverVar = f.Receiver
		f.ReceiverType = f.Src.FullType()
		return
	}

	// Handle type specification (e.g., "*UserService", "Service")
	receiverType := f.Receiver

	// Generate variable name from type
	f.ReceiverVar = generateReceiverVarName(receiverType)
	f.ReceiverType = receiverType
}

// isSimpleIdentifier checks if a string is a simple Go identifier.
func isSimpleIdentifier(s string) bool {
	if s == "" {
		return false
	}

	first, size := utf8.DecodeRuneInString(s)
	if !unicode.IsLetter(first) && first != '_' {
		return false
	}

	for i := size; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
		i += size
	}

	return true
}

// generateReceiverVarName generates a receiver variable name from a type specification.
// Examples:
// - "*UserService" -> "u"
// - "Service" -> "s"
// - "pkg.UserService" -> "u".
func generateReceiverVarName(receiverType string) string {
	// Remove pointer prefix
	typeName := strings.TrimPrefix(receiverType, "*")

	// Get the last part if it's a qualified name
	parts := strings.Split(typeName, ".")
	typeName = parts[len(parts)-1]

	// Convert to lowercase first letter
	if typeName == "" {
		return "r" // Default receiver name
	}

	first, size := utf8.DecodeRuneInString(typeName)
	if size == 0 {
		return "r"
	}

	return strings.ToLower(string(first))
}
