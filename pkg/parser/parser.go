package parser

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"os"
	"regexp"
	"strings"

	gonanoid "github.com/matoous/go-nanoid"
	"github.com/reedom/convergen/pkg/builder"
	"github.com/reedom/convergen/pkg/logger"
	"github.com/reedom/convergen/pkg/option"
	"github.com/reedom/convergen/pkg/util"
	"golang.org/x/tools/go/packages"
)

const buildTag = "convergen"

type Parser struct {
	file    *ast.File
	fset    *token.FileSet
	pkg     *packages.Package
	opts    option.Options
	imports util.ImportNames
	intf    *types.TypeName
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
		return nil, logger.Errorf("%v: failed to load package information: \n%w", srcPath, err)
	}

	return &Parser{
		fset:    fileSet,
		file:    fileSrc,
		pkg:     pkgs[0],
		opts:    option.NewOptions(),
		imports: util.NewImportNames(fileSrc.Imports),
	}, nil
}

func (p *Parser) Parse() ([]*builder.MethodEntry, error) {
	intf, err := p.extractIntfEntry()
	if err != nil {
		return nil, err
	}

	p.intf = intf.intf
	return p.parseMethods(intf)
}

func (p *Parser) CreateBuilder() *builder.FunctionBuilder {
	return builder.NewFunctionBuilder(p.file, p.fset, p.pkg, p.imports)
}

func (p *Parser) GenerateBaseCode() (pre string, post string, err error) {
	util.RemoveMatchComments(p.file, reGoBuildGen)

	// Remove doc comment of the interface.
	// And also find the range pos of the interface in the code.
	nodes, _ := util.ToAstNode(p.file, p.intf)
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
			break
		}
	}

	// Insert markers.
	marker, _ := gonanoid.Nanoid()
	util.InsertComment(p.file, marker, minPos)
	util.InsertComment(p.file, marker, maxPos)

	var buf bytes.Buffer
	err = printer.Fprint(&buf, p.fset, p.file)
	if err != nil {
		return
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

	pre, post, _ = strings.Cut(base, marker)
	return
}
