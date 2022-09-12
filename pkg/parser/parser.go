package parser

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"regexp"
	"strings"

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
	astRemoveMatchComments(p.entryFile.file, reGoBuildGen)
	err := p.parseconvergen()
	return err
}

func (p *Parser) parseconvergen() error {
	e := p.entryFile
	intf, err := e.getInterface()
	if err != nil {
		return err
	}

	//intfDocComment := astGetDocCommentOn(e.file, intf)

	iface, ok := intf.Type().Underlying().(*types.Interface)
	if !ok {
		panic("???")
	}

	mset := types.NewMethodSet(iface)
	for i := 0; i < mset.Len(); i++ {
		err = p.parseMethod()
		meth := mset.At(i).Obj()
		//cg := astGetDocCommentOn(e.file, meth)
		sig := types.TypeString(meth.Type(), (*types.Package).Name)
		fmt.Printf("func %s%s \n",
			meth.Name(),
			strings.TrimPrefix(sig, "func"))
	}

	return nil
}
