package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"regexp"

	"github.com/reedom/convergen/pkg/parser/option"
	"golang.org/x/tools/go/packages"
)

const intfName = "Convergen"
const buildTag = "convergen"

type Parser struct {
	file *ast.File
	fset *token.FileSet
	pkg  *packages.Package
	opt  *option.GlobalOption
}

type intfEntry struct {
	intf      *types.TypeName
	notations []*ast.Comment
	methods   []*methodEntry
	functions []*funcEntry
}

type methodEntry struct {
	method    types.Object
	notations []*ast.Comment
	src       *types.Tuple
	dst       *types.Tuple
}

type funcEntry struct {
	fun       *types.Object
	notations []*ast.Comment
}

const parserLoadMode = packages.NeedName | packages.NeedImports | packages.NeedDeps |
	packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo

func NewParser(srcPath string) (*Parser, error) {
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

	return &Parser{
		fset: fileSet,
		file: file,
		pkg:  pkgs[0],
		opt:  option.NewGlobalOption(),
	}, nil
}

var reGoBuildGen = regexp.MustCompile(`\s*//\s*((go:generate\b|build convergen\b)|\+build convergen)`)

func (p *Parser) Parse() error {
	//astRemoveMatchComments(p.file, reGoBuildGen)
	intfEntry, err := p.extractIntfEntry()
	if err != nil {
		return err
	}
	fmt.Printf("%#v\n", intfEntry)
	return nil
}

func (p *Parser) extractIntfEntry() (*intfEntry, error) {
	var intf *types.TypeName
	intf = findInterface(p.pkg.Types.Scope(), intfName)
	if intf == nil {
		return nil, fmt.Errorf("%v: %v interface not found", p.fset.Position(p.file.Package), intfName)
	}

	docComment := astGetDocCommentOn(p.file, intf)
	notations := astExtractMatchComments(docComment, reNotation)
	fmt.Printf("@@@ intf notations %v\n", len(notations))

	methods, err := p.extractIntfMethods(intf)
	if err != nil {
		return nil, err
	}

	return &intfEntry{
		intf:      intf,
		notations: notations,
		methods:   methods,
		functions: nil,
	}, nil
}

func (p *Parser) extractIntfMethods(intf *types.TypeName) ([]*methodEntry, error) {
	iface, ok := intf.Type().Underlying().(*types.Interface)
	if !ok {
		return nil, fmt.Errorf("%v: %v is not interface", p.fset.Position(intf.Pos()), intf.Name())
	}

	methods := make([]*methodEntry, 0)
	mset := types.NewMethodSet(iface)
	for i := 0; i < mset.Len(); i++ {
		method, err := p.extractMethodEntry(mset.At(i).Obj())
		if err != nil {
			return nil, err
		}
		methods = append(methods, method)
	}
	return methods, nil
}

func (p *Parser) extractMethodEntry(method types.Object) (*methodEntry, error) {
	signature, ok := method.Type().(*types.Signature)
	if !ok {
		return nil, fmt.Errorf(`%v: expected signature but %#v`, p.fset.Position(method.Pos()), method)
	}

	if signature.Params().Len() == 0 {
		return nil, fmt.Errorf(`%v: method must have one or more arguments as copy source`, p.fset.Position(method.Pos()))
	}
	if signature.Results().Len() == 0 {
		return nil, fmt.Errorf(`%v: method must have one or more return values as copy destination`, p.fset.Position(method.Pos()))
	}

	docComment := astGetDocCommentOn(p.file, method)
	notations := astExtractMatchComments(docComment, reNotation)
	fmt.Printf("@@@ method notations %v\n", len(notations))
	return &methodEntry{
		method:    method,
		notations: notations,
		src:       signature.Params(),
		dst:       signature.Results(),
	}, nil
}
