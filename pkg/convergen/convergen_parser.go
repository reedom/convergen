package convergen

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"
	"unicode"

	"github.com/reedom/convergen/pkg/convergen/option"
)

const intfName = "Convergen"

type intfEntry struct {
	intf      *types.TypeName
	notations []*ast.Comment
}

type methodEntry struct {
	name      string
	method    types.Object
	notations []*ast.Comment
	src       *types.Tuple
	dst       *types.Tuple
}

type funcEntry struct {
	fun       *types.Object
	notations []*ast.Comment
}

func (p *Convergen) ExtractIntfEntry() error {
	var intf *types.TypeName
	intf = findInterface(p.pkg.Types.Scope(), intfName)
	if intf == nil {
		return fmt.Errorf("%v: %v interface not found", p.fset.Position(p.file.Package), intfName)
	}

	docComment := astGetDocCommentOn(p.file, intf)
	notations := astExtractMatchComments(docComment, reNotation)
	err := p.parseIntfNotations(notations)
	if err != nil {
		return err
	}

	p.intfEntry = &intfEntry{
		intf:      intf,
		notations: notations,
	}
	return nil
}

func (p *Convergen) extractIntfMethods(intf *types.TypeName) ([]*methodEntry, error) {
	iface, ok := intf.Type().Underlying().(*types.Interface)
	if !ok {
		return nil, fmt.Errorf("%v: %v is not interface", p.fset.Position(intf.Pos()), intf.Name())
	}

	methods := make([]*methodEntry, 0)
	mset := types.NewMethodSet(iface)
	for i := 0; i < mset.Len(); i++ {
		method, err := p.extractMethodEntry(mset.At(i).Obj())
		if err != nil {
			return nil, err
		}
		methods = append(methods, method)
	}
	return methods, nil
}

func (p *Convergen) getTypeSignature(t types.Type) string {
	switch typ := t.(type) {
	case *types.Pointer:
		return "*" + p.getTypeSignature(typ.Elem())
	case *types.Named:
		pkgPath := typ.Obj().Pkg().Path()
		pkgName, ok := p.imports[pkgPath]
		fmt.Printf("XXX %v, %v\n", pkgPath, pkgName)
		if ok {
			return fmt.Sprintf("%v.%v", pkgName, typ.Obj().Name())
		}
		return typ.Obj().Name()
	}
	return "???"
}

func (p *Convergen) extractMethodEntry(method types.Object) (*methodEntry, error) {
	signature, ok := method.Type().(*types.Signature)
	if !ok {
		return nil, fmt.Errorf(`%v: expected signature but %#v`, p.fset.Position(method.Pos()), method)
	}

	if signature.Params().Len() == 0 {
		return nil, fmt.Errorf(`%v: method must have one or more arguments as copy source`, p.fset.Position(method.Pos()))
	}
	if signature.Results().Len() == 0 {
		return nil, fmt.Errorf(`%v: method must have one or more return values as copy destination`, p.fset.Position(method.Pos()))
	}

	docComment := astGetDocCommentOn(p.file, method)
	notations := astExtractMatchComments(docComment, reNotation)
	fmt.Printf("### %v\n", p.getTypeSignature(signature.Params().At(0).Type()))
	p.parseIt(p.pkg.Types.Scope(), signature.Params().At(0))

	return &methodEntry{
		name:      method.Name(),
		method:    method,
		notations: notations,
		src:       signature.Params(),
		dst:       signature.Results(),
	}, nil
}

func (p *Convergen) lookupField(path string) error {
	parts := strings.Split(path, ".")
	switch len(parts) {
	case 1:
		// Must be a field name.

	}
	return nil
}

func (p *Convergen) parseIntfNotations(notations []*ast.Comment) error {
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
			} else if style, ok := option.NewDstVarStyleFromValue(args[0]); !ok {
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

type varEntry struct {
	v *types.Var
}

type typeEntry struct {
	isPointer bool
	elem      *typeEntry
}

func (p *Convergen) parseIt(scope *types.Scope, at *types.Var) {
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
