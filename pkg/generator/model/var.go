package model

import (
	"strings"
)

type Var struct {
	Name    string
	PkgName string
	Type    string
	Pointer bool
}

// IsPkgExternal reports whether the Type is defined in an external package.
func (v Var) IsPkgExternal() bool {
	return v.PkgName != ""
}

// FullType creates a complete type expression string that can be used for var declaration.
// E.g. "*model.Pet" for "var x *model.Pet".
func (v Var) FullType() string {
	var sb strings.Builder
	if v.Pointer {
		sb.WriteString("*")
	}
	if v.PkgName != "" {
		sb.WriteString(v.PkgName)
		sb.WriteString(".")
	}
	sb.WriteString(v.Type)
	return sb.String()
}

// PtrLessFullType creates a complete type expression string but omits the pointer "*" symbol at the top.
func (v Var) PtrLessFullType() string {
	var sb strings.Builder
	if v.PkgName != "" {
		sb.WriteString(v.PkgName)
		sb.WriteString(".")
	}
	sb.WriteString(v.Type)
	return sb.String()
}
