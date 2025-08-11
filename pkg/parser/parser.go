package parser

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"regexp"

	"golang.org/x/tools/go/packages"

	"github.com/reedom/convergen/v9/pkg/builder"
	"github.com/reedom/convergen/v9/pkg/builder/model"
	"github.com/reedom/convergen/v9/pkg/logger"
	"github.com/reedom/convergen/v9/pkg/option"
	"github.com/reedom/convergen/v9/pkg/util"
)

const buildTag = "convergen"

// Static errors for err113 compliance.
var (
	ErrParseError                 = errors.New("parse error")
	ErrInterfacePositionsNotFound = errors.New("interface positions not found")
)

// Parser represents a parser for Go source files that contain convergen interface annotations.
// It provides functionality to parse source code, extract convergen interfaces and their methods,
// resolve types, and generate base code for the conversion pipeline.
//
// The parser supports both legacy synchronous parsing and modern concurrent processing modes,
// depending on the configuration provided. It maintains an internal AST representation of the
// source file and provides methods to extract domain models for the generation pipeline.
//
// Example usage:
//
//	parser, err := NewParser("source.go", "generated.go")
//	if err != nil {
//	    return err
//	}
//	methods, err := parser.Parse()
//	if err != nil {
//	    return err
//	}
//	builder := parser.CreateBuilder()
//	baseCode, err := parser.GenerateBaseCode()
type Parser struct {
	srcPath       string            // The path to the source file being parsed.
	file          *ast.File         // The parsed AST of the source file.
	fset          *token.FileSet    // The token file set used for parsing.
	pkg           *packages.Package // The package information for the parsed file.
	opts          option.Options    // The options for the parser.
	imports       util.ImportNames  // The import names used in the parsed file.
	intfEntries   []*intfEntry      // The interface entries parsed from the file.
	packageLoader *PackageLoader    // Concurrent package loader (optional)
	config        *Config           // Parser configuration
}

// parserLoadMode is a packages.Load mode that loads types and syntax trees.
const parserLoadMode = packages.NeedName | packages.NeedImports | packages.NeedDeps |
	packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo

// NewParser creates a new parser for convergen annotations using default configuration.
// It analyzes the source file at srcPath and prepares for code generation to dstPath.
//
// This is the recommended entry point for most use cases as it provides sensible defaults.
// For advanced configuration options, use NewParserWithConfig instead.
//
// Parameters:
//   - srcPath: Path to the Go source file containing convergen interface annotations
//   - dstPath: Path where the generated code will be written (used for import resolution)
//
// Returns:
//   - *Parser: A configured parser ready for use
//   - error: Any error encountered during parser initialization
//
// Example:
//
//	parser, err := NewParser("models.go", "models_gen.go")
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewParser(srcPath, dstPath string) (*Parser, error) {
	return NewParserWithConfig(srcPath, dstPath, nil)
}

// NewParserWithConfig creates a new parser with custom configuration for advanced use cases.
// This function provides fine-grained control over parsing behavior, performance settings,
// and feature enablement.
//
// The parser can operate in different modes based on the configuration:
//   - Concurrent processing: Enables worker pools for package loading and method processing
//   - Legacy mode: Traditional synchronous parsing for simple use cases
//   - Custom timeouts: Configurable timeouts for type resolution operations
//   - Memory management: Configurable cache sizes and memory thresholds
//
// Parameters:
//   - srcPath: Path to the Go source file containing convergen interface annotations
//   - dstPath: Path where the generated code will be written (used for import resolution)
//   - config: Parser configuration options (nil for defaults)
//
// Returns:
//   - *Parser: A configured parser ready for use
//   - error: Any error encountered during parser initialization
//
// Example:
//
//	config := &ParserConfig{
//	    EnableConcurrentLoading: true,
//	    MaxConcurrentWorkers:    8,
//	    TypeResolutionTimeout:   30 * time.Second,
//	}
//	parser, err := NewParserWithConfig("models.go", "models_gen.go", config)
func NewParserWithConfig(srcPath, dstPath string, config *Config) (*Parser, error) {
	validConfig := EnsureValidConfig(config)

	// Use concurrent package loading if enabled
	if validConfig.EnableConcurrentLoading {
		return newParserWithConcurrentLoading(srcPath, dstPath, validConfig)
	}

	// Fallback to legacy loading for compatibility
	return newParserLegacy(srcPath, dstPath, validConfig)
}

// newParserWithConcurrentLoading creates parser using concurrent package loading.
func newParserWithConcurrentLoading(srcPath, dstPath string, config *Config) (*Parser, error) {
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
// This function coordinates the legacy parsing workflow by delegating to specialized functions.
func newParserLegacy(srcPath, dstPath string, config *Config) (*Parser, error) {
	// Prepare file statistics for selective parsing
	srcStat, dstStat, err := getFileStatistics(srcPath, dstPath)
	if err != nil {
		return nil, err
	}

	// Create package loading configuration
	fileSet := token.NewFileSet()
	parseContext := &legacyParseContext{
		srcStat: srcStat,
		dstStat: dstStat,
	}

	packageConfig := createPackageConfig(fileSet, parseContext)

	// Load package with custom parsing logic
	pkg, sourceFile, err := loadPackageWithLegacyParsing(packageConfig, srcPath, parseContext)
	if err != nil {
		return nil, err
	}

	// Construct parser instance
	return buildParserFromLegacyLoad(fileSet, sourceFile, pkg, config), nil
}

// legacyParseContext holds context information for legacy parsing operations.
type legacyParseContext struct {
	srcStat    os.FileInfo
	dstStat    os.FileInfo
	sourceFile *ast.File
	parseErr   error
}

// getFileStatistics retrieves file information for source and destination paths.
// Returns file stats needed for selective parsing logic.
func getFileStatistics(srcPath, dstPath string) (srcStat, dstStat os.FileInfo, err error) {
	srcStat, err = os.Stat(srcPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to stat source file %s: %w", srcPath, err)
	}

	dstStat, _ = os.Stat(dstPath) // Destination may not exist, ignore error
	return srcStat, dstStat, nil
}

// createPackageConfig creates a packages.Config with custom parsing logic.
// The custom ParseFile function handles selective file parsing based on file statistics.
func createPackageConfig(fileSet *token.FileSet, parseContext *legacyParseContext) *packages.Config {
	return &packages.Config{
		Mode:       parserLoadMode,
		BuildFlags: []string{"-tags", buildTag},
		Fset:       fileSet,
		ParseFile:  createLegacyParseFileFunc(parseContext),
	}
}

// createLegacyParseFileFunc creates a ParseFile function that implements legacy parsing logic.
// This function decides whether to parse files based on their relationship to source/destination.
func createLegacyParseFileFunc(parseContext *legacyParseContext) func(*token.FileSet, string, []byte) (*ast.File, error) {
	return func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
		stat, err := os.Stat(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to stat file %s: %w", filename, err)
		}

		// Skip destination file if it exists to avoid circular parsing
		if parseContext.dstStat != nil && os.SameFile(stat, parseContext.dstStat) {
			return nil, nil
		}

		// Parse non-source files without comments
		if !os.SameFile(stat, parseContext.srcStat) {
			return parser.ParseFile(fset, filename, src, 0)
		}

		// Parse source file with comments for annotation processing
		file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
		if err != nil {
			parseContext.parseErr = fmt.Errorf("failed to parse file %s: %w", filename, err)
			return nil, parseContext.parseErr
		}

		parseContext.sourceFile = file
		return file, nil
	}
}

// loadPackageWithLegacyParsing loads the package using the configured parsing logic.
// Returns the loaded package and the source file AST.
func loadPackageWithLegacyParsing(cfg *packages.Config, srcPath string, parseContext *legacyParseContext) (*packages.Package, *ast.File, error) {
	pkgs, err := packages.Load(cfg, "file="+srcPath)
	if err != nil {
		return nil, nil, fmt.Errorf("%v: failed to load type information: %w", srcPath, err)
	}

	if len(pkgs) == 0 {
		return nil, nil, fmt.Errorf("%v: %w", srcPath, ErrFailedToLoadPackageInformation)
	}

	if parseContext.sourceFile == nil && parseContext.parseErr != nil {
		return nil, nil, fmt.Errorf("%v: %w: %v", srcPath, ErrParseError, parseContext.parseErr)
	}

	return pkgs[0], parseContext.sourceFile, nil
}

// buildParserFromLegacyLoad constructs a Parser instance from legacy loading results.
// This function centralizes the Parser struct initialization logic.
func buildParserFromLegacyLoad(fileSet *token.FileSet, sourceFile *ast.File, pkg *packages.Package, config *Config) *Parser {
	return &Parser{
		srcPath: fileSet.Position(sourceFile.Pos()).Filename,
		fset:    fileSet,
		file:    sourceFile,
		pkg:     pkg,
		opts:    option.NewOptions(),
		imports: util.NewImportNames(sourceFile.Imports),
		config:  config,
	}
}

// Parse analyzes the source code and extracts convergen interface methods and their annotations.
// This is the main entry point for parsing operations and should be called after creating
// a parser instance.
//
// The parsing process follows these steps:
//  1. Discovers all convergen interfaces in the source file
//  2. Extracts methods from each interface with their annotations
//  3. Resolves cross-references between converter functions
//  4. Returns structured method information for the generation pipeline
//
// Returns:
//   - []*model.MethodsInfo: Slice of method information grouped by interface
//   - error: Any error encountered during parsing (syntax errors, type resolution failures, etc.)
//
// Example:
//
//	methodsInfo, err := parser.Parse()
//	if err != nil {
//	    return fmt.Errorf("parsing failed: %w", err)
//	}
//	for _, info := range methodsInfo {
//	    fmt.Printf("Interface marker: %s, Methods: %d\n", info.Marker, len(info.Methods))
//	}
func (p *Parser) Parse() ([]*model.MethodsInfo, error) {
	entries, err := p.findConvergenEntries()
	if err != nil {
		return nil, err
	}

	var allMethods []*model.MethodEntry

	list := make([]*model.MethodsInfo, 0, len(entries))

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

// CreateBuilder creates a new function builder configured with the parser's context.
// The builder is used in the generation pipeline to create converter function implementations
// based on the parsed method information.
//
// This method should be called after Parse() to ensure the parser has processed the source
// file and established the necessary context (AST, package info, imports).
//
// Returns:
//   - *builder.FunctionBuilder: A configured builder ready for code generation
//
// Example:
//
//	methodsInfo, err := parser.Parse()
//	if err != nil {
//	    return err
//	}
//	builder := parser.CreateBuilder()
//	// Use builder in generation pipeline...
func (p *Parser) CreateBuilder() *builder.FunctionBuilder {
	return builder.NewFunctionBuilder(p.file, p.fset, p.pkg, p.imports)
}

// GenerateBaseCode generates the base code without convergen annotations.
// The code is stripped of convergen annotations and the doc comments of interfaces.
// The resulting code can be used as a starting point for the code generation process.
//
// The generation process:
//  1. Removes build generation comments from the AST
//  2. Inserts position markers around interface definitions
//  3. Renders the modified AST to source code
//  4. Replaces interface definitions with simple markers
//
// Returns:
//   - string: The generated base code with interfaces replaced by markers
//   - error: Any error encountered during code generation
//
// Example output:
//
//	Original: type Convergen interface { ConvertUser(u User) UserModel }
//	Result:   // <<unique-marker-id>>
func (p *Parser) GenerateBaseCode() (code string, err error) {
	util.RemoveMatchComments(p.file, reGoBuildGen)

	// Insert position markers around each interface definition
	if err := p.insertInterfaceMarkers(); err != nil {
		return "", fmt.Errorf("failed to insert interface markers: %w", err)
	}

	// Render the modified AST to source code
	sourceCode, err := p.renderASTToCode()
	if err != nil {
		return "", fmt.Errorf("failed to render AST to code: %w", err)
	}

	// Replace interface definitions with simple markers
	return p.replaceInterfacesWithMarkers(sourceCode), nil
}

// insertInterfaceMarkers inserts position markers around interface definitions.
// This prepares the AST for later replacement of interface code with simple markers.
func (p *Parser) insertInterfaceMarkers() error {
	for _, entry := range p.intfEntries {
		minPos, maxPos, err := p.findInterfacePositions(entry)
		if err != nil {
			return fmt.Errorf("failed to find positions for interface %s: %w", entry.marker, err)
		}

		// Insert markers at interface boundaries
		util.InsertComment(p.file, entry.marker, minPos)
		util.InsertComment(p.file, entry.marker, maxPos)
	}
	return nil
}

// findInterfacePositions determines the start and end positions of an interface definition.
// Returns the minimum and maximum token positions that bound the interface.
func (p *Parser) findInterfacePositions(entry *intfEntry) (minPos, maxPos token.Pos, err error) {
	nodes, _ := util.ToAstNode(p.file, entry.intf)

	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.GenDecl:
			ast.Inspect(n, func(node ast.Node) bool {
				if node == nil {
					return true
				}

				if f, ok := node.(*ast.FieldList); ok {
					minPos, maxPos = updatePositionRange(minPos, maxPos, f.Pos(), f.Closing)
				}

				return true
			})
		}
	}

	if minPos == 0 {
		return 0, 0, ErrInterfacePositionsNotFound
	}

	return minPos, maxPos, nil
}

// updatePositionRange updates the min and max position range.
func updatePositionRange(minPos, maxPos, newStart, newEnd token.Pos) (token.Pos, token.Pos) {
	if minPos == 0 {
		return newStart, newEnd
	}
	if newStart < minPos {
		minPos = newStart
	}
	if maxPos < newEnd {
		maxPos = newEnd
	}
	return minPos, maxPos
}

// renderASTToCode converts the modified AST back to Go source code.
func (p *Parser) renderASTToCode() (string, error) {
	var buf bytes.Buffer

	err := printer.Fprint(&buf, p.fset, p.file)
	if err != nil {
		return "", fmt.Errorf("failed to print AST: %w", err)
	}

	return buf.String(), nil
}

// replaceInterfacesWithMarkers replaces interface definitions with simple markers.
// This converts code like:
//
//	type Convergen <<marker>>interface { ... } <<marker>>
//
// Into:
//
//	<<marker>>
func (p *Parser) replaceInterfacesWithMarkers(sourceCode string) string {
	result := sourceCode

	for _, entry := range p.intfEntries {
		result = p.replaceInterfaceWithMarker(result, entry.marker)
	}

	return result
}

// replaceInterfaceWithMarker replaces a single interface definition with its marker.
func (p *Parser) replaceInterfaceWithMarker(sourceCode, marker string) string {
	// Escape the marker for regex usage
	reMarker := regexp.QuoteMeta(marker)

	// Match everything from the interface declaration to the closing marker
	// Pattern: any_text + marker + any_content + marker
	re := regexp.MustCompile(`.+` + reMarker + ".*(\n|.)*?" + reMarker)

	return re.ReplaceAllString(sourceCode, marker)
}
