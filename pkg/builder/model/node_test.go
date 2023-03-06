package model_test

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/reedom/convergen/pkg/builder/model"
	"github.com/reedom/convergen/pkg/option"
	"github.com/stretchr/testify/assert"
)

func TestRootNode(t *testing.T) {
	typ := types.NewStruct(nil, nil)
	root := model.NewRootNode("dst", typ)

	assert.Nil(t, root.Parent())
	assert.Equal(t, "dst", root.ObjName())
	assert.False(t, root.ObjNullable())
	assert.Equal(t, typ, root.ExprType())
	assert.False(t, root.ReturnsError())
	assert.Equal(t, "dst", root.AssignExpr())
	assert.Equal(t, "", root.MatcherExpr())
	assert.Equal(t, "dst", root.NullCheckExpr())
}

func TestScalarNode(t *testing.T) {
	parent := model.NewRootNode("dst", types.NewPointer(types.NewStruct(nil, nil)))
	typ := types.NewNamed(types.NewTypeName(token.NoPos, nil, "Name", types.Typ[types.String]), nil, nil)
	node := model.NewScalarNode(parent, "Name", typ)

	assert.Equal(t, parent, node.Parent())
	assert.Equal(t, "Name", node.ObjName())
	assert.False(t, node.ObjNullable())
	assert.Equal(t, typ, node.ExprType())
	assert.False(t, node.ReturnsError())
	assert.Equal(t, "dst", node.AssignExpr())
	assert.Equal(t, "", node.MatcherExpr())
	assert.Equal(t, "dst", node.NullCheckExpr())
}

func TestConverterNode(t *testing.T) {
	parent := model.NewRootNode("dst", types.NewPointer(types.NewStruct(nil, nil)))
	typ := types.NewNamed(types.NewTypeName(token.NoPos, nil, "Name", types.Typ[types.String]), nil, nil)
	arg := model.NewScalarNode(parent, "Name", typ)

	fc := option.NewFieldConverter("myConverter", "srcField", "dstField", token.NoPos)
	argType := types.Typ[types.Int]
	retType := types.Typ[types.String]
	fc.Set(argType, retType, true)

	node := model.NewConverterNode(arg, fc)

	assert.Equal(t, parent, node.Parent())
	assert.Equal(t, arg.ObjName(), node.ObjName())
	assert.Equal(t, arg.ObjNullable(), node.ObjNullable())
	assert.Equal(t, retType, node.ExprType())
	assert.True(t, node.ReturnsError())
	assert.Equal(t, "myConverter(dst)", node.AssignExpr())
	assert.Equal(t, "", node.MatcherExpr())
	assert.Equal(t, "myConverter(dst)", node.NullCheckExpr())
}

func TestTypecastEntry(t *testing.T) {
	innerNode := model.NewScalarNode(nil, "score", types.Typ[types.Int])
	castType := types.Universe.Lookup("string").Type()

	// Test creating a new TypecastEntry with valid arguments.
	node, ok := model.NewTypecast(nil, nil, castType, innerNode)
	assert.True(t, ok)

	// Test that the object name and nullable properties are inherited from the inner node.
	assert.Equal(t, "score", node.ObjName())
	assert.Equal(t, false, node.ObjNullable())

	// Test the ExprType() method.
	assert.Equal(t, castType, node.ExprType())

	// Test the AssignExpr() method.
	assert.Equal(t, "string(score)", node.AssignExpr())

	// Test the MatcherExpr() method.
	assert.Equal(t, innerNode.MatcherExpr(), node.MatcherExpr())

	// Test the NullCheckExpr() method.
	assert.Equal(t, innerNode.NullCheckExpr(), node.NullCheckExpr())

	// Test the ReturnsError() method.
	assert.Equal(t, false, node.ReturnsError())

	// Test creating a new TypecastEntry with invalid arguments.
	_, ok = model.NewTypecast(nil, nil, nil, innerNode)
	assert.False(t, ok)
}

func TestStringerEntry(t *testing.T) {
	inner := model.NewScalarNode(nil, "Name", types.Universe.Lookup("int").Type())
	entry := model.NewStringer(inner)

	assert.Equal(t, "Name", entry.ObjName())
	assert.Nil(t, entry.Parent())
	assert.Equal(t, types.Universe.Lookup("string").Type(), entry.ExprType())
	assert.Equal(t, "Name.String()", entry.AssignExpr())
	assert.Equal(t, "", entry.MatcherExpr())
	assert.Equal(t, "Name", entry.NullCheckExpr())
	assert.False(t, entry.ReturnsError())
	assert.False(t, entry.ObjNullable())
}
