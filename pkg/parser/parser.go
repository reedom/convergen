package parser

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"regexp"

	"github.com/reedom/convergen/pkg/builder"
	"github.com/reedom/convergen/pkg/logger"
	"github.com/reedom/convergen/pkg/option"
	"github.com/reedom/convergen/pkg/util"
	"golang.org/x/tools/go/packages"
)

const buildTag = "convergen"

type Parser struct {
	srcPath     string
	file        *ast.File
	fset        *token.FileSet
	pkg         *packages.Package
	opts        option.Options
	imports     util.ImportNames
	intfEntries []*intfEntry
}

type MethodsInfo struct {
	Marker  string
	Methods []*builder.MethodEntry
}

const parserLoadMode = packages.NeedName | packages.NeedImports | packages.NeedDeps |
	packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo

func NewParser(srcPath string) (*Parser, error) {
	fileSet := token.NewFileSet()
	var fileSrc *ast.File

	srcStat, err := os.Stat(srcPath)
	if err != nil {
		return nil, err
	}

	cfg := &packages.Config{
		Mode:       parserLoadMode,
		BuildFlags: []string{"-tags", buildTag},
		Fset:       fileSet,
		ParseFile: func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
			stat, err := os.Stat(filename)
			if err != nil {
				return nil, err
			}
			if !os.SameFile(stat, srcStat) {
				return parser.ParseFile(fset, filename, src, 0)
			}

			file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
			if err == nil {
				fileSrc = file
			}
			return file, err
		},
	}
	pkgs, err := packages.Load(cfg, "file="+srcPath)
	if err != nil {
		return nil, logger.Errorf("%v: failed to load type information: \n%w", srcPath, err)
	}
	if len(pkgs) == 0 {
		return nil, logger.Errorf("%v: failed to load package information", srcPath)
	}

	return &Parser{
		srcPath: fileSet.Position(fileSrc.Pos()).Filename,
		fset:    fileSet,
		file:    fileSrc,
		pkg:     pkgs[0],
		opts:    option.NewOptions(),
		imports: util.NewImportNames(fileSrc.Imports),
	}, nil
}

func (p *Parser) Parse() ([]*MethodsInfo, error) {
	entries, err := p.findConvergenEntries()
	if err != nil {
		return nil, err
	}

	list := make([]*MethodsInfo, 0)
	for _, entry := range entries {
		methods, err := p.parseMethods(entry)
		if err != nil {
			return nil, err
		}
		info := &MethodsInfo{
			Marker:  entry.marker,
			Methods: methods,
		}
		list = append(list, info)
	}

	p.intfEntries = entries
	return list, nil
}

func (p *Parser) CreateBuilder() *builder.FunctionBuilder {
	return builder.NewFunctionBuilder(p.file, p.fset, p.pkg, p.imports)
}

func (p *Parser) GenerateBaseCode() (code string, err error) {
	util.RemoveMatchComments(p.file, reGoBuildGen)

	// Remove doc comment of the interface.
	// And also find the range pos of the interface in the code.
	for _, entry := range p.intfEntries {
		nodes, _ := util.ToAstNode(p.file, entry.intf)
		var minPos, maxPos token.Pos

		for _, node := range nodes {
			switch n := node.(type) {
			case *ast.GenDecl:
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
			}
		}

		// Insert markers.
		util.InsertComment(p.file, entry.marker, minPos)
		util.InsertComment(p.file, entry.marker, maxPos)
	}

	var buf bytes.Buffer
	err = printer.Fprint(&buf, p.fset, p.file)
	if err != nil {
		return
	}

	base := buf.String()
	// Now each interfaces is marked with two <<marker>>s like below:
	//
	//	    type Convergen <<marker>>interface {
	//	      DomainToModel(pet *mx.Pet) *mx.Pet
	//      }   <<marker>>
	//
	// And then we're going to convert it to:
	//
	//	    <<marker>>

	for _, entry := range p.intfEntries {
		reMarker := regexp.QuoteMeta(entry.marker)
		re := regexp.MustCompile(`.+` + reMarker + ".*(\n|.)*?" + reMarker)
		if !re.MatchString(base) {
			fmt.Printf("@@@ NOT MATCH %v\n", entry.marker)
		}
		base = re.ReplaceAllString(base, entry.marker)
	}

	return base, nil
}
