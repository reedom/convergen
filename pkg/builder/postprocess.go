package builder

import (
	"go/types"

	"github.com/reedom/convergen/pkg/generator/model"
	"github.com/reedom/convergen/pkg/logger"
	"github.com/reedom/convergen/pkg/util"
)

func (p *FunctionBuilder) buildPostProcess(m *MethodEntry, src *types.Var, dst *types.Var, returnsError bool) (*model.Manipulator, error) {
	if m.Opts.PostProcess == nil {
		return nil, nil
	}

	pp := m.Opts.PostProcess

	ret := &model.Manipulator{}
	ret.Pkg, _ = p.imports.LookupName(pp.Func.Pkg().Path())
	ret.Name = pp.Func.Name()
	ret.ReturnsError = pp.ReturnsError

	if ret.Pkg != "" && !pp.Func.Exported() {
		return nil, logger.Errorf("%v: postprocess function %v is not exported", p.fset.Position(pp.Pos), ret.FuncName())
	}

	if pp.ReturnsError != returnsError {
		return nil, logger.Errorf("%v: cannot use postprocess function %v due to mismatch of returning error", p.fset.Position(pp.Pos), ret.FuncName())
	}

	if !types.AssignableTo(util.DerefPtr(pp.DstSide), util.DerefPtr(dst.Type())) {
		return nil, logger.Errorf("%v: postprocess function %v 1st arg type mismatch", p.fset.Position(pp.Pos), ret.FuncName())
	}

	if !types.AssignableTo(util.DerefPtr(pp.SrcSide), util.DerefPtr(src.Type())) {
		return nil, logger.Errorf("%v: postprocess function %v 2nd arg type mismatch", p.fset.Position(pp.Pos), ret.FuncName())
	}

	ret.IsSrcPtr = util.IsPtr(pp.SrcSide)
	ret.IsDstPtr = util.IsPtr(pp.DstSide)

	return ret, nil
}
