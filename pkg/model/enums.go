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
