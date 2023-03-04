package option

import (
	"fmt"
	"regexp"
	"strings"
)

// Regular expression pattern to match text inside the parentheses
var reFromParen = regexp.MustCompile(`\(.*`)

// IdentMatcher is used to match field or method names in a struct.
type IdentMatcher struct {
	// pattern is the string pattern to match field or method names.
	pattern string
	// paths contains the separated field or method names.
	paths []string
}

// NewIdentMatcher creates a new IdentMatcher instance with a given pattern.
func NewIdentMatcher(pattern string) *IdentMatcher {
	return &IdentMatcher{
		pattern: pattern,
		paths:   strings.Split(pattern, "."),
	}
}

// Match returns true if the given ident string matches the IdentMatcher's pattern.
// If exactCase is false, it matches case-insensitively.
func (m *IdentMatcher) Match(ident string, exactCase bool) bool {
	if exactCase {
		return m.pattern == ident
	}
	return strings.EqualFold(m.pattern, ident)
}

// PartialMatch returns true if the given ident string partially matches the IdentMatcher's pattern.
// If exactCase is false, it matches case-insensitively.
func (m *IdentMatcher) PartialMatch(ident string, exactCase bool) bool {
	// Construct a partial pattern by joining all the path segments until the last one that
	//fits into the length of ident.
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

// ForGetter returns true if the path at the given index represents a method that returns a value.
func (m *IdentMatcher) ForGetter(at int) bool {
	return strings.HasSuffix(m.paths[at], "()")
}

// ExprAt returns an expression at the path index.
// If it is of a method, it should contain parens like "GetValue()".
func (m *IdentMatcher) ExprAt(at int) string {
	return m.paths[at]
}

// NameAt returns the name of the field or method at the path index.
// If it is a method, it removes the parentheses.
func (m *IdentMatcher) NameAt(at int) string {
	return reFromParen.ReplaceAllString(m.paths[at], "")
}

// PathLen returns the length of the path.
func (m *IdentMatcher) PathLen() int {
	return len(m.paths)
}

func (m *IdentMatcher) String() string {
	return fmt.Sprintf(`IdentMatcher{pattern: "%v"}`, m.pattern)
}
