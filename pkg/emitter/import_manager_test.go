package emitter

import (
	"context"
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestImportManager_AnalyzeImports(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultEmitterConfig()
	importMgr := NewImportManager(config, logger)

	// Create test generated code with various import needs
	generatedCode := &GeneratedCode{
		PackageName: "testpkg",
		Methods: []*MethodCode{
			{
				Name:      "TestMethod",
				Signature: "func TestMethod() error",
				Body:      `return fmt.Errorf("test error")`,
				Imports: []*Import{
					{Path: "fmt", Standard: true, Required: true},
				},
			},
		},
		BaseCode: "// Base code",
	}

	ctx := context.Background()
	analysis, err := importMgr.AnalyzeImports(ctx, generatedCode)
	if err != nil {
		t.Fatalf("AnalyzeImports failed: %v", err)
	}

	if analysis == nil {
		t.Fatal("AnalyzeImports returned nil")
	}

	if len(analysis.RequiredImports) == 0 {
		t.Error("AnalyzeImports should detect required imports")
	}

	// Test with nil input
	_, err = importMgr.AnalyzeImports(ctx, nil)
	if err == nil {
		t.Error("AnalyzeImports should fail with nil input")
	}

	// Test with empty methods
	emptyCode := &GeneratedCode{
		PackageName: "testpkg",
		Methods:     []*MethodCode{},
	}

	analysis2, err := importMgr.AnalyzeImports(ctx, emptyCode)
	if err != nil {
		t.Fatalf("AnalyzeImports with empty methods failed: %v", err)
	}

	if analysis2 == nil {
		t.Fatal("AnalyzeImports should handle empty methods")
	}
}

func TestImportManager_GenerateImports(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultEmitterConfig()
	importMgr := NewImportManager(config, logger)

	// Create test analysis
	analysis := &ImportAnalysis{
		RequiredImports: []*Import{
			{Path: "fmt", Standard: true, Required: true, Used: true},
			{Path: "strings", Standard: true, Required: true, Used: true},
			{Path: "github.com/example/pkg", Standard: false, Required: true, Used: true},
		},
		StandardLibs: []*Import{
			{Path: "fmt", Standard: true, Required: true, Used: true},
			{Path: "strings", Standard: true, Required: true, Used: true},
		},
		ThirdPartyLibs: []*Import{
			{Path: "github.com/example/pkg", Standard: false, Required: true, Used: true},
		},
	}

	ctx := context.Background()
	imports, err := importMgr.GenerateImports(ctx, analysis)
	if err != nil {
		t.Fatalf("GenerateImports failed: %v", err)
	}

	if imports == nil {
		t.Fatal("GenerateImports returned nil")
	}

	if len(imports.Imports) == 0 {
		t.Error("GenerateImports should generate import declarations")
	}

	// Test with nil analysis
	_, err = importMgr.GenerateImports(ctx, nil)
	if err == nil {
		t.Error("GenerateImports should fail with nil analysis")
	}

	// Test with conflicting imports
	conflictAnalysis := &ImportAnalysis{
		RequiredImports: []*Import{
			{Path: "fmt", Standard: true},
			{Path: "github.com/example/fmt", Standard: false},
		},
		ConflictingNames: map[string][]*Import{
			"fmt": {
				{Path: "fmt", Standard: true},
				{Path: "github.com/example/fmt", Standard: false},
			},
		},
	}

	imports2, err := importMgr.GenerateImports(ctx, conflictAnalysis)
	if err != nil {
		t.Fatalf("GenerateImports with conflicts failed: %v", err)
	}

	if imports2 == nil {
		t.Fatal("GenerateImports should handle conflicts")
	}
}

func TestImportManager_ResolveConflicts(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultEmitterConfig()
	importMgr := NewImportManager(config, logger)

	// Create conflicting imports
	imports := []*Import{
		{Path: "fmt", Standard: true, Used: true},
		{Path: "github.com/example/fmt", Standard: false, Used: true},
		{Path: "github.com/another/fmt", Standard: false, Used: true},
	}

	resolved, err := importMgr.ResolveConflicts(imports)
	if err != nil {
		t.Fatalf("ResolveConflicts failed: %v", err)
	}

	if resolved == nil {
		t.Fatal("ResolveConflicts returned nil")
	}

	if len(resolved) != len(imports) {
		t.Error("ResolveConflicts should preserve import count")
	}

	// Test with no conflicts
	noConflicts := []*Import{
		{Path: "fmt", Standard: true, Used: true},
		{Path: "strings", Standard: true, Used: true},
	}

	resolved2, err := importMgr.ResolveConflicts(noConflicts)
	if err != nil {
		t.Fatalf("ResolveConflicts with no conflicts failed: %v", err)
	}

	if len(resolved2) != len(noConflicts) {
		t.Error("ResolveConflicts should handle no conflicts")
	}

	// Test with empty imports
	empty, err := importMgr.ResolveConflicts([]*Import{})
	if err != nil {
		t.Fatalf("ResolveConflicts with empty imports failed: %v", err)
	}

	if len(empty) != 0 {
		t.Error("ResolveConflicts should handle empty imports")
	}
}

func TestImportManager_OptimizeImports(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultEmitterConfig()
	importMgr := NewImportManager(config, logger)

	// Create imports that can be optimized
	imports := []*Import{
		{Path: "fmt", Standard: true, Used: true},
		{Path: "strings", Standard: true, Used: false}, // Unused
		{Path: "github.com/example/pkg", Standard: false, Used: true},
		{Path: "github.com/unused/pkg", Standard: false, Used: false}, // Unused
	}

	optimized, err := importMgr.OptimizeImports(imports)
	if err != nil {
		t.Fatalf("OptimizeImports failed: %v", err)
	}

	if optimized == nil {
		t.Fatal("OptimizeImports returned nil")
	}

	// Test with empty imports
	empty, err := importMgr.OptimizeImports([]*Import{})
	if err != nil {
		t.Fatalf("OptimizeImports with empty imports failed: %v", err)
	}

	if len(empty) != 0 {
		t.Error("OptimizeImports should handle empty imports")
	}
}

func TestImportManager_PrivateMethods(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultEmitterConfig()
	importMgr := NewImportManager(config, logger).(*ConcreteImportManager)

	// Test AddImport (0% coverage)
	imp := &Import{Path: "fmt", Standard: true, Used: true}
	imports := []*Import{}
	result := importMgr.AddImport(imports, imp)
	if len(result) == 0 {
		t.Error("AddImport should add import to collection")
	}

	// Test isStandardLibrary (0% coverage)
	tests := []struct {
		path     string
		expected bool
	}{
		{"fmt", true},
		{"strings", true},
		{"context", true},
		{"github.com/example/pkg", false},
		{"example.com/pkg", false},
		{"", false},
	}

	for _, tt := range tests {
		result := importMgr.isStandardLibrary(tt.path)
		if result != tt.expected {
			t.Errorf("isStandardLibrary(%q) = %v, want %v", tt.path, result, tt.expected)
		}
	}

	// Test isLocalImport (0% coverage)
	localTests := []struct {
		path     string
		expected bool
	}{
		{"./pkg", true},
		{"../pkg", true},
		{"./internal/pkg", true},
		{"fmt", false},
		{"github.com/example/pkg", false},
	}

	for _, tt := range localTests {
		result := importMgr.isLocalImport(tt.path)
		if result != tt.expected {
			t.Errorf("isLocalImport(%q) = %v, want %v", tt.path, result, tt.expected)
		}
	}

	// Test getPackageName (0% coverage)
	pkgTests := []struct {
		path     string
		expected string
	}{
		{"fmt", "fmt"},
		{"encoding/json", "json"},
		{"github.com/example/pkg", "pkg"},
		{"github.com/example/some-pkg", "some-pkg"},
		{"", ""},
	}

	for _, tt := range pkgTests {
		result := importMgr.getPackageName(tt.path)
		if result != tt.expected {
			t.Errorf("getPackageName(%q) = %q, want %q", tt.path, result, tt.expected)
		}
	}

	// Test getImportKey (0% coverage)
	keyTests := []struct {
		imp      *Import
		expected string
	}{
		{&Import{Path: "fmt"}, "fmt"},
		{&Import{Path: "fmt", Alias: "f"}, "f"},
		{&Import{Path: "github.com/example/pkg"}, "pkg"},
	}

	for _, tt := range keyTests {
		result := importMgr.getImportKey(tt.imp)
		if result != tt.expected {
			t.Errorf("getImportKey(%+v) = %q, want %q", tt.imp, result, tt.expected)
		}
	}

	// Test generateAlias (0% coverage)
	aliasTests := []struct {
		path        string
		packageName string
	}{
		{"fmt", "fmt"},
		{"github.com/example/pkg", "pkg"},
		{"github.com/very/long/package/name", "name"},
	}

	for _, tt := range aliasTests {
		result := importMgr.generateAlias(tt.path, tt.packageName)
		if result == "" {
			t.Errorf("generateAlias(%q, %q) should return non-empty alias", tt.path, tt.packageName)
		}
		t.Logf("generateAlias(%q, %q) = %q", tt.path, tt.packageName, result)
	}

	// Test isValidIdentifier (0% coverage)
	idTests := []struct {
		name     string
		expected bool
	}{
		{"fmt", true},
		{"package", false}, // Go keyword
		{"func", false},    // Go keyword
		{"validName", true},
		{"123invalid", false}, // Starts with number
		{"valid_name", true},
		{"", false},
	}

	for _, tt := range idTests {
		result := importMgr.isValidIdentifier(tt.name)
		if result != tt.expected {
			t.Errorf("isValidIdentifier(%q) = %v, want %v", tt.name, result, tt.expected)
		}
	}

	// Test isImportUsed (0% coverage)
	usageTests := []struct {
		imp  *Import
		code string
	}{
		{&Import{Path: "fmt"}, `fmt.Println("hello")`},
		{&Import{Path: "strings", Alias: "str"}, `str.Join([]string{}, "")`},
		{&Import{Path: "unused"}, `fmt.Println("hello")`},
	}

	for _, tt := range usageTests {
		result := importMgr.isImportUsed(tt.imp, tt.code)
		t.Logf("isImportUsed(%+v, %q) = %v", tt.imp, tt.code, result)
	}

	// Test RemoveUnusedImports (0% coverage)
	unusedImports := []*Import{
		{Path: "fmt", Used: true},
		{Path: "unused", Used: false},
	}

	sourceCode := "fmt.Println(\"hello\")"
	filtered := importMgr.RemoveUnusedImports(unusedImports, sourceCode)
	t.Logf("RemoveUnusedImports filtered %d imports to %d", len(unusedImports), len(filtered))

	// Test resolveConflictGroup (0% coverage)
	conflictGroup := []*Import{
		{Path: "fmt", Standard: true},
		{Path: "github.com/example/fmt", Standard: false},
	}

	resolved := importMgr.resolveConflictGroup("fmt", conflictGroup)
	if len(resolved) != len(conflictGroup) {
		t.Error("resolveConflictGroup should preserve import count")
	}

	// Test detectConflicts - needs ImportAnalysis
	analysis := &ImportAnalysis{
		RequiredImports: []*Import{
			{Path: "fmt", Standard: true},
			{Path: "strings", Standard: true},
			{Path: "github.com/example/fmt", Standard: false}, // Conflicts with "fmt"
		},
		ConflictingNames: make(map[string][]*Import),
	}

	importMgr.detectConflicts(analysis)
	t.Logf("detectConflicts processed analysis")

	// Test analyzeOptimizations - needs ImportAnalysis and GeneratedCode
	testCode := &GeneratedCode{
		PackageName: "test",
		Methods:     []*MethodCode{},
	}
	importMgr.analyzeOptimizations(analysis, testCode)
	t.Logf("analyzeOptimizations processed analysis")

	// Test sortImportsByPath - void function
	unsorted := []*Import{
		{Path: "strings"},
		{Path: "fmt"},
		{Path: "context"},
	}

	importMgr.sortImportsByPath(unsorted)
	t.Logf("sortImportsByPath processed %d imports", len(unsorted))

	// Test generateImportSource
	sourceTests := []struct {
		name        string
		declaration *ImportDeclaration
	}{
		{
			name: "standard_only",
			declaration: &ImportDeclaration{
				Imports: []*Import{
					{Path: "fmt", Standard: true, Used: true},
				},
				StandardLibs: []*Import{
					{Path: "fmt", Standard: true, Used: true},
				},
			},
		},
		{
			name: "with_aliases",
			declaration: &ImportDeclaration{
				Imports: []*Import{
					{Path: "fmt", Alias: "f", Standard: true, Used: true},
				},
				StandardLibs: []*Import{
					{Path: "fmt", Alias: "f", Standard: true, Used: true},
				},
			},
		},
		{
			name: "empty",
			declaration: &ImportDeclaration{
				Imports: []*Import{},
			},
		},
	}

	for _, tt := range sourceTests {
		t.Run(tt.name, func(t *testing.T) {
			result := importMgr.generateImportSource(tt.declaration)
			t.Logf("generateImportSource for %s: %s", tt.name, result)
		})
	}
}

func TestImportManager_EdgeCases(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultEmitterConfig()
	importMgr := NewImportManager(config, logger)

	// Test with malformed import paths
	malformedImports := []*Import{
		{Path: "", Standard: false},              // Empty path
		{Path: "invalid..path", Standard: false}, // Invalid format
	}

	_, err := importMgr.ResolveConflicts(malformedImports)
	if err != nil {
		t.Logf("ResolveConflicts with malformed imports failed as expected: %v", err)
	}

	// Test with circular import dependencies (edge case)
	circularImports := []*Import{
		{Path: "pkg1", Standard: false, Used: true},
		{Path: "pkg2", Standard: false, Used: true},
	}

	_, err = importMgr.OptimizeImports(circularImports)
	if err != nil {
		t.Logf("OptimizeImports with potential circular deps failed: %v", err)
	}

	// Test with very long import paths
	longPathImport := &Import{
		Path:     "github.com/very/very/very/long/package/path/that/might/cause/issues",
		Standard: false,
		Used:     true,
	}

	analysis := &ImportAnalysis{
		RequiredImports: []*Import{longPathImport},
	}

	ctx := context.Background()
	_, err = importMgr.GenerateImports(ctx, analysis)
	if err != nil {
		t.Fatalf("GenerateImports with long path failed: %v", err)
	}
}
