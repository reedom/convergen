package builder

import (
	"go/types"

	gmodel "github.com/reedom/convergen/pkg/generator/model"
	"github.com/reedom/convergen/pkg/logger"
	"github.com/reedom/convergen/pkg/option"
	"github.com/reedom/convergen/pkg/util"
)

func (p *FunctionBuilder) buildManipulator(m *option.Manipulator, src *types.Var, dst *types.Var, retError bool) (*gmodel.Manipulator, error) {
	if m == nil {
		return nil, nil
	}

	ret := &gmodel.Manipulator{}
	ret.Pkg, _ = p.imports.LookupName(m.Func.Pkg().Path())
	ret.Name = m.Func.Name()
	ret.RetError = m.RetError

	if ret.Pkg != "" && !m.Func.Exported() {
		return nil, logger.Errorf("%v: postprocess function %v is not exported", p.fset.Position(m.Pos), ret.FuncName())
	}

	if m.RetError && !retError {
		return nil, logger.Errorf("%v: cannot use postprocess function %v due to mismatch of returning error", p.fset.Position(m.Pos), ret.FuncName())
	}

	if !types.AssignableTo(util.DerefPtr(m.DstSide), util.DerefPtr(dst.Type())) {
		return nil, logger.Errorf("%v: postprocess function %v 1st arg type mismatch", p.fset.Position(m.Pos), ret.FuncName())
	}

	if !types.AssignableTo(util.DerefPtr(m.SrcSide), util.DerefPtr(src.Type())) {
		return nil, logger.Errorf("%v: postprocess function %v 2nd arg type mismatch", p.fset.Position(m.Pos), ret.FuncName())
	}

	ret.IsSrcPtr = util.IsPtr(m.SrcSide)
	ret.IsDstPtr = util.IsPtr(m.DstSide)

	return ret, nil
}
