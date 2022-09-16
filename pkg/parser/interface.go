package parser

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"
	"unicode"

	"github.com/reedom/convergen/pkg/model"
	"github.com/reedom/convergen/pkg/parser/option"
)

const intfName = "Convergen"

type intfEntry struct {
	intf       *types.TypeName
	docComment *ast.CommentGroup
	notations  []*ast.Comment
}

type funcEntry struct {
	fun       *types.Object
	notations []*ast.Comment
}

// extractIntfEntry looks up the setup interface with the name of intfName("Convergen") and also
// parses convergen notations from the interface's doc comment.
func (p *Parser) extractIntfEntry() (*intfEntry, error) {
	//intf, err := astFindIntfEntry(p.file, p.fset, p.pkg.Types.Scope(), intfName)
	intf, err := p.findIntfEntry(p.pkg.Types.Scope(), intfName)
	if err != nil {
		return nil, err
	}

	docComment := getDocCommentOn(p.file, intf)
	notations := astExtractMatchComments(docComment, reNotation)
	err = p.parseIntfNotations(notations)
	if err != nil {
		return nil, err
	}

	return &intfEntry{
		intf:       intf,
		docComment: docComment,
		notations:  notations,
	}, nil
}

// findIntfEntry looks up the setup interface with the specific name and returns it.
func (p *Parser) findIntfEntry(scope *types.Scope, name string) (*types.TypeName, error) {
	if typ := scope.Lookup(name); typ != nil {
		if intf, ok := typ.(*types.TypeName); ok {
			if _, ok = intf.Type().Underlying().(*types.Interface); ok {
				return intf, nil
			}
			return nil, fmt.Errorf("%v: %v it not interface", p.fset.Position(p.file.Package), name)
		}
	}
	return nil, fmt.Errorf("%v: %v interface not found", p.fset.Position(p.file.Package), name)
}

func (p *Parser) parseIntfNotations(notations []*ast.Comment) error {
	for _, n := range notations {
		m := reNotation.FindStringSubmatch(n.Text)
		var args []string
		if len(m) == 3 {
			args = strings.Fields(m[2])
		}

		switch m[1] {
		case "opt:style":
			if args == nil {
				return fmt.Errorf("%v: needs <style> arg", p.fset.Position(n.Pos()))
			} else if style, ok := model.NewDstVarStyleFromValue(args[0]); !ok {
				return fmt.Errorf("%v: invalid <style> arg", p.fset.Position(n.Pos()))
			} else {
				p.opt.Style = style
			}
		case "opt:match":
			if args == nil {
				return fmt.Errorf("%v: needs <order> arg", p.fset.Position(n.Pos()))
			} else if order, ok := option.FieldMatchOrderFromValue(args[0]); !ok {
				return fmt.Errorf("%v: invalid <order> arg", p.fset.Position(n.Pos()))
			} else {
				p.opt.FieldMatchOrder = order
			}
		case "opt:nocase":
			p.opt.ExactCase = true
		case "rcv":
			if args == nil {
				return fmt.Errorf("%v: needs name for the receiver", p.fset.Position(n.Pos()))
			} else if !isValidIdentifier(args[0]) {
				return fmt.Errorf("%v: invalid ident", p.fset.Position(n.Pos()))
			}
			p.opt.Receiver = args[0]
		case "skip":
			if args == nil {
				return fmt.Errorf("%v: needs <field> arg", p.fset.Position(n.Pos()))
			}
			matcher, err := option.NewIdentMatcher(args[0], p.opt.ExactCase)
			if err != nil {
				return fmt.Errorf("%v: invalid <field> arg", p.fset.Position(n.Pos()))
			}
			p.opt.Skip = append(p.opt.Skip, matcher)
		case "map":
			if len(args) < 2 {
				return fmt.Errorf("%v: needs <src> <dst> args", p.fset.Position(n.Pos()))
			}
			matcher, err := option.NewFieldMatcher(args[0], args[1], p.opt.ExactCase)
			if err != nil {
				return fmt.Errorf("%v: invalid <field> arg", p.fset.Position(n.Pos()))
			}
			p.opt.Matchers = append(p.opt.Matchers, matcher)
		case "conv":
			if len(args) < 2 {
				return fmt.Errorf("%v: needs <src> <dst> args", p.fset.Position(n.Pos()))
			}
			scope, obj := p.pkg.Types.Scope().LookupParent(args[0], n.Pos())
			fmt.Printf("@@@ lookup %v, %#v, %#v\n", args[0], scope, obj)
			obj = p.pkg.Types.Scope().Lookup(args[0])
			fmt.Printf("@@@ lookup %v, %#v\n", args[0], obj)
			inner := p.pkg.Types.Scope().Innermost(n.Pos())
			scope, obj = inner.LookupParent("domain", n.Pos())
			fmt.Printf("@@@! lookup %v, %#v, %#v\n", args[0], scope, obj)
		default:
			fmt.Printf("@@@ notation %v\n", m[1])
		}
	}
	return nil
}

func (p *Parser) parseIt(scope *types.Scope, at *types.Var) {
	signature, ok := at.Type().(*types.Signature)
	if ok {
		fmt.Printf("--- NAME: %v\n", signature.String())
	}
	tt, err := findField(at.Pkg(), at.Type(), lookupFieldOpt{
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
