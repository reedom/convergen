package parser

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

type entryFile struct {
	srcPath string
	file    *ast.File
	fileSet *token.FileSet
	pkg     *types.Package
}

func (e *entryFile) getInterface() (*types.TypeName, error) {
	intf := findInterface(e.pkg.Scope(), "Convergen")
	if intf == nil {
		return nil, fmt.Errorf(`"Convergen" interface not found in %v`, e.srcPath)
	}
	return intf, nil
}
