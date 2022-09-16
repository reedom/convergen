package parser

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

var errNotFound = errors.New("not found")

type lookupFieldOpt struct {
	exactCase     bool
	supportsError bool
	pattern       string
}

// toAstNode converts types.Object to []ast.Node.
func toAstNode(file *ast.File, obj types.Object) (path []ast.Node, exact bool) {
	return astutil.PathEnclosingInterval(file, obj.Pos(), obj.Pos())
}

func isErrorType(t types.Type) bool {
	return t.String() == "error"
}

func removeObject(file *ast.File, obj types.Object) {
	nodes, _ := toAstNode(file, obj)
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.GenDecl:
			if n.Doc != nil {
				n.Doc.List = nil
			}
			astRemoveDecl(file, obj.Name())
		}
	}
	return
}

// getDocCommentOn retrieves doc comments that relate to nodes.
func getDocCommentOn(file *ast.File, obj types.Object) *ast.CommentGroup {
	nodes, _ := toAstNode(file, obj)
	if nodes == nil {
		return nil
	}

	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.GenDecl:
			if n.Doc != nil {
				return n.Doc
			}
		case *ast.FuncDecl:
			if n.Doc != nil {
				return n.Doc
			}
		case *ast.TypeSpec:
			if n.Doc != nil {
				return n.Doc
			}
		case *ast.Field:
			if n.Doc != nil {
				return n.Doc
			}
		}
	}
	return nil
}

func findField(fset *token.FileSet, pkg *types.Package, t types.Type, opt lookupFieldOpt) (types.Object, error) {
	switch typ := t.(type) {
	case *types.Pointer:
		return findField(fset, pkg, typ.Elem(), opt)
	case *types.Named:
		return findFieldInternal(fset, pkg, typ.Obj().Pkg(), typ.Underlying(), opt, strings.Split(opt.pattern, "."))
	}
	return findFieldInternal(fset, pkg, pkg, t, opt, strings.Split(opt.pattern, "."))
}

func findFieldInternal(fset *token.FileSet, pkg, typePkg *types.Package, t types.Type, opt lookupFieldOpt, pattern []string) (types.Object, error) {
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

			ok, err := pathMatch(pattern[0], m.Name(), opt.exactCase)
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

			ok, err := pathMatch(pattern[0], e.Name(), opt.exactCase)
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
		logger.Printf("findField for *types.Basic is not implemented")
	case *types.Array:
		logger.Printf("findField for *types.Array is not implemented")
	case *types.Slice:
		logger.Printf("findField for *types.Slice is not implemented")
	case *types.Map:
		logger.Printf("findField for *types.Map is not implemented")
	case *types.Chan:
		logger.Printf("findField for *types.Chan is not implemented")
	case *types.Interface:
		logger.Printf("findField for *types.Interface is not implemented")
	case *types.Tuple:
		logger.Printf("findField for *types.Tuple is not implemented")
	case *types.Signature:
		logger.Printf("findField for *types.Signature is not implemented")
	}
	logger.Printf("field pattern mismatch. pattern=%#v, t=%v\n", pattern, t)
	return nil, errNotFound
}

func pathMatch(pattern, name string, exactCase bool) (bool, error) {
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
