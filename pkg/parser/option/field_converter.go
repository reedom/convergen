package option

import (
	"go/token"
)

type FieldConverter struct {
	m         *NameMatcher
	converter string
}

func NewFieldConverter(converter, src, dst string, exactCase bool, pos token.Pos) (*FieldConverter, error) {
	m, err := NewNameMatcher(src, dst, exactCase, pos)
	if err != nil {
		return nil, err
	}

	return &FieldConverter{
		m:         m,
		converter: converter,
	}, nil
}

func (c *FieldConverter) Match(src, dst string, exactCase bool) bool {
	return c.m.Match(src, dst, exactCase)
}

func (c *FieldConverter) Converter() string {
	return c.converter
}

func (m *FieldConverter) Src() *IdentMatcher {
	return m.m.src
}

func (m *FieldConverter) Dst() *IdentMatcher {
	return m.m.dst
}

func (c *FieldConverter) Pos() token.Pos {
	return c.m.pos
}
