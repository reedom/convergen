package option

import (
	"fmt"
	"go/token"
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
	if dst == "" {
		dstM = srcM
	} else {
		dstM, err = NewIdentMatcher(dst, exactCase)
		if err != nil {
			return nil, fmt.Errorf("dst regexp is invalid")
		}
	}

	return &NameMatcher{src: srcM, dst: dstM, pos: pos}, nil
}

func (m *NameMatcher) Match(src, dst string, exactCase bool) bool {
	return m.src.Match(src, exactCase) && m.dst.Match(dst, exactCase)
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
