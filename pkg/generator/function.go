package generator

import (
	"fmt"
	"strings"

	"github.com/reedom/convergen/pkg/generator/model"
)

// FuncToString generates the string representation of a given Function.
// The generated string can be used to represent the Function as Go code.
// The function generates a doc comment (if any), the function signature,
// the variable declarations (if any), the assignment statements, and the return statement.
// The function uses ManipulatorToString to generate the string representation of manipulators.
func (g *Generator) FuncToString(f *model.Function) string {
	var sb strings.Builder

	// doc comment
	for i := range f.Comments {
		sb.WriteString(f.Comments[i])
		sb.WriteString("\n")
	}

	// "func"
	sb.WriteString("func ")

	if f.Receiver != "" {
		// "func (r *MyStruct)"
		sb.WriteString("(")
		sb.WriteString(f.Receiver)
		sb.WriteString(" ")
		sb.WriteString(f.Src.FullType())
		sb.WriteString(") ")
	}

	// "func (r *SrcModel) Name("
	if f.FuncCutPrefix == "" {
		sb.WriteString(f.Name)
	} else {
		// cut prefix, multi funcs with reciever CAN have same function name
		sb.WriteString(strings.TrimPrefix(f.Name, f.FuncCutPrefix))
	}
	sb.WriteString("(")

	if f.DstVarStyle == model.DstVarArg {
		// "func Name(dst *DstModel"
		sb.WriteString(f.Dst.Name)
		sb.WriteString(" *")
		sb.WriteString(f.Dst.PtrLessFullType())
		if f.Receiver == "" {
			// "func Name(dst *DstModel, "
			sb.WriteString(", ")
		}
	}

	if f.Receiver == "" {
		// "func Name(dst *DstModel, src *SrcModel"
		sb.WriteString(f.Src.Name)
		sb.WriteString(" ")
		sb.WriteString(f.Src.FullType())
	}
	// "func Name(dst *DstModel, src *SrcModel)"
	sb.WriteString(") ")

	checkSrc := func() {
		if f.Src.Pointer {
			sb.WriteString(fmt.Sprintf("if %s == nil {\n", f.Src.Name))
			sb.WriteString("return\n")
			sb.WriteString("}\n\n")
		}
	}

	initDst := func() {
		if f.Dst.Pointer {
			// "dst = &DstModel{}"
			sb.WriteString(f.Dst.Name)
			sb.WriteString(" = ")
			if f.Dst.Pointer {
				sb.WriteString("&")
			}
			sb.WriteString(f.Dst.PtrLessFullType())
			sb.WriteString("{}\n")
		}
	}

	if f.DstVarStyle == model.DstVarReturn {
		// "func Name(src *SrcModel) (dst *DstModel"
		sb.WriteString("(")
		sb.WriteString(f.Dst.Name)
		sb.WriteString(" ")
		sb.WriteString(f.Dst.FullType())
		if f.RetError {
			// "func Name(src *SrcModel) (dst *DstModel, err error"
			sb.WriteString(", err error")
		}

		// "func Name(src *SrcModel) (dst *DstModel) {"
		sb.WriteString(") {\n")
	} else {
		if f.RetError {
			// "func Name(dst *DstModel, src *SrcModel) (err error) {"
			sb.WriteString("(err error) {\n")
		} else {
			// "func Name(dst *DstModel, src *SrcModel) {"
			sb.WriteString("{\n")
		}

		initDst = func() {} // arg 不需要初始化 dst
	}

	if f.PreProcess != nil {
		if f.PreProcess.RetError { // 最高优先级, 比如nil要返回错误
			initDst()
			sb.WriteString(g.ManipulatorToString(f.PreProcess, f.Src, f.Dst))
			sb.WriteString("\n")
			checkSrc() // 检查 == nil
		} else {
			checkSrc()
			initDst()
			sb.WriteString(g.ManipulatorToString(f.PreProcess, f.Src, f.Dst))
		}
	} else {
		checkSrc()
		initDst()
	}

	for i := range f.Assignments {
		sb.WriteString(AssignmentToString(f, f.Assignments[i]))
	}
	if f.PostProcess != nil {
		sb.WriteString(g.ManipulatorToString(f.PostProcess, f.Src, f.Dst))
	}
	if f.RetError || f.DstVarStyle == model.DstVarReturn {
		sb.WriteString("\nreturn\n")
	}
	sb.WriteString("}\n\n")
	return sb.String()
}
