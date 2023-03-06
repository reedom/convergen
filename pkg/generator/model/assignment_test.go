package model_test

import (
	"testing"

	"github.com/reedom/convergen/pkg/generator/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkipField(t *testing.T) {
	t.Parallel()
	sf := model.SkipField{
		LHS: "foo",
	}

	t.Run("String", func(t *testing.T) {
		expected := "// skip: foo\n"
		actual := sf.String()
		assert.Equal(t, expected, actual)
	})

	t.Run("RetError", func(t *testing.T) {
		actual := sf.RetError()
		require.False(t, actual)
	})
}

func TestNoMatchField(t *testing.T) {
	t.Parallel()
	nmf := model.NoMatchField{
		LHS: "foo",
	}

	t.Run("String", func(t *testing.T) {
		expected := "// no match: foo\n"
		actual := nmf.String()
		assert.Equal(t, expected, actual)
	})

	t.Run("RetError", func(t *testing.T) {
		actual := nmf.RetError()
		require.False(t, actual)
	})
}

func TestSimpleField(t *testing.T) {
	t.Parallel()
	sf := model.SimpleField{
		LHS:   "foo",
		RHS:   "bar",
		Error: true,
	}

	t.Run("String", func(t *testing.T) {
		expected := "foo, err = bar\n"
		actual := sf.String()
		assert.Equal(t, expected, actual)
	})

	t.Run("RetError", func(t *testing.T) {
		actual := sf.RetError()
		require.True(t, actual)
	})
}

func TestNestStruct(t *testing.T) {
	t.Parallel()
	ns := model.NestStruct{
		InitExpr:      "initExpr",
		NullCheckExpr: "nullCheckExpr",
		Contents: []model.Assignment{
			&model.SimpleField{LHS: "foo", RHS: "bar", Error: false},
			&model.SkipField{LHS: "baz"},
			&model.NoMatchField{LHS: "qux"},
		},
	}

	t.Run("String", func(t *testing.T) {
		expected := `if nullCheckExpr != nil {
initExpr
foo = bar
// skip: baz
// no match: qux
}
`
		actual := ns.String()
		assert.Equal(t, expected, actual)
	})

	t.Run("RetError", func(t *testing.T) {
		actual := ns.RetError()
		require.False(t, actual)
	})
}

func TestSliceAssignment(t *testing.T) {
	t.Parallel()
	sa := model.SliceAssignment{
		LHS: "foo",
		RHS: "bar",
		Typ: "[]int",
	}

	t.Run("String", func(t *testing.T) {
		expected := `if bar != nil {
foo = make([]int, len(bar))
copy(foo, bar)
}
`
		actual := sa.String()
		assert.Equal(t, expected, actual)
	})

	t.Run("RetError", func(t *testing.T) {
		actual := sa.RetError()
		require.False(t, actual)
	})
}

func TestSliceTypecastAssignment(t *testing.T) {
	t.Parallel()
	sta := model.SliceTypecastAssignment{
		LHS:  "foo",
		RHS:  "bar",
		Typ:  "[]string",
		Cast: "string",
	}

	t.Run("String", func(t *testing.T) {
		expected := `if bar != nil {
foo = make([]string, len(bar))
for i, e := range bar{
foo[i] = string(e)
}
}
`
		actual := sta.String()
		assert.Equal(t, expected, actual)
	})

	t.Run("RetError", func(t *testing.T) {
		actual := sta.RetError()
		require.False(t, actual)
	})
}
