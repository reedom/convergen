package util

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path"
	"strings"

	"github.com/reedom/convergen/pkg/logger"
	"golang.org/x/tools/go/ast/astutil"
)

var ErrNotFound = errors.New("not found")

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

func DereferencePtr(t types.Type) types.Type {
	if ptr, ok := t.(*types.Pointer); ok {
		return ptr.Elem()
	}
	return t
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
	return
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

func FindField(fset *token.FileSet, pkg *types.Package, t types.Type, opt LookupFieldOpt) (types.Object, error) {
	switch typ := t.(type) {
	case *types.Pointer:
		return FindField(fset, pkg, typ.Elem(), opt)
	case *types.Named:
		return findFieldInternal(fset, pkg, typ.Obj().Pkg(), typ.Underlying(), opt, strings.Split(opt.Pattern, "."))
	}
	return findFieldInternal(fset, pkg, pkg, t, opt, strings.Split(opt.Pattern, "."))
}

func findFieldInternal(fset *token.FileSet, pkg, typePkg *types.Package, t types.Type, opt LookupFieldOpt, pattern []string) (types.Object, error) {
	if pattern[0] == "" {
		return nil, fmt.Errorf("invalid pattern")
	}

	switch typ := t.(type) {
	case *types.Pointer:
		return findFieldInternal(fset, pkg, typePkg, typ.Elem(), opt, pattern)
	case *types.Named:
		for i := 0; i < typ.NumMethods(); i++ {
			m := typ.Method(i)
			if pkg.Name() != typePkg.Name() && !m.Exported() {
				continue
			}

			ok, err := PathMatch(pattern[0], m.Name(), opt.ExactCase)
			if err != nil {
				return nil, err
			}
			if ok {
				if len(pattern) == 1 {
					return m, nil
				} else {
					return findFieldInternal(fset, pkg, typePkg, m.Type(), opt, pattern[1:])
				}
			}
		}
		return findFieldInternal(fset, pkg, typ.Obj().Pkg(), typ.Underlying(), opt, pattern)
	case *types.Struct:
		for i := 0; i < typ.NumFields(); i++ {
			e := typ.Field(i)
			if pkg.Name() != typePkg.Name() && !e.Exported() {
				continue
			}

			ok, err := PathMatch(pattern[0], e.Name(), opt.ExactCase)
			if err != nil {
				return nil, err
			}
			if ok {
				if len(pattern) == 1 {
					return e, nil
				} else {
					return findFieldInternal(fset, pkg, typePkg, e.Type(), opt, pattern[1:])
				}
			}
		}
	case *types.Basic:
		logger.Printf("FindField for *types.Basic is not implemented")
	case *types.Array:
		logger.Printf("FindField for *types.Array is not implemented")
	case *types.Slice:
		logger.Printf("FindField for *types.Slice is not implemented")
	case *types.Map:
		logger.Printf("FindField for *types.Map is not implemented")
	case *types.Chan:
		logger.Printf("FindField for *types.Chan is not implemented")
	case *types.Interface:
		logger.Printf("FindField for *types.Interface is not implemented")
	case *types.Tuple:
		logger.Printf("FindField for *types.Tuple is not implemented")
	case *types.Signature:
		logger.Printf("FindField for *types.Signature is not implemented")
	}
	logger.Printf("field pattern mismatch. pattern=%#v, t=%v\n", pattern, t)
	return nil, ErrNotFound
}

func PathMatch(pattern, name string, exactCase bool) (bool, error) {
	if exactCase {
		return path.Match(pattern, name)
	}
	return path.Match(strings.ToLower(pattern), strings.ToLower(name))
}

type walkStructCallback = func(pkg *types.Package, obj types.Object, namePath string) (done bool, err error)

type walkStructOpt struct {
	exactCase     bool
	supportsError bool
}

type walkStructWork struct {
	varName   string
	pkg       *types.Package
	cb        walkStructCallback
	isPointer bool
	namePath  namePathType
	done      *bool
}

type namePathType string

func (n namePathType) String() string {
	return string(n)
}

func (n namePathType) add(name string) namePathType {
	return namePathType(n.String() + "." + name)
}

func (w walkStructWork) withNamePath(namePath namePathType) walkStructWork {
	work := w
	work.namePath = namePath
	return work
}

func walkStruct(varName string, pkg *types.Package, t types.Type, cb walkStructCallback, opt walkStructOpt) error {
	done := false
	work := walkStructWork{
		varName:   varName,
		pkg:       pkg,
		cb:        cb,
		isPointer: false,
		namePath:  namePathType(varName),
		done:      &done,
	}

	switch typ := t.(type) {
	case *types.Pointer:
		return walkStruct(varName, pkg, typ.Elem(), cb, opt)
	case *types.Named:
		return walkStructInternal(typ.Obj().Pkg(), typ.Underlying(), opt, work)
	}

	return walkStructInternal(pkg, t, opt, work)
}

func walkStructInternal(typePkg *types.Package, t types.Type, opt walkStructOpt, work walkStructWork) error {
	//isPointer := work.isPointer
	work.isPointer = false

	switch typ := t.(type) {
	case *types.Pointer:
		work.isPointer = true
		return walkStructInternal(typePkg, typ.Elem(), opt, work)
	case *types.Named:
		return walkStructInternal(typ.Obj().Pkg(), typ.Underlying(), opt, work)
	case *types.Struct:
		for i := 0; i < typ.NumFields(); i++ {
			f := typ.Field(i)
			namePath := work.namePath.add(f.Name())
			done, err := work.cb(typePkg, f, namePath.String())
			if err != nil {
				return err
			}
			if done {
				continue
			}

			// Down to the f.Type() hierarchy.
			err = walkStructInternal(typePkg, f.Type(), opt, work.withNamePath(namePath))
			if err != nil {
				return err
			}
		}
		return nil
	}

	fmt.Printf("@@@ %#v\n", t)
	return nil
}

func IterateMethods(t types.Type, cb func(*types.Func) (done bool, err error)) error {
	if ptr, ok := t.(*types.Pointer); ok {
		return IterateMethods(ptr.Elem(), cb)
	}

	named, ok := t.(*types.Named)
	if !ok {
		return ErrNotFound
	}

	for i := 0; i < named.NumMethods(); i++ {
		m := named.Method(i)
		done, err := cb(m)
		if done || err != nil {
			return err
		}
	}
	return nil
}

func IterateFields(t types.Type, cb func(*types.Var) (done bool, err error)) error {
	if ptr, ok := t.(*types.Pointer); ok {
		return IterateFields(ptr.Elem(), cb)
	}
	if named, ok := t.(*types.Named); ok {
		return IterateFields(named.Underlying(), cb)
	}

	strct, ok := t.Underlying().(*types.Struct)
	if !ok {
		return ErrNotFound
	}

	for i := 0; i < strct.NumFields(); i++ {
		m := strct.Field(i)
		done, err := cb(m)
		if done || err != nil {
			return err
		}
	}
	return nil
}

func GetMethodReturnTypes(m *types.Func) (*types.Tuple, bool) {
	sig := m.Type().(*types.Signature)
	num := sig.Results().Len()
	if num == 0 || 2 < num {
		return nil, false
	}

	return sig.Results(), true
}
