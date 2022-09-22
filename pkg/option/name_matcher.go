package option

import (
	"go/token"
)

type NameMatcher struct {
	src *IdentMatcher
	dst *IdentMatcher
	pos token.Pos
}

func NewNameMatcher(src, dst string, pos token.Pos) *NameMatcher {
	srcM := NewIdentMatcher(src)

	var dstM *IdentMatcher
	if dst == "" {
		dstM = srcM
	} else {
		dstM = NewIdentMatcher(dst)
	}

	return &NameMatcher{src: srcM, dst: dstM, pos: pos}
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
