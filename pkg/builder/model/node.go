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

type RootNode struct {
	name string
	typ  types.Type
}

func NewRootNode(name string, typ types.Type) RootNode {
	return RootNode{name: name, typ: typ}
}

func (n RootNode) Parent() Node {
	return nil
}

func (n RootNode) ObjName() string {
	return n.name
}

func (n RootNode) ObjNullable() bool {
	return util.IsPtr(n.typ)
}

func (n RootNode) ExprType() types.Type {
	return n.typ
}

func (n RootNode) ReturnsError() bool {
	return false
}

func (n RootNode) AssignExpr() string {
	return n.name
}

func (n RootNode) MatcherExpr() string {
	return ""
}

func (n RootNode) NullCheckExpr() string {
	return n.name
}

type ScalarNode struct {
	// parent refers the parent of the struct if nested. Can be nil.
	parent Node
	// name is either a variable name for a root struct or field name in a struct.
	name string
	// typ is the type of the subject.
	typ types.Type
}

func NewScalarNode(parent Node, name string, typ types.Type) Node {
	return ScalarNode{
		parent: parent,
		name:   name,
		typ:    typ,
	}
}

func (n ScalarNode) Parent() Node {
	return nil
}

func (n ScalarNode) ObjName() string {
	return n.name
}

func (n ScalarNode) ObjNullable() bool {
	return util.IsPtr(n.typ)
}

func (n ScalarNode) ExprType() types.Type {
	return n.typ
}

func (n ScalarNode) ReturnsError() bool {
	return false
}

func (n ScalarNode) AssignExpr() string {
	if n.parent != nil {
		return n.parent.AssignExpr()
	}
	return n.name
}

func (n ScalarNode) MatcherExpr() string {
	if n.parent != nil {
		return n.parent.MatcherExpr()
	}
	return ""
}

func (n ScalarNode) NullCheckExpr() string {
	if n.parent != nil {
		return n.parent.NullCheckExpr()
	}
	return n.name
}

type ConverterNode struct {
	arg       Node
	pkgName   string
	converter *option.FieldConverter
}

func NewConverterNode(arg Node, converter *option.FieldConverter) Node {
	return ConverterNode{
		arg:       arg,
		converter: converter,
	}
}

func (n ConverterNode) Parent() Node {
	return n.arg.Parent()
}

func (n ConverterNode) ObjName() string {
	return n.arg.ObjName()
}

func (n ConverterNode) ObjNullable() bool {
	return n.arg.ObjNullable()
}

func (n ConverterNode) ExprType() types.Type {
	return n.converter.RetType()
}

func (n ConverterNode) ReturnsError() bool {
	return n.converter.RetError()
}

func (n ConverterNode) AssignExpr() string {
	refStr := ""
	if !util.IsPtr(n.arg.ExprType()) && util.IsPtr(n.converter.ArgType()) {
		refStr = "&"
	}
	return fmt.Sprintf("%v(%v%v)", n.converter.Converter(), refStr, n.arg.AssignExpr())
}

func (n ConverterNode) MatcherExpr() string {
	return n.arg.MatcherExpr()
}

func (n ConverterNode) NullCheckExpr() string {
	return n.AssignExpr()
}

type TypecastEntry struct {
	inner Node
	typ   types.Type
	expr  string
}

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

func (n TypecastEntry) ObjName() string {
	return n.inner.ObjName()
}

func (n TypecastEntry) Parent() Node {
	return n.inner.Parent()
}

func (n TypecastEntry) ExprType() types.Type {
	return n.typ
}

func (n TypecastEntry) AssignExpr() string {
	return fmt.Sprintf("%v(%v)", n.expr, n.inner.AssignExpr())
}

func (n TypecastEntry) MatcherExpr() string {
	return n.inner.MatcherExpr()
}

func (n TypecastEntry) NullCheckExpr() string {
	return n.inner.NullCheckExpr()
}

func (n TypecastEntry) ReturnsError() bool {
	return false
}

func (n TypecastEntry) ObjNullable() bool {
	return n.inner.ObjNullable()
}

type StringerEntry struct {
	inner Node
}

func NewStringer(inner Node) Node {
	return StringerEntry{inner: inner}
}

func (e StringerEntry) ObjName() string {
	return e.inner.ObjName()
}

func (e StringerEntry) Parent() Node {
	return e.inner.Parent()
}

func (e StringerEntry) ExprType() types.Type {
	return types.Universe.Lookup("string").Type()
}

func (e StringerEntry) AssignExpr() string {
	return fmt.Sprintf("%v.String()", e.inner.AssignExpr())
}

func (e StringerEntry) MatcherExpr() string {
	return e.inner.MatcherExpr()
}

func (e StringerEntry) NullCheckExpr() string {
	return e.inner.NullCheckExpr()
}

func (e StringerEntry) ReturnsError() bool {
	return false
}

func (e StringerEntry) ObjNullable() bool {
	return e.inner.ObjNullable()
}
