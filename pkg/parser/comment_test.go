package parser

import (
	"go/ast"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/reedom/convergen/pkg/model"
	"github.com/reedom/convergen/pkg/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotations(t *testing.T) {
	t.Parallel()

	t.Run("common with ValidOpsIntf", func(t *testing.T) {
		t.Parallel()
		testCommonNotations(t, option.ValidOpsIntf)
	})
	t.Run("common with ValidOpsMethod", func(t *testing.T) {
		t.Parallel()
		testCommonNotations(t, option.ValidOpsMethod)
	})
	t.Run("methods", func(t *testing.T) {
		t.Parallel()
		testMethodNotations(t)
	})
}

func testCommonNotations(t *testing.T, validOpts map[string]struct{}) {
	cases := []struct {
		notation string
		expected func(options *option.Options)
	}{
		{
			notation: ":style arg",
			expected: func(opt *option.Options) { opt.Style = model.DstVarArg },
		},
		{
			notation: ":style return",
			expected: func(opt *option.Options) { opt.Style = model.DstVarReturn },
		},
		{
			notation: ":match tag",
			expected: func(opt *option.Options) { opt.Rule = model.MatchRuleTag },
		},
		{
			notation: ":match name",
			expected: func(opt *option.Options) { opt.Rule = model.MatchRuleName },
		},
		{
			notation: ":match none",
			expected: func(opt *option.Options) { opt.Rule = model.MatchRuleNone },
		},
		{
			notation: ":case:off",
			expected: func(opt *option.Options) { opt.ExactCase = false },
		},
		{
			notation: ":case",
			expected: func(opt *option.Options) { opt.ExactCase = true },
		},
		{
			notation: ":getter:off",
			expected: func(opt *option.Options) { opt.Getter = false },
		},
		{
			notation: ":getter",
			expected: func(opt *option.Options) { opt.Getter = true },
		},
		{
			notation: ":stringer",
			expected: func(opt *option.Options) { opt.Stringer = true },
		},
		{
			notation: ":stringer:off",
			expected: func(opt *option.Options) { opt.Stringer = false },
		},
		{
			notation: ":typecast",
			expected: func(opt *option.Options) { opt.Typecast = true },
		},
		{
			notation: ":typecast:off",
			expected: func(opt *option.Options) { opt.Typecast = false },
		},
	}

	p, err := NewParser("../../tests/fixtures/usecase/getter/setup.go")
	require.Nil(t, err)

	expected := option.NewOptions()
	for _, tt := range cases {
		notations := []*ast.Comment{{Text: "// " + tt.notation}}
		actual := expected
		err := p.parseNotationInComments(notations, validOpts, &actual)
		assert.Nil(t, err)

		tt.expected(&expected)
		assertOptionsEquals(t, expected, actual, tt.notation)
	}
}

func testMethodNotations(t *testing.T) {
	cases := []struct {
		notation  string
		validator func(options option.Options) bool
	}{
		{
			notation:  ":rcv r",
			validator: func(opt option.Options) bool { return opt.Receiver == "r" },
		},
		{
			notation:  ":skip Name",
			validator: func(opt option.Options) bool { return len(opt.SkipFields) == 1 },
		},
		{
			notation:  ":map ID UserID",
			validator: func(opt option.Options) bool { return len(opt.NameMapper) == 1 },
		},
	}

	p, err := NewParser("../../tests/fixtures/usecase/getter/setup.go")
	require.Nil(t, err)

	for _, tt := range cases {
		actual := option.NewOptions()
		notations := []*ast.Comment{{Text: "// " + tt.notation}}
		err := p.parseNotationInComments(notations, option.ValidOpsMethod, &actual)
		assert.Nil(t, err)

		assert.True(t, tt.validator(actual))
	}
}

func assertOptionsEquals(t *testing.T, a, b option.Options, msg string) {
	t.Helper()
	cmpOpts := []cmp.Option{
		cmp.AllowUnexported(option.Options{}),
	}

	assert.True(
		t,
		cmp.Equal(a, b, cmpOpts...),
		cmp.Diff(a, b, cmpOpts...),
		msg,
	)
}
