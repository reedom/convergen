package generator

import (
	"strings"

	"github.com/reedom/convergen/pkg/generator/model"
)

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
