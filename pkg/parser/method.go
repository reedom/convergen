package parser

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"regexp"

	"github.com/reedom/convergen/pkg/logger"
	"github.com/reedom/convergen/pkg/model"
)

var reGoBuildGen = regexp.MustCompile(`\s*//\s*(go:(generate\b|build convergen\b)|\+build convergen)`)
var ErrAbort = errors.New("abort")

type methodEntry struct {
	method      types.Object // Also a *types.Signature
	docComment  *ast.CommentGroup
	notations   []*ast.Comment
	dstVarStyle model.DstVarStyle
	receiver    string
	src         *types.Tuple
	dst         *types.Tuple
}

func (p *Parser) parseMethods(intf *intfEntry) ([]*model.Function, error) {
	iface := intf.intf.Type().Underlying().(*types.Interface)
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

func (p *Parser) extractMethodEntry(method types.Object) (*methodEntry, error) {
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
	cleanUp()

	return &methodEntry{
		method:      method,
		docComment:  docComment,
		notations:   notations,
		dstVarStyle: model.DstVarReturn,
		src:         signature.Params(),
		dst:         signature.Results(),
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
		a, err := p.createAssign(f, dstVar, src.Type(), srcVar, m.method.Pos())
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
		Receiver:     m.receiver,
		Src:          srcVar,
		Dst:          dstVar,
		DstVarStyle:  model.DstVarReturn,
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

func isMethodAssignableTo(m *types.Func, dst types.Type, returnsError bool) bool {
	sig := m.Type().(*types.Signature)
	num := sig.Results().Len()
	if num == 0 || 2 < num {
		return false
	}

	r := sig.Results().At(0).Type()
	if !types.AssignableTo(r, dst) {
		return false
	}
	return num == 1 || returnsError && isErrorType(sig.Results().At(1).Type())
}

func (p *Parser) createAssign(dst *types.Var, dstVar model.Var, srcType types.Type, srcVar model.Var, pos token.Pos) (*model.Assignment, error) {
	name := dst.Name()

	named, ok := srcType.(*types.Named)
	if !ok {
		if ptr, ok := srcType.(*types.Pointer); ok {
			named, ok = ptr.Elem().(*types.Named)
		}
	}

	var strct *types.Struct

	if named != nil {
		strct, ok = named.Underlying().(*types.Struct)
		if !ok {
			if ptr, ok := named.Underlying().(*types.Pointer); ok {
				strct, ok = ptr.Elem().(*types.Struct)
			}
		}

		for i := 0; i < named.NumMethods(); i++ {
			m := named.Method(i)
			if name != m.Name() {
				continue
			}

			if isMethodAssignableTo(m, dst.Type(), false) {
				logger.Printf("%v: assignment found, %v.%v() [%v] to %v.%v [%v]",
					p.fset.Position(dst.Pos()), srcVar.Name, m.Name(), m.Type().Underlying().String(),
					dstVar.Name, dst.Name(), dst.Type().String())
				return &model.Assignment{
					LHS: fmt.Sprintf("%v.%v", dstVar.Name, name),
					RHS: model.SimpleField{Path: fmt.Sprintf("%v.%v()", srcVar.Name, m.Name())},
				}, nil
			}
			break
		}
	}

	if strct == nil {
		strct, ok = srcType.Underlying().(*types.Struct)
		if !ok {
			return nil, logger.Errorf("%v: src value is not a struct", p.fset.Position(pos))
		}
	}

	for i := 0; i < strct.NumFields(); i++ {
		f := strct.Field(i)
		if name != f.Name() {
			continue
		}
		if types.AssignableTo(f.Type(), dst.Type()) {
			logger.Printf("%v: assignment found, %v.%v [%v] to %v.%v [%v]",
				p.fset.Position(dst.Pos()), srcVar.Name, f.Name(), f.Type().String(),
				dstVar.Name, dst.Name(), dst.Type().String())
			return &model.Assignment{
				LHS: fmt.Sprintf("%v.%v", dstVar.Name, name),
				RHS: model.SimpleField{Path: fmt.Sprintf("%v.%v", srcVar.Name, f.Name())},
			}, nil
		}
		break
	}

	logger.Printf("%v: no assignment for %v.%v [%v]",
		p.fset.Position(dst.Pos()),
		dstVar.Name, dst.Name(), dst.Type().String())
	return nil, errNotFound
}
