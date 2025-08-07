package model_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/reedom/convergen/v8/pkg/generator/model"
)

func TestDstVarStyle(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		expected := "return"
		actual := model.DstVarReturn.String()
		assert.Equal(t, expected, actual)

		expected = "arg"
		actual = model.DstVarArg.String()
		assert.Equal(t, expected, actual)

		expected = "struct_literal"
		actual = model.DstVarStructLiteral.String()
		assert.Equal(t, expected, actual)
	})

	t.Run("IsStructLiteral", func(t *testing.T) {
		assert.False(t, model.DstVarReturn.IsStructLiteral())
		assert.False(t, model.DstVarArg.IsStructLiteral())
		assert.True(t, model.DstVarStructLiteral.IsStructLiteral())
	})

	t.Run("DstVarStyleValues", func(t *testing.T) {
		expected := []model.DstVarStyle{model.DstVarReturn, model.DstVarArg, model.DstVarStructLiteral}
		actual := model.DstVarStyleValues
		assert.Equal(t, expected, actual)
	})

	t.Run("NewDstVarStyleFromValue", func(t *testing.T) {
		expected := model.DstVarReturn
		actual, ok := model.NewDstVarStyleFromValue("return")
		require.True(t, ok)
		assert.Equal(t, expected, actual)

		expected = model.DstVarArg
		actual, ok = model.NewDstVarStyleFromValue("arg")
		require.True(t, ok)
		assert.Equal(t, expected, actual)

		expected = model.DstVarStructLiteral
		actual, ok = model.NewDstVarStyleFromValue("struct_literal")
		require.True(t, ok)
		assert.Equal(t, expected, actual)

		_, ok = model.NewDstVarStyleFromValue("invalid")
		require.False(t, ok)
	})
}

func TestMatchRuleValues(t *testing.T) {
	t.Run("MatchRuleValues", func(t *testing.T) {
		assert.ElementsMatch(t, []model.MatchRule{
			model.MatchRuleName,
			model.MatchRuleTag,
			model.MatchRuleNone,
		}, model.MatchRuleValues)
	})

	t.Run("NewMatchRuleFromValue", func(t *testing.T) {
		rule, ok := model.NewMatchRuleFromValue("name")
		assert.True(t, ok)
		assert.Equal(t, model.MatchRuleName, rule)

		rule, ok = model.NewMatchRuleFromValue("tag")
		assert.True(t, ok)
		assert.Equal(t, model.MatchRuleTag, rule)

		rule, ok = model.NewMatchRuleFromValue("none")
		assert.True(t, ok)
		assert.Equal(t, model.MatchRuleNone, rule)

		rule, ok = model.NewMatchRuleFromValue("invalid")
		assert.False(t, ok)
		assert.Equal(t, model.MatchRule(""), rule)
	})
}

func TestOutputStyle(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		expected := "auto"
		actual := model.OutputStyleAuto.String()
		assert.Equal(t, expected, actual)

		expected = "struct_literal"
		actual = model.OutputStyleStructLiteral.String()
		assert.Equal(t, expected, actual)

		expected = "traditional"
		actual = model.OutputStyleTraditional.String()
		assert.Equal(t, expected, actual)
	})

	t.Run("OutputStyleValues", func(t *testing.T) {
		expected := []model.OutputStyle{model.OutputStyleAuto, model.OutputStyleStructLiteral, model.OutputStyleTraditional}
		actual := model.OutputStyleValues
		assert.Equal(t, expected, actual)
	})

	t.Run("NewOutputStyleFromValue", func(t *testing.T) {
		expected := model.OutputStyleAuto
		actual, ok := model.NewOutputStyleFromValue("auto")
		require.True(t, ok)
		assert.Equal(t, expected, actual)

		expected = model.OutputStyleStructLiteral
		actual, ok = model.NewOutputStyleFromValue("struct_literal")
		require.True(t, ok)
		assert.Equal(t, expected, actual)

		expected = model.OutputStyleTraditional
		actual, ok = model.NewOutputStyleFromValue("traditional")
		require.True(t, ok)
		assert.Equal(t, expected, actual)

		_, ok = model.NewOutputStyleFromValue("invalid")
		require.False(t, ok)
	})
}
