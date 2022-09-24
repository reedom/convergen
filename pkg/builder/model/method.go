package model

import (
	"go/ast"
	"go/types"

	"github.com/reedom/convergen/pkg/option"
)

type MethodSignature struct {
	Name     string
	Recv     types.Type // can be nil
	Ret      types.Type // can be nil
	RetError bool
	Args     []types.Type
}

type MethodsInfo struct {
	Marker  string
	Methods []*MethodEntry
}

type MethodEntry struct {
	Method     types.Object // Also a *types.Signature
	Opts       option.Options
	DocComment *ast.CommentGroup
	Src        *types.Tuple
	Dst        *types.Tuple
}
