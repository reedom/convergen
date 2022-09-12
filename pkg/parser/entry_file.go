package parser

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/reedom/loki/pkg/parser/option"
)

type entryFile struct {
	srcPath    string
	file       *ast.File
	fileSet    *token.FileSet
	pkg        *types.Package
	opt        *option.GlobalOption
	docComment ast.CommentGroup
}

type parsedMethod struct {
	method    types.Object
	comments  []*ast.Comment
	notations []*notation
}

func (e *entryFile) getInterface() (*types.TypeName, error) {
	intf := findInterface(e.pkg.Scope(), "convergen")
	if intf == nil {
		return nil, fmt.Errorf(`"convergen" interface not found in %v`, e.srcPath)
	}
	return intf, nil
}

func (e *entryFile) parseDocComment(intf *types.TypeName) {
	docComment := astGetDocCommentOn(e.file, intf)
	if docComment == nil || len(docComment.List) == 0 {
		return
	}

	notations := make([]notation, 0)
	for _, comment := range docComment.List {
		m := reNotation.FindStringSubmatch(comment.Text)
	}
}

func (e *entryFile) parseLoki() error {
	intf, err := e.getInterface()
	if err != nil {
		return err
	}

	e.parseDocComment(intf)

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
