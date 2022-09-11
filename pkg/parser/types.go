package parser

import (
	"go/types"
)

func findInterface(scope *types.Scope, name string) *types.TypeName {
	typ := scope.Lookup(name)
	if typ == nil {
		return nil
	}
	intf, ok := typ.(*types.TypeName)
	if !ok {
		return nil
	}
	return intf
}
