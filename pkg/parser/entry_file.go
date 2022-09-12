package parser

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

type entryFile struct {
	srcPath string
	file    *ast.File
	fileSet *token.FileSet
	pkg     *types.Package
}

type parsedMethod struct {
	method    types.Object
	comments  []*ast.Comment
	notations []*notation
	from
}

type notation struct {
	name string
}

func (e *entryFile) getInterface() (*types.TypeName, error) {
	intf := findInterface(e.pkg.Scope(), "loki")
	if intf == nil {
		return nil, fmt.Errorf(`"loki" interface not found in %v`, e.srcPath)
	}
	return intf, nil
}

func (e *entryFile) parseloki() error {
	intf, err := e.getInterface()
	if err != nil {
		return err
	}

	//intfDocComment := astGetDocCommentOn(e.file, intf)

	iface, ok := intf.Type().Underlying().(*types.Interface)
	if !ok {
		panic("???")
	}

	mset := types.NewMethodSet(iface)
	for i := 0; i < mset.Len(); i++ {
		method := mset.At(i).Obj()
		err = e.parseMethod(method)
	}

	return nil
}

func (e *entryFile) parseMethod(method types.Object) error {
	//cg := astGetDocCommentOn(e.file, meth)
	sig := types.TypeString(method.Type(), (*types.Package).Name)
	fmt.Printf("func %s%s \n",
		method.Name(),
		strings.TrimPrefix(sig, "func"))

	return nil
}
