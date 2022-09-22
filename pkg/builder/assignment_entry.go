package builder

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/reedom/convergen/pkg/model"
)

type dstFieldEntry struct {
	parent *dstFieldEntry
	model.Var
	field *types.Var
}

func (f dstFieldEntry) fieldName() string {
	return f.field.Name()
}

func (f dstFieldEntry) fieldPath() string {
	if f.parent != nil {
		return fmt.Sprintf("%v.%v", f.parent.fieldPath(), f.field.Name())
	}
	return f.field.Name()
}

func (f dstFieldEntry) fieldType() types.Type {
	return f.field.Type()
}

func (f dstFieldEntry) lhsExpr() string {
	if f.parent != nil {
		return fmt.Sprintf("%v.%v", f.parent.lhsExpr(), f.fieldName())
	}
	return fmt.Sprintf("%v.%v", f.Name, f.fieldName())
}

func (f dstFieldEntry) isFieldExported() bool {
	return f.PkgName == "" || ast.IsExported(f.fieldName())
}

type srcStructEntry struct {
	parent *srcStructEntry
	model.Var
	strct *types.Var
}

func (s srcStructEntry) strctType() types.Type {
	return s.strct.Type()
}

func (s srcStructEntry) root() srcStructEntry {
	if s.parent != nil {
		return s.parent.root()
	}
	return s
}

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

func (s srcStructEntry) isRecursive(newChild *types.Var) bool {
	if s.strct.Id() == newChild.Id() {
		return true
	}
	if s.parent == nil {
		return false
	}
	return s.parent.isRecursive(newChild)
}
