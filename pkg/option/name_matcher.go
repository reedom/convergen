package option

import (
	"go/token"
)

// NameMatcher matches source and destination names.
type NameMatcher struct {
	src *IdentMatcher // The matcher for the source identifier.
	dst *IdentMatcher // The matcher for the destination identifier.
	pos token.Pos     // The position of the name matcher in the file.
}

// NewNameMatcher creates a new NameMatcher instance.
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

// Match matches source and destination with the given names.
func (m *NameMatcher) Match(src, dst string, exactCase bool) bool {
	return m.src.Match(src, exactCase) && m.dst.Match(dst, exactCase)
}

// Src returns the source IdentMatcher.
func (m *NameMatcher) Src() *IdentMatcher {
	return m.src
}

// Dst returns the destination IdentMatcher.
func (m *NameMatcher) Dst() *IdentMatcher {
	return m.dst
}

// Pos returns the token.Pos of NameMatcher.
func (m *NameMatcher) Pos() token.Pos {
	return m.pos
}
