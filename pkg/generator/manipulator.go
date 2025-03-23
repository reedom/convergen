package generator

import (
	"strings"

	"github.com/reedom/convergen/pkg/generator/model"
)

// ManipulatorToString returns a string representation of the given Manipulator.
// It generates a function call that performs the manipulation and returns the result as a string.
// Parameters:
// - m: the Manipulator to be converted into a string representation.
// - src: the source Var that corresponds to the Manipulator's first argument.
// - dst: the destination Var that corresponds to the Manipulator's second argument.
// Returns:
// - a string that represents the function call to the Manipulator.
func (g *Generator) ManipulatorToString(m *model.Manipulator, src, dst model.Var, args []model.Var) string {
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

	if m.HasAdditionalArgs {
		for _, arg := range args {
			sb.WriteString(", ")
			sb.WriteString(arg.Name)
		}
	}
	sb.WriteString(")\n")

	if m.RetError {
		sb.WriteString("if err != nil {\nreturn\n}\n")
	}

	return sb.String()
}
