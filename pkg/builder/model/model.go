package model

import (
	"go/types"
)

type Node interface {
	// Parent returns the container of the node or nil.
	Parent() Node
	// ObjName returns the ident of the leaf element.
	// E.g. "Status" in both of dst.User.Status or dst.User.Status().
	ObjName() string
	// ObjNullable indicates whether the node itself is a pointer type so that can be a nil at runtime.
	ObjNullable() bool
	// AssignExpr returns a value evaluate expression for assignment.
	// E.g. "dst.User.Name", "dst.User.Status()", "strconv.Itoa(dst.User.Score())", etc.
	AssignExpr() string
	// MatcherExpr returns a value evaluate expression for assignment but omits the root variable name.
	// E.g. "User.Status()" in "dst.User.Status()".
	MatcherExpr() string
	// NullCheckExpr returns a value evaluate expression for null check conditional.
	// E.g. "dst.Node.Child".
	NullCheckExpr() string
	// ExprType returns the evaluated result type of the node.
	// E.g. that type "dst.User.Status()" returns. An expr may in converter form, as "strconv.Itoa(dst.User.Status())".
	ExprType() types.Type
	// ReturnsError indicates whether the expression return an error object as the second returning value.
	ReturnsError() bool
}
