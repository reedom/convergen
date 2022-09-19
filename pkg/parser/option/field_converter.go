package option

import (
	"fmt"
	"go/token"
	"go/types"
)

type FieldConverter struct {
	m         *NameMatcher
	converter string

	argType      types.Type
	retType      types.Type
	returnsError bool
}

func NewFieldConverter(converter, src, dst string, argType, retType types.Type, returnError bool, pos token.Pos) (*FieldConverter, error) {
	m, err := NewNameMatcher(src, dst, true, pos)
	if err != nil {
		return nil, err
	}

	return &FieldConverter{
		m:            m,
		converter:    converter,
		argType:      argType,
		retType:      retType,
		returnsError: returnError,
	}, nil
}

func (c *FieldConverter) Match(src, dst string) bool {
	return c.m.Match(src, dst, true)
}

func (c *FieldConverter) Converter() string {
	return c.converter
}

func (c *FieldConverter) Src() *IdentMatcher {
	return c.m.src
}

func (c *FieldConverter) Dst() *IdentMatcher {
	return c.m.dst
}

func (c *FieldConverter) Pos() token.Pos {
	return c.m.pos
}

func (c *FieldConverter) ArgType() types.Type {
	return c.argType
}

func (c *FieldConverter) RetType() types.Type {
	return c.retType
}

func (c *FieldConverter) ReturnsError() bool {
	return c.returnsError
}

func (c *FieldConverter) RHSExpr(arg string) string {
	return fmt.Sprintf("%v(%v)", c.converter, arg)
}
