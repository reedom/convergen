package model

import (
	"fmt"
	"go/types"

	"github.com/reedom/convergen/pkg/option"
	"github.com/reedom/convergen/pkg/util"
)

type Node interface {
	// Parent returns the container of the node or nil.
	Parent() Node

	// ObjName returns the ident of the leaf element.
	// For example, it returns "Status" in both of dst.User.Status or dst.User.Status().
	ObjName() string

	// ObjNullable indicates whether the node itself is a pointer type so that it can be nil at runtime.
	ObjNullable() bool

	// AssignExpr returns a value evaluate expression for assignment.
	// For example, it returns "dst.User.Name", "dst.User.Status()", "strconv.Itoa(dst.User.Score())", etc.
	AssignExpr() string

	// MatcherExpr returns a value evaluate expression for assignment but omits the root variable name.
	// For example, it returns "User.Status()" in "dst.User.Status()".
	MatcherExpr() string

	// NullCheckExpr returns a value evaluate expression for null check conditional.
	// For example, it returns "dst.Node.Child".
	NullCheckExpr() string

	// ExprType returns the evaluated result type of the node.
	// For example, it returns the type that "dst.User.Status()" returns.
	// An expression may be in converter form, such as "strconv.Itoa(dst.User.Status())".
	ExprType() types.Type

	// ReturnsError indicates whether the expression returns an error object as the second returning value.
	ReturnsError() bool
}

// RootNode is a special node that represents the root of the expression tree.
type RootNode struct {
	name string
	typ  types.Type
}

// NewRootNode creates a new RootNode.
func NewRootNode(name string, typ types.Type) RootNode {
	return RootNode{name: name, typ: typ}
}

// Parent returns the container of the node or nil.
func (n RootNode) Parent() Node {
	return nil
}

// ObjName returns the ident of the leaf element.
// For example, it returns "Status" in both of dst.User.Status or dst.User.Status().
func (n RootNode) ObjName() string {
	return n.name
}

// ObjNullable indicates whether the node itself is a pointer type so that it can be nil at runtime.
func (n RootNode) ObjNullable() bool {
	return util.IsPtr(n.typ)
}

// ExprType returns the evaluated result type of the node.
// For example, it returns the type that "dst.User.Status()" returns.
// An expression may be in converter form, such as "strconv.Itoa(dst.User.Status())".
func (n RootNode) ExprType() types.Type {
	return n.typ
}

// ReturnsError indicates whether the expression returns an error object as the second returning value.
func (n RootNode) ReturnsError() bool {
	return false
}

// AssignExpr returns a value evaluate expression for assignment.
// For example, it returns "dst.User.Name", "dst.User.Status()", "strconv.Itoa(dst.User.Score())", etc.
func (n RootNode) AssignExpr() string {
	return n.name
}

// MatcherExpr returns a value evaluate expression for assignment but omits the root variable name.
// For example, it returns "User.Status()" in "dst.User.Status()".
func (n RootNode) MatcherExpr() string {
	return ""
}

// NullCheckExpr returns a value evaluate expression for null check conditional.
// For example, it returns "dst.Node.Child".
func (n RootNode) NullCheckExpr() string {
	return n.name
}

// ScalarNode is a node that represents a leaf element of the expression tree.
type ScalarNode struct {
	// parent refers the parent of the struct if nested. Can be nil.
	parent Node
	// name is either a variable name for a root struct or field name in a struct.
	name string
	// typ is the type of the subject.
	typ types.Type
}

// NewScalarNode creates a new ScalarNode.
func NewScalarNode(parent Node, name string, typ types.Type) Node {
	return ScalarNode{
		parent: parent,
		name:   name,
		typ:    typ,
	}
}

// Parent returns the container of the node or nil.
func (n ScalarNode) Parent() Node {
	return n.parent
}

// ObjName returns the ident of the leaf element.
// For example, it returns "Status" in both of dst.User.Status or dst.User.Status().
func (n ScalarNode) ObjName() string {
	return n.name
}

// ObjNullable indicates whether the node itself is a pointer type so that it can be nil at runtime.
func (n ScalarNode) ObjNullable() bool {
	return util.IsPtr(n.typ)
}

// ExprType returns the evaluated result type of the node.
// For example, it returns the type that "dst.User.Status()" returns.
// An expression may be in converter form, such as "strconv.Itoa(dst.User.Status())".
func (n ScalarNode) ExprType() types.Type {
	return n.typ
}

// ReturnsError indicates whether the expression returns an error object as the second returning value.
func (n ScalarNode) ReturnsError() bool {
	return false
}

// AssignExpr returns a value evaluate expression for assignment.
// For example, it returns "dst.User.Name", "dst.User.Status()", "strconv.Itoa(dst.User.Score())", etc.
func (n ScalarNode) AssignExpr() string {
	if n.parent != nil {
		return n.parent.AssignExpr()
	}
	return n.name
}

// MatcherExpr returns a value evaluate expression for assignment but omits the root variable name.
// For example, it returns "User.Status()" in "dst.User.Status()".
func (n ScalarNode) MatcherExpr() string {
	if n.parent != nil {
		return n.parent.MatcherExpr()
	}
	return ""
}

// NullCheckExpr returns a value evaluate expression for null check conditional.
// For example, it returns "dst.Node.Child".
func (n ScalarNode) NullCheckExpr() string {
	if n.parent != nil {
		return n.parent.NullCheckExpr()
	}
	return n.name
}

// ConverterNode is a node that represents a converter function.
type ConverterNode struct {
	arg Node
	//pkgName   string
	converter *option.FieldConverter
}

// NewConverterNode creates a new ConverterNode.
func NewConverterNode(arg Node, converter *option.FieldConverter) Node {
	return ConverterNode{
		arg:       arg,
		converter: converter,
	}
}

// Parent returns the container of the node or nil.
func (n ConverterNode) Parent() Node {
	return n.arg.Parent()
}

// ObjName returns the ident of the leaf element.
// For example, it returns "Status" in both of dst.User.Status or dst.User.Status().
func (n ConverterNode) ObjName() string {
	return n.arg.ObjName()
}

// ObjNullable indicates whether the node itself is a pointer type so that it can be nil at runtime.
func (n ConverterNode) ObjNullable() bool {
	return n.arg.ObjNullable()
}

// ExprType returns the evaluated result type of the node.
// For example, it returns the type that "dst.User.Status()" returns.
// An expression may be in converter form, such as "strconv.Itoa(dst.User.Status())".
func (n ConverterNode) ExprType() types.Type {
	return n.converter.RetType()
}

// ReturnsError indicates whether the expression returns an error object as the second returning value.
func (n ConverterNode) ReturnsError() bool {
	return n.converter.RetError()
}

// AssignExpr returns a value evaluate expression for assignment.
// For example, it returns "dst.User.Name", "dst.User.Status()", "strconv.Itoa(dst.User.Score())", etc.
func (n ConverterNode) AssignExpr() string {
	refStr := ""
	if !util.IsPtr(n.arg.ExprType()) && util.IsPtr(n.converter.ArgType()) {
		refStr = "&"
	}
	return fmt.Sprintf("%v(%v%v)", n.converter.Converter(), refStr, n.arg.AssignExpr())
}

// MatcherExpr returns a value evaluate expression for assignment but omits the root variable name.
// For example, it returns "User.Status()" in "dst.User.Status()".
func (n ConverterNode) MatcherExpr() string {
	return n.arg.MatcherExpr()
}

// NullCheckExpr returns a value evaluate expression for null check conditional.
// For example, it returns "dst.Node.Child".
func (n ConverterNode) NullCheckExpr() string {
	return n.AssignExpr()
}

// TypecastEntry is a node that represents a typecast expression.
type TypecastEntry struct {
	inner Node
	typ   types.Type
	expr  string
}

// NewTypecast creates a new TypecastEntry.
func NewTypecast(scope *types.Scope, imports util.ImportNames, t types.Type, inner Node) (Node, bool) {
	var expr string
	switch typ := util.DerefPtr(t).(type) {
	case *types.Named:
		// If the type is defined within the current package.
		if scope.Lookup(typ.Obj().Name()) != nil {
			expr = typ.Obj().Name()
		} else if pkgName, ok := imports.LookupName(typ.Obj().Pkg().Path()); ok {
			expr = fmt.Sprintf("%v.%v", pkgName, typ.Obj().Name())
		} else {
			expr = fmt.Sprintf("%v.%v", typ.Obj().Pkg().Name(), typ.Obj().Name())
		}
	case *types.Basic:
		expr = t.String()
	default:
		return nil, false
	}

	return TypecastEntry{inner: inner, typ: t, expr: expr}, true
}

// ObjName returns the ident of the leaf element.
// For example, it returns "Status" in both of dst.User.Status or dst.User.Status().
func (n TypecastEntry) ObjName() string {
	return n.inner.ObjName()
}

// Parent returns the container of the node or nil.
func (n TypecastEntry) Parent() Node {
	return n.inner.Parent()
}

// ExprType returns the evaluated result type of the node.
// For example, it returns the type that "dst.User.Status()" returns.
// An expression may be in converter form, such as "strconv.Itoa(dst.User.Status())".
func (n TypecastEntry) ExprType() types.Type {
	return n.typ
}

// AssignExpr returns a value evaluate expression for assignment.
// For example, it returns "dst.User.Name", "dst.User.Status()", "strconv.Itoa(dst.User.Score())", etc.
func (n TypecastEntry) AssignExpr() string {
	return fmt.Sprintf("%v(%v)", n.expr, n.inner.AssignExpr())
}

// MatcherExpr returns a value evaluate expression for assignment but omits the root variable name.
// For example, it returns "User.Status()" in "dst.User.Status()".
func (n TypecastEntry) MatcherExpr() string {
	return n.inner.MatcherExpr()
}

// NullCheckExpr returns a value evaluate expression for null check conditional.
// For example, it returns "dst.Node.Child".
func (n TypecastEntry) NullCheckExpr() string {
	return n.inner.NullCheckExpr()
}

// ReturnsError indicates whether the expression returns an error object as the second returning value.
func (n TypecastEntry) ReturnsError() bool {
	return false
}

// ObjNullable indicates whether the node itself is a pointer type so that it can be nil at runtime.
func (n TypecastEntry) ObjNullable() bool {
	return n.inner.ObjNullable()
}

// StringerEntry is a node that represents a Stringer interface.
type StringerEntry struct {
	inner Node
}

// NewStringer creates a new StringerEntry.
func NewStringer(inner Node) Node {
	return StringerEntry{inner: inner}
}

// ObjName returns the ident of the leaf element.
// For example, it returns "Status" in both of dst.User.Status or dst.User.Status().
func (e StringerEntry) ObjName() string {
	return e.inner.ObjName()
}

// Parent returns the container of the node or nil.
func (e StringerEntry) Parent() Node {
	return e.inner.Parent()
}

// ExprType returns the evaluated result type of the node.
// For example, it returns the type that "dst.User.Status()" returns.
// An expression may be in converter form, such as "strconv.Itoa(dst.User.Status())".
func (e StringerEntry) ExprType() types.Type {
	return types.Universe.Lookup("string").Type()
}

// AssignExpr returns a value evaluate expression for assignment.
// For example, it returns "dst.User.Name", "dst.User.Status()", "strconv.Itoa(dst.User.Score())", etc.
func (e StringerEntry) AssignExpr() string {
	return fmt.Sprintf("%v.String()", e.inner.AssignExpr())
}

// MatcherExpr returns a value evaluate expression for assignment but omits the root variable name.
// For example, it returns "User.Status()" in "dst.User.Status()".
func (e StringerEntry) MatcherExpr() string {
	return e.inner.MatcherExpr()
}

// NullCheckExpr returns a value evaluate expression for null check conditional.
// For example, it returns "dst.Node.Child".
func (e StringerEntry) NullCheckExpr() string {
	return e.inner.NullCheckExpr()
}

// ReturnsError indicates whether the expression returns an error object as the second returning value.
func (e StringerEntry) ReturnsError() bool {
	return false
}

// ObjNullable indicates whether the node itself is a pointer type so that it can be nil at runtime.
func (e StringerEntry) ObjNullable() bool {
	return e.inner.ObjNullable()
}
