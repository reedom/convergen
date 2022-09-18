package parser

import (
	"fmt"
	"go/ast"
	"go/types"
	"unicode"

	"github.com/reedom/convergen/pkg/logger"
)

const intfName = "Convergen"

type intfEntry struct {
	intf *types.TypeName
	opts options
}

type funcEntry struct {
	fun       *types.Object
	notations []*ast.Comment
}

// extractIntfEntry looks up the setup interface with the name of intfName("Convergen") and also
// parses convergen notations from the interface's doc comment.
func (p *Parser) extractIntfEntry() (*intfEntry, error) {
	intf, err := p.findIntfEntry(p.pkg.Types.Scope(), intfName)
	if err != nil {
		return nil, err
	}

	docComment, cleanUp := getDocCommentOn(p.file, intf)
	notations := astExtractMatchComments(docComment, reNotation)
	docComment.List = nil
	cleanUp()

	opts := newOptions()
	err = p.parseNotationInComments(notations, validOpsIntf, &opts)
	if err != nil {
		return nil, err
	}

	return &intfEntry{
		intf: intf,
		opts: opts,
	}, nil
}

// findIntfEntry looks up the setup interface with the specific name and returns it.
func (p *Parser) findIntfEntry(scope *types.Scope, name string) (*types.TypeName, error) {
	if typ := scope.Lookup(name); typ != nil {
		if intf, ok := typ.(*types.TypeName); ok {
			if _, ok = intf.Type().Underlying().(*types.Interface); ok {
				logger.Printf("%v: %v interface found", p.fset.Position(intf.Pos()), name)
				return intf, nil
			}
			return nil, logger.Errorf("%v: %v it not interface", p.fset.Position(intf.Pos()), name)
		}
	}
	return nil, logger.Errorf("%v: %v interface not found", p.fset.Position(p.file.Package), name)
}

func (p *Parser) parseIt(scope *types.Scope, at *types.Var) {
	signature, ok := at.Type().(*types.Signature)
	if ok {
		fmt.Printf("--- NAME: %v\n", signature.String())
	}
	tt, err := findField(p.fset, at.Pkg(), at.Type(), lookupFieldOpt{
		exactCase:     true,
		supportsError: false,
		pattern:       "Category.ID",
	})
	if err != nil && err != errNotFound {
		panic(err)
	}
	fmt.Printf("--- FOUND: %v\n", tt)

	switch typ := at.Type().(type) {
	case *types.Named:
		fmt.Printf("--- methods: %v\n", typ.NumMethods())
		for i := 0; i < typ.NumMethods(); i++ {
			method := typ.Method(i)
			fmt.Printf("--- method: %v\n", method.Name())
		}
	case *types.Pointer:
		switch typ2 := typ.Elem().(type) {
		case *types.Named:
			fmt.Printf("--- methods: %v\n", typ2.NumMethods())
			for i := 0; i < typ2.NumMethods(); i++ {
				method := typ2.Method(i)
				fmt.Printf("--- method: %v\n", method.Name())

			}
			fmt.Printf("--- ul: %#v\n\n", typ2.Underlying())
		default:
			fmt.Println("----- uh??")
			fmt.Printf("@@@ parseIt: %#v\n, %#v\n", typ.Elem().String(), at.Type())
		}
	default:
		fmt.Println("----- uh?")
		fmt.Printf("@@@ parseIt: %#v\n, %#v\n", at.Type().String(), at.Type())
	}
}

func isValidIdentifier(id string) bool {
	for i, r := range id {
		if !unicode.IsLetter(r) &&
			!(i > 0 && unicode.IsDigit(r)) {
			return false
		}
	}
	return id != ""
}
