package generator

import (
	"strings"

	"github.com/reedom/convergen/pkg/generator/model"
)

func (g *Generator) ManipulatorToString(m *model.Manipulator, src, dst model.Var) string {
	var sb strings.Builder
	if m.RetError {
		sb.WriteString("err = ")
	}
	if m.Pkg != "" {
		sb.WriteString(m.Pkg)
		sb.WriteString(".")
	}
	sb.WriteString(m.Name)
	sb.WriteString("(")

	if dst.Pointer != m.IsDstPtr {
		if dst.Pointer {
			sb.WriteString("*")
		} else {
			sb.WriteString("&")
		}
	}
	sb.WriteString(dst.Name)
	sb.WriteString(", ")

	if src.Pointer != m.IsSrcPtr {
		if src.Pointer {
			sb.WriteString("*")
		} else {
			sb.WriteString("&")
		}
	}
	sb.WriteString(src.Name)
	sb.WriteString(")\n")

	if m.RetError {
		sb.WriteString("if err != nil {\nreturn\n}\n")
	}

	return sb.String()
}
