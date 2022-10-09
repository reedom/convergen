package builder

import (
	"go/ast"
	"go/token"
	"go/types"

	bmodel "github.com/reedom/convergen/pkg/builder/model"
	gmodel "github.com/reedom/convergen/pkg/generator/model"
	"github.com/reedom/convergen/pkg/logger"
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

func (p *FunctionBuilder) CreateFunctions(methods []*bmodel.MethodEntry) ([]*gmodel.Function, error) {
	functions := make([]*gmodel.Function, len(methods))
	var err error
	for i, method := range methods {
		functions[i], err = p.CreateFunction(method)
		if err != nil {
			return nil, err
		}
	}
	return functions, nil
}

func (p *FunctionBuilder) CreateFunction(m *bmodel.MethodEntry) (*gmodel.Function, error) {
	comments := util.ToTextList(m.DocComment)
	src := m.SrcVar()
	dst := m.DstVar()

	if util.IsInvalidType(src.Type()) {
		return nil, logger.Errorf("%v: src type is not defined. make sure to be imported", p.fset.Position(src.Pos()))
	}
	if util.IsInvalidType(src.Type()) {
		return nil, logger.Errorf("%v: dst type is not defined. make sure to be imported", p.fset.Position(dst.Pos()))
	}
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
		if srcVar.External {
			return nil, logger.Errorf("%v: an external package type cannot be a receiver", p.fset.Position(m.Method.Pos()))
		}
		srcVar.Name = m.Opts.Receiver
	}

	var assignments []gmodel.Assignment
	var err error
	if m.Opts.Reverse {
		builder := newAssignmentBuilder(p, m, srcVar, dstVar)
		assignments, err = builder.build(src, dst)
	} else {
		builder := newAssignmentBuilder(p, m, dstVar, srcVar)
		assignments, err = builder.build(dst, src)
	}
	if err != nil {
		return nil, err
	}

	preProcess, err := p.buildManipulator(m.Opts.PreProcess, src, dst, m.RetError())
	if err != nil {
		return nil, err
	}
	postProcess, err := p.buildManipulator(m.Opts.PostProcess, src, dst, m.RetError())
	if err != nil {
		return nil, err
	}

	fn := &gmodel.Function{
		Name:        m.Method.Name(),
		Comments:    comments,
		Receiver:    m.Opts.Receiver,
		Src:         srcVar,
		Dst:         dstVar,
		DstVarStyle: m.Opts.Style,
		RetError:    m.RetError(),
		Assignments: assignments,
		PreProcess:  preProcess,
		PostProcess: postProcess,
	}

	return fn, nil
}

func (p *FunctionBuilder) createVar(v *types.Var, defName string) gmodel.Var {
	name := v.Name()
	if name == "" {
		name = defName
	}

	typ, isPtr := util.Deref(v.Type())
	return gmodel.Var{
		Name:     name,
		Type:     p.imports.TypeName(typ),
		Pointer:  isPtr,
		External: p.imports.IsExternal(typ),
	}
}
