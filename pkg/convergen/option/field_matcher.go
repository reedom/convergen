package option

import (
	"fmt"
	"strings"
)

type FieldMatcher struct {
	src *IdentMatcher
	dst *IdentMatcher
}

func NewFieldMatcher(src, dst string, exactCase bool) (*FieldMatcher, error) {
	srcM, err := NewIdentMatcher(src, exactCase)
	if err != nil {
		return nil, fmt.Errorf("src regexp is invalid")
	}

	var dstM *IdentMatcher
	if dst != "" {
		dstM, err = NewIdentMatcher(dst, exactCase)
		if err != nil {
			return nil, fmt.Errorf("dst regexp is invalid")
		}
	}

	return &FieldMatcher{src: srcM, dst: dstM}, nil
}

func (m *FieldMatcher) Match(src, dst string, exactCase bool) bool {
	if !m.src.Match(src, exactCase) {
		return false
	}
	if m.dst != nil {
		return m.dst.Match(dst, exactCase)
	}

	// If m.dst is nil, compare each name part of src and dst.
	srcName := extractName(src)
	dstName := extractName(dst)
	if !exactCase {
		srcName = strings.ToLower(srcName)
		dstName = strings.ToLower(dstName)
	}
	return srcName != "" && srcName == dstName
}

func extractName(fullName string) string {
	i := strings.LastIndex(fullName, ".")
	return fullName[i+1:]
}
