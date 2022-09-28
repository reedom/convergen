package model

import (
	"strings"
)

type Assignment interface {
	String() string
	RetError() bool
}

// SkipField indicates that the field is skipped due to a :skip notation.
type SkipField struct {
	LHS string
}

func (s SkipField) String() string {
	var sb strings.Builder
	sb.WriteString("// skip: ")
	sb.WriteString(s.LHS)
	sb.WriteString("\n")
	return sb.String()
}

func (s SkipField) RetError() bool {
	return false
}

// NoMatchField indicates that the field is skipped while there was no matching fields or getters.
type NoMatchField struct {
	LHS string
}

func (s NoMatchField) String() string {
	var sb strings.Builder
	sb.WriteString("// no match: ")
	sb.WriteString(s.LHS)
	sb.WriteString("\n")
	return sb.String()
}

func (s NoMatchField) RetError() bool {
	return false
}

// SimpleField represents an RHS expression.
type SimpleField struct {
	LHS   string
	RHS   string
	Error bool
}

func (s SimpleField) String() string {
	var sb strings.Builder
	sb.WriteString(s.LHS)
	if s.Error {
		sb.WriteString(", err")
	}
	sb.WriteString(" = ")
	sb.WriteString(s.RHS)
	sb.WriteString("\n")
	return sb.String()
}

func (s SimpleField) RetError() bool {
	return s.Error
}
