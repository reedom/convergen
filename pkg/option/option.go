package option

import (
	"strings"

	"github.com/reedom/convergen/pkg/generator/model"
)

// Options represents the conversion options.
type Options struct {
	Style               model.DstVarStyle // Style of the destination variable name
	Rule                model.MatchRule   // Matching rule for fields
	ExactCase           bool              // Whether to match fields with exact case sensitivity
	Getter              bool              // Whether to use getter methods to access fields
	Stringer            bool              // Whether to use stringer methods to convert values to strings
	Typecast            bool              // Whether to use explicit typecasts when converting values
	Receiver            string            // Receiver name for method generation
	Reverse             bool              // Whether to reverse the order of struct tags
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
	if o.ExactCase {
		return a == b
	}
	return strings.EqualFold(a, b)
}

// ValidOpsIntf is a set of valid conversion option keys for interface-level conversion.
var ValidOpsIntf = map[string]struct{}{
	"convergen":    {},
	"style":        {},
	"match":        {},
	"case":         {},
	"case:off":     {},
	"getter":       {},
	"getter:off":   {},
	"stringer":     {},
	"stringer:off": {},
	"typecast":     {},
	"typecast:off": {},
}

// ValidOpsMethod is a set of valid conversion option keys for method-level conversion.
var ValidOpsMethod = map[string]struct{}{
	"style":        {},
	"match":        {},
	"case":         {},
	"case:off":     {},
	"getter":       {},
	"getter:off":   {},
	"stringer":     {},
	"stringer:off": {},
	"typecast":     {},
	"typecast:off": {},
	"recv":         {},
	"reverse":      {},
	"skip":         {},
	"map":          {},
	"tag":          {},
	"conv":         {},
	"conv:type":    {},
	"conv:with":    {},
	"literal":      {},
	"preprocess":   {},
	"postprocess":  {},
}
