package parser

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"regexp"

	"golang.org/x/tools/go/packages"
)

const buildTag = "convergen"

type Parser struct {
	pkgs      []*packages.Package
	entryFile *entryFile
}

const parserLoadMode = packages.NeedName | packages.NeedImports | packages.NeedDeps |
	packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo |
	packages.NeedFiles

func NewParser(srcPath string, src any) (*Parser, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, srcPath, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the source file: %v\n%w", srcPath, err)
	}

	if src == nil {
		cfg := &packages.Config{Mode: parserLoadMode, BuildFlags: []string{"-tags", buildTag}, Fset: fset}
		pkgs, err := packages.Load(cfg, srcPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load type information: \n%w", err)
		}
		return &Parser{
			pkgs: pkgs,
			entryFile: &entryFile{
				srcPath: srcPath,
				file:    pkgs[0].Syntax[0],
				fileSet: fset,
				pkg:     pkgs[0].Types,
			},
		}, nil
	}

	cfg := types.Config{Importer: importer.Default()}
	pkg, err := cfg.Check("comment.go", fset, []*ast.File{file}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load type information: \n%w", err)
	}

	return &Parser{
		entryFile: &entryFile{
			srcPath: srcPath,
			file:    file,
			fileSet: fset,
			pkg:     pkg,
		},
	}, nil
}

var reGoBuildGen = regexp.MustCompile(`\s*//\s*((go:generate\b|build convergen\b)|\+build convergen)`)

func (p *Parser) Parse() error {
	e := p.entryFile

	astRemoveMatchComments(e.file, reGoBuildGen)

	intf, err := e.getInterface()
	if err != nil {
		return err
	}

	fmt.Println(e.fileSet.Position(intf.Pos()))
	nodes, _ := astToNodes(e.file, intf)
	fmt.Printf("@@@ nodes: %v\n", len(nodes))
	intfDocComment := astGetDocCommentOn(e.file, intf)
	for _, comment := range intfDocComment.List {
		fmt.Println(comment.Text)
	}

	return nil
}
