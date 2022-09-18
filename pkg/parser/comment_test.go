package parser

import (
	"go/ast"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/reedom/convergen/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotations(t *testing.T) {
	t.Parallel()

	t.Run("common with validOpsIntf", func(t *testing.T) {
		t.Parallel()
		testCommonNotations(t, validOpsIntf)
	})
	t.Run("common with validOpsMethod", func(t *testing.T) {
		t.Parallel()
		testCommonNotations(t, validOpsMethod)
	})
	t.Run("methods", func(t *testing.T) {
		t.Parallel()
		testMethodNotations(t)
	})
}

func testCommonNotations(t *testing.T, validOpts map[string]struct{}) {
	cases := []struct {
		notation string
		expected func(*options)
	}{
		{
			notation: ":style arg",
			expected: func(opt *options) { opt.Style = model.DstVarArg },
		},
		{
			notation: ":style return",
			expected: func(opt *options) { opt.Style = model.DstVarReturn },
		},
		{
			notation: ":match tag",
			expected: func(opt *options) { opt.rule = model.MatchRuleTag },
		},
		{
			notation: ":match name",
			expected: func(opt *options) { opt.rule = model.MatchRuleName },
		},
		{
			notation: ":match none",
			expected: func(opt *options) { opt.rule = model.MatchRuleNone },
		},
		{
			notation: ":case:off",
			expected: func(opt *options) { opt.exactCase = false },
		},
		{
			notation: ":case",
			expected: func(opt *options) { opt.exactCase = true },
		},
		{
			notation: ":getter:off",
			expected: func(opt *options) { opt.getter = false },
		},
		{
			notation: ":getter",
			expected: func(opt *options) { opt.getter = true },
		},
		{
			notation: ":stringer",
			expected: func(opt *options) { opt.stringer = true },
		},
		{
			notation: ":stringer:off",
			expected: func(opt *options) { opt.stringer = false },
		},
		{
			notation: ":typecast",
			expected: func(opt *options) { opt.typecast = true },
		},
		{
			notation: ":typecast:off",
			expected: func(opt *options) { opt.typecast = false },
		},
	}

	p, err := NewParser("../../tests/fixtures/usecase/getter/setup.go")
	require.Nil(t, err)

	expected := newOptions()
	for _, tt := range cases {
		notations := []*ast.Comment{{Text: "// " + tt.notation}}
		actual := expected.copyForMethods()
		err := p.parseNotationInComments(notations, validOpts, &actual)
		assert.Nil(t, err)

		tt.expected(&expected)
		assertOptionsEquals(t, expected, actual, tt.notation)
	}
}

func testMethodNotations(t *testing.T) {
	cases := []struct {
		notation  string
		validator func(options) bool
	}{
		{
			notation:  ":rcv r",
			validator: func(opt options) bool { return opt.receiver == "r" },
		},
		{
			notation:  ":skip Name",
			validator: func(opt options) bool { return len(opt.skipFields) == 1 },
		},
		{
			notation:  ":map ID UserID",
			validator: func(opt options) bool { return len(opt.nameMapper) == 1 },
		},
	}

	p, err := NewParser("../../tests/fixtures/usecase/getter/setup.go")
	require.Nil(t, err)

	for _, tt := range cases {
		actual := newOptions()
		notations := []*ast.Comment{{Text: "// " + tt.notation}}
		err := p.parseNotationInComments(notations, validOpsMethod, &actual)
		assert.Nil(t, err)

		assert.True(t, tt.validator(actual))
	}
}

func assertOptionsEquals(t *testing.T, a, b options, msg string) {
	t.Helper()
	cmpOpts := []cmp.Option{
		cmp.AllowUnexported(options{}),
	}

	assert.True(
		t,
		cmp.Equal(a, b, cmpOpts...),
		cmp.Diff(a, b, cmpOpts...),
		msg,
	)
}
