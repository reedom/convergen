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

func (v Var) IsPkgExternal() bool {
	return v.PkgName != ""
}

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

func (v Var) PtrLessFullType() string {
	var sb strings.Builder
	if v.PkgName != "" {
		sb.WriteString(v.PkgName)
		sb.WriteString(".")
	}
	sb.WriteString(v.Type)
	return sb.String()
}
