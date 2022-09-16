package generator

import (
	"strings"

	"github.com/reedom/convergen/pkg/model"
)

func AssignmentToString(a *model.Assignment) string {
	var sb strings.Builder

	sb.WriteString(a.LHS)
	if a.RHS.ReturnsError() {
		sb.WriteString("err")
	}
	sb.WriteString(" = ")
	sb.WriteString(a.RHS.String())
	sb.WriteString("\n")
	if a.RHS.ReturnsError() {
		sb.WriteString("if err != nil {\nreturn err\n}\n")
	}
	return sb.String()
}
