package model_test

import (
	"go/types"
	"testing"

	"github.com/reedom/convergen/v8/pkg/builder/model"
	"github.com/stretchr/testify/assert"
)

func TestStructFieldNode(t *testing.T) {
	// Create a parent node.
	parent := model.NewRootNode("dst", types.NewPointer(types.NewNamed(
		types.NewTypeName(0, nil, "MyStruct", nil),
		nil,
		nil,
	)))
	// Create a field node.
	field := types.NewField(0, nil, "MyField", types.Typ[types.String], false)
	fieldNode := model.NewStructFieldNode(parent, field)

	assert.Equal(t, parent, fieldNode.Parent())
	assert.Equal(t, "MyField", fieldNode.ObjName())
	assert.False(t, fieldNode.ObjNullable())
	assert.Equal(t, types.Typ[types.String], fieldNode.ExprType())
	assert.False(t, fieldNode.ReturnsError())
	assert.Equal(t, "dst.MyField", fieldNode.AssignExpr())
	assert.Equal(t, "MyField", fieldNode.MatcherExpr())
	assert.Equal(t, "dst.MyField", fieldNode.NullCheckExpr())
}

func TestStructMethodNode(t *testing.T) {
	// Create a parent node.
	parent := model.NewRootNode("dst", types.NewPointer(types.NewNamed(
		types.NewTypeName(0, nil, "MyStruct", nil),
		nil,
		nil,
	)))
	// Create a method node.
	params := types.NewTuple(types.NewParam(0, nil, "", types.Typ[types.Int]))
	results := types.NewTuple(types.NewParam(0, nil, "", types.Typ[types.Int]))
	sig := types.NewSignatureType(nil, nil, nil, params, results, false)

	method := types.NewFunc(0, nil, "MyMethod", sig)
	methodNode := model.NewStructMethodNode(parent, method)

	assert.Equal(t, "MyMethod", methodNode.ObjName())
	assert.Equal(t, parent, methodNode.Parent())
	assert.Equal(t, types.Typ[types.Int], methodNode.ExprType())
	assert.Equal(t, "dst.MyMethod()", methodNode.AssignExpr())
	assert.Equal(t, "MyMethod()", methodNode.MatcherExpr())
	assert.Equal(t, "dst.MyMethod()", methodNode.NullCheckExpr())
	assert.False(t, methodNode.ReturnsError())
	assert.False(t, methodNode.ObjNullable())
}
