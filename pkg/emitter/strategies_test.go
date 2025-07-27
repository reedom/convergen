package emitter

import (
	"context"
	"reflect"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"

	"github.com/reedom/convergen/v8/pkg/domain"
	"github.com/reedom/convergen/v8/pkg/executor"
)

func TestCompositeLiteralStrategy(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultEmitterConfig()
	strategy := NewCompositeLiteralStrategy(config, logger)

	if strategy.Name() != "composite_literal" {
		t.Errorf("Expected strategy name 'composite_literal', got '%s'", strategy.Name())
	}

	// Create test method with proper domain constructors
	sourceType := domain.NewBasicType("User", reflect.Struct)
	destType := domain.NewBasicType("UserDTO", reflect.Struct)
	method, err := domain.NewMethod("ConvertSimple", sourceType, destType)
	if err != nil {
		t.Fatalf("Failed to create method: %v", err)
	}

	// Test with simple method that can be handled
	simpleMethod := &domain.MethodResult{
		Method:      method,
		Success:     true,
		Error:       nil,
		Code:        "",
		ProcessedAt: time.Now(),
		DurationMS:  5,
		Metadata: map[string]interface{}{
			"Name": &executor.FieldResult{
				FieldID:      "Name",
				Success:      true,
				Result:       "src.Name",
				StrategyUsed: "direct",
				Duration:     time.Millisecond,
			},
			"Email": &executor.FieldResult{
				FieldID:      "Email",
				Success:      true,
				Result:       "src.Email",
				StrategyUsed: "direct",
				Duration:     time.Millisecond,
			},
		},
	}

	if !strategy.CanHandle(simpleMethod) {
		t.Error("Composite literal strategy should handle simple methods")
	}

	// Test code generation
	templateData := &TemplateData{
		Method: simpleMethod,
		Config: config,
	}

	ctx := context.Background()
	code, err := strategy.GenerateCode(ctx, simpleMethod, templateData)

	if err != nil {
		t.Fatalf("Code generation failed: %v", err)
	}

	if code == "" {
		t.Error("Generated code should not be empty")
	}

	t.Logf("Composite literal code:\n%s", code)

	// Test complexity calculation
	complexity := strategy.GetComplexity(simpleMethod)
	if complexity == nil {
		t.Error("Complexity metrics should not be nil")
	}

	if complexity.RecommendedStrategy != StrategyCompositeLiteral {
		t.Errorf("Expected recommended strategy to be composite literal, got %s",
			complexity.RecommendedStrategy)
	}

	// Test required imports
	imports := strategy.GetRequiredImports(simpleMethod)
	// Composite literals typically don't require additional imports
	if len(imports) > 0 {
		t.Logf("Composite literal strategy imports: %d", len(imports))
	}
}

func TestCompositeLiteralStrategy_CannotHandle(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultEmitterConfig()
	config.MaxFieldsForComposite = 2 // Low threshold for testing
	strategy := NewCompositeLiteralStrategy(config, logger)

	// Create test method for too many fields
	sourceType := domain.NewBasicType("Source", reflect.Struct)
	destType := domain.NewBasicType("Dest", reflect.Struct)
	method, err := domain.NewMethod("ConvertMany", sourceType, destType)
	if err != nil {
		t.Fatalf("Failed to create method: %v", err)
	}

	// Test with too many fields
	manyFieldsMethod := &domain.MethodResult{
		Method:  method,
		Success: true,
		Metadata: map[string]interface{}{
			"Field1": &executor.FieldResult{FieldID: "Field1", Success: true, StrategyUsed: "direct"},
			"Field2": &executor.FieldResult{FieldID: "Field2", Success: true, StrategyUsed: "direct"},
			"Field3": &executor.FieldResult{FieldID: "Field3", Success: true, StrategyUsed: "direct"},
		},
	}

	if strategy.CanHandle(manyFieldsMethod) {
		t.Error("Composite literal strategy should not handle methods with too many fields")
	}

	// Test with error fields
	errorMethod2, err := domain.NewMethod("ConvertWithError", sourceType, destType)
	if err != nil {
		t.Fatalf("Failed to create error method: %v", err)
	}

	errorMethod := &domain.MethodResult{
		Method:  errorMethod2,
		Success: false,
		Metadata: map[string]interface{}{
			"Field1": &executor.FieldResult{
				FieldID:      "Field1",
				Success:      false,
				Error:        &executor.ExecutionError{FieldID: "Field1", Error: "error"},
				StrategyUsed: "converter",
			},
		},
	}

	if strategy.CanHandle(errorMethod) {
		t.Error("Composite literal strategy should not handle methods with errors")
	}

	// Test with nil method
	if strategy.CanHandle(nil) {
		t.Error("Composite literal strategy should not handle nil methods")
	}
}

func TestAssignmentBlockStrategy(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultEmitterConfig()
	strategy := NewAssignmentBlockStrategy(config, logger)

	if strategy.Name() != "assignment_block" {
		t.Errorf("Expected strategy name 'assignment_block', got '%s'", strategy.Name())
	}

	// Create test method
	sourceType := domain.NewBasicType("Source", reflect.Struct)
	destType := domain.NewBasicType("Dest", reflect.Struct)
	method, err := domain.NewMethod("ConvertAny", sourceType, destType)
	if err != nil {
		t.Fatalf("Failed to create method: %v", err)
	}

	// Assignment block strategy should handle any method
	simpleMethod := &domain.MethodResult{
		Method:  method,
		Success: true,
		Metadata: map[string]interface{}{
			"Field1": &executor.FieldResult{
				FieldID:      "Field1",
				Success:      true,
				Result:       "src.Field1",
				StrategyUsed: "direct",
				Duration:     time.Millisecond,
			},
		},
	}

	if !strategy.CanHandle(simpleMethod) {
		t.Error("Assignment block strategy should handle any method")
	}

	if strategy.CanHandle(nil) {
		t.Error("Assignment block strategy should not handle nil")
	}

	// Test with complex method including errors
	complexMethod2, err := domain.NewMethod("ConvertComplex", sourceType, destType)
	if err != nil {
		t.Fatalf("Failed to create complex method: %v", err)
	}

	complexMethod := &domain.MethodResult{
		Method:  complexMethod2,
		Success: false,
		Metadata: map[string]interface{}{
			"SuccessField": &executor.FieldResult{
				FieldID:      "SuccessField",
				Success:      true,
				Result:       "src.SuccessField",
				StrategyUsed: "direct",
				Duration:     time.Millisecond,
			},
			"ErrorField": &executor.FieldResult{
				FieldID:      "ErrorField",
				Success:      false,
				Error:        &executor.ExecutionError{FieldID: "ErrorField", Error: "conversion failed"},
				StrategyUsed: "converter",
				Duration:     10 * time.Millisecond,
			},
		},
	}

	templateData := &TemplateData{
		Method: complexMethod,
		Config: config,
	}

	ctx := context.Background()
	code, err := strategy.GenerateCode(ctx, complexMethod, templateData)

	if err != nil {
		t.Fatalf("Assignment block code generation failed: %v", err)
	}

	if code == "" {
		t.Error("Generated code should not be empty")
	}

	t.Logf("Assignment block code:\n%s", code)

	// Test complexity calculation
	complexity := strategy.GetComplexity(complexMethod)
	if complexity == nil {
		t.Error("Complexity metrics should not be nil")
	}

	if complexity.ErrorFields == 0 {
		t.Error("Should detect error fields in complexity analysis")
	}

	// Test required imports (should include fmt for error handling)
	imports := strategy.GetRequiredImports(complexMethod)
	fmtImportFound := false
	for _, imp := range imports {
		if imp.Path == "fmt" {
			fmtImportFound = true
			break
		}
	}

	if !fmtImportFound {
		t.Error("Assignment block strategy should require fmt import for error handling")
	}
}

func TestMixedApproachStrategy(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultEmitterConfig()
	strategy := NewMixedApproachStrategy(config, logger)

	if strategy.Name() != "mixed_approach" {
		t.Errorf("Expected strategy name 'mixed_approach', got '%s'", strategy.Name())
	}

	// Create test method
	sourceType := domain.NewBasicType("Source", reflect.Struct)
	destType := domain.NewBasicType("Dest", reflect.Struct)
	method, err := domain.NewMethod("ConvertMixed", sourceType, destType)
	if err != nil {
		t.Fatalf("Failed to create method: %v", err)
	}

	// Test with method that has both simple and complex fields
	mixedMethod := &domain.MethodResult{
		Method:  method,
		Success: true,
		Metadata: map[string]interface{}{
			"SimpleField1": &executor.FieldResult{
				FieldID:      "SimpleField1",
				Success:      true,
				Result:       "src.SimpleField1",
				StrategyUsed: "direct",
				Duration:     time.Millisecond,
			},
			"SimpleField2": &executor.FieldResult{
				FieldID:      "SimpleField2",
				Success:      true,
				Result:       "src.SimpleField2",
				StrategyUsed: "literal",
				Duration:     time.Millisecond,
			},
			"ComplexField": &executor.FieldResult{
				FieldID:      "ComplexField",
				Success:      true,
				Result:       "converter.Convert(src.ComplexField)",
				StrategyUsed: "converter",
				Duration:     5 * time.Millisecond,
			},
			"ErrorField": &executor.FieldResult{
				FieldID:      "ErrorField",
				Success:      false,
				Error:        &executor.ExecutionError{FieldID: "ErrorField", Error: "failed"},
				StrategyUsed: "converter",
				Duration:     10 * time.Millisecond,
			},
		},
	}

	if !strategy.CanHandle(mixedMethod) {
		t.Error("Mixed approach strategy should handle methods with mixed complexity")
	}

	// Test code generation
	templateData := &TemplateData{
		Method: mixedMethod,
		Config: config,
	}

	ctx := context.Background()
	code, err := strategy.GenerateCode(ctx, mixedMethod, templateData)

	if err != nil {
		t.Fatalf("Mixed approach code generation failed: %v", err)
	}

	if code == "" {
		t.Error("Generated code should not be empty")
	}

	t.Logf("Mixed approach code:\n%s", code)

	// Test complexity calculation
	complexity := strategy.GetComplexity(mixedMethod)
	if complexity == nil {
		t.Error("Complexity metrics should not be nil")
	}

	if complexity.RecommendedStrategy != StrategyMixedApproach {
		t.Errorf("Expected recommended strategy to be mixed approach, got %s",
			complexity.RecommendedStrategy)
	}
}

func TestMixedApproachStrategy_CannotHandle(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultEmitterConfig()
	strategy := NewMixedApproachStrategy(config, logger)

	// Create test method
	sourceType := domain.NewBasicType("Source", reflect.Struct)
	destType := domain.NewBasicType("Dest", reflect.Struct)
	method, err := domain.NewMethod("ConvertFew", sourceType, destType)
	if err != nil {
		t.Fatalf("Failed to create method: %v", err)
	}

	// Test with too few fields
	fewFieldsMethod := &domain.MethodResult{
		Method:  method,
		Success: true,
		Metadata: map[string]interface{}{
			"Field1": &executor.FieldResult{FieldID: "Field1", Success: true, StrategyUsed: "direct"},
			"Field2": &executor.FieldResult{FieldID: "Field2", Success: true, StrategyUsed: "direct"},
		},
	}

	if strategy.CanHandle(fewFieldsMethod) {
		t.Error("Mixed approach strategy should not handle methods with too few fields")
	}

	// Create method for all simple fields test
	method2, err := domain.NewMethod("ConvertAllSimple", sourceType, destType)
	if err != nil {
		t.Fatalf("Failed to create all simple method: %v", err)
	}

	// Test with all simple fields
	allSimpleMethod := &domain.MethodResult{
		Method:  method2,
		Success: true,
		Metadata: map[string]interface{}{
			"Field1": &executor.FieldResult{FieldID: "Field1", Success: true, StrategyUsed: "direct"},
			"Field2": &executor.FieldResult{FieldID: "Field2", Success: true, StrategyUsed: "direct"},
			"Field3": &executor.FieldResult{FieldID: "Field3", Success: true, StrategyUsed: "literal"},
		},
	}

	// This might or might not be handled depending on the exact logic
	canHandle := strategy.CanHandle(allSimpleMethod)
	t.Logf("Mixed approach can handle all simple fields: %v", canHandle)

	// Test with nil method
	if strategy.CanHandle(nil) {
		t.Error("Mixed approach strategy should not handle nil methods")
	}
}

func TestStrategies_RequiredImports(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultEmitterConfig()

	// Create test method
	sourceType := domain.NewBasicType("Source", reflect.Struct)
	destType := domain.NewBasicType("Dest", reflect.Struct)
	method, err := domain.NewMethod("ConvertWithErrors", sourceType, destType)
	if err != nil {
		t.Fatalf("Failed to create method: %v", err)
	}

	// Test method with error handling needs
	errorMethod := &domain.MethodResult{
		Method:  method,
		Success: false,
		Metadata: map[string]interface{}{
			"ErrorField": &executor.FieldResult{
				FieldID:      "ErrorField",
				Success:      false,
				Error:        &executor.ExecutionError{FieldID: "ErrorField", Error: "failed"},
				StrategyUsed: "converter",
			},
		},
	}

	strategies := []GenerationStrategy{
		NewCompositeLiteralStrategy(config, logger),
		NewAssignmentBlockStrategy(config, logger),
		NewMixedApproachStrategy(config, logger),
	}

	for _, strategy := range strategies {
		imports := strategy.GetRequiredImports(errorMethod)
		t.Logf("%s strategy requires %d imports", strategy.Name(), len(imports))

		// Check if fmt import is included for error handling
		fmtFound := false
		for _, imp := range imports {
			if imp.Path == "fmt" && imp.Standard && imp.Required {
				fmtFound = true
				break
			}
		}

		// Assignment block and mixed approach should require fmt for error handling
		if (strategy.Name() == "assignment_block" || strategy.Name() == "mixed_approach") && !fmtFound {
			t.Errorf("%s strategy should require fmt import for error handling", strategy.Name())
		}
	}
}

func TestStrategies_ComplexityMetrics(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultEmitterConfig()

	// Create test methods
	sourceType := domain.NewBasicType("Source", reflect.Struct)
	destType := domain.NewBasicType("Dest", reflect.Struct)
	simpleMethodObj, err := domain.NewMethod("Simple", sourceType, destType)
	if err != nil {
		t.Fatalf("Failed to create simple method: %v", err)
	}

	complexMethodObj, err := domain.NewMethod("Complex", sourceType, destType)
	if err != nil {
		t.Fatalf("Failed to create complex method: %v", err)
	}

	// Create methods with different complexity levels
	simpleMethod := &domain.MethodResult{
		Method:  simpleMethodObj,
		Success: true,
		Metadata: map[string]interface{}{
			"Field1": &executor.FieldResult{FieldID: "Field1", Success: true, StrategyUsed: "direct"},
		},
	}

	complexMethod := &domain.MethodResult{
		Method:  complexMethodObj,
		Success: false,
		Metadata: map[string]interface{}{
			"Field1": &executor.FieldResult{
				FieldID:      "Field1",
				Success:      false,
				Error:        &executor.ExecutionError{FieldID: "Field1", Error: "error"},
				StrategyUsed: "converter",
			},
			"Field2": &executor.FieldResult{FieldID: "Field2", Success: true, StrategyUsed: "converter"},
			"Field3": &executor.FieldResult{FieldID: "Field3", Success: true, StrategyUsed: "expression"},
		},
	}

	strategies := []GenerationStrategy{
		NewCompositeLiteralStrategy(config, logger),
		NewAssignmentBlockStrategy(config, logger),
		NewMixedApproachStrategy(config, logger),
	}

	for _, strategy := range strategies {
		// Test simple method complexity
		simpleComplexity := strategy.GetComplexity(simpleMethod)
		if simpleComplexity == nil {
			t.Errorf("%s strategy should return complexity metrics", strategy.Name())
			continue
		}

		// Test complex method complexity
		complexComplexity := strategy.GetComplexity(complexMethod)
		if complexComplexity == nil {
			t.Errorf("%s strategy should return complexity metrics for complex method", strategy.Name())
			continue
		}

		t.Logf("%s strategy - Simple: %f, Complex: %f",
			strategy.Name(), simpleComplexity.ComplexityScore, complexComplexity.ComplexityScore)

		// Complex method should generally have higher complexity score
		if complexComplexity.ComplexityScore <= simpleComplexity.ComplexityScore {
			t.Logf("Note: %s strategy shows complex method has same or lower complexity than simple", strategy.Name())
		}
	}
}

func TestStrategies_EdgeCases(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultEmitterConfig()

	strategies := []GenerationStrategy{
		NewCompositeLiteralStrategy(config, logger),
		NewAssignmentBlockStrategy(config, logger),
		NewMixedApproachStrategy(config, logger),
	}

	// Create test method
	sourceType := domain.NewBasicType("Source", reflect.Struct)
	destType := domain.NewBasicType("Dest", reflect.Struct)
	method, err := domain.NewMethod("Empty", sourceType, destType)
	if err != nil {
		t.Fatalf("Failed to create empty method: %v", err)
	}

	// Test with empty method
	emptyMethod := &domain.MethodResult{
		Method:   method,
		Success:  true,
		Metadata: map[string]interface{}{},
	}

	for _, strategy := range strategies {
		t.Logf("Testing %s strategy with empty method", strategy.Name())

		canHandle := strategy.CanHandle(emptyMethod)
		t.Logf("%s can handle empty method: %v", strategy.Name(), canHandle)

		if canHandle {
			templateData := &TemplateData{
				Method: emptyMethod,
				Config: config,
			}

			ctx := context.Background()
			code, err := strategy.GenerateCode(ctx, emptyMethod, templateData)

			if err != nil {
				t.Errorf("%s strategy failed with empty method: %v", strategy.Name(), err)
			} else {
				t.Logf("%s generated for empty method:\n%s", strategy.Name(), code)
			}
		}

		complexity := strategy.GetComplexity(emptyMethod)
		if complexity == nil {
			t.Errorf("%s strategy should return complexity for empty method", strategy.Name())
		}

		imports := strategy.GetRequiredImports(emptyMethod)
		t.Logf("%s strategy requires %d imports for empty method", strategy.Name(), len(imports))
	}
}
