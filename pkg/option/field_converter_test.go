package option_test

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/reedom/convergen/v8/pkg/option"
	"github.com/stretchr/testify/assert"
)

func TestFieldConverter(t *testing.T) {
	// Create a new FieldConverter.
	fc := option.NewFieldConverter("myConverter", "srcField", "dstField", token.NoPos)

	// Assert that the converter and identifiers were set correctly.
	assert.Equal(t, "myConverter", fc.Converter())
	assert.Equal(t, "srcField", fc.Src().ExprAt(0))
	assert.Equal(t, "dstField", fc.Dst().ExprAt(0))
	assert.Equal(t, token.NoPos, fc.Pos())

	// Set the types and assert that they were set correctly.
	argType := types.Typ[types.Int]
	retType := types.Typ[types.String]
	fc.Set(argType, retType, true)
	assert.Equal(t, argType, fc.ArgType())
	assert.Equal(t, retType, fc.RetType())
	assert.True(t, fc.RetError())

	// Test the Match function.
	assert.True(t, fc.Match("srcField", "dstField"))
	assert.False(t, fc.Match("srcField2", "dstField2"))

	// Test the RHSExpr function.
	assert.Equal(t, "myConverter(42)", fc.RHSExpr("42"))
}
