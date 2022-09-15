package convergen

import (
	"errors"
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"regexp"
	"strings"

	. "github.com/dave/jennifer/jen"
	"github.com/reedom/convergen/pkg/convergen/option"
)

var reGoBuildGen = regexp.MustCompile(`\s*//\s*((go:generate\b|build convergen\b)|\+build convergen)`)
var ErrAbort = errors.New("abort")

type methodEntry struct {
	name        string
	method      types.Object // Also a *types.Signature
	notations   []*ast.Comment
	dstVarStyle option.DstVarStyle
	receiver    string
	src         *types.Tuple
	dst         *types.Tuple
}

func (p *Convergen) Generate(filePath string) error {
	jen := NewFile(filePath)

	iface := p.intfEntry.intf.Type().Underlying().(*types.Interface)
	mset := types.NewMethodSet(iface)
	methods := make([]*methodEntry, 0)
	for i := 0; i < mset.Len(); i++ {
		method, err := p.extractMethodEntry(mset.At(i).Obj())
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
			continue
		}
		methods = append(methods, method)
	}
	if len(methods) < mset.Len() {
		return ErrAbort
	}

	for _, method := range methods {
		if err := p.GenerateMethod(jen, method); err != nil {
			return err
		}
	}

	return nil
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

	return &methodEntry{
		name:        method.Name(),
		method:      method,
		notations:   notations,
		dstVarStyle: option.DstVarReturn,
		src:         signature.Params(),
		dst:         signature.Results(),
	}, nil
}

func (p *Convergen) getTypeSignature(t types.Type) string {
	switch typ := t.(type) {
	case *types.Pointer:
		return "*" + p.getTypeSignature(typ.Elem())
	case *types.Named:
		pkgPath := typ.Obj().Pkg().Path()
		pkgName, ok := p.imports[pkgPath]
		if ok {
			return fmt.Sprintf("%v.%v", pkgName, typ.Obj().Name())
		}
		return typ.Obj().Name()
	case *types.Basic:
		return typ.Name()
	}
	panic(t)
}

func (p *Convergen) GenerateMethod(jen *File, m *methodEntry) error {
	fn := jen.Func()

	sig := m.method.Type().(*types.Signature)
	src := sig.Params().At(0)
	dst := sig.Results().At(0)

	// Receiver
	srcType := p.getTypeSignature(src.Type())
	if m.receiver != "" {
		fn.Add(Params(Id(m.receiver).Id(srcType)))
	}

	// func name
	fn.Add(Id(m.method.Name()))

	// func args
	args := make([]Code, 0)
	dstType := p.getTypeSignature(dst.Type())
	if m.dstVarStyle == option.DstVarArg {
		args = append(
			args,
			Id("dst").Op("*").Id(strings.Replace(dstType, "*", "", 1)),
		)
	}
	if m.receiver == "" {
		args = append(
			args,
			Id("src").Op("*").Id(strings.Replace(srcType, "*", "", 1)),
		)
	}
	fn.Add(Params(args...))

	// func return value
	if m.dstVarStyle == option.DstVarReturn {
		fn.Add(Id(dstType))
	}

	fn.Add(Block(
		Return(Nil()),
	))
	fmt.Println(fn.GoString())
	return nil
}
