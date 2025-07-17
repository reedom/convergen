package generator

import (
	"strings"

	"github.com/reedom/convergen/v8/pkg/generator/model"
)

// AssignmentToString returns the string representation of the assignment.
func AssignmentToString(f *model.Function, a model.Assignment) string {
	var sb strings.Builder
	sb.WriteString(a.String())
	if a.RetError() {
		if f.DstVarStyle == model.DstVarReturn && f.Dst.Pointer {
			sb.WriteString("if err != nil {\nreturn nil, err\n}\n")
		} else {
			sb.WriteString("if err != nil {\nreturn\n}\n")
		}
	}
	return sb.String()
}
