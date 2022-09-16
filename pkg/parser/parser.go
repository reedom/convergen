package parser

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"regexp"

	"github.com/matoous/go-nanoid"
	"github.com/reedom/convergen/pkg/model"
	"github.com/reedom/convergen/pkg/parser/option"
	"golang.org/x/tools/go/packages"
)

const buildTag = "convergen"

type Parser struct {
	file    *ast.File
	fset    *token.FileSet
	pkg     *packages.Package
	opt     *option.GlobalOption
	imports importNames
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
		fset:    fileSet,
		file:    file,
		pkg:     pkgs[0],
		opt:     option.NewGlobalOption(),
		imports: newImportNames(file.Imports),
	}, nil
}

func (p *Parser) Parse() (*model.Code, error) {
	intf, err := p.extractIntfEntry()
	if err != nil {
		return nil, err
	}

	functions, err := p.parseMethods(intf)
	if err != nil {
		return nil, err
	}

	astRemoveMatchComments(p.file, reGoBuildGen)
	marker, _ := gonanoid.Nanoid()
	base, err := p.generateBaseCode(intf, marker)
	if err != nil {
		return nil, err
	}

	return &model.Code{
		Base:      base,
		Marker:    marker,
		Functions: functions,
	}, nil
}

func (p *Parser) generateBaseCode(intf *intfEntry, marker string) (string, error) {
	// Remove doc comment of the interface.
	// And also find the range pos of the interface in the code.
	nodes, _ := toAstNode(p.file, intf.intf)
	var minPos, maxPos token.Pos

	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.GenDecl:
			if n.Doc != nil {
				n.Doc.List = nil
			}
			ast.Inspect(n, func(node ast.Node) bool {
				if node == nil {
					return true
				}
				if f, ok := node.(*ast.FieldList); ok {
					if minPos == 0 {
						minPos = f.Pos()
						maxPos = f.Closing
					} else if f.Pos() < minPos {
						minPos = f.Pos()
					} else if maxPos < f.Closing {
						maxPos = f.Closing
					}
				}
				return true
			})
			break
		}
	}

	// Insert markers.
	astInsertComment(p.file, marker, minPos)
	astInsertComment(p.file, marker, maxPos)

	var buf bytes.Buffer
	err := printer.Fprint(&buf, p.fset, p.file)
	if err != nil {
		return "", err
	}
	base := buf.String()

	// Now "base" contains code like this:
	//
	//    package simple
	//
	//	  import (
	//	    mx "github.com/reedom/convergen/pkg/fixtures/data/ddd/model"
	//    )
	//
	//	  type Convergen <<marker>>interface {
	//	    DomainToModel(pet *mx.Pet) *mx.Pet
	//    }   <<marker>>
	//
	// And then we're going to convert it to:
	//
	//    package simple
	//
	//	  import (
	//	    mx "github.com/reedom/convergen/pkg/fixtures/data/ddd/model"
	//    )
	//
	//	  <<marker>>

	reMarker := regexp.QuoteMeta(marker)
	re := regexp.MustCompile(`.+` + reMarker + ".*(\n|.)*?" + reMarker)
	base = re.ReplaceAllString(base, marker)

	return base, nil
}
