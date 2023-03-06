package option

import (
	"fmt"
	"go/token"
	"go/types"
)

// FieldConverter represents a converter for a single field of the source and destination types.
type FieldConverter struct {
	m         *NameMatcher // A name matcher that matches the name of the source and destination fields.
	converter string       // The name of the converter function.

	argType  types.Type // The type of the converter's argument.
	retType  types.Type // The type of the converter's return value.
	retError bool       // Indicates whether the converter returns an error.
}

// NewFieldConverter creates a new FieldConverter with the given parameters.
func NewFieldConverter(converter, src, dst string, pos token.Pos) *FieldConverter {
	return &FieldConverter{
		m:         NewNameMatcher(src, dst, pos),
		converter: converter,
	}
}

// Set sets the types of the FieldConverter's argument and return value, as well as whether the converter returns an error.
func (c *FieldConverter) Set(argType, retType types.Type, returnError bool) {
	c.argType = argType
	c.retType = retType
	c.retError = returnError
}

// Match returns true if the given source and destination field names match the FieldConverter's name matcher.
func (c *FieldConverter) Match(src, dst string) bool {
	return c.m.Match(src, dst, true)
}

// Converter returns the name of the converter function.
func (c *FieldConverter) Converter() string {
	return c.converter
}

// Src returns the FieldConverter's source identifier matcher.
func (c *FieldConverter) Src() *IdentMatcher {
	return c.m.src
}

// Dst returns the FieldConverter's destination identifier matcher.
func (c *FieldConverter) Dst() *IdentMatcher {
	return c.m.dst
}

// Pos returns the position of the FieldConverter.
func (c *FieldConverter) Pos() token.Pos {
	return c.m.pos
}

// ArgType returns the type of the converter's argument.
func (c *FieldConverter) ArgType() types.Type {
	return c.argType
}

// RetType returns the type of the converter's return value.
func (c *FieldConverter) RetType() types.Type {
	return c.retType
}

// RetError returns true if the converter returns an error.
func (c *FieldConverter) RetError() bool {
	return c.retError
}

// RHSExpr returns the right-hand side expression of the FieldConverter for a given argument.
func (c *FieldConverter) RHSExpr(arg string) string {
	return fmt.Sprintf("%v(%v)", c.converter, arg)
}
