package model

type DstVarStyle string

func (s DstVarStyle) String() string {
	return string(s)
}

const (
	DstVarReturn = DstVarStyle("return")
	DstVarArg    = DstVarStyle("arg")
)

var DstVarStyleValues = []DstVarStyle{DstVarReturn, DstVarArg}

func NewDstVarStyleFromValue(v string) (DstVarStyle, bool) {
	for _, style := range DstVarStyleValues {
		if style.String() == v {
			return style, true
		}
	}
	return DstVarStyle(""), false
}

type MatchRule string

func (s MatchRule) String() string {
	return string(s)
}

const (
	MatchRuleName = MatchRule("name")
	MatchRuleTag  = MatchRule("tag")
	MatchRuleNone = MatchRule("none")
)

var MatchRuleValues = []MatchRule{MatchRuleName, MatchRuleTag, MatchRuleNone}

func NewMatchRuleFromValue(v string) (MatchRule, bool) {
	for _, rule := range MatchRuleValues {
		if rule.String() == v {
			return rule, true
		}
	}
	return MatchRule(""), false
}
