package option_test

import (
	"testing"

	"github.com/reedom/convergen/v8/pkg/option"
)

func TestIdentMatcher_Match(t *testing.T) {
	testCases := []struct {
		name       string
		pattern    string
		ident      string
		exactCase  bool
		wantResult bool
	}{
		{"Exact case match", "foo", "foo", true, true},
		{"Exact case mismatch", "foo", "bar", true, false},
		{"Case-insensitive match", "foo", "Foo", false, true},
		{"Case-insensitive mismatch", "foo", "Bar", false, false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := option.NewIdentMatcher(tc.pattern)
			got := m.Match(tc.ident, tc.exactCase)
			if got != tc.wantResult {
				t.Errorf("m.Match(%q, %v) = %v; want %v", tc.ident, tc.exactCase, got, tc.wantResult)
			}
		})
	}
}

func TestIdentMatcher_PartialMatch(t *testing.T) {
	testCases := []struct {
		name       string
		pattern    string
		ident      string
		exactCase  bool
		wantResult bool
	}{
		{"Exact case match", "foo.bar", "foo.bar", true, true},
		{"Exact case mismatch", "foo.bar", "foo.baz", true, false},
		{"Case-insensitive match", "foo.bar", "Foo.Bar", false, true},
		{"Case-insensitive mismatch", "foo.bar", "Foo.Baz", false, false},
		{"Partial match", "foo.bar.baz", "foo.bar.baz.qux", true, true},
		{"Partial mismatch", "foo.bar.baz", "foo.bar.qux.baz", true, false},
		{"Partial match case-insensitive", "foo.bar.baz", "foo.Bar.baz.Qux", false, true},
		{"Partial mismatch case-insensitive", "foo.bar.baz", "foo.bar.Qux.baz", false, false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := option.NewIdentMatcher(tc.pattern)
			got := m.PartialMatch(tc.ident, tc.exactCase)
			if got != tc.wantResult {
				t.Errorf("m.PartialMatch(%q, %v) = %v; want %v", tc.ident, tc.exactCase, got, tc.wantResult)
			}
		})
	}
}
