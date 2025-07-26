package parser

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"regexp"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v8/pkg/domain"
)

// BaseCodeGenerator generates base code by stripping convergen annotations
type BaseCodeGenerator struct {
	fileSet *token.FileSet
	logger  *zap.Logger
}

// NewBaseCodeGenerator creates a new base code generator
func NewBaseCodeGenerator(fileSet *token.FileSet, logger *zap.Logger) *BaseCodeGenerator {
	return &BaseCodeGenerator{
		fileSet: fileSet,
		logger:  logger,
	}
}

// generateBaseCode generates clean Go code without convergen annotations
func (p *ASTParser) generateBaseCode(file *ast.File, interfaces []*InterfaceInfo) (string, error) {
	// Create a copy of the AST to avoid modifying the original
	fileCopy := p.copyASTFile(file)

	// Remove build tags and convergen comments
	p.removeConvergenComments(fileCopy)

	// Process each interface to insert markers and clean up
	for _, interfaceInfo := range interfaces {
		if err := p.processInterfaceForBaseCode(fileCopy, interfaceInfo); err != nil {
			return "", fmt.Errorf("failed to process interface %s: %w", interfaceInfo.Object.Name(), err)
		}
	}

	// Generate the code
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, p.fileSet, fileCopy); err != nil {
		return "", fmt.Errorf("failed to print AST: %w", err)
	}

	baseCode := buf.String()

	// Post-process to replace interface blocks with markers
	for _, interfaceInfo := range interfaces {
		baseCode = p.replaceInterfaceWithMarker(baseCode, interfaceInfo.Marker)
	}

	return baseCode, nil
}

// copyASTFile creates a deep copy of an AST file
func (p *ASTParser) copyASTFile(file *ast.File) *ast.File {
	// This is a simplified copy - in a full implementation,
	// you'd need a complete AST deep copy function
	return &ast.File{
		Doc:        file.Doc,
		Package:    file.Package,
		Name:       file.Name,
		Decls:      p.copyDecls(file.Decls),
		Scope:      file.Scope,
		Imports:    file.Imports,
		Unresolved: file.Unresolved,
		Comments:   p.copyComments(file.Comments),
	}
}

// copyDecls creates a copy of declarations
func (p *ASTParser) copyDecls(decls []ast.Decl) []ast.Decl {
	copied := make([]ast.Decl, len(decls))
	for i, decl := range decls {
		copied[i] = p.copyDecl(decl)
	}
	return copied
}

// copyDecl creates a copy of a single declaration
func (p *ASTParser) copyDecl(decl ast.Decl) ast.Decl {
	switch d := decl.(type) {
	case *ast.GenDecl:
		return &ast.GenDecl{
			Doc:    d.Doc,
			TokPos: d.TokPos,
			Tok:    d.Tok,
			Lparen: d.Lparen,
			Specs:  p.copySpecs(d.Specs),
			Rparen: d.Rparen,
		}
	case *ast.FuncDecl:
		return &ast.FuncDecl{
			Doc:  d.Doc,
			Recv: d.Recv,
			Name: d.Name,
			Type: d.Type,
			Body: d.Body,
		}
	default:
		return decl
	}
}

// copySpecs creates a copy of specifications
func (p *ASTParser) copySpecs(specs []ast.Spec) []ast.Spec {
	copied := make([]ast.Spec, len(specs))
	for i, spec := range specs {
		copied[i] = p.copySpec(spec)
	}
	return copied
}

// copySpec creates a copy of a single specification
func (p *ASTParser) copySpec(spec ast.Spec) ast.Spec {
	switch s := spec.(type) {
	case *ast.TypeSpec:
		return &ast.TypeSpec{
			Doc:     s.Doc,
			Name:    s.Name,
			Assign:  s.Assign,
			Type:    s.Type,
			Comment: s.Comment,
		}
	case *ast.ImportSpec:
		return &ast.ImportSpec{
			Doc:     s.Doc,
			Name:    s.Name,
			Path:    s.Path,
			Comment: s.Comment,
			EndPos:  s.EndPos,
		}
	case *ast.ValueSpec:
		return &ast.ValueSpec{
			Doc:     s.Doc,
			Names:   s.Names,
			Type:    s.Type,
			Values:  s.Values,
			Comment: s.Comment,
		}
	default:
		return spec
	}
}

// copyComments creates a copy of comment groups
func (p *ASTParser) copyComments(comments []*ast.CommentGroup) []*ast.CommentGroup {
	copied := make([]*ast.CommentGroup, len(comments))
	for i, comment := range comments {
		copied[i] = p.copyCommentGroup(comment)
	}
	return copied
}

// copyCommentGroup creates a copy of a comment group
func (p *ASTParser) copyCommentGroup(group *ast.CommentGroup) *ast.CommentGroup {
	if group == nil {
		return nil
	}

	comments := make([]*ast.Comment, len(group.List))
	for i, comment := range group.List {
		comments[i] = &ast.Comment{
			Slash: comment.Slash,
			Text:  comment.Text,
		}
	}

	return &ast.CommentGroup{
		List: comments,
	}
}

// removeConvergenComments removes convergen-related comments from the AST
func (p *ASTParser) removeConvergenComments(file *ast.File) {
	// Patterns to match and remove
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\s*//\s*(go:(generate\b|build convergen\b)|\+build convergen)`),
		regexp.MustCompile(`\s*//\s*:convergen\b`),
		regexp.MustCompile(`\s*//\s*:\w+\s*.*`), // All annotation comments
	}

	// Remove matching comments
	file.Comments = p.filterComments(file.Comments, patterns)

	// Also clean up doc comments on declarations
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			genDecl.Doc = p.filterCommentGroup(genDecl.Doc, patterns)
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					typeSpec.Doc = p.filterCommentGroup(typeSpec.Doc, patterns)
					typeSpec.Comment = p.filterCommentGroup(typeSpec.Comment, patterns)
				}
			}
		}
	}
}

// filterComments removes comments that match any of the given patterns
func (p *ASTParser) filterComments(comments []*ast.CommentGroup, patterns []*regexp.Regexp) []*ast.CommentGroup {
	var filtered []*ast.CommentGroup

	for _, group := range comments {
		if filteredGroup := p.filterCommentGroup(group, patterns); filteredGroup != nil {
			filtered = append(filtered, filteredGroup)
		}
	}

	return filtered
}

// filterCommentGroup filters individual comments within a comment group
func (p *ASTParser) filterCommentGroup(group *ast.CommentGroup, patterns []*regexp.Regexp) *ast.CommentGroup {
	if group == nil {
		return nil
	}

	var filteredComments []*ast.Comment

	for _, comment := range group.List {
		shouldRemove := false
		for _, pattern := range patterns {
			if pattern.MatchString(comment.Text) {
				shouldRemove = true
				break
			}
		}

		if !shouldRemove {
			filteredComments = append(filteredComments, comment)
		}
	}

	if len(filteredComments) == 0 {
		return nil
	}

	return &ast.CommentGroup{
		List: filteredComments,
	}
}

// processInterfaceForBaseCode processes an interface for base code generation
func (p *ASTParser) processInterfaceForBaseCode(file *ast.File, interfaceInfo *InterfaceInfo) error {
	// Find the interface declaration in the AST
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if typeSpec.Name.Name == interfaceInfo.Object.Name() {
						// Remove doc comments from the interface
						if genDecl.Doc != nil {
							genDecl.Doc = nil
						}
						if typeSpec.Doc != nil {
							typeSpec.Doc = nil
						}

						// Insert marker comments
						p.insertMarkerComments(file, typeSpec, interfaceInfo.Marker)
						return nil
					}
				}
			}
		}
	}

	return fmt.Errorf("interface %s not found in AST", interfaceInfo.Object.Name())
}

// insertMarkerComments inserts marker comments around the interface
func (p *ASTParser) insertMarkerComments(file *ast.File, typeSpec *ast.TypeSpec, marker string) {
	// This is a simplified implementation
	// In practice, you'd need to carefully insert comments at the right positions
	markerComment := &ast.Comment{
		Slash: typeSpec.Pos(),
		Text:  fmt.Sprintf("// %s", marker),
	}

	// Create a comment group with the marker
	markerGroup := &ast.CommentGroup{
		List: []*ast.Comment{markerComment},
	}

	// Add to the file's comments
	file.Comments = append(file.Comments, markerGroup)
}

// replaceInterfaceWithMarker performs final string replacement to clean up interface blocks
func (p *ASTParser) replaceInterfaceWithMarker(code, marker string) string {
	// Create regex to match the interface declaration with markers
	escapedMarker := regexp.QuoteMeta(marker)

	// Pattern to match everything between two marker comments
	pattern := fmt.Sprintf(`(?s).*?%s.*?\n.*?%s`, escapedMarker, escapedMarker)
	re := regexp.MustCompile(pattern)

	// Replace with just the marker comment
	replacement := fmt.Sprintf("// %s", marker)

	return re.ReplaceAllString(code, replacement)
}

// resolveCrossReferences resolves cross-references between methods
func (p *ASTParser) resolveCrossReferences(ctx context.Context, methods []*domain.Method) error {
	// Create a map of method names to methods for quick lookup
	methodMap := make(map[string]*domain.Method)
	for _, method := range methods {
		methodMap[method.Name] = method
	}

	// Process each method to resolve converter references
	for _, method := range methods {
		if err := p.resolveMethodConverters(method, methodMap); err != nil {
			return fmt.Errorf("failed to resolve converters for method %s: %w", method.Name, err)
		}
	}

	return nil
}

// resolveMethodConverters resolves converter function references within a method
func (p *ASTParser) resolveMethodConverters(method *domain.Method, methodMap map[string]*domain.Method) error {
	// Check each field mapping for converter references
	for _, mapping := range method.FieldMappings() {
		if mapping.StrategyName == "converter" && mapping.Config.Converter != nil {
			converterName := mapping.Config.Converter.Name
			// Try to find the converter method
			if converterMethod, exists := methodMap[converterName]; exists {
				// Validate that the converter method is compatible
				if err := p.validateConverterCompatibility(mapping, converterMethod); err != nil {
					return fmt.Errorf("converter %s is not compatible: %w", converterName, err)
				}

				p.logger.Debug("resolved converter reference",
					zap.String("method", method.Name),
					zap.String("converter", converterName))
			}
		}
	}

	return nil
}

// validateConverterCompatibility checks if a method can be used as a converter
func (p *ASTParser) validateConverterCompatibility(mapping *domain.FieldMapping, converter *domain.Method) error {
	// Check that the converter has the right signature
	params := converter.SourceParams()
	returns := converter.DestinationReturns()

	if len(params) != 1 {
		return fmt.Errorf("converter must have exactly one parameter")
	}

	if len(returns) < 1 || len(returns) > 2 {
		return fmt.Errorf("converter must have 1 or 2 return values")
	}

	// Check type compatibility
	sourceType := mapping.Source.Type
	destType := mapping.Dest.Type
	converterSourceType := params[0].Type
	converterDestType := returns[0].Type

	if !sourceType.AssignableTo(converterSourceType) {
		return fmt.Errorf("source type %s not assignable to converter input %s",
			sourceType.Name(), converterSourceType.Name())
	}

	if !converterDestType.AssignableTo(destType) {
		return fmt.Errorf("converter output %s not assignable to destination type %s",
			converterDestType.Name(), destType.Name())
	}

	// If two returns, second must be error
	if len(returns) == 2 {
		if returns[1].Type.Name() != "error" {
			return fmt.Errorf("second return value must be error type")
		}
	}

	return nil
}
