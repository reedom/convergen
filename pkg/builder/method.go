package builder

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/reedom/convergen/pkg/logger"
	"github.com/reedom/convergen/pkg/model"
	"github.com/reedom/convergen/pkg/option"
	"github.com/reedom/convergen/pkg/util"
	"golang.org/x/tools/go/packages"
)

type FunctionBuilder struct {
	file    *ast.File
	fset    *token.FileSet
	pkg     *packages.Package
	imports util.ImportNames
}

func NewFunctionBuilder(
	file *ast.File,
	fset *token.FileSet,
	pkg *packages.Package,
	imports util.ImportNames,
) *FunctionBuilder {
	return &FunctionBuilder{
		file:    file,
		fset:    fset,
		pkg:     pkg,
		imports: imports,
	}
}

type MethodEntry struct {
	Method     types.Object // Also a *types.Signature
	Opts       option.Options
	DocComment *ast.CommentGroup
	Src        *types.Tuple
	Dst        *types.Tuple
}

func (p *FunctionBuilder) CreateFunctions(methods []*MethodEntry) ([]*model.Function, error) {
	functions := make([]*model.Function, len(methods))
	var err error
	for i, method := range methods {
		functions[i], err = p.CreateFunction(method)
		if err != nil {
			return nil, err
		}
	}
	return functions, nil
}

func (p *FunctionBuilder) CreateFunction(m *MethodEntry) (*model.Function, error) {
	sig := m.Method.Type().(*types.Signature)
	comments := util.ToTextList(m.DocComment)
	returnsError := 1 < sig.Results().Len() &&
		util.IsErrorType(sig.Results().At(sig.Results().Len()-1).Type())

	src := sig.Params().At(0)
	dst := sig.Results().At(0)

	if !util.IsStructType(util.DerefPtr(src.Type())) {
		return nil, logger.Errorf("%v: src type should be a struct but %v", p.fset.Position(dst.Pos()), src.Type().Underlying().String())
	}
	if !util.IsStructType(util.DerefPtr(dst.Type())) {
		return nil, logger.Errorf("%v: dst type should be a struct but %v", p.fset.Position(dst.Pos()), dst.Type().Underlying().String())
	}

	srcDefName := "src"
	dstDefName := "dst"
	if m.Opts.Reverse {
		srcDefName, dstDefName = dstDefName, srcDefName
	}

	srcVar := p.createVar(src, srcDefName)
	dstVar := p.createVar(dst, dstDefName)

	if m.Opts.Receiver != "" {
		if srcVar.IsPkgExternal() {
			return nil, logger.Errorf("%v: an external package type cannot be a receiver", p.fset.Position(m.Method.Pos()))
		}
		srcVar.Name = m.Opts.Receiver
	}

	builder := newAssignmentBuilder(p, m.Method.Pos(), m.Opts)
	var assignments []*model.Assignment
	var err error
	if m.Opts.Reverse {
		assignments, err = builder.build(dstVar, dst, srcVar, src.Type())
	} else {
		assignments, err = builder.build(srcVar, src, dstVar, dst.Type())
	}
	if err != nil {
		return nil, err
	}

	postProcess, err := p.buildPostProcess(m, src, dst, returnsError)
	if err != nil {
		return nil, err
	}

	fn := &model.Function{
		Name:         m.Method.Name(),
		Comments:     comments,
		Receiver:     m.Opts.Receiver,
		Src:          srcVar,
		Dst:          dstVar,
		DstVarStyle:  m.Opts.Style,
		ReturnsError: returnsError,
		Assignments:  assignments,
		PostProcess:  postProcess,
	}

	return fn, nil
}

func (p *FunctionBuilder) createVar(v *types.Var, defName string) model.Var {
	mv := model.Var{Name: v.Name()}
	if mv.Name == "" {
		mv.Name = defName
	}

	p.parseVarType(v.Type(), &mv)
	return mv
}

func (p *FunctionBuilder) parseVarType(t types.Type, varModel *model.Var) {
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
