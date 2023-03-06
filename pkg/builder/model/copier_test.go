package model_test

import (
	"go/types"
	"testing"

	"github.com/reedom/convergen/pkg/builder/model"
)

func TestMarkHandle(t *testing.T) {
	// Create a Copier with some types
	c := model.NewCopier("myCopier", types.NewPointer(types.Typ[types.String]), types.NewPointer(types.Typ[types.Int]))

	// Test with matching types
	lhs := types.NewPointer(types.Typ[types.String])
	rhs := types.NewPointer(types.Typ[types.Int])
	if !c.MarkHandle(lhs, rhs) {
		t.Error("Expected to handle types")
	}
	if c.HandleCount != 2 {
		t.Errorf("Expected HandleCount to be 2, but got %d", c.HandleCount)
	}

	// Test with non-matching types
	lhs = types.NewPointer(types.Typ[types.Bool])
	rhs = types.NewPointer(types.Typ[types.Int])
	if c.MarkHandle(lhs, rhs) {
		t.Error("Expected to not handle types")
	}
	if c.HandleCount != 2 {
		t.Errorf("Expected HandleCount to still be 2, but got %d", c.HandleCount)
	}
}
