package parser

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"regexp"

	"github.com/reedom/convergen/pkg/logger"
	"github.com/reedom/convergen/pkg/model"
	"golang.org/x/tools/go/loader"
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

	comments := make([]string, len(m.docComment.List))
	for i := range m.docComment.List {
		comments[i] = m.docComment.List[i].Text
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

	for i := 0; i < strct.NumFields(); i++ {
		f := strct.Field(i)
		a, err := p.createAssign(m.opts, f, dstVar, src.Type(), srcVar, m.method.Pos())
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

func (p *Parser) createAssign(opts options, dst *types.Var, dstVar model.Var, srcType types.Type, srcVar model.Var, pos token.Pos) (*model.Assignment, error) {
	name := dst.Name()
	dstVarName := fmt.Sprintf("%v.%v", dstVar.Name, name)

	if dstVar.PkgName != "" && !ast.IsExported(name) {
		logger.Printf("%v: skip %v.%v while it is not an exported field",
			p.fset.Position(dst.Pos()), dstVar.Name, dst.Name())
		return nil, errNotFound
	}

	// :skip notation
	if opts.shouldSkip(dstVarName) {
		logger.Printf("%v: skip %v.%v [%v]",
			p.fset.Position(dst.Pos()), dstVar.Name, dst.Name(), dst.Type().String())
		return &model.Assignment{
			LHS: dstVarName,
			RHS: model.SkipField{},
		}, nil
	}

	// Handle getters
	var a *model.Assignment
	if opts.getter {
		err := iterateMethods(srcType, func(m *types.Func) (done bool, err error) {
			if !opts.compareFieldName(name, m.Name()) {
				return
			}
			if srcVar.IsPkgExternal() && !ast.IsExported(m.Name()) {
				return
			}

			retTypes, ok := getMethodReturnTypes(m)
			if !ok || !compliesGetter(retTypes, false) {
				return
			}

			retType := retTypes.At(0).Type()
			if types.AssignableTo(retType, dst.Type()) {
				logger.Printf("%v: assignment found, %v.%v() [%v] to %v.%v [%v]",
					p.fset.Position(dst.Pos()), srcVar.Name, m.Name(), m.Type().Underlying().String(),
					dstVar.Name, dst.Name(), dst.Type().String())
				a = &model.Assignment{
					LHS: fmt.Sprintf("%v.%v", dstVar.Name, name),
					RHS: model.SimpleField{Path: fmt.Sprintf("%v.%v()", srcVar.Name, m.Name())},
				}
				return true, nil
			}

			// :stringer notation
			if opts.stringer && supportsStringer(retType, dst.Type()) {
				logger.Printf("%v: assignment found, %v.%v().String() to %v.%v [%v]",
					p.fset.Position(dst.Pos()), srcVar.Name, m.Name(),
					dstVar.Name, dst.Name(), dst.Type().String())
				a = &model.Assignment{
					LHS: fmt.Sprintf("%v.%v", dstVar.Name, name),
					RHS: model.SimpleField{Path: fmt.Sprintf("%v.%v().String()", srcVar.Name, m.Name())},
				}
				return true, nil
			}

			// :typecast notation
			if opts.typecast && types.ConvertibleTo(retType, dst.Type()) {
				logger.Printf("%v: assignment found, %v(%v.%v) to %v.%v",
					p.fset.Position(dst.Pos()), dst.Type().String(), srcVar.Name, dstVar.Name, dst.Name(),
					dstVar.Name, dst.Name())
				a = &model.Assignment{
					LHS: fmt.Sprintf("%v.%v", dstVar.Name, name),
					RHS: model.SimpleField{Path: fmt.Sprintf("%v(%v.%v)", dst.Type().String(), srcVar.Name, m.Name())},
				}
				return true, nil
			}

			return
		})
		if err != nil && err != errNotFound {
			return nil, err
		}
		if a != nil {
			return a, nil
		}
	}

	// Field name mapping
	err := iterateFields(srcType, func(f *types.Var) (done bool, err error) {
		if !opts.compareFieldName(name, f.Name()) {
			return
		}
		if srcVar.IsPkgExternal() && !ast.IsExported(f.Name()) {
			return
		}

		if types.AssignableTo(f.Type(), dst.Type()) {
			logger.Printf("%v: assignment found, %v.%v [%v] to %v.%v [%v]",
				p.fset.Position(dst.Pos()), srcVar.Name, f.Name(), f.Type().String(),
				dstVar.Name, dst.Name(), dst.Type().String())
			a = &model.Assignment{
				LHS: fmt.Sprintf("%v.%v", dstVar.Name, name),
				RHS: model.SimpleField{Path: fmt.Sprintf("%v.%v", srcVar.Name, f.Name())},
			}
			return true, nil
		}

		// :stringer notation
		if opts.stringer && supportsStringer(f.Type(), dst.Type()) {
			logger.Printf("%v: assignment found, %v.%v.String() to %v.%v [%v]",
				p.fset.Position(dst.Pos()), srcVar.Name, f.Name(), f.Type().String(),
				dstVar.Name, dst.Name(), dst.Type().String())
			a = &model.Assignment{
				LHS: fmt.Sprintf("%v.%v", dstVar.Name, name),
				RHS: model.SimpleField{Path: fmt.Sprintf("%v.%v.String()", srcVar.Name, f.Name())},
			}
			return true, nil
		}

		// :typecast notation
		if opts.typecast && types.ConvertibleTo(f.Type(), dst.Type()) {
			logger.Printf("%v: assignment found, %v(%v.%v) to %v.%v",
				p.fset.Position(dst.Pos()), dst.Type().String(), srcVar.Name, f.Name(), f.Type().String(),
				dstVar.Name, dst.Name())
			a = &model.Assignment{
				LHS: fmt.Sprintf("%v.%v", dstVar.Name, name),
				RHS: model.SimpleField{Path: fmt.Sprintf("%v(%v.%v)", dst.Type().String(), srcVar.Name, f.Name())},
			}
			return true, nil
		}
		return
	})
	if err == errNotFound {
		return nil, logger.Errorf("%v: src value is not a struct", p.fset.Position(pos))
	} else if err != nil {
		return nil, err
	}
	if a != nil {
		return a, nil
	}

	logger.Printf("%v: no assignment for %v.%v [%v]",
		p.fset.Position(dst.Pos()), dstVar.Name, dst.Name(), dst.Type().String())
	return &model.Assignment{
		LHS: fmt.Sprintf("%v.%v", dstVar.Name, name),
		RHS: model.NoMatchField{},
	}, nil
}

func compliesGetter(retTypes *types.Tuple, returnsError bool) bool {
	num := retTypes.Len()
	if num == 0 || 2 < num {
		return false
	}
	return num == 1 || returnsError && isErrorType(retTypes.At(1).Type())
}

var stringer *types.Interface

func supportsStringer(src types.Type, dst types.Type) bool {
	strType := types.Universe.Lookup("string").Type()
	if !types.AssignableTo(strType, dst) {
		return false
	}

	if stringer == nil {
		conf := loader.Config{ParserMode: parser.ParseComments}
		conf.Import("fmt")
		lprog, _ := conf.Load()
		t := lprog.Package("fmt").Pkg.Scope().Lookup("Stringer").Type()
		stringer = t.Underlying().(*types.Interface)
	}

	return types.Implements(src, stringer)
}
