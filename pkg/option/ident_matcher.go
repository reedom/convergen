package option

import (
	"fmt"
	"regexp"
	"strings"
)

type IdentMatcher struct {
	pattern   string
	re        *regexp.Regexp
	exactCase bool
}

// NewIdentMatcher creates a new IdentMatcher instance.
// If pattern is wrapped with slashes like "/……/", the instance will use regexp match. Otherwise it will be
// an exact word match.
// With the latter and if exactCase is false, it will apply a case-insensitive match. (For regexp patterns
// exactCase won't take effect.)
func NewIdentMatcher(pattern string, exactCase bool) (*IdentMatcher, error) {
	re, err := compileRegexp(pattern, exactCase)
	if err != nil {
		return nil, err
	}
	return &IdentMatcher{
		pattern:   pattern,
		re:        re,
		exactCase: exactCase,
	}, nil
}

func (m *IdentMatcher) Match(ident string, exactCase bool) bool {
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

func compileRegexp(pattern string, exactCase bool) (*regexp.Regexp, error) {
	var expr string
	if strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/") && 2 <= len(pattern) {
		expr = pattern[1 : len(pattern)-1]
	} else {
		expr = fmt.Sprintf("^%v$", regexp.QuoteMeta(pattern))
		if !exactCase {
			expr = strings.ToLower(expr)
		}
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, fmt.Errorf("invalid regexp")
	}
	return re, nil
}
