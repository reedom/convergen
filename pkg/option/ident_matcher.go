package option

import (
	"fmt"
	"regexp"
	"strings"
)

var reFromParen = regexp.MustCompile(`\(.*`)

type IdentMatcher struct {
	pattern string
	paths   []string
}

// NewIdentMatcher creates a new IdentMatcher instance.
func NewIdentMatcher(pattern string) *IdentMatcher {
	return &IdentMatcher{
		pattern: pattern,
		paths:   strings.Split(pattern, "."),
	}
}

func (m *IdentMatcher) Match(ident string, exactCase bool) bool {
	if exactCase {
		return m.pattern == ident
	}
	return strings.ToLower(m.pattern) == strings.ToLower(ident)
}

func (m *IdentMatcher) PartialMatch(ident string, exactCase bool) bool {
	partial := m.paths[0]
	for i := 1; i < len(m.paths) && len(partial)+len(m.paths[i]) < len(ident); i++ {
		partial += "."
		partial += m.paths[i]
	}

	if exactCase {
		return strings.HasPrefix(ident, partial)
	}

	return strings.HasPrefix(strings.ToLower(ident), strings.ToLower(partial))
}

func (m *IdentMatcher) ForGetter(at int) bool {
	return strings.HasSuffix(m.paths[at], "()")
}

// ExprAt returns an expression at the path index.
// If it is of a method, it should contain parens like "GetValue()".
func (m *IdentMatcher) ExprAt(at int) string {
	return m.paths[at]
}

// NameAt returns a name of field or method at the path index.
func (m *IdentMatcher) NameAt(at int) string {
	return reFromParen.ReplaceAllString(m.paths[at], "")
}

func (m *IdentMatcher) PathLen() int {
	return len(m.paths)
}

func (m *IdentMatcher) String() string {
	return fmt.Sprintf(`IdentMatcher{pattern: "%v"}`, m.pattern)
}
