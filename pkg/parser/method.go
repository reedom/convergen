package parser

import (
	"errors"
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"regexp"

	"github.com/reedom/convergen/pkg/logger"
	"github.com/reedom/convergen/pkg/model"
)

var reGoBuildGen = regexp.MustCompile(`\s*//\s*(go:(generate\b|build convergen\b)|\+build convergen)`)
var ErrAbort = errors.New("abort")

type methodEntry struct {
	method     types.Object // Also a *types.Signature
	opts       options
	docComment *ast.CommentGroup
	src        *types.Tuple
	dst        *types.Tuple
}

func (p *Parser) parseMethods(intf *intfEntry) ([]*model.Function, error) {
	iface := intf.intf.Type().Underlying().(*types.Interface)
	opts := intf.opts.copyForMethods()
	mset := types.NewMethodSet(iface)
	methods := make([]*methodEntry, 0)
	for i := 0; i < mset.Len(); i++ {
		method, err := p.extractMethodEntry(mset.At(i).Obj(), opts)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
			continue
		}
		methods = append(methods, method)
	}
	if len(methods) < mset.Len() {
		return nil, ErrAbort
	}

	functions := make([]*model.Function, 0)
	for _, method := range methods {
		fn, err := p.CreateFunction(method)
		if err != nil {
			return nil, err
		}
		functions = append(functions, fn)
	}

	return functions, nil
}

func (p *Parser) extractMethodEntry(method types.Object, opts options) (*methodEntry, error) {
	signature, ok := method.Type().(*types.Signature)
	if !ok {
		return nil, logger.Errorf(`%v: expected signature but %#v`, p.fset.Position(method.Pos()), method)
	}

	if signature.Params().Len() == 0 {
		return nil, logger.Errorf(`%v: method must have one or more arguments as copy source`, p.fset.Position(method.Pos()))
	}
	if signature.Results().Len() == 0 {
		return nil, logger.Errorf(`%v: method must have one or more return values as copy destination`, p.fset.Position(method.Pos()))
	}

	docComment, cleanUp := getDocCommentOn(p.file, method)
	notations := astExtractMatchComments(docComment, reNotation)
	err := p.parseNotationInComments(notations, validOpsMethod, &opts)
	if err != nil {
		return nil, err
	}

	cleanUp()

	return &methodEntry{
		method:     method,
		opts:       opts,
		docComment: docComment,
		src:        signature.Params(),
		dst:        signature.Results(),
	}, nil
}

func (p *Parser) getTypeSignature(t types.Type) string {
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

func (p *Parser) CreateFunction(m *methodEntry) (*model.Function, error) {
	sig := m.method.Type().(*types.Signature)
	hasError := 1 < sig.Results().Len() &&
		isErrorType(sig.Results().At(sig.Results().Len()-1).Type())

	var comments []string
	if m.docComment != nil {
		comments = make([]string, len(m.docComment.List))
		for i := range m.docComment.List {
			comments[i] = m.docComment.List[i].Text
		}
	}
	src := sig.Params().At(0)
	dst := sig.Results().At(0)

	srcVar := p.createVar(src, "src")
	if m.opts.receiver != "" {
		if srcVar.PkgName != "" {
			return nil, logger.Errorf("%v: an external package type cannot be a receiver", p.fset.Position(m.method.Pos()))
		}
		srcVar.Name = m.opts.receiver
	}
	dstVar := p.createVar(dst, "dst")

	assignments := make([]*model.Assignment, 0)
	strct, ok := dst.Type().Underlying().(*types.Struct)
	if !ok {
		if ptr, ok := dst.Type().Underlying().(*types.Pointer); ok {
			strct, ok = ptr.Elem().Underlying().(*types.Struct)
			if !ok {
				return nil, logger.Errorf("%v: dst type should be a struct but %v", p.fset.Position(dst.Pos()), dst.Type().String())
			}
		}
	}

	srcStrct := srcStructEntry{
		Var:   srcVar,
		strct: src,
	}
	builder := newAssignmentBuilder(p, m.method.Pos(), m.opts, srcStrct)
	for i := 0; i < strct.NumFields(); i++ {
		dstField := dstFieldEntry{
			Var:   dstVar,
			field: strct.Field(i),
		}
		a, err := builder.create(dstField)
		if err == errNotFound {
			continue
		}
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, a)
	}

	fn := &model.Function{
		Name:         m.method.Name(),
		Comments:     comments,
		Receiver:     m.opts.receiver,
		Src:          srcVar,
		Dst:          dstVar,
		DstVarStyle:  m.opts.Style,
		ReturnsError: hasError,
		Assignments:  assignments,
	}

	return fn, nil
}

func (p *Parser) createVar(v *types.Var, defName string) model.Var {
	mv := model.Var{Name: v.Name()}
	if mv.Name == "" {
		mv.Name = defName
	}

	p.parseVarType(v.Type(), &mv)
	return mv
}

func (p *Parser) parseVarType(t types.Type, varModel *model.Var) {
	switch typ := t.(type) {
	case *types.Pointer:
		varModel.Pointer = true
		p.parseVarType(typ.Elem(), varModel)
	case *types.Named:
		if pkgName, ok := p.imports[typ.Obj().Pkg().Path()]; ok {
			varModel.PkgName = pkgName
		}
		varModel.Type = typ.Obj().Name()
	case *types.Basic:
		varModel.Type = typ.Name()
	default:
		panic(t)
	}
}
