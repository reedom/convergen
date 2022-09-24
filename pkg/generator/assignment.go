package generator

import (
	"strings"

	"github.com/reedom/convergen/pkg/generator/model"
)

func AssignmentToString(f *model.Function, a *model.Assignment) string {
	var sb strings.Builder

	switch a.RHS.(type) {
	case model.SkipField:
		sb.WriteString("// skip: ")
		sb.WriteString(a.LHS)
		sb.WriteString("\n")
	case model.NoMatchField:
		sb.WriteString("// no match: ")
		sb.WriteString(a.LHS)
		sb.WriteString("\n")
	default:
		sb.WriteString(a.LHS)
		if a.RHS.ReturnsError() {
			sb.WriteString(", err")
		}
		sb.WriteString(" = ")
		sb.WriteString(a.RHS.String())
		sb.WriteString("\n")
		if a.RHS.ReturnsError() {
			if f.DstVarStyle == model.DstVarReturn && f.Dst.Pointer {
				sb.WriteString("if err != nil {\nreturn nil, err\n}\n")
			} else {
				sb.WriteString("if err != nil {\nreturn\n}\n")
			}
		}
	}
	return sb.String()
}
