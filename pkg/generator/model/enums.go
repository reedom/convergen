// Package model provides the data structures used to represent the generated code.
package model

// DstVarStyle represents the style of destination variable in a function signature.
type DstVarStyle string

// String returns the string representation of the destination variable style.
func (s DstVarStyle) String() string {
	return string(s)
}

// IsStructLiteral returns true if the destination variable style is struct literal.
func (s DstVarStyle) IsStructLiteral() bool {
	return s == DstVarStructLiteral
}

const (
	// DstVarReturn indicates that the destination variable is a return
	// value in a function signature.
	DstVarReturn = DstVarStyle("return")
	// DstVarArg indicates that the destination variable is an argument
	// in a function signature.
	DstVarArg = DstVarStyle("arg")
	// DstVarStructLiteral indicates that the destination variable is created
	// using struct literal syntax.
	DstVarStructLiteral = DstVarStyle("struct_literal")
)

// DstVarStyleValues is a slice of all possible destination variable styles.
var DstVarStyleValues = []DstVarStyle{DstVarReturn, DstVarArg, DstVarStructLiteral}

// NewDstVarStyleFromValue creates a new DstVarStyle instance from the
// given value string.
func NewDstVarStyleFromValue(v string) (DstVarStyle, bool) {
	for _, style := range DstVarStyleValues {
		if style.String() == v {
			return style, true
		}
	}

	return "", false
}

// MatchRule represents the field matching rule.
type MatchRule string

// String returns the string representation of the match rule.
func (s MatchRule) String() string {
	return string(s)
}

const (
	// MatchRuleName indicates that the field name is used as the matching criteria.
	MatchRuleName = MatchRule("name")
	// MatchRuleTag indicates that the field tag is used as the matching criteria.
	MatchRuleTag = MatchRule("tag")
	// MatchRuleNone indicates that there is no matching criteria for the field.
	MatchRuleNone = MatchRule("none")
)

// MatchRuleValues is a slice of all possible field matching rules.
var MatchRuleValues = []MatchRule{MatchRuleName, MatchRuleTag, MatchRuleNone}

// NewMatchRuleFromValue creates a new MatchRule instance from the given value string.
func NewMatchRuleFromValue(v string) (MatchRule, bool) {
	for _, rule := range MatchRuleValues {
		if rule.String() == v {
			return rule, true
		}
	}

	return "", false
}

// OutputStyle represents the output style for code generation.
type OutputStyle string

// String returns the string representation of the output style.
func (s OutputStyle) String() string {
	return string(s)
}

const (
	// OutputStyleAuto indicates that the output style should be determined automatically
	// based on compatibility analysis.
	OutputStyleAuto = OutputStyle("auto")
	// OutputStyleStructLiteral indicates that the output should use struct literal syntax.
	OutputStyleStructLiteral = OutputStyle("struct_literal")
	// OutputStyleTraditional indicates that the output should use traditional assignment syntax.
	OutputStyleTraditional = OutputStyle("traditional")
)

// OutputStyleValues is a slice of all possible output styles.
var OutputStyleValues = []OutputStyle{OutputStyleAuto, OutputStyleStructLiteral, OutputStyleTraditional}

// NewOutputStyleFromValue creates a new OutputStyle instance from the given value string.
func NewOutputStyleFromValue(v string) (OutputStyle, bool) {
	for _, style := range OutputStyleValues {
		if style.String() == v {
			return style, true
		}
	}

	return "", false
}
