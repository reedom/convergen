package option

import (
	"go/token"
)

// LiteralSetter sets the literal value to the destination identified by dst.
type LiteralSetter struct {
	dst     *IdentMatcher // The IdentMatcher for the destination.
	literal string        // The literal value to be set.
	pos     token.Pos     // The position of the setter in the source code.
}

// NewLiteralSetter creates a new LiteralSetter instance.
func NewLiteralSetter(dst, literal string, pos token.Pos) *LiteralSetter {
	dstM := NewIdentMatcher(dst)
	return &LiteralSetter{dst: dstM, literal: literal, pos: pos}
}

// Match checks whether the destination matches the pattern of the IdentMatcher.
func (m *LiteralSetter) Match(dst string, exactCase bool) bool {
	return m.dst.Match(dst, exactCase)
}

// Dst returns the IdentMatcher instance for the destination.
func (m *LiteralSetter) Dst() *IdentMatcher {
	return m.dst
}

// Literal returns the literal value.
func (m *LiteralSetter) Literal() string {
	return m.literal
}

// Pos returns the position of the setter in the source code.
func (m *LiteralSetter) Pos() token.Pos {
	return m.pos
}
