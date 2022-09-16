package model

import (
	"fmt"
	"strings"
)

type Assignment struct {
	LHS string
	RHS AssignmentRHS
}

type AssignmentRHS interface {
	String() string
	ReturnsError() bool
}

type SimpleField struct {
	Path string
}

func (s SimpleField) String() string {
	return s.Path
}

func (s SimpleField) ReturnsError() bool {
	return false
}

type Converter struct {
	Func  string
	Error bool
	Arg   string
}

func (c Converter) String() string {
	return fmt.Sprintf("%v(%v)", c.Func, c.Arg)
}

func (c Converter) ReturnsError() bool {
	return c.Error
}

type StructAssignment struct {
	Struct string
	Fields []Assignment
}

func (c StructAssignment) String() string {
	var sb strings.Builder
	sb.WriteString(c.Struct)
	sb.WriteString("{\n")
	sb.WriteString("}\n")
	return sb.String()
}

func (c StructAssignment) ReturnsError() bool {
	return false
}
