package model

import (
	"go/ast"
	"go/types"

	"github.com/reedom/convergen/pkg/generator/model"
	"github.com/reedom/convergen/pkg/option"
	"github.com/reedom/convergen/pkg/util"
)

type MethodsInfo struct {
	Marker  string
	Methods []*MethodEntry
}

type MethodEntry struct {
	Method     types.Object
	Opts       option.Options
	DocComment *ast.CommentGroup
}

func (m *MethodEntry) Name() string {
	return m.Method.Name()
}

func (m *MethodEntry) Recv() types.Type {
	if m.Opts.Receiver == "" {
		return nil
	}

	sig := m.Method.Type().(*types.Signature)
	if sig.Params().Len() == 0 {
		return nil
	}
	return sig.Params().At(0).Type()
}

func (m *MethodEntry) Args() []types.Type {
	var list []types.Type

	sig := m.Method.Type().(*types.Signature)
	params := sig.Params()
	results := sig.Results()

	if m.Opts.Style == model.DstVarArg {
		for i := 0; i < results.Len(); i++ {
			list = append(list, results.At(i).Type())
		}
	}

	i := 0
	if m.Opts.Receiver != "" {
		i = 1
	}
	for ; i < params.Len(); i++ {
		list = append(list, params.At(i).Type())
	}
	return list
}

func (m *MethodEntry) Results() []types.Type {
	var list []types.Type

	sig := m.Method.Type().(*types.Signature)
	results := sig.Results()

	for i := 0; i < results.Len(); i++ {
		if m.Opts.Style == model.DstVarReturn {
			list = append(list, results.At(i).Type())
		} else {
			t := results.At(i).Type()
			if util.IsErrorType(t) {
				list = append(list, t)
			}
		}
	}
	return list
}

func (m *MethodEntry) RetError() bool {
	ret := m.Results()
	return 0 < len(ret) && util.IsErrorType(ret[len(ret)-1])
}

// SrcVar returns a variable that is a copy source.
// It assumes that there is only one source variable.
func (m *MethodEntry) SrcVar() *types.Var {
	sig := m.Method.Type().(*types.Signature)
	params := sig.Params()
	if params.Len() == 0 {
		return nil
	}
	return params.At(0)
}

// DstVar returns a variable that is a copy destination.
// It assumes that there is only one destination variable.
func (m *MethodEntry) DstVar() *types.Var {
	sig := m.Method.Type().(*types.Signature)
	results := sig.Results()
	if results.Len() == 0 {
		return nil
	}
	return results.At(0)
}
