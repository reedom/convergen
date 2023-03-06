package option

import (
	"fmt"
	"regexp"
	"strings"
)

// PatternMatcher matches a string against a pattern.
type PatternMatcher struct {
	pattern   string         // The pattern to match against
	re        *regexp.Regexp // The compiled regular expression from pattern
	exactCase bool           // Whether to match case-sensitively or not
}

// NewPatternMatcher creates a new PatternMatcher instance.
func NewPatternMatcher(pattern string, exactCase bool) (*PatternMatcher, error) {
	re, err := compileRegexp(pattern, exactCase)
	if err != nil {
		return nil, err
	}
	return &PatternMatcher{
		pattern:   pattern,
		re:        re,
		exactCase: exactCase,
	}, nil
}

// Match checks whether the input string matches the pattern.
func (m *PatternMatcher) Match(ident string, exactCase bool) bool {
	if m.exactCase != exactCase {
		m.re, _ = compileRegexp(m.pattern, exactCase)
		m.exactCase = exactCase
	}

	s := ident
	if !exactCase {
		s = strings.ToLower(s)
	}
	return m.re.MatchString(s)
}

// compileRegexp compiles the given pattern into a regular expression.
// If exactCase is false, the pattern is case-insensitive.
func compileRegexp(pattern string, exactCase bool) (*regexp.Regexp, error) {
	var expr string
	if strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/") && 2 <= len(pattern) {
		expr = pattern[1 : len(pattern)-1]
	} else {
		expr = fmt.Sprintf("^%v$", regexp.QuoteMeta(pattern))
	}
	if !exactCase {
		expr = strings.ToLower(expr)
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, fmt.Errorf("invalid regexp")
	}
	return re, nil
}
