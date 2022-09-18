package option

type FieldConverter struct {
	m         *NameMatcher
	converter string
	hasError  bool
}

func NewFieldConverter(src, dst string, exactCase bool, converter string, hasError bool) (*FieldConverter, error) {
	m, err := NewNameMatcher(src, dst, exactCase)
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
