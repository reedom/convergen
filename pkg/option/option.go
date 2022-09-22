package option

import (
	"strings"

	"github.com/reedom/convergen/pkg/model"
)

type Options struct {
	Style model.DstVarStyle
	Rule  model.MatchRule

	ExactCase bool
	Getter    bool
	Stringer  bool
	Typecast  bool

	Receiver    string
	SkipFields  []*IdentMatcher
	NameMapper  []*NameMatcher
	Converters  []*FieldConverter
	PostProcess *Postprocess
}

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

func (o Options) ShouldSkip(fieldName string) bool {
	for _, skip := range o.SkipFields {
		if skip.Match(fieldName, o.ExactCase) {
			return true
		}
	}
	return false
}

func (o Options) CompareFieldName(a, b string) bool {
	if o.ExactCase {
		return a == b
	}
	return strings.ToLower(a) == strings.ToLower(b)
}

var ValidOpsIntf = map[string]struct{}{
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
	"rcv":          {},
	"skip":         {},
	"map":          {},
	"tag":          {},
	"conv":         {},
	"conv:type":    {},
	"conv:with":    {},
	"postprocess":  {},
}
