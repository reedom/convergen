package emitter

import (
	"context"
	"errors"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Static errors for err113 compliance.
var (
	ErrCodeNil                        = errors.New("code cannot be nil")
	ErrSourceCodeEmpty                = errors.New("source code is empty")
	ErrSourceCodeNotProperlyFormatted = errors.New("source code is not properly formatted")
	ErrImportDeclarationNil           = errors.New("import declaration cannot be nil")
	ErrMethodCodeNil                  = errors.New("method code cannot be nil")
)

// FormatManager handles code formatting and style enforcement.
type FormatManager interface {
	// FormatCode formats the complete generated code
	FormatCode(ctx context.Context, code *GeneratedCode) (*GeneratedCode, error)

	// ApplyGoFormat applies standard Go formatting
	ApplyGoFormat(source string) (string, error)

	// OptimizeLayout optimizes code layout and structure
	OptimizeLayout(code *GeneratedCode) error

	// ValidateFormat validates formatting compliance
	ValidateFormat(source string) error

	// FormatImports formats import declarations
	FormatImports(imports *ImportDeclaration) (*ImportDeclaration, error)
}

// ConcreteFormatManager implements FormatManager.
type ConcreteFormatManager struct {
	config    *FormatConfig
	logger    *zap.Logger
	goImports GoImportsProcessor
	goFmt     GoFmtProcessor
	linter    CodeLinter
}

// FormatConfig defines formatting preferences.
type FormatConfig struct {
	IndentStyle         string        `json:"indent_style"`
	LineWidth           int           `json:"line_width"`
	UseGoImports        bool          `json:"use_goimports"`
	UseGoFmt            bool          `json:"use_gofmt"`
	SortImports         bool          `json:"sort_imports"`
	GroupImports        bool          `json:"group_imports"`
	RemoveUnusedImports bool          `json:"remove_unused_imports"`
	FormatComments      bool          `json:"format_comments"`
	EnforceLineWidth    bool          `json:"enforce_line_width"`
	PreserveBlankLines  bool          `json:"preserve_blank_lines"`
	MaxBlankLines       int           `json:"max_blank_lines"`
	ValidationTimeout   time.Duration `json:"validation_timeout"`
}

// GoImportsProcessor handles goimports processing.
type GoImportsProcessor interface {
	Process(source string) (string, error)
	ProcessWithOptions(source string, options *GoImportsOptions) (string, error)
}

// GoFmtProcessor handles gofmt processing.
type GoFmtProcessor interface {
	Format(source string) (string, error)
	FormatWithTabWidth(source string, tabWidth int) (string, error)
}

// CodeLinter performs code quality checks.
type CodeLinter interface {
	Lint(source string) (*LintResult, error)
	LintWithRules(source string, rules []string) (*LintResult, error)
}

// GoImportsOptions configures goimports behavior.
type GoImportsOptions struct {
	LocalPrefix string `json:"local_prefix"`
	FormatOnly  bool   `json:"format_only"`
	Comments    bool   `json:"comments"`
	TabIndent   bool   `json:"tab_indent"`
	TabWidth    int    `json:"tab_width"`
}

// LintResult contains linting results.
type LintResult struct {
	Issues      []LintIssue `json:"issues"`
	Warnings    []string    `json:"warnings"`
	Errors      []string    `json:"errors"`
	Suggestions []string    `json:"suggestions"`
	Score       float64     `json:"score"`
}

// LintIssue represents a single linting issue.
type LintIssue struct {
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Message  string `json:"message"`
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Fix      string `json:"fix,omitempty"`
}

// NewFormatManager creates a new format manager.
func NewFormatManager(config *Config, logger *zap.Logger) FormatManager {
	formatConfig := &FormatConfig{
		IndentStyle:         config.IndentStyle,
		LineWidth:           config.LineWidth,
		UseGoImports:        true,
		UseGoFmt:            true,
		SortImports:         true,
		GroupImports:        true,
		RemoveUnusedImports: true,
		FormatComments:      true,
		EnforceLineWidth:    false, // Flexible for generated code
		PreserveBlankLines:  false,
		MaxBlankLines:       2,
		ValidationTimeout:   5 * time.Second,
	}

	return &ConcreteFormatManager{
		config:    formatConfig,
		logger:    logger,
		goImports: NewGoImportsProcessor(logger),
		goFmt:     NewGoFmtProcessor(logger),
		linter:    NewCodeLinter(logger),
	}
}

// FormatCode formats the complete generated code.
func (fm *ConcreteFormatManager) FormatCode(ctx context.Context, code *GeneratedCode) (*GeneratedCode, error) {
	if code == nil {
		return nil, ErrGeneratedCodeNil
	}

	fm.logger.Debug("formatting generated code",
		zap.String("package", code.PackageName),
		zap.Int("methods", len(code.Methods)))

	startTime := time.Now()

	// Create a copy to avoid modifying the original
	formattedCode := &GeneratedCode{
		PackageName: code.PackageName,
		Imports:     code.Imports,
		Methods:     make([]*MethodCode, len(code.Methods)),
		BaseCode:    code.BaseCode,
		Metadata:    code.Metadata,
		Metrics:     code.Metrics,
	}

	// Copy methods
	copy(formattedCode.Methods, code.Methods)

	// Format imports first
	if code.Imports != nil {
		formattedImports, err := fm.FormatImports(code.Imports)
		if err != nil {
			fm.logger.Warn("import formatting failed", zap.Error(err))
		} else {
			formattedCode.Imports = formattedImports
		}
	}

	// Format individual method code
	for i, method := range formattedCode.Methods {
		if err := fm.formatMethodCode(method); err != nil {
			fm.logger.Warn("method formatting failed",
				zap.String("method", method.Name),
				zap.Error(err))
		}

		formattedCode.Methods[i] = method
	}

	// Assemble complete source code
	sourceCode := fm.assembleSourceCode(formattedCode)

	// Apply Go formatting
	if fm.config.UseGoFmt || fm.config.UseGoImports {
		formattedSource, err := fm.ApplyGoFormat(sourceCode)
		if err != nil {
			fm.logger.Warn("Go formatting failed", zap.Error(err))
		} else {
			sourceCode = formattedSource
		}
	}

	// Optimize layout
	if err := fm.OptimizeLayout(formattedCode); err != nil {
		fm.logger.Warn("layout optimization failed", zap.Error(err))
	}

	// Validate formatting if enabled
	if fm.config.UseGoFmt {
		if err := fm.ValidateFormat(sourceCode); err != nil {
			fm.logger.Warn("format validation failed", zap.Error(err))
		}
	}

	formattedCode.Source = sourceCode

	// Update metrics
	if formattedCode.Metrics != nil {
		formattedCode.Metrics.FormattingTime = time.Since(startTime)
		formattedCode.Metrics.LinesGenerated = strings.Count(sourceCode, "\n") + 1
	}

	fm.logger.Debug("code formatting completed",
		zap.Duration("duration", time.Since(startTime)),
		zap.Int("lines", strings.Count(sourceCode, "\n")+1))

	return formattedCode, nil
}

// ApplyGoFormat applies standard Go formatting tools.
func (fm *ConcreteFormatManager) ApplyGoFormat(source string) (string, error) {
	var formattedSource = source

	var err error

	// Apply gofmt first
	if fm.config.UseGoFmt {
		formattedSource, err = fm.goFmt.Format(formattedSource)
		if err != nil {
			return source, fmt.Errorf("gofmt failed: %w", err)
		}
	}

	// Apply goimports
	if fm.config.UseGoImports {
		options := &GoImportsOptions{
			TabIndent: true,
			TabWidth:  4,
			Comments:  fm.config.FormatComments,
		}

		formattedSource, err = fm.goImports.ProcessWithOptions(formattedSource, options)
		if err != nil {
			return formattedSource, fmt.Errorf("goimports failed: %w", err)
		}
	}

	return formattedSource, nil
}

// OptimizeLayout optimizes code layout and structure.
func (fm *ConcreteFormatManager) OptimizeLayout(code *GeneratedCode) error {
	if code == nil {
		return ErrCodeNil
	}

	// Optimize method ordering for readability
	fm.optimizeMethodOrdering(code.Methods)

	// Optimize blank line usage
	if !fm.config.PreserveBlankLines {
		fm.optimizeBlankLines(code)
	}

	// Optimize import organization
	if code.Imports != nil && fm.config.GroupImports {
		fm.optimizeImportGrouping(code.Imports)
	}

	return nil
}

// ValidateFormat validates that source code meets formatting standards.
func (fm *ConcreteFormatManager) ValidateFormat(source string) error {
	if source == "" {
		return ErrSourceCodeEmpty
	}

	// Parse the source to ensure it's valid Go code
	fset := token.NewFileSet()

	_, err := parser.ParseFile(fset, "", source, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("source code parsing failed: %w", err)
	}

	// Validate that gofmt produces the same result
	if fm.config.UseGoFmt {
		formatted, err := fm.goFmt.Format(source)
		if err != nil {
			return fmt.Errorf("format validation failed: %w", err)
		}

		if formatted != source {
			return ErrSourceCodeNotProperlyFormatted
		}
	}

	return nil
}

// FormatImports formats import declarations.
func (fm *ConcreteFormatManager) FormatImports(imports *ImportDeclaration) (*ImportDeclaration, error) {
	if imports == nil {
		return nil, ErrImportDeclarationNil
	}

	formattedImports := &ImportDeclaration{
		Imports:        make([]*Import, len(imports.Imports)),
		StandardLibs:   make([]*Import, 0),
		ThirdPartyLibs: make([]*Import, 0),
		LocalImports:   make([]*Import, 0),
	}

	// Copy all imports
	copy(formattedImports.Imports, imports.Imports)

	// Remove unused imports if configured
	if fm.config.RemoveUnusedImports {
		formattedImports.Imports = fm.filterUsedImports(formattedImports.Imports)
	}

	// Categorize imports
	for _, imp := range formattedImports.Imports {
		if imp.Standard {
			formattedImports.StandardLibs = append(formattedImports.StandardLibs, imp)
		} else if imp.Local {
			formattedImports.LocalImports = append(formattedImports.LocalImports, imp)
		} else {
			formattedImports.ThirdPartyLibs = append(formattedImports.ThirdPartyLibs, imp)
		}
	}

	// Sort imports within each group
	if fm.config.SortImports {
		sort.Slice(formattedImports.StandardLibs, func(i, j int) bool {
			return formattedImports.StandardLibs[i].Path < formattedImports.StandardLibs[j].Path
		})
		sort.Slice(formattedImports.ThirdPartyLibs, func(i, j int) bool {
			return formattedImports.ThirdPartyLibs[i].Path < formattedImports.ThirdPartyLibs[j].Path
		})
		sort.Slice(formattedImports.LocalImports, func(i, j int) bool {
			return formattedImports.LocalImports[i].Path < formattedImports.LocalImports[j].Path
		})
	}

	// Generate formatted import block
	formattedImports.Source = fm.generateImportBlock(formattedImports)

	return formattedImports, nil
}

// Helper methods

func (fm *ConcreteFormatManager) formatMethodCode(method *MethodCode) error {
	if method == nil {
		return ErrMethodCodeNil
	}

	// Format method body
	if method.Body != "" {
		formatted := fm.formatCodeBlock(method.Body)
		method.Body = formatted
	}

	// Format error handling code
	if method.ErrorHandling != "" {
		formatted := fm.formatCodeBlock(method.ErrorHandling)
		method.ErrorHandling = formatted
	}

	// Format field code
	for _, field := range method.Fields {
		if field.Assignment != "" {
			field.Assignment = fm.formatCodeBlock(field.Assignment)
		}

		if field.ErrorCheck != "" {
			field.ErrorCheck = fm.formatCodeBlock(field.ErrorCheck)
		}
	}

	return nil
}

func (fm *ConcreteFormatManager) formatCodeBlock(code string) string {
	lines := strings.Split(code, "\n")

	formatted := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			formatted = append(formatted, "")
			continue
		}

		// Apply indentation
		indent := strings.Repeat(fm.config.IndentStyle, 1)
		formatted = append(formatted, indent+trimmed)
	}

	return strings.Join(formatted, "\n")
}

func (fm *ConcreteFormatManager) assembleSourceCode(code *GeneratedCode) string {
	var source strings.Builder

	// Package declaration
	source.WriteString(fmt.Sprintf("package %s\n\n", code.PackageName))

	// Imports
	if code.Imports != nil && len(code.Imports.Imports) > 0 {
		source.WriteString(code.Imports.Source)
		source.WriteString("\n\n")
	}

	// Base code (existing code)
	if code.BaseCode != "" {
		source.WriteString(code.BaseCode)

		if !strings.HasSuffix(code.BaseCode, "\n") {
			source.WriteString("\n")
		}

		source.WriteString("\n")
	}

	// Generated methods
	for i, method := range code.Methods {
		// Method documentation
		if method.Documentation != "" {
			source.WriteString(method.Documentation)
		}

		// Method signature and body
		source.WriteString(method.Signature)
		source.WriteString(" {\n")

		if method.Body != "" {
			source.WriteString(method.Body)
		}

		source.WriteString("}\n")

		// Add blank line between methods (except for the last one)
		if i < len(code.Methods)-1 {
			source.WriteString("\n")
		}
	}

	return source.String()
}

func (fm *ConcreteFormatManager) optimizeMethodOrdering(methods []*MethodCode) {
	// Sort methods alphabetically for consistency
	sort.Slice(methods, func(i, j int) bool {
		return methods[i].Name < methods[j].Name
	})
}

func (fm *ConcreteFormatManager) optimizeBlankLines(code *GeneratedCode) {
	// This would implement blank line optimization logic
	// For now, it's a placeholder
}

func (fm *ConcreteFormatManager) optimizeImportGrouping(imports *ImportDeclaration) {
	// Import grouping is already handled in FormatImports
	// This could add additional optimizations
}

func (fm *ConcreteFormatManager) filterUsedImports(imports []*Import) []*Import {
	var used []*Import

	for _, imp := range imports {
		if imp.Used {
			used = append(used, imp)
		}
	}

	return used
}

func (fm *ConcreteFormatManager) generateImportBlock(imports *ImportDeclaration) string {
	if len(imports.Imports) == 0 {
		return ""
	}

	var block strings.Builder

	block.WriteString("import (\n")

	// Standard library imports
	if len(imports.StandardLibs) > 0 {
		for _, imp := range imports.StandardLibs {
			if imp.Alias != "" {
				block.WriteString(fmt.Sprintf("\t%s \"%s\"\n", imp.Alias, imp.Path))
			} else {
				block.WriteString(fmt.Sprintf("\t\"%s\"\n", imp.Path))
			}
		}

		if len(imports.ThirdPartyLibs) > 0 || len(imports.LocalImports) > 0 {
			block.WriteString("\n")
		}
	}

	// Third-party imports
	if len(imports.ThirdPartyLibs) > 0 {
		for _, imp := range imports.ThirdPartyLibs {
			if imp.Alias != "" {
				block.WriteString(fmt.Sprintf("\t%s \"%s\"\n", imp.Alias, imp.Path))
			} else {
				block.WriteString(fmt.Sprintf("\t\"%s\"\n", imp.Path))
			}
		}

		if len(imports.LocalImports) > 0 {
			block.WriteString("\n")
		}
	}

	// Local imports
	if len(imports.LocalImports) > 0 {
		for _, imp := range imports.LocalImports {
			if imp.Alias != "" {
				block.WriteString(fmt.Sprintf("\t%s \"%s\"\n", imp.Alias, imp.Path))
			} else {
				block.WriteString(fmt.Sprintf("\t\"%s\"\n", imp.Path))
			}
		}
	}

	block.WriteString(")")

	return block.String()
}

// Default implementations for processors

type DefaultGoImportsProcessor struct {
	logger *zap.Logger
}

func NewGoImportsProcessor(logger *zap.Logger) GoImportsProcessor {
	return &DefaultGoImportsProcessor{logger: logger}
}

func (p *DefaultGoImportsProcessor) Process(source string) (string, error) {
	return p.ProcessWithOptions(source, &GoImportsOptions{})
}

func (p *DefaultGoImportsProcessor) ProcessWithOptions(source string, options *GoImportsOptions) (string, error) {
	// This would integrate with the actual goimports tool
	// For now, return the source as-is
	return source, nil
}

type DefaultGoFmtProcessor struct {
	logger *zap.Logger
}

func NewGoFmtProcessor(logger *zap.Logger) GoFmtProcessor {
	return &DefaultGoFmtProcessor{logger: logger}
}

func (p *DefaultGoFmtProcessor) Format(source string) (string, error) {
	return p.FormatWithTabWidth(source, 4)
}

func (p *DefaultGoFmtProcessor) FormatWithTabWidth(source string, tabWidth int) (string, error) {
	// Use Go's format package
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "", source, parser.ParseComments)
	if err != nil {
		return source, fmt.Errorf("parsing failed: %w", err)
	}

	var buf strings.Builder
	if err := format.Node(&buf, fset, file); err != nil {
		return source, fmt.Errorf("formatting failed: %w", err)
	}

	return buf.String(), nil
}

type DefaultCodeLinter struct {
	logger *zap.Logger
}

func NewCodeLinter(logger *zap.Logger) CodeLinter {
	return &DefaultCodeLinter{logger: logger}
}

func (l *DefaultCodeLinter) Lint(source string) (*LintResult, error) {
	return l.LintWithRules(source, []string{"basic"})
}

func (l *DefaultCodeLinter) LintWithRules(source string, rules []string) (*LintResult, error) {
	// Basic validation - ensure the code parses
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "", source, parser.ParseComments)

	result := &LintResult{
		Issues:      []LintIssue{},
		Warnings:    []string{},
		Errors:      []string{},
		Suggestions: []string{},
		Score:       100.0,
	}

	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.Score = 0.0
	}

	return result, nil
}
