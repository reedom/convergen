package convergen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/reedom/convergen/pkg/convergen/option"
	"golang.org/x/tools/go/packages"
)

const buildTag = "convergen"

type Convergen struct {
	file    *ast.File
	fset    *token.FileSet
	pkg     *packages.Package
	opt     *option.GlobalOption
	imports importNames
}

const parserLoadMode = packages.NeedName | packages.NeedImports | packages.NeedDeps |
	packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo

func NewConvergen(srcPath string) (*Convergen, error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, srcPath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the source file: %v\n%w", srcPath, err)
	}

	cfg := &packages.Config{Mode: parserLoadMode, BuildFlags: []string{"-tags", buildTag}, Fset: fileSet}
	pkgs, err := packages.Load(cfg, srcPath)
	if err != nil {
		return nil, fmt.Errorf("%v: failed to load type information: \n%w", srcPath, err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("%v: failed to load package information: \n%w", srcPath, err)
	}

	if 0 < len(pkgs[0].Syntax) {
		file = pkgs[0].Syntax[0]
	}

	return &Convergen{
		fset:    fileSet,
		file:    file,
		pkg:     pkgs[0],
		opt:     option.NewGlobalOption(),
		imports: newImportNames(file.Imports),
	}, nil
}
