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

// IsErrorType returns true if the given type is an error type.
func IsErrorType(t types.Type) bool {
	return t.String() == "error"
}

// IsInvalidType returns true if the given type is an invalid type.
func IsInvalidType(t types.Type) bool {
	if typ, ok := DerefPtr(t).Underlying().(*types.Basic); ok {
		return typ.Kind() == types.Invalid
	}
	return false
}

// IsSliceType returns true if the given type is a slice type.
func IsSliceType(t types.Type) bool {
	_, ok := t.(*types.Slice)
	return ok
}

// IsBasicType returns true if the given type is a basic type.
func IsBasicType(t types.Type) bool {
	_, ok := t.(*types.Basic)
	return ok
}

// IsStructType returns true if the given type is a struct type.
func IsStructType(t types.Type) bool {
	_, ok := t.Underlying().(*types.Struct)
	return ok
}

// IsNamedType returns true if the given type is a named type.
func IsNamedType(t types.Type) bool {
	_, ok := t.(*types.Named)
	return ok
}

// IsFunc returns true if the given type is a func type.
func IsFunc(obj types.Object) bool {
	_, ok := obj.(*types.Func)
	return ok
}

// IsPtr returns true if the given type is a pointer type.
func IsPtr(t types.Type) bool {
	_, ok := t.(*types.Pointer)
	return ok
}

// DerefPtr dereferences a *Pointer type and returns its base type.
func DerefPtr(typ types.Type) types.Type {
	if ptr, ok := typ.(*types.Pointer); ok {
		return ptr.Elem()
	}
	return typ
}

// Deref dereferences a type if it is a *Pointer type and returns its base type and true.
// Otherwise, it returns (typ, false).
func Deref(typ types.Type) (types.Type, bool) {
	if ptr, ok := typ.(*types.Pointer); ok {
		return ptr.Elem(), true
	}
	return typ, false
}

// PkgOf returns the package of the given type.
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

// SliceElement returns the type of the element in a slice type.
func SliceElement(t types.Type) types.Type {
	if slice, ok := t.(*types.Slice); ok {
		return slice.Elem()
	}
	return nil
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

// ToTextList returns a list of strings representing the comments in a CommentGroup.
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

// PathMatch returns true if the name matches the pattern.
func PathMatch(pattern, name string, exactCase bool) (bool, error) {
	if exactCase {
		return path.Match(pattern, name)
	}
	return path.Match(strings.ToLower(pattern), strings.ToLower(name))
}

// FindMethod returns the method with the given name in the given type.
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

// IterateMethods iterates over the methods of the given type and calls the callback for each one.
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

// FindField returns the field with the given name from the given type.
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

// IterateFields iterates over the fields of the given type and calls the callback for each one.
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

// GetMethodReturnTypes returns the return types of the given method.
func GetMethodReturnTypes(m *types.Func) (*types.Tuple, bool) {
	sig := m.Type().(*types.Signature)
	num := sig.Results().Len()
	if num == 0 || 2 < num {
		return nil, false
	}

	return sig.Results(), true
}

// ParseGetterReturnTypes returns the return types of the given method.
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

// StringType returns the string type in the universe scope.
func StringType() types.Type {
	return types.Universe.Lookup("string").Type()
}

// CompliesGetter checks whether the given function complies with the requirements of a getter function.
// A getter function must have no input parameters and must return exactly one non-error value.
func CompliesGetter(m *types.Func) bool {
	sig := m.Type().(*types.Signature)
	if sig.Params().Len() != 0 {
		return false
	}
	num := sig.Results().Len()
	return num == 1 && !IsErrorType(sig.Results().At(0).Type())
}

// CompliesStringer checks if the given type is a Stringer compliant type,
// which has a method "String()" that takes no arguments and returns a string.
func CompliesStringer(src types.Type) bool {
	named, ok := DerefPtr(src).(*types.Named)
	if !ok {
		return false
	}

	obj, _, _ := types.LookupFieldOrMethod(named, false, named.Obj().Pkg(), "String")
	if obj == nil {
		return false
	}

	sig, ok := obj.Type().(*types.Signature)
	if !ok {
		return false
	}

	return sig.Params().Len() == 0 &&
		sig.Results().Len() == 1 &&
		sig.Results().At(0).Type().String() == "string"
}
