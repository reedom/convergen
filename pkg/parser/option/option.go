package option

import (
	"fmt"
)

type DstVarStyle string

const (
	DstVarReturn = DstVarStyle("return")
	DstVarArg    = DstVarStyle("arg")
)

type FieldMatchSrc string

const (
	FieldMatchField  = FieldMatchSrc("field")
	FieldMatchGetter = FieldMatchSrc("getter")
)

type GlobalOption struct {
	Style           DstVarStyle
	FieldMatchOrder []FieldMatchSrc
	NoCase          bool
	Converters      []any
}

type MethodOption struct {
	Style           DstVarStyle
	FieldMatchOrder []FieldMatchSrc
	NoCase          bool
	PostProcess     string
	Skip            []IdentMatcher
	Matchers        []any
	Converters      []any
}

func (o *MethodOption) AddMatcher(m any) {
	switch m.(type) {
	case *FieldMatcher:
		o.Matchers = append(o.Matchers, m)
	default:
		panic(fmt.Sprintf("unknown matcher: %q", m))
	}
}

func (o *MethodOption) AddConverter(c any) {
	switch c.(type) {
	case *FieldConverter:
		o.Converters = append(o.Converters, c)
	default:
		panic(fmt.Sprintf("unknown converter: %q", c))
	}
}

type FieldOption struct {
}
