package builder

import (
	"fmt"
	"go/types"

	gmodel "github.com/reedom/convergen/v8/pkg/generator/model"
	"github.com/reedom/convergen/v8/pkg/logger"
	"github.com/reedom/convergen/v8/pkg/option"
	"github.com/reedom/convergen/v8/pkg/util"
)

// buildManipulator builds a gmodel.Manipulator based on the given Manipulator
// option, source and destination variables, and retError.
// It checks that the function is valid and the types of its arguments match
// the source and destination variables.
// If the Manipulator is nil, it returns nil and no error.
func (p *FunctionBuilder) buildManipulator(
	m *option.Manipulator,
	src, dst *types.Var,
	additionalArgs []*types.Var,
	retError bool,
) (*gmodel.Manipulator, error) {
	if m == nil {
		return nil, nil
	}

	ret := &gmodel.Manipulator{}
	ret.Pkg, _ = p.imports.LookupName(m.Func.Pkg().Path())
	ret.Name = m.Func.Name()
	ret.RetError = m.RetError

	if ret.Pkg != "" && !m.Func.Exported() {
		return nil, logger.Errorf("%v: manipulator function %v is not exported", p.fset.Position(m.Pos), ret.FuncName())
	}

	if m.RetError && !retError {
		return nil, logger.Errorf("%v: cannot use manipulator function %v due to mismatch of returning error", p.fset.Position(m.Pos), ret.FuncName())
	}

	if !types.AssignableTo(util.DerefPtr(m.DstSide), util.DerefPtr(dst.Type())) {
		return nil, logger.Errorf("%v: manipulator function %v 1st arg type mismatch", p.fset.Position(m.Pos), ret.FuncName())
	}

	if !types.AssignableTo(util.DerefPtr(m.SrcSide), util.DerefPtr(src.Type())) {
		return nil, logger.Errorf("%v: manipulator function %v 2nd arg type mismatch", p.fset.Position(m.Pos), ret.FuncName())
	}

	if 0 < len(m.AdditionalArgs) {
		if len(m.AdditionalArgs) != len(additionalArgs) {
			return nil, logger.Errorf("%v: manipulator function %v additional args count mismatch", p.fset.Position(m.Pos), ret.FuncName())
		}
		for i, arg := range m.AdditionalArgs {
			if !types.AssignableTo(arg, additionalArgs[i].Type()) {
				return nil, logger.Errorf("%v: manipulator function %v %s arg type mismatch", p.fset.Position(m.Pos), ret.FuncName(), ordinalNumber(i+3))
			}
		}
		ret.HasAdditionalArgs = true
	}
	ret.IsSrcPtr = util.IsPtr(m.SrcSide)
	ret.IsDstPtr = util.IsPtr(m.DstSide)

	return ret, nil
}

func ordinalNumber(n int) string {
	if 11 <= n && n <= 13 {
		return fmt.Sprintf("%dth", n)
	}
	switch n % 10 {
	case 1:
		return fmt.Sprintf("%dst", n)
	case 2:
		return fmt.Sprintf("%dnd", n)
	case 3:
		return fmt.Sprintf("%drd", n)
	default:
		return fmt.Sprintf("%dth", n)
	}
}
