package model

import (
	"fmt"
	"go/types"

	"github.com/reedom/convergen/pkg/util"
)

type StructFieldNode struct {
	// parent refers the parent struct type entry.
	parent Node
	// field refers a leaf Field.
	field *types.Var
}

func NewStructFieldNode(container Node, field *types.Var) StructFieldNode {
	return StructFieldNode{
		parent: container,
		field:  field,
	}
}

func (n StructFieldNode) Parent() Node {
	return n.parent
}

func (n StructFieldNode) Field() *types.Var {
	return n.field
}

func (n StructFieldNode) ObjName() string {
	return n.field.Name()
}

func (n StructFieldNode) ObjNullable() bool {
	return util.IsPtr(n.field.Type())
}

func (n StructFieldNode) ExprType() types.Type {
	return n.field.Type()
}

func (n StructFieldNode) ReturnsError() bool {
	return false
}

func (n StructFieldNode) AssignExpr() string {
	return fmt.Sprintf("%v.%v", n.parent.AssignExpr(), n.field.Name())
}

func (n StructFieldNode) MatcherExpr() string {
	parentExpr := n.parent.MatcherExpr()
	if parentExpr == "" {
		return n.field.Name()
	}
	return fmt.Sprintf("%v.%v", parentExpr, n.field.Name())
}

func (n StructFieldNode) NullCheckExpr() string {
	return fmt.Sprintf("%v.%v", n.parent.AssignExpr(), n.field.Name())
}

type StructMethodNode struct {
	// container refers the container struct type entry.
	container Node
	// method refers a leaf Field that type is a func.
	method *types.Func
}

func NewStructMethodNode(container Node, method *types.Func) StructMethodNode {
	return StructMethodNode{
		container: container,
		method:    method,
	}
}

func (n StructMethodNode) ObjName() string {
	return n.method.Name()
}

func (n StructMethodNode) Parent() Node {
	return n.container
}

func (n StructMethodNode) ExprType() types.Type {
	sig := n.method.Type().(*types.Signature)
	return sig.Results().At(0).Type()
}

func (n StructMethodNode) AssignExpr() string {
	return fmt.Sprintf("%v.%v()", n.container.AssignExpr(), n.method.Name())
}

func (n StructMethodNode) MatcherExpr() string {
	parentExpr := n.container.MatcherExpr()
	if parentExpr == "" {
		return n.method.Name() + "()"
	}
	return fmt.Sprintf("%v.%v()", parentExpr, n.method.Name())
}

func (n StructMethodNode) NullCheckExpr() string {
	return fmt.Sprintf("%v.%v()", n.container.AssignExpr(), n.method.Name())
}

func (n StructMethodNode) ReturnsError() bool {
	sig := n.method.Type().(*types.Signature)
	return sig.Results().Len() == 2
}

func (n StructMethodNode) ObjNullable() bool {
	return util.IsPtr(n.ExprType())
}
