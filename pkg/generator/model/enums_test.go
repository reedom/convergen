package model_test

import (
	"testing"

	"github.com/reedom/convergen/pkg/generator/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDstVarStyle(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		expected := "return"
		actual := model.DstVarReturn.String()
		assert.Equal(t, expected, actual)

		expected = "arg"
		actual = model.DstVarArg.String()
		assert.Equal(t, expected, actual)
	})

	t.Run("DstVarStyleValues", func(t *testing.T) {
		expected := []model.DstVarStyle{model.DstVarReturn, model.DstVarArg}
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
