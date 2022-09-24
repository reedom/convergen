package util

import (
	"go/ast"
	"go/types"
	"path"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

type LookupFieldOpt struct {
	ExactCase     bool
	SupportsError bool
	Pattern       string
}

// ToAstNode converts types.Object to []ast.Node.
func ToAstNode(file *ast.File, obj types.Object) (path []ast.Node, exact bool) {
	return astutil.PathEnclosingInterval(file, obj.Pos(), obj.Pos())
}

func IsErrorType(t types.Type) bool {
	return t.String() == "error"
}

func IsStructType(t types.Type) bool {
	_, ok := t.Underlying().(*types.Struct)
	return ok
}

func IsNamedType(t types.Type) bool {
	_, ok := t.(*types.Named)
	return ok
}

func IsFunc(obj types.Object) bool {
	_, ok := obj.(*types.Func)
	return ok
}

func IsPtr(t types.Type) bool {
	_, ok := t.(*types.Pointer)
	return ok
}

// DerefPtr dereferences typ if it is a *Pointer and returns its base.
func DerefPtr(typ types.Type) types.Type {
	if ptr, ok := typ.(*types.Pointer); ok {
		return ptr.Elem()
	}
	return typ
}

func PkgOf(t types.Type) *types.Package {
	switch typ := t.(type) {
	case *types.Pointer:
		return PkgOf(typ.Elem())
	case *types.Named:
		return typ.Obj().Pkg()
	default:
		return nil
	}
}

// Deref dereferences typ if it is a *Pointer and returns its base and true.
// Otherwise it returns (typ, false).
func Deref(typ types.Type) (types.Type, bool) {
	if ptr, ok := typ.(*types.Pointer); ok {
		return ptr.Elem(), true
	}
	return typ, false
}

func RemoveObject(file *ast.File, obj types.Object) {
	nodes, _ := ToAstNode(file, obj)
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.GenDecl:
			if n.Doc != nil {
				n.Doc.List = nil
			}
			RemoveDecl(file, obj.Name())
		}
	}
}

// GetDocCommentOn retrieves doc comments that relate to nodes.
func GetDocCommentOn(file *ast.File, obj types.Object) (cg *ast.CommentGroup, cleanUp func()) {
	nodes, _ := ToAstNode(file, obj)
	if nodes == nil {
		return nil, func() {}
	}

	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.GenDecl:
			if n.Doc != nil {
				return n.Doc, func() {
					if len(n.Doc.List) == 0 {
						n.Doc = nil
					}
				}
			}
		case *ast.FuncDecl:
			if n.Doc != nil {
				return n.Doc, func() {
					if len(n.Doc.List) == 0 {
						n.Doc = nil
					}
				}
			}
		case *ast.TypeSpec:
			if n.Doc != nil {
				return n.Doc, func() {
					if len(n.Doc.List) == 0 {
						n.Doc = nil
					}
				}
			}
		case *ast.Field:
			if n.Doc != nil {
				return n.Doc, func() {
					if len(n.Doc.List) == 0 {
						n.Doc = nil
					}
				}
			}
		case *ast.File:
			if n.Doc != nil {
				return n.Doc, func() {
					if len(n.Doc.List) == 0 {
						n.Doc = nil
					}
				}
			}
		}
	}
	return nil, func() {}
}

func ToTextList(doc *ast.CommentGroup) []string {
	if doc == nil || len(doc.List) == 0 {
		return []string{}
	}

	list := make([]string, len(doc.List))
	for i := range doc.List {
		list[i] = doc.List[i].Text
	}
	return list
}

func PathMatch(pattern, name string, exactCase bool) (bool, error) {
	if exactCase {
		return path.Match(pattern, name)
	}
	return path.Match(strings.ToLower(pattern), strings.ToLower(name))
}

func FindMethod(t types.Type, name string, exactCase bool) (method *types.Func) {
	if !exactCase {
		name = strings.ToLower(name)
	}

	IterateMethods(t, func(m *types.Func) (done bool) {
		found := false
		if exactCase {
			found = m.Name() == name
		} else {
			found = strings.ToLower(m.Name()) == name
		}
		if found {
			method = m
		}
		return found
	})
	return
}

func IterateMethods(t types.Type, cb func(*types.Func) (done bool)) {
	typ := DerefPtr(t)
	named, ok := typ.(*types.Named)
	if !ok {
		return
	}

	for i := 0; i < named.NumMethods(); i++ {
		m := named.Method(i)
		if cb(m) {
			return
		}
	}
}

func FindField(t types.Type, name string, exactCase bool) (field *types.Var) {
	if !exactCase {
		name = strings.ToLower(name)
	}

	IterateFields(t, func(f *types.Var) (done bool) {
		found := false
		if exactCase {
			found = f.Name() == name
		} else {
			found = strings.ToLower(f.Name()) == name
		}
		if found {
			field = f
		}
		return found
	})
	return
}

func IterateFields(t types.Type, cb func(*types.Var) (done bool)) {
	typ := DerefPtr(t)
	if named, ok := typ.(*types.Named); ok {
		typ = named.Underlying()
	}

	strct, ok := typ.Underlying().(*types.Struct)
	if !ok {
		return
	}

	for i := 0; i < strct.NumFields(); i++ {
		m := strct.Field(i)
		if cb(m) {
			return
		}
	}
}

func GetMethodReturnTypes(m *types.Func) (*types.Tuple, bool) {
	sig := m.Type().(*types.Signature)
	num := sig.Results().Len()
	if num == 0 || 2 < num {
		return nil, false
	}

	return sig.Results(), true
}

func ParseGetterReturnTypes(m *types.Func) (ret types.Type, retError, ok bool) {
	sig := m.Type().(*types.Signature)
	num := sig.Results().Len()
	if num == 0 || 2 < num {
		return
	}
	if num == 2 {
		if !IsErrorType(sig.Results().At(1).Type()) {
			return
		}
	}

	return sig.Results().At(0).Type(), num == 2, true
}
