package emitter

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"go.uber.org/zap"
)

// Static errors for err113 compliance.
var (
	ErrImportAnalysisNil = errors.New("import analysis cannot be nil")
)

// ImportManager manages Go import statements with automatic detection and optimization.
type ImportManager interface {
	// AnalyzeImports analyzes code and determines required imports
	AnalyzeImports(ctx context.Context, code *GeneratedCode) (*ImportAnalysis, error)

	// GenerateImports creates import declarations from analysis
	GenerateImports(ctx context.Context, analysis *ImportAnalysis) (*ImportDeclaration, error)

	// ResolveConflicts resolves import name conflicts
	ResolveConflicts(imports []*Import) ([]*Import, error)

	// OptimizeImports optimizes import organization and usage
	OptimizeImports(imports []*Import) ([]*Import, error)

	// AddImport adds a new import to the collection
	AddImport(imports []*Import, newImport *Import) []*Import

	// RemoveUnusedImports removes imports that are not used
	RemoveUnusedImports(imports []*Import, sourceCode string) []*Import
}

// ConcreteImportManager implements ImportManager.
type ConcreteImportManager struct {
	config           *Config
	logger           *zap.Logger
	standardLibs     map[string]bool
	aliasCounter     map[string]int
	conflictResolver *ConflictResolver
}

// ConflictResolver handles import name conflicts.
type ConflictResolver struct {
	reservedNames   map[string]bool
	aliasPatterns   []AliasPattern
	conflictHistory map[string]string
}

// AliasPattern defines patterns for generating import aliases.
type AliasPattern struct {
	Pattern     string `json:"pattern"`
	Priority    int    `json:"priority"`
	Description string `json:"description"`
}

// NewImportManager creates a new import manager.
func NewImportManager(config *Config, logger *zap.Logger) ImportManager {
	manager := &ConcreteImportManager{
		config:       config,
		logger:       logger,
		standardLibs: createStandardLibMap(),
		aliasCounter: make(map[string]int),
		conflictResolver: &ConflictResolver{
			reservedNames:   createReservedNamesMap(),
			aliasPatterns:   createDefaultAliasPatterns(),
			conflictHistory: make(map[string]string),
		},
	}

	return manager
}

// AnalyzeImports analyzes code and determines required imports.
func (im *ConcreteImportManager) AnalyzeImports(ctx context.Context, code *GeneratedCode) (*ImportAnalysis, error) {
	if code == nil {
		return nil, ErrGeneratedCodeNil
	}

	im.logger.Debug("analyzing imports for generated code",
		zap.String("package", code.PackageName),
		zap.Int("methods", len(code.Methods)))

	analysis := &ImportAnalysis{
		RequiredImports:        make([]*Import, 0),
		ConflictingNames:       make(map[string][]*Import),
		UnusedImports:          make([]*Import, 0),
		StandardLibs:           make([]*Import, 0),
		ThirdPartyLibs:         make([]*Import, 0),
		LocalImports:           make([]*Import, 0),
		OptimizationsSuggested: make([]string, 0),
	}

	// Collect imports from all methods
	importMap := make(map[string]*Import)

	for _, method := range code.Methods {
		for _, imp := range method.Imports {
			if existing, exists := importMap[imp.Path]; exists {
				// Mark as used if any method uses it
				existing.Used = existing.Used || imp.Used
			} else {
				// Create a copy to avoid modifying the original
				importCopy := &Import{
					Path:     imp.Path,
					Alias:    imp.Alias,
					Used:     imp.Used,
					Standard: im.isStandardLibrary(imp.Path),
					Local:    im.isLocalImport(imp.Path),
					Required: imp.Required,
				}
				importMap[imp.Path] = importCopy
			}
		}
	}

	// Convert map to slice
	for _, imp := range importMap {
		analysis.RequiredImports = append(analysis.RequiredImports, imp)
	}

	// Categorize imports
	for _, imp := range analysis.RequiredImports {
		switch {
		case imp.Standard:
			analysis.StandardLibs = append(analysis.StandardLibs, imp)
		case imp.Local:
			analysis.LocalImports = append(analysis.LocalImports, imp)
		default:
			analysis.ThirdPartyLibs = append(analysis.ThirdPartyLibs, imp)
		}
	}

	// Detect conflicts
	im.detectConflicts(analysis)

	// Analyze for optimizations
	im.analyzeOptimizations(analysis, code)

	im.logger.Debug("import analysis completed",
		zap.Int("total_imports", len(analysis.RequiredImports)),
		zap.Int("standard_libs", len(analysis.StandardLibs)),
		zap.Int("third_party", len(analysis.ThirdPartyLibs)),
		zap.Int("local", len(analysis.LocalImports)),
		zap.Int("conflicts", len(analysis.ConflictingNames)))

	return analysis, nil
}

// GenerateImports creates import declarations from analysis.
func (im *ConcreteImportManager) GenerateImports(ctx context.Context, analysis *ImportAnalysis) (*ImportDeclaration, error) {
	if analysis == nil {
		return nil, ErrImportAnalysisNil
	}

	im.logger.Debug("generating import declarations",
		zap.Int("required_imports", len(analysis.RequiredImports)))

	// Resolve conflicts first
	resolvedImports, err := im.ResolveConflicts(analysis.RequiredImports)
	if err != nil {
		return nil, fmt.Errorf("conflict resolution failed: %w", err)
	}

	// Optimize imports
	optimizedImports, err := im.OptimizeImports(resolvedImports)
	if err != nil {
		im.logger.Warn("import optimization failed", zap.Error(err))

		optimizedImports = resolvedImports
	}

	// Create import declaration
	declaration := &ImportDeclaration{
		Imports:        optimizedImports,
		StandardLibs:   make([]*Import, 0),
		ThirdPartyLibs: make([]*Import, 0),
		LocalImports:   make([]*Import, 0),
	}

	// Categorize optimized imports
	for _, imp := range optimizedImports {
		switch {
		case imp.Standard:
			declaration.StandardLibs = append(declaration.StandardLibs, imp)
		case imp.Local:
			declaration.LocalImports = append(declaration.LocalImports, imp)
		default:
			declaration.ThirdPartyLibs = append(declaration.ThirdPartyLibs, imp)
		}
	}

	// Sort imports within each category
	im.sortImportsByPath(declaration.StandardLibs)
	im.sortImportsByPath(declaration.ThirdPartyLibs)
	im.sortImportsByPath(declaration.LocalImports)

	// Generate import source code
	declaration.Source = im.generateImportSource(declaration)

	im.logger.Debug("import declarations generated",
		zap.Int("final_imports", len(declaration.Imports)),
		zap.Int("standard", len(declaration.StandardLibs)),
		zap.Int("third_party", len(declaration.ThirdPartyLibs)),
		zap.Int("local", len(declaration.LocalImports)))

	return declaration, nil
}

// ResolveConflicts resolves import name conflicts.
func (im *ConcreteImportManager) ResolveConflicts(imports []*Import) ([]*Import, error) {
	if len(imports) == 0 {
		return imports, nil
	}

	im.logger.Debug("resolving import conflicts",
		zap.Int("imports", len(imports)))

	// Group imports by package name to detect conflicts
	nameGroups := make(map[string][]*Import)

	for _, imp := range imports {
		packageName := im.getPackageName(imp.Path)
		nameGroups[packageName] = append(nameGroups[packageName], imp)
	}

	// Resolve conflicts
	resolved := make([]*Import, 0, len(imports))
	conflictCount := 0

	for packageName, group := range nameGroups {
		if len(group) == 1 {
			// No conflict
			resolved = append(resolved, group[0])
		} else {
			// Conflict detected
			conflictCount++
			resolvedGroup := im.resolveConflictGroup(packageName, group)
			resolved = append(resolved, resolvedGroup...)
		}
	}

	im.logger.Debug("conflict resolution completed",
		zap.Int("conflicts_resolved", conflictCount),
		zap.Int("resolved_imports", len(resolved)))

	return resolved, nil
}

// OptimizeImports optimizes import organization and usage.
func (im *ConcreteImportManager) OptimizeImports(imports []*Import) ([]*Import, error) {
	if len(imports) == 0 {
		return imports, nil
	}

	im.logger.Debug("optimizing imports",
		zap.Int("imports", len(imports)))

	optimized := make([]*Import, 0, len(imports))

	// Remove duplicates
	seen := make(map[string]*Import)

	for _, imp := range imports {
		key := im.getImportKey(imp)
		if existing, exists := seen[key]; exists {
			// Merge import information
			existing.Used = existing.Used || imp.Used
			existing.Required = existing.Required || imp.Required
		} else {
			seen[key] = &Import{
				Path:     imp.Path,
				Alias:    imp.Alias,
				Used:     imp.Used,
				Standard: imp.Standard,
				Local:    imp.Local,
				Required: imp.Required,
			}
		}
	}

	// Convert back to slice
	for _, imp := range seen {
		optimized = append(optimized, imp)
	}

	// Sort for deterministic output
	sort.Slice(optimized, func(i, j int) bool {
		return optimized[i].Path < optimized[j].Path
	})

	im.logger.Debug("import optimization completed",
		zap.Int("original", len(imports)),
		zap.Int("optimized", len(optimized)))

	return optimized, nil
}

// AddImport adds a new import to the collection.
func (im *ConcreteImportManager) AddImport(imports []*Import, newImport *Import) []*Import {
	if newImport == nil {
		return imports
	}

	// Check if import already exists
	for _, existing := range imports {
		if existing.Path == newImport.Path {
			// Update existing import
			existing.Used = existing.Used || newImport.Used
			existing.Required = existing.Required || newImport.Required

			if newImport.Alias != "" {
				existing.Alias = newImport.Alias
			}

			return imports
		}
	}

	// Add new import
	return append(imports, newImport)
}

// RemoveUnusedImports removes imports that are not used.
func (im *ConcreteImportManager) RemoveUnusedImports(imports []*Import, sourceCode string) []*Import {
	if sourceCode == "" {
		return imports
	}

	im.logger.Debug("removing unused imports",
		zap.Int("total_imports", len(imports)))

	used := make([]*Import, 0, len(imports))

	for _, imp := range imports {
		if im.isImportUsed(imp, sourceCode) {
			imp.Used = true
			used = append(used, imp)
		}
	}

	im.logger.Debug("unused imports removed",
		zap.Int("remaining", len(used)),
		zap.Int("removed", len(imports)-len(used)))

	return used
}

// Helper methods

func (im *ConcreteImportManager) isStandardLibrary(importPath string) bool {
	// Check if it's in the standard library map
	if im.standardLibs[importPath] {
		return true
	}

	// Additional heuristics for standard library detection
	parts := strings.Split(importPath, "/")
	if len(parts) > 0 {
		// Standard library packages don't contain dots in the first component
		firstPart := parts[0]
		if !strings.Contains(firstPart, ".") {
			// Common standard library prefixes
			standardPrefixes := []string{
				"archive", "bufio", "builtin", "bytes", "compress", "container",
				"context", "crypto", "database", "debug", "encoding", "errors",
				"expvar", "flag", "fmt", "go", "hash", "html", "image", "index",
				"io", "log", "math", "mime", "net", "os", "path", "plugin",
				"reflect", "regexp", "runtime", "sort", "strconv", "strings",
				"sync", "syscall", "testing", "text", "time", "unicode", "unsafe",
			}

			for _, prefix := range standardPrefixes {
				if firstPart == prefix || strings.HasPrefix(importPath, prefix+"/") {
					return true
				}
			}
		}
	}

	return false
}

func (im *ConcreteImportManager) isLocalImport(importPath string) bool {
	// Heuristic: local imports typically contain the module name or start with "./"
	return strings.HasPrefix(importPath, "./") ||
		strings.HasPrefix(importPath, "../") ||
		strings.Contains(importPath, "github.com/reedom/convergen")
}

func (im *ConcreteImportManager) getPackageName(importPath string) string {
	// Extract package name from import path
	parts := strings.Split(importPath, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return importPath
}

func (im *ConcreteImportManager) getImportKey(imp *Import) string {
	if imp.Alias != "" {
		return fmt.Sprintf("%s as %s", imp.Path, imp.Alias)
	}

	return imp.Path
}

func (im *ConcreteImportManager) detectConflicts(analysis *ImportAnalysis) {
	nameMap := make(map[string][]*Import)

	for _, imp := range analysis.RequiredImports {
		name := im.getPackageName(imp.Path)
		nameMap[name] = append(nameMap[name], imp)
	}

	for name, imports := range nameMap {
		if len(imports) > 1 {
			analysis.ConflictingNames[name] = imports
		}
	}
}

func (im *ConcreteImportManager) analyzeOptimizations(analysis *ImportAnalysis, _ *GeneratedCode) {
	// Suggest optimizations based on import patterns
	if len(analysis.RequiredImports) > 10 {
		analysis.OptimizationsSuggested = append(analysis.OptimizationsSuggested,
			"Consider grouping related functionality to reduce import count")
	}

	if len(analysis.ConflictingNames) > 3 {
		analysis.OptimizationsSuggested = append(analysis.OptimizationsSuggested,
			"Consider using more descriptive package aliases to reduce conflicts")
	}
}

func (im *ConcreteImportManager) resolveConflictGroup(packageName string, group []*Import) []*Import {
	// Sort by path for deterministic resolution
	sort.Slice(group, func(i, j int) bool {
		return group[i].Path < group[j].Path
	})

	resolved := make([]*Import, len(group))

	for i, imp := range group {
		resolved[i] = &Import{
			Path:     imp.Path,
			Used:     imp.Used,
			Standard: imp.Standard,
			Local:    imp.Local,
			Required: imp.Required,
		}

		if i == 0 && imp.Alias == "" {
			// First import keeps the original name
			resolved[i].Alias = ""
		} else {
			// Generate alias for conflicting imports
			resolved[i].Alias = im.generateAlias(imp.Path, packageName)
		}
	}

	return resolved
}

func (im *ConcreteImportManager) generateAlias(importPath, packageName string) string {
	// Check conflict history first
	if alias, exists := im.conflictResolver.conflictHistory[importPath]; exists {
		return alias
	}

	// Generate alias based on path components
	parts := strings.Split(importPath, "/")

	var alias string
	if len(parts) >= 2 {
		// Use last two components
		alias = parts[len(parts)-2] + parts[len(parts)-1]
	} else {
		// Use package name with suffix
		im.aliasCounter[packageName]++
		alias = fmt.Sprintf("%s%d", packageName, im.aliasCounter[packageName])
	}

	// Clean alias (remove special characters)
	alias = strings.ReplaceAll(alias, "-", "")
	alias = strings.ReplaceAll(alias, ".", "")

	// Ensure it's a valid Go identifier
	if !im.isValidIdentifier(alias) {
		alias = "pkg" + alias
	}

	// Store in history
	im.conflictResolver.conflictHistory[importPath] = alias

	return alias
}

func (im *ConcreteImportManager) isValidIdentifier(name string) bool {
	if name == "" {
		return false
	}

	// Simple check: starts with letter, contains only letters and numbers
	first := rune(name[0])
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
		return false
	}

	for _, r := range name[1:] {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}

	return true
}

func (im *ConcreteImportManager) isImportUsed(imp *Import, sourceCode string) bool {
	if imp.Required {
		return true
	}

	packageName := im.getPackageName(imp.Path)
	if imp.Alias != "" {
		packageName = imp.Alias
	}

	// Simple usage detection
	return strings.Contains(sourceCode, packageName+".")
}

func (im *ConcreteImportManager) sortImportsByPath(imports []*Import) {
	sort.Slice(imports, func(i, j int) bool {
		return imports[i].Path < imports[j].Path
	})
}

func (im *ConcreteImportManager) generateImportSource(declaration *ImportDeclaration) string {
	if len(declaration.Imports) == 0 {
		return ""
	}

	if len(declaration.Imports) == 1 {
		return im.formatSingleImport(declaration.Imports[0])
	}

	return im.formatMultipleImports(declaration)
}

// formatSingleImport formats a single import statement.
func (im *ConcreteImportManager) formatSingleImport(imp *Import) string {
	if imp.Alias != "" {
		return fmt.Sprintf("import %s \"%s\"", imp.Alias, imp.Path)
	}
	return fmt.Sprintf("import \"%s\"", imp.Path)
}

// formatMultipleImports formats multiple imports with proper grouping.
func (im *ConcreteImportManager) formatMultipleImports(declaration *ImportDeclaration) string {
	var lines []string
	lines = append(lines, "import (")

	// Add import groups with spacing
	lines = im.addImportGroup(lines, declaration.StandardLibs, false)
	lines = im.addImportGroup(lines, declaration.ThirdPartyLibs, im.needsSpacing(declaration, 1))
	lines = im.addImportGroup(lines, declaration.LocalImports, im.needsSpacing(declaration, 2))

	lines = append(lines, ")")
	return strings.Join(lines, "\n")
}

// addImportGroup adds a group of imports to the lines with optional spacing.
func (im *ConcreteImportManager) addImportGroup(lines []string, imports []*Import, addSpacing bool) []string {
	if len(imports) == 0 {
		return lines
	}

	if addSpacing {
		lines = append(lines, "")
	}

	for _, imp := range imports {
		lines = append(lines, im.formatImportLine(imp))
	}

	return lines
}

// formatImportLine formats a single import line with proper indentation.
func (im *ConcreteImportManager) formatImportLine(imp *Import) string {
	if imp.Alias != "" {
		return fmt.Sprintf("\t%s \"%s\"", imp.Alias, imp.Path)
	}
	return fmt.Sprintf("\t\"%s\"", imp.Path)
}

// needsSpacing determines if spacing is needed between import groups.
func (im *ConcreteImportManager) needsSpacing(declaration *ImportDeclaration, groupIndex int) bool {
	switch groupIndex {
	case 1: // Third-party after standard
		return len(declaration.StandardLibs) > 0
	case 2: // Local after third-party or standard
		return len(declaration.ThirdPartyLibs) > 0 || len(declaration.StandardLibs) > 0
	default:
		return false
	}
}

// Default data initialization

func createStandardLibMap() map[string]bool {
	// Standard library packages (partial list)
	stdLibs := []string{
		"bufio", "bytes", "context", "crypto", "encoding", "errors", "fmt",
		"io", "log", "math", "net", "os", "path", "reflect", "sort",
		"strconv", "strings", "sync", "time", "unicode", "unsafe",
		"archive/tar", "archive/zip", "compress/gzip", "crypto/md5", "crypto/sha1",
		"encoding/json", "encoding/xml", "net/http", "net/url", "path/filepath",
		"text/template", "go/ast", "go/parser", "go/token", "go/format",
	}

	stdLibMap := make(map[string]bool)
	for _, lib := range stdLibs {
		stdLibMap[lib] = true
	}

	return stdLibMap
}

func createReservedNamesMap() map[string]bool {
	// Go reserved words and commonly used names
	reserved := []string{
		"break", "case", "chan", "const", "continue", "default", "defer",
		"else", "fallthrough", "for", "func", "go", "goto", "if", "import",
		"interface", "map", "package", "range", "return", "select", "struct",
		"switch", "type", "var", "true", "false", "nil", "iota",
		// Common package names to avoid
		"main", "test", "init", "string", "int", "error",
	}

	reservedMap := make(map[string]bool)
	for _, word := range reserved {
		reservedMap[word] = true
	}

	return reservedMap
}

func createDefaultAliasPatterns() []AliasPattern {
	return []AliasPattern{
		{Pattern: "packagename + version", Priority: 1, Description: "Add version suffix"},
		{Pattern: "abbreviation", Priority: 2, Description: "Use common abbreviations"},
		{Pattern: "prefix + packagename", Priority: 3, Description: "Add descriptive prefix"},
	}
}
