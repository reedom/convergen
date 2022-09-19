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

// SkipField indicates that the field is skipped due to a :skip notation.
type SkipField struct {
}

func (s SkipField) String() string {
	return ""
}

func (s SkipField) ReturnsError() bool {
	return false
}

// NoMatchField indicates that the field is skipped while there was no matching fields or getters.
type NoMatchField struct {
}

func (s NoMatchField) String() string {
	return ""
}

func (s NoMatchField) ReturnsError() bool {
	return false
}

// SimpleField represents an RHS expression.
type SimpleField struct {
	Path  string
	Error bool
}

func (s SimpleField) String() string {
	return s.Path
}

func (s SimpleField) ReturnsError() bool {
	return s.Error
}

// Converter represents an RHS expression that uses a converter function.
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
