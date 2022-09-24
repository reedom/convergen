package builder

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/reedom/convergen/pkg/generator/model"
)

// dstFieldEntry contains assignment LHS(left hand side) information.
type dstFieldEntry struct {
	// parent refers a parent path in a nested data. Can be nil.
	parent *dstFieldEntry
	// model.Var represents an LHS variable for the root entry.
	// For descendants, it contains parent's object information.
	model.Var
	// field refers a leaf field.
	field *types.Var
}

// fieldType returns the type of the leaf field.
func (f dstFieldEntry) fieldType() types.Type {
	return f.field.Type()
}

// fieldName returns the ident of the leaf field.
// E.g. "Status" in dst.User.Status.
func (f dstFieldEntry) fieldName() string {
	return f.field.Name()
}

// fieldPath returns a path string of field or method chain without the variable name(model.Var.Name).
// E.g. "User.Status" in dst.User.Status.
func (f dstFieldEntry) fieldPath() string {
	if f.parent != nil {
		return fmt.Sprintf("%v.%v", f.parent.fieldPath(), f.field.Name())
	}
	return f.field.Name()
}

// lhsExpr returns a path string of field or method chain.
// E.g. "dst.User.Status"
func (f dstFieldEntry) lhsExpr() string {
	if f.parent != nil {
		return fmt.Sprintf("%v.%v", f.parent.lhsExpr(), f.fieldName())
	}
	return fmt.Sprintf("%v.%v", f.Name, f.fieldName())
}

// isFieldAccessible reports whether the leaf field is accessible from the "setup package".
func (f dstFieldEntry) isFieldAccessible() bool {
	return f.PkgName == "" || ast.IsExported(f.fieldName())
}

// srcStructEntry contains assignment RHS(right hand side) information.
type srcStructEntry struct {
	// parent refers a parent path in a nested data. Can be nil.
	parent *srcStructEntry
	// model.Var represents an RHS variable for the root entry.
	// For descendants, it contains parent's object information.
	model.Var
	// strct refers a type definition of a struct.
	strct *types.Var
}

// strctType returns type of the leaf struct.
func (s srcStructEntry) strctType() types.Type {
	return s.strct.Type()
}

// root returns the root ancestor.
func (s srcStructEntry) root() srcStructEntry {
	if s.parent != nil {
		return s.parent.root()
	}
	return s
}

// fieldPath returns a path string of field or method chain without the variable name(model.Var.Name).
// E.g. "User.Section" in src.User.Section.
func (s srcStructEntry) fieldPath(obj types.Object) string {
	switch obj.(type) {
	case *types.Var:
		if s.parent != nil {
			return fmt.Sprintf("%v.%v", s.parent.fieldPath(s.strct), obj.Name())
		}
		return obj.Name()
	case *types.Func:
		if s.parent != nil {
			return fmt.Sprintf("%v.%v()", s.parent.fieldPath(s.strct), obj.Name())
		}
		return obj.Name()
	default:
		panic(fmt.Sprintf("not implemented for %#v", obj))
	}
}

// rhsExpr returns a path string of field or method chain.
// E.g. "src.User.Section"
func (s srcStructEntry) rhsExpr(obj types.Object) string {
	if obj == nil {
		if s.parent != nil {
			return s.parent.rhsExpr(s.strct)
		}
		return s.Name
	}

	switch obj.(type) {
	case *types.Var:
		if s.parent != nil {
			return fmt.Sprintf("%v.%v", s.parent.rhsExpr(s.strct), obj.Name())
		}
		return fmt.Sprintf("%v.%v", s.Name, obj.Name())
	case *types.Func:
		if s.parent != nil {
			return fmt.Sprintf("%v.%v()", s.parent.rhsExpr(s.strct), obj.Name())
		}
		return fmt.Sprintf("%v.%v()", s.Name, obj.Name())
	default:
		panic(fmt.Sprintf("not implemented for %#v", obj))
	}
}

// isRecursive reports whether it'd form a recursive structure if it added the newChild.
func (s srcStructEntry) isRecursive(newChild *types.Var) bool {
	if s.strct.Id() == newChild.Id() {
		return true
	}
	if s.parent == nil {
		return false
	}
	return s.parent.isRecursive(newChild)
}
