package option

import (
	"fmt"
	"go/token"
	"strings"
)

type NameMatcher struct {
	src *IdentMatcher
	dst *IdentMatcher
	pos token.Pos
}

func NewNameMatcher(src, dst string, exactCase bool, pos token.Pos) (*NameMatcher, error) {
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

	return &NameMatcher{src: srcM, dst: dstM, pos: pos}, nil
}

func (m *NameMatcher) Match(src, dst string, exactCase bool) bool {
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

func (m *NameMatcher) Src() *IdentMatcher {
	return m.src
}

func (m *NameMatcher) Dst() *IdentMatcher {
	return m.dst
}

func (m *NameMatcher) Pos() token.Pos {
	return m.pos
}

func extractName(fullName string) string {
	i := strings.LastIndex(fullName, ".")
	return fullName[i+1:]
}
