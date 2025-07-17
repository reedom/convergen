package model

import (
	"go/ast"
	"go/types"

	"github.com/reedom/convergen/v8/pkg/generator/model"
	"github.com/reedom/convergen/v8/pkg/option"
	"github.com/reedom/convergen/v8/pkg/util"
)

// MethodsInfo contains a list of MethodEntry.
type MethodsInfo struct {
	Marker  string
	Methods []*MethodEntry
}

// MethodEntry contains a method information.
type MethodEntry struct {
	Method     types.Object
	Opts       option.Options
	DocComment *ast.CommentGroup
}

// Name returns the method name.
func (m *MethodEntry) Name() string {
	return m.Method.Name()
}

// Recv returns the receiver type.
func (m *MethodEntry) Recv() types.Type {
	if m.Opts.Receiver == "" {
		return nil
	}

	sig := m.Method.Type().(*types.Signature)
	if sig.Recv() == nil {
		return nil
	}
	return sig.Recv().Type()
}

// Results returns the result types.
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

// RetError returns true if the last result is an error.
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

// AdditionalArgVars returns the additional arguments.
func (m *MethodEntry) AdditionalArgVars() []*types.Var {
	sig := m.Method.Type().(*types.Signature)
	params := make([]*types.Var, sig.Params().Len())
	for i := 0; i < sig.Params().Len(); i++ {
		params[i] = sig.Params().At(i)
	}
	if len(params) <= 1 {
		return nil
	}
	return params[1:]
}
