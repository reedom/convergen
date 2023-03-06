package model

// DstVarStyle represents the style of destination variable in a function signature.
type DstVarStyle string

// String returns the string representation of the destination variable style.
func (s DstVarStyle) String() string {
	return string(s)
}

const (
	// DstVarReturn indicates that the destination variable is a return
	// value in a function signature.
	DstVarReturn = DstVarStyle("return")
	// DstVarArg indicates that the destination variable is an argument
	// in a function signature.
	DstVarArg = DstVarStyle("arg")
)

// DstVarStyleValues is a slice of all possible destination variable styles.
var DstVarStyleValues = []DstVarStyle{DstVarReturn, DstVarArg}

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
