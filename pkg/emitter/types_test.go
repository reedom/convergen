package emitter

import (
	"reflect"
	"testing"
	"time"

	"github.com/reedom/convergen/v8/pkg/domain"
	"github.com/reedom/convergen/v8/pkg/executor"
)

func TestOptimizationLevel_String(t *testing.T) {
	tests := []struct {
		level    OptimizationLevel
		expected string
	}{
		{OptimizationNone, "none"},
		{OptimizationBasic, "basic"},
		{OptimizationAggressive, "aggressive"},
		{OptimizationMaximal, "maximal"},
		{OptimizationLevel(999), "unknown"},
	}

	for _, tt := range tests {
		result := tt.level.String()
		if result != tt.expected {
			t.Errorf("OptimizationLevel(%d).String() = %q, want %q", tt.level, result, tt.expected)
		}
	}
}

func TestConstructionStrategy_String(t *testing.T) {
	tests := []struct {
		strategy ConstructionStrategy
		expected string
	}{
		{StrategyCompositeLiteral, "composite_literal"},
		{StrategyAssignmentBlock, "assignment_block"},
		{StrategyMixedApproach, "mixed_approach"},
		{StrategyCustomTemplate, "custom_template"},
		{ConstructionStrategy(999), "unknown"},
	}

	for _, tt := range tests {
		result := tt.strategy.String()
		if result != tt.expected {
			t.Errorf("ConstructionStrategy(%d).String() = %q, want %q", tt.strategy, result, tt.expected)
		}
	}
}

func TestGeneratedCode(t *testing.T) {
	code := &GeneratedCode{
		PackageName: "testpkg",
		Imports: &ImportDeclaration{
			Imports: []*Import{
				{Path: "fmt", Standard: true, Used: true},
			},
		},
		Methods: []*MethodCode{
			{
				Name:      "TestMethod",
				Signature: "func TestMethod()",
				Body:      "return nil",
			},
		},
		BaseCode: "// Base code",
		Source:   "package testpkg\nfunc TestMethod() { return nil }",
		Metadata: &GenerationMetadata{
			GenerationTime: time.Now(),
		},
		Metrics: NewGenerationMetrics(),
	}

	if code.PackageName != "testpkg" {
		t.Error("GeneratedCode should preserve package name")
	}

	if len(code.Methods) != 1 {
		t.Error("GeneratedCode should preserve methods")
	}

	if code.Imports == nil {
		t.Error("GeneratedCode should handle imports")
	}

	if code.Metrics == nil {
		t.Error("GeneratedCode should have metrics")
	}
}

func TestMethodCode(t *testing.T) {
	method := &MethodCode{
		Name:          "ConvertUser",
		Signature:     "func ConvertUser(src *User) (*UserDTO, error)",
		Body:          "return &UserDTO{Name: src.Name}, nil",
		ErrorHandling: "if err != nil { return nil, err }",
		Documentation: "// ConvertUser converts User to UserDTO",
		Imports: []*Import{
			{Path: "fmt", Standard: true, Used: true},
		},
		Complexity: NewComplexityMetrics(),
		Strategy:   StrategyCompositeLiteral,
		Fields: []*FieldCode{
			{
				Name:       "Name",
				Assignment: "Name: src.Name",
				Strategy:   "direct",
			},
		},
	}

	if method.Name != "ConvertUser" {
		t.Error("MethodCode should preserve name")
	}

	if method.Strategy != StrategyCompositeLiteral {
		t.Error("MethodCode should preserve strategy")
	}

	if len(method.Fields) != 1 {
		t.Error("MethodCode should preserve fields")
	}

	if method.Complexity == nil {
		t.Error("MethodCode should have complexity metrics")
	}
}

func TestFieldCode(t *testing.T) {
	field := &FieldCode{
		Name:         "UserName",
		Assignment:   "UserName: src.Name",
		Declaration:  "UserName string",
		ErrorCheck:   "if src.Name == \"\" { return err }",
		Imports:      []*Import{{Path: "strings", Standard: true}},
		Dependencies: []string{"Name"},
		Order:        1,
		Strategy:     "direct",
	}

	if field.Name != "UserName" {
		t.Error("FieldCode should preserve name")
	}

	if field.Order != 1 {
		t.Error("FieldCode should preserve order")
	}

	if field.Strategy != "direct" {
		t.Error("FieldCode should preserve strategy")
	}

	if len(field.Dependencies) != 1 {
		t.Error("FieldCode should preserve dependencies")
	}
}

func TestImportDeclaration(t *testing.T) {
	imports := &ImportDeclaration{
		Imports: []*Import{
			{Path: "fmt", Standard: true, Used: true},
			{Path: "strings", Standard: true, Used: true},
			{Path: "github.com/example/pkg", Standard: false, Used: true},
		},
		StandardLibs: []*Import{
			{Path: "fmt", Standard: true, Used: true},
			{Path: "strings", Standard: true, Used: true},
		},
		ThirdPartyLibs: []*Import{
			{Path: "github.com/example/pkg", Standard: false, Used: true},
		},
		LocalImports: []*Import{},
		Source:       "import (\n\t\"fmt\"\n\t\"strings\"\n)",
	}

	if len(imports.Imports) != 3 {
		t.Error("ImportDeclaration should preserve all imports")
	}

	if len(imports.StandardLibs) != 2 {
		t.Error("ImportDeclaration should categorize standard libs")
	}

	if len(imports.ThirdPartyLibs) != 1 {
		t.Error("ImportDeclaration should categorize third party libs")
	}

	if len(imports.LocalImports) != 0 {
		t.Error("ImportDeclaration should handle empty local imports")
	}
}

func TestImport(t *testing.T) {
	imp := &Import{
		Path:     "fmt",
		Alias:    "",
		Used:     true,
		Standard: true,
		Local:    false,
		Required: true,
	}

	if imp.Path != "fmt" {
		t.Error("Import should preserve path")
	}

	if !imp.Standard {
		t.Error("Import should preserve standard flag")
	}

	if !imp.Used {
		t.Error("Import should preserve used flag")
	}

	if imp.Local {
		t.Error("Import should preserve local flag")
	}

	// Test with alias
	aliasedImport := &Import{
		Path:     "github.com/example/pkg",
		Alias:    "epkg",
		Standard: false,
		Local:    false,
		Used:     true,
	}

	if aliasedImport.Alias != "epkg" {
		t.Error("Import should preserve alias")
	}
}

func TestGenerationMetadata(t *testing.T) {
	now := time.Now()
	metadata := &GenerationMetadata{
		GenerationTime:     now,
		CompletionTime:     now.Add(time.Second),
		GenerationDuration: time.Second,
		EmitterVersion:     "v1.0.0",
		ConfigHash:         "abc123",
		SourceFileHash:     "def456",
		Environment: map[string]string{
			"GO_VERSION": "1.21",
			"OS":         "linux",
		},
	}

	if metadata.GenerationTime.IsZero() {
		t.Error("GenerationMetadata should preserve generation time")
	}

	if metadata.GenerationDuration != time.Second {
		t.Error("GenerationMetadata should preserve duration")
	}

	if metadata.EmitterVersion != "v1.0.0" {
		t.Error("GenerationMetadata should preserve version")
	}

	if len(metadata.Environment) != 2 {
		t.Error("GenerationMetadata should preserve environment")
	}
}

func TestNewGenerationMetrics(t *testing.T) {
	metrics := NewGenerationMetrics()

	if metrics == nil {
		t.Error("NewGenerationMetrics should return metrics")
		return
	}

	if metrics.MethodsGenerated != 0 {
		t.Error("NewGenerationMetrics should initialize with zero values")
	}

	if metrics.FieldsGenerated != 0 {
		t.Error("NewGenerationMetrics should initialize fields with zero")
	}

	if metrics.ErrorsEncountered != 0 {
		t.Error("NewGenerationMetrics should initialize errors with zero")
	}
}

func TestNewComplexityMetrics(t *testing.T) {
	complexity := NewComplexityMetrics()

	if complexity == nil {
		t.Error("NewComplexityMetrics should return metrics")
		return
	}

	if complexity.RecommendedStrategy != StrategyCompositeLiteral {
		t.Error("NewComplexityMetrics should default to composite literal strategy")
	}
}

func TestComplexityMetrics_IsComplex(t *testing.T) {
	tests := []struct {
		name     string
		metrics  *ComplexityMetrics
		expected bool
	}{
		{
			name:     "simple_case",
			metrics:  &ComplexityMetrics{ComplexityScore: 30.0},
			expected: false,
		},
		{
			name:     "high_complexity_score",
			metrics:  &ComplexityMetrics{ComplexityScore: 60.0},
			expected: true,
		},
		{
			name:     "has_error_fields",
			metrics:  &ComplexityMetrics{ErrorFields: 1},
			expected: true,
		},
		{
			name:     "many_nested_fields",
			metrics:  &ComplexityMetrics{NestedFields: 3},
			expected: true,
		},
		{
			name:     "high_cyclomatic_complexity",
			metrics:  &ComplexityMetrics{CyclomaticComplexity: 15},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.metrics.IsComplex()
			if result != tt.expected {
				t.Errorf("IsComplex() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestComplexityMetrics_ShouldUseComposite(t *testing.T) {
	tests := []struct {
		name     string
		strategy ConstructionStrategy
		expected bool
	}{
		{
			name:     "composite_literal",
			strategy: StrategyCompositeLiteral,
			expected: true,
		},
		{
			name:     "assignment_block",
			strategy: StrategyAssignmentBlock,
			expected: false,
		},
		{
			name:     "mixed_approach",
			strategy: StrategyMixedApproach,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := &ComplexityMetrics{RecommendedStrategy: tt.strategy}

			result := metrics.ShouldUseComposite()
			if result != tt.expected {
				t.Errorf("ShouldUseComposite() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewMetrics(t *testing.T) {
	metrics := NewMetrics()

	if metrics == nil {
		t.Error("NewMetrics should return metrics")
		return
	}

	if metrics.StrategyUsage == nil {
		t.Error("NewMetrics should initialize StrategyUsage map")
	}

	if metrics.StrategyPerformance == nil {
		t.Error("NewMetrics should initialize StrategyPerformance map")
	}

	if metrics.OptimizationsApplied == nil {
		t.Error("NewMetrics should initialize OptimizationsApplied map")
	}

	if metrics.ErrorsByType == nil {
		t.Error("NewMetrics should initialize ErrorsByType map")
	}
}

func TestMetrics_RecordGeneration(t *testing.T) {
	metrics := NewMetrics()
	initialMethods := metrics.TotalMethods

	methodCode := &MethodCode{
		Name: "TestMethod",
		Body: "line1\nline2\nline3",
	}

	metrics.RecordGeneration(methodCode, "testpkg", []*MethodCode{methodCode})

	if metrics.TotalMethods != initialMethods+1 {
		t.Error("RecordGeneration should increment TotalMethods")
	}

	if metrics.TotalLines == 0 {
		t.Error("RecordGeneration should count lines")
	}

	// Test with nil method
	metrics.RecordGeneration(nil, "testpkg", []*MethodCode{})

	if metrics.TotalMethods != initialMethods+2 {
		t.Error("RecordGeneration should handle nil method")
	}
}

func TestMetrics_GetSnapshot(t *testing.T) {
	metrics := NewMetrics()
	metrics.TotalMethods = 10
	metrics.TotalLines = 100

	snapshot := metrics.GetSnapshot()

	if snapshot == nil {
		t.Error("GetSnapshot should return snapshot")
		return
	}

	if snapshot.TotalMethods != 10 {
		t.Error("GetSnapshot should copy TotalMethods")
	}

	if snapshot.TotalLines != 100 {
		t.Error("GetSnapshot should copy TotalLines")
	}

	// Verify it's a copy, not the same instance
	if snapshot == metrics {
		t.Error("GetSnapshot should return a copy, not the original")
	}
}

func TestTemplateData(t *testing.T) {
	sourceType := domain.NewBasicType("User", reflect.Struct)
	destType := domain.NewBasicType("UserDTO", reflect.Struct)

	method, err := domain.NewMethod("ConvertUser", sourceType, destType)
	if err != nil {
		t.Fatalf("Failed to create method: %v", err)
	}

	methodResult := &domain.MethodResult{
		Method:  method,
		Success: true,
	}

	templateData := &TemplateData{
		Method: methodResult,
		Fields: []*executor.FieldResult{
			{
				FieldID:      "Name",
				Success:      true,
				Result:       "src.Name",
				StrategyUsed: "direct",
			},
		},
		Package: "testpkg",
		Imports: []*Import{
			{Path: "fmt", Standard: true, Used: true},
		},
		Config: DefaultConfig(),
		Metadata: map[string]interface{}{
			"version": "1.0.0",
		},
		HelperFunctions: map[string]interface{}{
			"titleCase": func(s string) string { return s },
		},
	}

	if templateData.Method == nil {
		t.Error("TemplateData should preserve method")
	}

	if len(templateData.Fields) != 1 {
		t.Error("TemplateData should preserve fields")
	}

	if templateData.Package != "testpkg" {
		t.Error("TemplateData should preserve package")
	}

	if len(templateData.Imports) != 1 {
		t.Error("TemplateData should preserve imports")
	}

	if templateData.Config == nil {
		t.Error("TemplateData should preserve config")
	}

	if len(templateData.Metadata) != 1 {
		t.Error("TemplateData should preserve metadata")
	}

	if len(templateData.HelperFunctions) != 1 {
		t.Error("TemplateData should preserve helper functions")
	}
}

func TestOrderedBuffer(t *testing.T) {
	buffer := &OrderedBuffer{}

	// Test adding items out of order
	buffer.Add(3, "third", "content")
	buffer.Add(1, "first", "content")
	buffer.Add(2, "second", "content")

	result := buffer.Generate()
	expected := "firstsecondthird"

	if result != expected {
		t.Errorf("OrderedBuffer.Generate() = %q, want %q", result, expected)
	}

	// Test that it's sorted
	if !buffer.sorted {
		t.Error("OrderedBuffer should be marked as sorted after Generate()")
	}

	// Test adding more items after sorting
	buffer.Add(0, "zero", "content")

	if buffer.sorted {
		t.Error("OrderedBuffer should be marked as unsorted after adding new items")
	}

	result2 := buffer.Generate()
	expected2 := "zerofirstsecondthird"

	if result2 != expected2 {
		t.Errorf("OrderedBuffer.Generate() after adding = %q, want %q", result2, expected2)
	}
}

func TestOrderedBuffer_EdgeCases(t *testing.T) {
	// Test empty buffer
	buffer := &OrderedBuffer{}

	result := buffer.Generate()
	if result != "" {
		t.Errorf("Empty OrderedBuffer.Generate() = %q, want empty string", result)
	}

	// Test single item
	buffer.Add(1, "single", "content")

	result = buffer.Generate()
	if result != "single" {
		t.Errorf("Single item OrderedBuffer.Generate() = %q, want %q", result, "single")
	}

	// Test duplicate orders (should maintain insertion order for same order)
	buffer2 := &OrderedBuffer{}
	buffer2.Add(1, "first-1", "content")
	buffer2.Add(1, "second-1", "content")
	result2 := buffer2.Generate()
	expected := "first-1second-1"

	if result2 != expected {
		t.Errorf("Duplicate order OrderedBuffer.Generate() = %q, want %q", result2, expected)
	}
}

func TestSimpleTemplateSystem(t *testing.T) {
	system := NewTemplateSystem()

	// Test Execute - simple implementation just returns template
	result, err := system.Execute("template content", nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result != "template content" {
		t.Errorf("Execute() = %q, want %q", result, "template content")
	}

	// Test HasTemplate - simple implementation always returns false
	hasTemplate := system.HasTemplate("test")
	if hasTemplate {
		t.Error("HasTemplate should return false for simple implementation")
	}

	// Test RegisterTemplate - simple implementation should not error
	err = system.RegisterTemplate("test", "content")
	if err != nil {
		t.Errorf("RegisterTemplate failed: %v", err)
	}
}

func TestNewCustomTemplate(t *testing.T) {
	template := NewCustomTemplate("test", "content")
	if template != "content" {
		t.Error("NewCustomTemplate should return content")
	}
}

func TestValidationTypes(t *testing.T) {
	// Test ValidationResult
	validationResult := &ValidationResult{
		Valid: true,
		Errors: []ValidationError{
			{
				Line:     10,
				Column:   5,
				Message:  "syntax error",
				Code:     "E001",
				Severity: "error",
				Context:  "func test() {",
			},
		},
		Warnings: []ValidationWarning{
			{
				Line:       15,
				Column:     3,
				Message:    "unused variable",
				Code:       "W001",
				Context:    "var x int",
				Suggestion: "remove unused variable",
			},
		},
		Suggestions: []ValidationSuggestion{
			{
				Line:        20,
				Column:      1,
				Message:     "use shorter variable name",
				Replacement: "n",
				Benefit:     "improved readability",
			},
		},
		Metrics: &ValidationMetrics{
			ValidationTime:   time.Millisecond * 100,
			LinesValidated:   50,
			ErrorsFound:      1,
			WarningsFound:    1,
			SuggestionsFound: 1,
		},
	}

	if !validationResult.Valid {
		t.Error("ValidationResult should preserve valid flag")
	}

	if len(validationResult.Errors) != 1 {
		t.Error("ValidationResult should preserve errors")
	}

	if len(validationResult.Warnings) != 1 {
		t.Error("ValidationResult should preserve warnings")
	}

	if len(validationResult.Suggestions) != 1 {
		t.Error("ValidationResult should preserve suggestions")
	}

	if validationResult.Metrics == nil {
		t.Error("ValidationResult should have metrics")
	}
}

func TestGenerationRequestResponse(t *testing.T) {
	// Test GenerationRequest
	sourceType := domain.NewBasicType("User", reflect.Struct)
	destType := domain.NewBasicType("UserDTO", reflect.Struct)

	method, err := domain.NewMethod("ConvertUser", sourceType, destType)
	if err != nil {
		t.Fatalf("Failed to create method: %v", err)
	}

	methodResult := &domain.MethodResult{
		Method:  method,
		Success: true,
	}

	executionResults := &domain.ExecutionResults{
		Methods: []*domain.MethodResult{methodResult},
		Success: true,
	}

	request := &GenerationRequest{
		ExecutionResults: executionResults,
		Config:           DefaultConfig(),
		Context: map[string]interface{}{
			"version": "1.0.0",
		},
		RequestID: "req-123",
		Timestamp: time.Now(),
	}

	if request.ExecutionResults == nil {
		t.Error("GenerationRequest should preserve execution results")
	}

	if request.Config == nil {
		t.Error("GenerationRequest should preserve config")
	}

	if request.RequestID != "req-123" {
		t.Error("GenerationRequest should preserve request ID")
	}

	// Test GenerationResponse
	response := &GenerationResponse{
		GeneratedCode: &GeneratedCode{
			PackageName: "testpkg",
		},
		Success: true,
		Errors: []GenerationError{
			{
				Phase:       "generation",
				Method:      "ConvertUser",
				Field:       "Name",
				Message:     "conversion failed",
				Code:        "G001",
				Severity:    "error",
				Context:     map[string]interface{}{"line": 10},
				Timestamp:   time.Now(),
				Recoverable: true,
			},
		},
		Warnings: []GenerationWarning{
			{
				Phase:      "validation",
				Method:     "ConvertUser",
				Field:      "Email",
				Message:    "field might be empty",
				Code:       "W001",
				Context:    map[string]interface{}{"line": 15},
				Timestamp:  time.Now(),
				Suggestion: "add nil check",
			},
		},
		RequestID:      "req-123",
		ProcessingTime: time.Millisecond * 500,
	}

	if response.GeneratedCode == nil {
		t.Error("GenerationResponse should preserve generated code")
	}

	if !response.Success {
		t.Error("GenerationResponse should preserve success flag")
	}

	if len(response.Errors) != 1 {
		t.Error("GenerationResponse should preserve errors")
	}

	if len(response.Warnings) != 1 {
		t.Error("GenerationResponse should preserve warnings")
	}

	if response.RequestID != "req-123" {
		t.Error("GenerationResponse should preserve request ID")
	}

	if response.ProcessingTime != time.Millisecond*500 {
		t.Error("GenerationResponse should preserve processing time")
	}
}

func TestConstants(t *testing.T) {
	// Test event constants
	if EventEmitStarted != "emit.started" {
		t.Error("EventEmitStarted should have correct value")
	}

	if EventEmitCompleted != "emit.completed" {
		t.Error("EventEmitCompleted should have correct value")
	}

	if EventEmitFailed != "emit.failed" {
		t.Error("EventEmitFailed should have correct value")
	}

	if EventValidationFailed != "emit.validation.failed" {
		t.Error("EventValidationFailed should have correct value")
	}

	// Test default constants
	if DefaultIndentStyle != "\t" {
		t.Error("DefaultIndentStyle should be tab")
	}

	if DefaultLineWidth != 120 {
		t.Error("DefaultLineWidth should be 120")
	}

	if DefaultMaxConcurrency != 8 {
		t.Error("DefaultMaxConcurrency should be 8")
	}

	if DefaultGenerationTimeout != 30*time.Second {
		t.Error("DefaultGenerationTimeout should be 30 seconds")
	}

	// Test template constants
	if TemplateCompositeLiteral != "composite_literal" {
		t.Error("TemplateCompositeLiteral should have correct value")
	}

	if TemplateAssignmentBlock != "assignment_block" {
		t.Error("TemplateAssignmentBlock should have correct value")
	}
}

func TestPerformanceSnapshot(t *testing.T) {
	snapshot := &PerformanceSnapshot{
		Timestamp:            time.Now(),
		GenerationsPerSecond: 10.5,
		MethodsPerSecond:     50.2,
		LinesPerSecond:       1000.7,
		MemoryUsage:          1024 * 1024,
		ConcurrencyLevel:     4,
		ErrorRate:            0.01,
	}

	if snapshot.GenerationsPerSecond != 10.5 {
		t.Error("PerformanceSnapshot should preserve GenerationsPerSecond")
	}

	if snapshot.MethodsPerSecond != 50.2 {
		t.Error("PerformanceSnapshot should preserve MethodsPerSecond")
	}

	if snapshot.LinesPerSecond != 1000.7 {
		t.Error("PerformanceSnapshot should preserve LinesPerSecond")
	}

	if snapshot.MemoryUsage != 1024*1024 {
		t.Error("PerformanceSnapshot should preserve MemoryUsage")
	}

	if snapshot.ConcurrencyLevel != 4 {
		t.Error("PerformanceSnapshot should preserve ConcurrencyLevel")
	}

	if snapshot.ErrorRate != 0.01 {
		t.Error("PerformanceSnapshot should preserve ErrorRate")
	}
}
