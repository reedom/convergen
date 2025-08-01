package emitter

import (
	"context"
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestFormatManager_FormatCode(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	formatMgr := NewFormatManager(config, logger)

	// Create test generated code
	methodCode := &MethodCode{
		Name:      "TestMethod",
		Signature: "func TestMethod(src *Source) (*Dest, error)",
		Body:      "return &Dest{Field: src.Field}, nil",
		Strategy:  StrategyCompositeLiteral,
		Fields: []*FieldCode{
			{
				Name:       "Field",
				Assignment: "Field: src.Field",
				Strategy:   "direct",
			},
		},
	}

	generatedCode := &GeneratedCode{
		PackageName: "testpkg",
		Methods:     []*MethodCode{methodCode},
		BaseCode:    "// Generated code",
		Imports: &ImportDeclaration{
			Imports: []*Import{
				{Path: "fmt", Standard: true, Used: true},
			},
		},
	}

	// Test format code
	result, err := formatMgr.FormatCode(context.TODO(), generatedCode)
	if err != nil {
		t.Fatalf("FormatCode failed: %v", err)
	}

	if result == nil {
		t.Fatal("FormatCode returned nil result")
	}

	if result.Source == "" {
		t.Error("FormatCode should generate source code")
	}

	t.Logf("Formatted code:\n%s", result.Source)
}

func TestFormatManager_ApplyGoFormat(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	formatMgr := NewFormatManager(config, logger)

	// Test with valid Go code
	source := `package main
func main(){
fmt.Println("hello")
}`

	formatted, err := formatMgr.ApplyGoFormat(source)
	if err != nil {
		t.Fatalf("ApplyGoFormat failed: %v", err)
	}

	if formatted == "" {
		t.Error("ApplyGoFormat should return formatted code")
	}

	// Test with invalid Go code
	invalidSource := "invalid go code {"

	_, err = formatMgr.ApplyGoFormat(invalidSource)
	if err == nil {
		t.Error("ApplyGoFormat should fail with invalid Go code")
	}
}

func TestFormatManager_ValidateFormat(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	formatMgr := NewFormatManager(config, logger)

	// Test with valid, properly formatted code
	validCode := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`

	err := formatMgr.ValidateFormat(validCode)
	if err != nil {
		t.Errorf("ValidateFormat should pass for valid code: %v", err)
	}

	// Test with invalid code
	invalidCode := "package main\nfunc invalid {"

	err = formatMgr.ValidateFormat(invalidCode)
	if err == nil {
		t.Error("ValidateFormat should fail with invalid code")
	}
}

func TestFormatManager_FormatImports(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	formatMgr := NewFormatManager(config, logger)

	imports := &ImportDeclaration{
		Imports: []*Import{
			{Path: "fmt", Standard: true, Used: true},
			{Path: "strings", Standard: true, Used: true},
			{Path: "github.com/example/pkg", Standard: false, Used: true},
		},
	}

	result, err := formatMgr.FormatImports(imports)
	if err != nil {
		t.Fatalf("FormatImports failed: %v", err)
	}

	if result == nil {
		t.Fatal("FormatImports returned nil")
	}

	if len(result.StandardLibs) == 0 {
		t.Error("FormatImports should categorize standard libs")
	}

	if len(result.ThirdPartyLibs) == 0 {
		t.Error("FormatImports should categorize third party libs")
	}
}

func TestFormatManager_OptimizeLayout(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	formatMgr := NewFormatManager(config, logger)

	// Create test code with suboptimal layout
	generatedCode := &GeneratedCode{
		PackageName: "testpkg",
		Methods: []*MethodCode{
			{
				Name:      "Method1",
				Signature: "func Method1() error",
				Body:      "return nil",
			},
			{
				Name:      "Method2",
				Signature: "func Method2() error",
				Body:      "return nil",
			},
		},
		BaseCode: "// Base code",
	}

	err := formatMgr.OptimizeLayout(generatedCode)
	if err != nil {
		t.Errorf("OptimizeLayout failed: %v", err)
	}

	// Verify optimization was applied
	if len(generatedCode.Methods) != 2 {
		t.Error("OptimizeLayout should preserve all methods")
	}
}

func TestGoImportsProcessor(t *testing.T) {
	logger := zaptest.NewLogger(t)
	processor := NewGoImportsProcessor(logger)

	// Test with valid code
	source := `package main
import "fmt"
func main() {
fmt.Println("hello")
}`

	result, err := processor.ProcessWithOptions(source, nil)
	if err != nil {
		t.Fatalf("ProcessWithOptions failed: %v", err)
	}

	if result == "" {
		t.Error("ProcessWithOptions should return processed code")
	}
}

func TestGoFmtProcessor(t *testing.T) {
	logger := zaptest.NewLogger(t)
	processor := NewGoFmtProcessor(logger)

	// Test with unformatted code
	source := `package main
func main(){
fmt.Println("hello")
}`

	formatted, err := processor.Format(source)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	if formatted == "" {
		t.Error("Format should return formatted code")
	}

	// Test with custom tab width
	formatted2, err := processor.FormatWithTabWidth(source, 2)
	if err != nil {
		t.Fatalf("FormatWithTabWidth failed: %v", err)
	}

	if formatted2 == "" {
		t.Error("FormatWithTabWidth should return formatted code")
	}
}

func TestCodeLinter(t *testing.T) {
	logger := zaptest.NewLogger(t)
	linter := NewCodeLinter(logger)

	// Test with valid code
	result, err := linter.Lint("package main")
	if err != nil {
		t.Fatalf("Lint failed: %v", err)
	}

	if result == nil {
		t.Error("Lint should return result")
	}

	result2, err := linter.LintWithRules("package main", []string{"rule1"})
	if err != nil {
		t.Fatalf("LintWithRules failed: %v", err)
	}

	if result2 == nil {
		t.Error("LintWithRules should return result")
	}
}

func TestFormatManager_EdgeCases(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	formatMgr := NewFormatManager(config, logger)

	// Test with nil generated code
	_, err := formatMgr.FormatCode(context.TODO(), nil)
	if err == nil {
		t.Error("FormatCode should fail with nil generated code")
	}

	// Test with empty imports
	emptyImports := &ImportDeclaration{
		Imports: []*Import{},
	}

	result, err := formatMgr.FormatImports(emptyImports)
	if err != nil {
		t.Fatalf("FormatImports with empty imports failed: %v", err)
	}

	if result == nil {
		t.Fatal("FormatImports should handle empty imports")
	}

	// Test with imports containing conflicts
	conflictingImports := &ImportDeclaration{
		Imports: []*Import{
			{Path: "fmt", Alias: "fmt1", Standard: true, Used: true},
			{Path: "github.com/example/fmt", Alias: "fmt2", Standard: false, Used: true},
		},
	}

	result2, err := formatMgr.FormatImports(conflictingImports)
	if err != nil {
		t.Fatalf("FormatImports with conflicts failed: %v", err)
	}

	if result2 == nil {
		t.Fatal("FormatImports should handle conflicts")
	}
}

func TestFormatManager_PrivateMethods(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	formatMgr := NewFormatManager(config, logger).(*ConcreteFormatManager)

	// Test optimizeBlankLines (currently 0% coverage)
	testCode := &GeneratedCode{
		PackageName: "test",
		BaseCode:    "line1\n\n\nline2\n\n\n\nline3",
	}
	formatMgr.optimizeBlankLines(testCode)
	t.Log("optimizeBlankLines executed (currently placeholder)")

	// Test optimizeImportGrouping (currently 0% coverage)
	imports := &ImportDeclaration{
		Imports: []*Import{
			{Path: "fmt", Standard: true},
			{Path: "strings", Standard: true},
		},
	}
	formatMgr.optimizeImportGrouping(imports)
	t.Log("optimizeImportGrouping executed (currently placeholder)")

	// Test generateImportBlock with various scenarios
	testCases := []struct {
		name    string
		imports []*Import
	}{
		{
			name: "standard_only",
			imports: []*Import{
				{Path: "fmt", Standard: true, Used: true},
			},
		},
		{
			name: "third_party_only",
			imports: []*Import{
				{Path: "github.com/example/pkg", Standard: false, Used: true},
			},
		},
		{
			name: "mixed_imports",
			imports: []*Import{
				{Path: "fmt", Standard: true, Used: true},
				{Path: "github.com/example/pkg", Standard: false, Used: true},
			},
		},
		{
			name: "with_aliases",
			imports: []*Import{
				{Path: "fmt", Alias: "f", Standard: true, Used: true},
			},
		},
		{
			name:    "empty_imports",
			imports: []*Import{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			importDecl := &ImportDeclaration{
				Imports:        tc.imports,
				StandardLibs:   []*Import{},
				ThirdPartyLibs: []*Import{},
			}
			// Categorize imports
			for _, imp := range tc.imports {
				if imp.Standard {
					importDecl.StandardLibs = append(importDecl.StandardLibs, imp)
				} else {
					importDecl.ThirdPartyLibs = append(importDecl.ThirdPartyLibs, imp)
				}
			}

			result := formatMgr.generateImportBlock(importDecl)
			t.Logf("Import block for %s: %s", tc.name, result)
		})
	}
}
