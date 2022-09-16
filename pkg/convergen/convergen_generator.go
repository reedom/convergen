package convergen

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"os"
	"regexp"

	"github.com/reedom/convergen/pkg/model"
)

var reGoBuildGen = regexp.MustCompile(`\s*//\s*((go:generate\b|build convergen\b)|\+build convergen)`)
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

func (p *Convergen) Generate() ([]*model.Function, error) {
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
		method:      method,
		docComment:  docComment,
		notations:   notations,
		dstVarStyle: model.DstVarReturn,
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

func (p *Convergen) CreateFunction(m *methodEntry) (*model.Function, error) {
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
			strct = ptr.Elem().Underlying().(*types.Struct)
		}
	}

	for i := 0; i < strct.NumFields(); i++ {
		f := strct.Field(i)
		a, err := p.createAssign(f, dstVar, strct, srcVar, m.method.Pos())
		if err == errNotFound {
			log.Printf("no assigment src found for %v.%v\n", p.getTypeSignature(dst.Type()), f.Name())
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

func (p *Convergen) createVar(v *types.Var, defName string) model.Var {
	mv := model.Var{Name: v.Name()}
	if mv.Name == "" {
		mv.Name = defName
	}

	p.parseVarType(v.Type(), &mv)
	return mv
}

func (p *Convergen) parseVarType(t types.Type, varModel *model.Var) {
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

func (p *Convergen) createAssign(dst *types.Var, dstVar model.Var, srcType types.Type, srcVar model.Var, pos token.Pos) (*model.Assignment, error) {
	name := dst.Name()

	named, ok := srcType.(*types.Named)
	if !ok {
		if ptr, ok := srcType.(*types.Pointer); ok {
			named, ok = ptr.Elem().(*types.Named)
		}
	}

	if named != nil {
		for i := 0; i < named.NumMethods(); i++ {
			m := named.Method(i)
			if name != m.Name() {
				continue
			}
			if types.AssignableTo(m.Type(), dst.Type()) {
				return &model.Assignment{
					LHS: fmt.Sprintf("%v.%v", dstVar.Name, name),
					RHS: model.SimpleField{Path: fmt.Sprintf("%v.%v()", srcVar.Name, m.Name())},
				}, nil
			}
			break
		}
	}

	strct, ok := srcType.Underlying().(*types.Struct)
	if !ok {
		return nil, fmt.Errorf("%v: src value is not a struct", p.fset.Position(pos))
	}

	for i := 0; i < strct.NumFields(); i++ {
		m := strct.Field(i)
		if name != m.Name() {
			continue
		}
		if types.AssignableTo(m.Type(), dst.Type()) {
			return &model.Assignment{
				LHS: fmt.Sprintf("%v.%v", dstVar.Name, name),
				RHS: model.SimpleField{Path: fmt.Sprintf("%v.%v", srcVar.Name, m.Name())},
			}, nil
		}
		break
	}

	return nil, errNotFound
}
