package generator

import (
	"strings"

	"github.com/reedom/convergen/pkg/model"
)

func FuncToString(f *model.Function) string {
	var sb strings.Builder
	for i := range f.Comments {
		sb.WriteString(f.Comments[i])
		sb.WriteString("\n")
	}
	sb.WriteString("func ")

	if f.Receiver != "" {
		sb.WriteString("(")
		sb.WriteString(f.Receiver)
		sb.WriteString(" ")
		sb.WriteString(f.Src.FullType())
		sb.WriteString(") ")
	}

	sb.WriteString(f.Name)
	sb.WriteString("(")

	if f.DstVarStyle == model.DstVarArg {
		sb.WriteString(f.Dst.Name)
		sb.WriteString(" *")
		sb.WriteString(f.Dst.PtrLessFullType())
		if f.Receiver == "" {
			sb.WriteString(", ")
		}
	}

	if f.Receiver == "" {
		sb.WriteString(f.Src.Name)
		sb.WriteString(" ")
		sb.WriteString(f.Src.FullType())
	}
	sb.WriteString(") ")

	if f.DstVarStyle == model.DstVarReturn {
		sb.WriteString("(")
		sb.WriteString(f.Dst.Name)
		sb.WriteString(" ")
		sb.WriteString(f.Dst.FullType())
		if f.ReturnsError {
			sb.WriteString(", err error")
		}
		sb.WriteString(") {\n")
		sb.WriteString(f.Dst.Name)
		sb.WriteString(" = ")
		if f.Dst.Pointer {
			sb.WriteString("&")
		}
		sb.WriteString(f.Dst.PtrLessFullType())
		sb.WriteString("{}\n")
	} else {
		if f.ReturnsError {
			sb.WriteString("(err error) {\n")
		} else {
			sb.WriteString("{\n")
		}
	}

	for i := range f.Assignments {
		sb.WriteString(AssignmentToString(f.Assignments[i]))
	}
	sb.WriteString("}\n")
	return sb.String()
}
