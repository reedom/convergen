package option

import (
	"go/token"
)

type LiteralSetter struct {
	dst     *IdentMatcher
	literal string
	pos     token.Pos
}

func NewLiteralSetter(dst, literal string, pos token.Pos) *LiteralSetter {
	dstM := NewIdentMatcher(dst)
	return &LiteralSetter{dst: dstM, literal: literal, pos: pos}
}

func (m *LiteralSetter) Match(dst string, exactCase bool) bool {
	return m.dst.Match(dst, exactCase)
}

func (m *LiteralSetter) Dst() *IdentMatcher {
	return m.dst
}

func (m *LiteralSetter) Literal() string {
	return m.literal
}

func (m *LiteralSetter) Pos() token.Pos {
	return m.pos
}
