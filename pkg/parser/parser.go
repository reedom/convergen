package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"regexp"

	"golang.org/x/tools/go/packages"
)

const buildTag = "convergen"

type Parser struct {
	srcPath string
	file    *ast.File
	fileSet *token.FileSet
	pkg     *packages.Package
}

const parserLoadMode = packages.NeedName | packages.NeedImports | packages.NeedDeps |
	packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo |
	packages.NeedEmbedFiles

func NewParser(srcPath string, src any) (*Parser, error) {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, srcPath, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the source file: %v\n%w", srcPath, err)
	}
	cfg := &packages.Config{Mode: parserLoadMode, BuildFlags: []string{"-tags", buildTag}}
	pkgs, err := packages.Load(cfg, srcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load type information: \n%w", err)
	}
	if len(pkgs) != 1 {
		panic("pkgs is more than 1")
	}
	if len(pkgs[0].Syntax) != 1 {
		panic("pkg.Syntax is more than 1")
	}

	return &Parser{
		srcPath: srcPath,
		file:    pkgs[0].Syntax[0],
		fileSet: fset,
		pkg:     pkgs[0],
	}, nil
}

func (p *Parser) File() *ast.File {
	return p.file
}

func (p *Parser) FileSet() *token.FileSet {
	return p.fileSet
}

func (p *Parser) Pkg() *packages.Package {
	return p.pkg
}

var reGoBuildGen = regexp.MustCompile(`\s*//\s*((go:generate\b|build convergen\b)|\+build convergen)`)

func (p *Parser) Parse() error {
	typ := p.pkg.Types.Scope().Lookup("Convergen")
	if typ == nil {
		return fmt.Errorf(`"Convergen" interface not found in %v`, p.srcPath)
	}
	intf, ok := typ.(*types.TypeName)
	if !ok {
		return fmt.Errorf(`"Convergen" interface not found in %v`, p.srcPath)
	}

	astRemoveMatchComments(p.file, reGoBuildGen)

	return nil
}
