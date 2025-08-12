package option

import (
	"strings"

	"github.com/reedom/convergen/v9/pkg/generator/model"
)

// Options represents the conversion options.
type Options struct {
	Style               model.DstVarStyle // Style of the destination variable name
	Rule                model.MatchRule   // Matching rule for fields
	ExactCase           bool              // Whether to match fields with exact case sensitivity
	Getter              bool              // Whether to use getter methods to access fields
	Stringer            bool              // Whether to use stringer methods to convert values to strings
	Typecast            bool              // Whether to use explicit typecasts when converting values
	Receiver            string            // Receiver specification for method generation (e.g., "c" or "*UserService")
	Reverse             bool              // Whether to reverse the order of struct tags
	StructLiteral       bool              // Whether to force struct literal generation
	NoStructLiteral     bool              // Whether to disable struct literal generation
	SkipFields          []*PatternMatcher // List of field names to skip during conversion
	NameMapper          []*NameMatcher    // List of field name mapping rules
	TemplatedNameMapper []*NameMatcher    // List of templated field name mapping rules
	Converters          []*FieldConverter // List of field conversion rules
	Literals            []*LiteralSetter  // List of literal value setting rules
	PreProcess          *Manipulator      // Manipulator to run before struct processing
	PostProcess         *Manipulator      // Manipulator to run after struct processing
}

// NewOptions returns a new Options instance.
func NewOptions() Options {
	return Options{
		Style:     model.DstVarReturn,
		Rule:      model.MatchRuleName,
		ExactCase: true,
		Getter:    false,
		Stringer:  false,
		Typecast:  false,
	}
}

// ShouldSkip returns true if the field with the given name should be skipped.
func (o Options) ShouldSkip(fieldName string) bool {
	for _, skip := range o.SkipFields {
		if skip.Match(fieldName, o.ExactCase) {
			return true
		}
	}

	return false
}

// CompareFieldName compares two field names.
func (o Options) CompareFieldName(a, b string) bool {
	// For MatchRuleNone, no fields should match
	if o.Rule == model.MatchRuleNone {
		return false
	}

	// Exact match (fastest path)
	if o.ExactCase {
		if a == b {
			return true
		}
	} else {
		if strings.EqualFold(a, b) {
			return true
		}
	}

	// For MatchRuleName, implement intelligent name matching
	if o.Rule == model.MatchRuleName {
		return o.matchFieldNames(a, b)
	}

	// For MatchRuleTag or other cases, fall back to exact/case-insensitive matching
	return false
}

// matchFieldNames implements intelligent field name matching for :match name annotation
func (o Options) matchFieldNames(srcName, dstName string) bool {
	// Try common prefix removal patterns
	prefixes := []string{
		"User", "Item", "Data", "Info", "Record", "Entity", "Model",
		"Src", "Source", "Dst", "Dest", "Destination", "Target",
	}

	compareNames := func(a, b string) bool {
		if o.ExactCase {
			return a == b
		}
		return strings.EqualFold(a, b)
	}

	// Try removing prefixes from source name
	for _, prefix := range prefixes {
		if o.ExactCase {
			if strings.HasPrefix(srcName, prefix) && compareNames(strings.TrimPrefix(srcName, prefix), dstName) {
				return true
			}
		} else {
			if strings.HasPrefix(strings.ToLower(srcName), strings.ToLower(prefix)) {
				trimmed := srcName[len(prefix):]
				if compareNames(trimmed, dstName) {
					return true
				}
			}
		}
	}

	// Try removing prefixes from destination name
	for _, prefix := range prefixes {
		if o.ExactCase {
			if strings.HasPrefix(dstName, prefix) && compareNames(srcName, strings.TrimPrefix(dstName, prefix)) {
				return true
			}
		} else {
			if strings.HasPrefix(strings.ToLower(dstName), strings.ToLower(prefix)) {
				trimmed := dstName[len(prefix):]
				if compareNames(srcName, trimmed) {
					return true
				}
			}
		}
	}

	// Try suffix matching patterns (e.g., Name in UserName)
	if o.ExactCase {
		if strings.HasSuffix(srcName, dstName) || strings.HasSuffix(dstName, srcName) {
			return true
		}
	} else {
		srcLower := strings.ToLower(srcName)
		dstLower := strings.ToLower(dstName)
		if strings.HasSuffix(srcLower, dstLower) || strings.HasSuffix(dstLower, srcLower) {
			return true
		}
	}

	return false
}

// ValidOpsIntf is a set of valid conversion option keys for interface-level conversion.
var ValidOpsIntf = map[string]struct{}{
	"convergen":         {},
	"style":             {},
	"match":             {},
	"case":              {},
	"case:off":          {},
	"getter":            {},
	"getter:off":        {},
	"stringer":          {},
	"stringer:off":      {},
	"typecast":          {},
	"typecast:off":      {},
	"struct-literal":    {},
	"no-struct-literal": {},
}

// ValidOpsMethod is a set of valid conversion option keys for method-level conversion.
var ValidOpsMethod = map[string]struct{}{
	"style":             {},
	"match":             {},
	"case":              {},
	"case:off":          {},
	"getter":            {},
	"getter:off":        {},
	"stringer":          {},
	"stringer:off":      {},
	"typecast":          {},
	"typecast:off":      {},
	"recv":              {},
	"reverse":           {},
	"struct-literal":    {},
	"no-struct-literal": {},
	"skip":              {},
	"map":               {},
	"tag":               {},
	"conv":              {},
	"conv:type":         {},
	"conv:with":         {},
	"literal":           {},
	"preprocess":        {},
	"postprocess":       {},
}
