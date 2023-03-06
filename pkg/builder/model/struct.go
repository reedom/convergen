package model

import (
	"fmt"
	"go/types"

	"github.com/reedom/convergen/pkg/util"
)

// StructFieldNode represents a struct field.
type StructFieldNode struct {
	// parent refers to the parent struct type entry.
	parent Node
	// field refers to the leaf Field.
	field *types.Var
}

// NewStructFieldNode creates a new StructFieldNode.
func NewStructFieldNode(container Node, field *types.Var) StructFieldNode {
	return StructFieldNode{
		parent: container,
		field:  field,
	}
}

// Parent returns the container of the node or nil.
func (n StructFieldNode) Parent() Node {
	return n.parent
}

// ObjName returns the ident of the leaf element.
// For example, it returns "Status" in both of dst.User.Status or dst.User.Status().
func (n StructFieldNode) ObjName() string {
	return n.field.Name()
}

// ObjNullable indicates whether the node itself is a pointer type so that it can be nil at runtime.
func (n StructFieldNode) ObjNullable() bool {
	return util.IsPtr(n.field.Type())
}

// ExprType returns the evaluated result type of the node.
// For example, it returns the type that "dst.User.Status()" returns.
// An expression may be in converter form, such as "strconv.Itoa(dst.User.Status())".
func (n StructFieldNode) ExprType() types.Type {
	return n.field.Type()
}

// ReturnsError indicates whether the expression returns an error object as the second returning value.
func (n StructFieldNode) ReturnsError() bool {
	return false
}

// AssignExpr returns a value evaluate expression for assignment.
// For example, it returns "dst.User.Name", "dst.User.Status()", "strconv.Itoa(dst.User.Score())", etc.
func (n StructFieldNode) AssignExpr() string {
	return fmt.Sprintf("%v.%v", n.parent.AssignExpr(), n.field.Name())
}

// MatcherExpr returns a value evaluate expression for assignment but omits the root variable name.
// For example, it returns "User.Status()" in "dst.User.Status()".
func (n StructFieldNode) MatcherExpr() string {
	parentExpr := n.parent.MatcherExpr()
	if parentExpr == "" {
		return n.field.Name()
	}
	return fmt.Sprintf("%v.%v", parentExpr, n.field.Name())
}

// NullCheckExpr returns a value evaluate expression for null check conditional.
// For example, it returns "dst.Node.Child".
func (n StructFieldNode) NullCheckExpr() string {
	return fmt.Sprintf("%v.%v", n.parent.AssignExpr(), n.field.Name())
}

// StructMethodNode represents a struct method.
type StructMethodNode struct {
	// container refers to the container struct type entry.
	container Node
	// method refers to the leaf field whose type is a func.
	method *types.Func
}

// NewStructMethodNode creates a new StructMethodNode.
func NewStructMethodNode(container Node, method *types.Func) StructMethodNode {
	return StructMethodNode{
		container: container,
		method:    method,
	}
}

// ObjName returns the ident of the leaf element.
// For example, it returns "Status" in both of dst.User.Status or dst.User.Status().
func (n StructMethodNode) ObjName() string {
	return n.method.Name()
}

// Parent returns the container of the node or nil.
func (n StructMethodNode) Parent() Node {
	return n.container
}

// ExprType returns the evaluated result type of the node.
// For example, it returns the type that "dst.User.Status()" returns.
// An expression may be in converter form, such as "strconv.Itoa(dst.User.Status())".
func (n StructMethodNode) ExprType() types.Type {
	sig := n.method.Type().(*types.Signature)
	return sig.Results().At(0).Type()
}

// AssignExpr returns a value evaluate expression for assignment.
// For example, it returns "dst.User.Name", "dst.User.Status()", "strconv.Itoa(dst.User.Score())", etc.
func (n StructMethodNode) AssignExpr() string {
	return fmt.Sprintf("%v.%v()", n.container.AssignExpr(), n.method.Name())
}

// MatcherExpr returns a value evaluate expression for assignment but omits the root variable name.
// For example, it returns "User.Status()" in "dst.User.Status()".
func (n StructMethodNode) MatcherExpr() string {
	parentExpr := n.container.MatcherExpr()
	if parentExpr == "" {
		return n.method.Name() + "()"
	}
	return fmt.Sprintf("%v.%v()", parentExpr, n.method.Name())
}

// NullCheckExpr returns a value evaluate expression for null check conditional.
// For example, it returns "dst.Node.Child".
func (n StructMethodNode) NullCheckExpr() string {
	return fmt.Sprintf("%v.%v()", n.container.AssignExpr(), n.method.Name())
}

// ReturnsError indicates whether the expression returns an error object as the second returning value.
func (n StructMethodNode) ReturnsError() bool {
	sig := n.method.Type().(*types.Signature)
	return sig.Results().Len() == 2
}

// ObjNullable indicates whether the node itself is a pointer type so that it can be nil at runtime.
func (n StructMethodNode) ObjNullable() bool {
	return util.IsPtr(n.ExprType())
}
