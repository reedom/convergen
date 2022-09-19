package option

import (
	"go/token"
)

type FieldConverter struct {
	m         *NameMatcher
	converter string
	hasError  bool
}

func NewFieldConverter(src, dst string, exactCase bool, converter string, hasError bool, pos token.Pos) (*FieldConverter, error) {
	m, err := NewNameMatcher(src, dst, exactCase, pos)
	if err != nil {
		return nil, err
	}

	return &FieldConverter{
		m:         m,
		converter: converter,
		hasError:  hasError,
	}, nil
}

func (c *FieldConverter) Match(src, dst string, exactCase bool) bool {
	return c.m.Match(src, dst, exactCase)
}

func (c *FieldConverter) Converter() string {
	return c.converter
}

func (c *FieldConverter) HasError() bool {
	return c.hasError
}

func (c *FieldConverter) Pos() token.Pos {
	return c.m.pos
}
