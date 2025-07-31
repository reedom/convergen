package parser

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"regexp"

	"golang.org/x/tools/go/packages"

	"github.com/reedom/convergen/v8/pkg/builder"
	"github.com/reedom/convergen/v8/pkg/builder/model"
	"github.com/reedom/convergen/v8/pkg/logger"
	"github.com/reedom/convergen/v8/pkg/option"
	"github.com/reedom/convergen/v8/pkg/util"
)

const buildTag = "convergen"

// Parser represents a parser for a Go source file that contains convergen blocks.
type Parser struct {
	srcPath       string            // The path to the source file being parsed.
	file          *ast.File         // The parsed AST of the source file.
	fset          *token.FileSet    // The token file set used for parsing.
	pkg           *packages.Package // The package information for the parsed file.
	opts          option.Options    // The options for the parser.
	imports       util.ImportNames  // The import names used in the parsed file.
	intfEntries   []*intfEntry      // The interface entries parsed from the file.
	packageLoader *PackageLoader    // Concurrent package loader (optional)
	config        *ParserConfig     // Parser configuration
}

// parserLoadMode is a packages.Load mode that loads types and syntax trees.
const parserLoadMode = packages.NeedName | packages.NeedImports | packages.NeedDeps |
	packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo

// NewParser returns a new parser for convergen annotations with default configuration.
func NewParser(srcPath, dstPath string) (*Parser, error) {
	return NewParserWithConfig(srcPath, dstPath, nil)
}

// NewParserWithConfig returns a new parser with custom configuration for performance tuning.
func NewParserWithConfig(srcPath, dstPath string, config *ParserConfig) (*Parser, error) {
	validConfig := EnsureValidConfig(config)

	// Use concurrent package loading if enabled
	if validConfig.EnableConcurrentLoading {
		return newParserWithConcurrentLoading(srcPath, dstPath, validConfig)
	}

	// Fallback to legacy loading for compatibility
	return newParserLegacy(srcPath, dstPath, validConfig)
}

// newParserWithConcurrentLoading creates parser using concurrent package loading.
func newParserWithConcurrentLoading(srcPath, dstPath string, config *ParserConfig) (*Parser, error) {
	// Create concurrent package loader
	loader := NewPackageLoader(config.MaxConcurrentWorkers, config.TypeResolutionTimeout)

	// Load package concurrently
	ctx, cancel := context.WithTimeout(context.Background(), config.TypeResolutionTimeout)
	defer cancel()

	result, err := loader.LoadPackageConcurrent(ctx, srcPath, dstPath)
	if err != nil {
		return nil, logger.Errorf("%v: concurrent package loading failed: %w", srcPath, err)
	}

	if result.Package == nil || result.File == nil {
		return nil, logger.Errorf("%v: package loading incomplete", srcPath)
	}

	return &Parser{
		srcPath:       result.FileSet.Position(result.File.Pos()).Filename,
		fset:          result.FileSet,
		file:          result.File,
		pkg:           result.Package,
		opts:          option.NewOptions(),
		imports:       util.NewImportNames(result.File.Imports),
		packageLoader: loader, // Store for potential reuse
		config:        config,
	}, nil
}

// newParserLegacy creates parser using legacy synchronous loading.
func newParserLegacy(srcPath, dstPath string, config *ParserConfig) (*Parser, error) {
	fileSet := token.NewFileSet()

	var fileSrc *ast.File

	srcStat, err := os.Stat(srcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat source file %s: %w", srcPath, err)
	}

	dstStat, _ := os.Stat(dstPath)

	var parseErr error

	cfg := &packages.Config{
		Mode:       parserLoadMode,
		BuildFlags: []string{"-tags", buildTag},
		Fset:       fileSet,
		ParseFile: func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
			stat, err := os.Stat(filename)
			if err != nil {
				return nil, fmt.Errorf("failed to stat file %s: %w", filename, err)
			}

			// If previously generation target file exists, skip reading it.
			if os.SameFile(stat, dstStat) {
				return nil, nil
			}

			if !os.SameFile(stat, srcStat) {
				return parser.ParseFile(fset, filename, src, 0)
			}

			file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
			if err != nil {
				parseErr = fmt.Errorf("failed to parse file %s: %w", filename, err)
				return nil, parseErr
			}
			fileSrc = file
			return file, nil
		},
	}

	pkgs, err := packages.Load(cfg, "file="+srcPath)
	if err != nil {
		return nil, logger.Errorf("%v: failed to load type information: \n%w", srcPath, err)
	}

	if len(pkgs) == 0 {
		return nil, logger.Errorf("%v: failed to load package information", srcPath)
	}

	if fileSrc == nil && parseErr != nil {
		return nil, logger.Errorf("%v: %v", srcPath, parseErr)
	}

	return &Parser{
		srcPath: fileSet.Position(fileSrc.Pos()).Filename,
		fset:    fileSet,
		file:    fileSrc,
		pkg:     pkgs[0],
		opts:    option.NewOptions(),
		imports: util.NewImportNames(fileSrc.Imports),
		config:  config,
	}, nil
}

// Parse parses convergen annotations in the source code.
func (p *Parser) Parse() ([]*model.MethodsInfo, error) {
	entries, err := p.findConvergenEntries()
	if err != nil {
		return nil, err
	}

	var allMethods []*model.MethodEntry

	var list []*model.MethodsInfo

	for _, entry := range entries {
		methods, err := p.parseMethods(entry)
		if err != nil {
			return nil, err
		}

		info := &model.MethodsInfo{
			Marker:  entry.marker,
			Methods: methods,
		}
		list = append(list, info)
		allMethods = append(allMethods, methods...)
	}

	// Resolve converters.
	// Some converters may refer to-be-generated functions that go/types doesn't contain
	// so that they are needed to be resolved manually.
	for _, method := range allMethods {
		for _, conv := range method.Opts.Converters {
			err = p.resolveConverters(allMethods, conv)
			if err != nil {
				return nil, err
			}
		}
	}

	p.intfEntries = entries

	return list, nil
}

// CreateBuilder creates a new function builder.
func (p *Parser) CreateBuilder() *builder.FunctionBuilder {
	return builder.NewFunctionBuilder(p.file, p.fset, p.pkg, p.imports)
}

// GenerateBaseCode generates the base code without convergen annotations.
// The code is stripped of convergen annotations and the doc comments of interfaces.
// The resulting code can be used as a starting point for the code generation process.
// GenerateBaseCode returns the resulting code as a string, or an error if the generation process fails.
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
		base = re.ReplaceAllString(base, entry.marker)
	}

	return base, nil
}
