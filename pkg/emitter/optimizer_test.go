package emitter

import (
	"context"
	"reflect"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"

	"github.com/reedom/convergen/v9/pkg/domain"
)

func TestCodeOptimizer_OptimizeCode(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	config.OptimizationLevel = OptimizationBasic
	metrics := NewMetrics()
	optimizer := NewCodeOptimizer(config, logger, metrics)

	// Create test method
	sourceType := domain.NewBasicType("Source", reflect.Struct)
	destType := domain.NewBasicType("Dest", reflect.Struct)

	_, err := domain.NewMethod("TestMethod", sourceType, destType)
	if err != nil {
		t.Fatalf("Failed to create method: %v", err)
	}

	// Create test generated code
	generatedCode := &GeneratedCode{
		PackageName: "testpkg",
		Methods: []*MethodCode{
			{
				Name:      "TestMethod",
				Signature: "func TestMethod() error",
				Body:      "return nil\n\n\n\nreturn nil",
			},
		},
		BaseCode: "// Base code",
		Metrics:  NewGenerationMetrics(),
	}

	ctx := context.Background()

	optimized, err := optimizer.OptimizeCode(ctx, generatedCode)
	if err != nil {
		t.Fatalf("OptimizeCode failed: %v", err)
	}

	if optimized == nil {
		t.Fatal("OptimizeCode returned nil")
	}

	if optimized == generatedCode {
		t.Error("OptimizeCode should return a copy, not the original")
	}

	// Test with nil input
	_, err = optimizer.OptimizeCode(ctx, nil)
	if err == nil {
		t.Error("OptimizeCode should fail with nil input")
	}

	// Test with optimization disabled
	config.OptimizationLevel = OptimizationNone
	optimizerNone := NewCodeOptimizer(config, logger, metrics)

	optimized2, err := optimizerNone.OptimizeCode(ctx, generatedCode)
	if err != nil {
		t.Fatalf("OptimizeCode with OptimizationNone failed: %v", err)
	}

	if optimized2 != generatedCode {
		t.Error("OptimizeCode with OptimizationNone should return original code")
	}
}

func TestCodeOptimizer_OptimizationLevels(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics := NewMetrics()

	levels := []OptimizationLevel{
		OptimizationBasic,
		OptimizationAggressive,
		OptimizationMaximal,
	}

	generatedCode := &GeneratedCode{
		PackageName: "testpkg",
		Methods: []*MethodCode{
			{
				Name:      "TestMethod",
				Signature: "func TestMethod() error",
				Body:      "converted_result := src.Field\nreturn nil",
			},
		},
		Metrics: NewGenerationMetrics(),
	}

	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			config := DefaultConfig()
			config.OptimizationLevel = level
			optimizer := NewCodeOptimizer(config, logger, metrics)

			ctx := context.Background()

			optimized, err := optimizer.OptimizeCode(ctx, generatedCode)
			if err != nil {
				t.Fatalf("OptimizeCode failed for level %s: %v", level, err)
			}

			if optimized == nil {
				t.Fatalf("OptimizeCode returned nil for level %s", level)
			}

			t.Logf("Optimization level %s completed", level)
		})
	}
}

func TestCodeOptimizer_OptimizeMethodCode(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	config.OptimizationLevel = OptimizationBasic
	metrics := NewMetrics()
	optimizer := NewCodeOptimizer(config, logger, metrics).(*ConcreteCodeOptimizer)

	// Test with method code
	method := &MethodCode{
		Name:          "TestMethod",
		Signature:     "func TestMethod() error",
		Body:          "converted_result := src.Field\n\n\nreturn nil",
		ErrorHandling: "temporary_err := err\nreturn temporary_err",
	}

	err := optimizer.OptimizeMethodCode(method)
	if err != nil {
		t.Fatalf("OptimizeMethodCode failed: %v", err)
	}

	// Test with nil method
	err = optimizer.OptimizeMethodCode(nil)
	if err == nil {
		t.Error("OptimizeMethodCode should fail with nil method")
	}

	// Test with optimization disabled
	config.OptimizationLevel = OptimizationNone
	optimizerNone := NewCodeOptimizer(config, logger, metrics).(*ConcreteCodeOptimizer)
	method2 := &MethodCode{
		Name: "TestMethod2",
		Body: "some code",
	}

	err = optimizerNone.OptimizeMethodCode(method2)
	if err != nil {
		t.Fatalf("OptimizeMethodCode with OptimizationNone failed: %v", err)
	}
}

func TestCodeOptimizer_EliminateDeadCode(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	config.EnableDeadCodeElim = true
	metrics := NewMetrics()
	optimizer := NewCodeOptimizer(config, logger, metrics).(*ConcreteCodeOptimizer)

	generatedCode := &GeneratedCode{
		Methods: []*MethodCode{
			{
				Name: "TestMethod",
				Body: "line1\n\n\nline2\n\n\nline3",
			},
		},
	}

	err := optimizer.EliminateDeadCode(generatedCode)
	if err != nil {
		t.Fatalf("EliminateDeadCode failed: %v", err)
	}

	// Test with dead code elimination disabled
	config.EnableDeadCodeElim = false
	optimizerDisabled := NewCodeOptimizer(config, logger, metrics).(*ConcreteCodeOptimizer)

	err = optimizerDisabled.EliminateDeadCode(generatedCode)
	if err != nil {
		t.Fatalf("EliminateDeadCode with disabled flag failed: %v", err)
	}
}

func TestCodeOptimizer_OptimizeVariableNames(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	config.EnableVarOptimization = true
	metrics := NewMetrics()
	optimizer := NewCodeOptimizer(config, logger, metrics).(*ConcreteCodeOptimizer)

	generatedCode := &GeneratedCode{
		Methods: []*MethodCode{
			{
				Name: "TestMethod",
				Body: "converted_result := src.Field\ntemporary_value := result_data",
			},
		},
	}

	err := optimizer.OptimizeVariableNames(generatedCode)
	if err != nil {
		t.Fatalf("OptimizeVariableNames failed: %v", err)
	}

	// Test with variable optimization disabled
	config.EnableVarOptimization = false
	optimizerDisabled := NewCodeOptimizer(config, logger, metrics).(*ConcreteCodeOptimizer)

	err = optimizerDisabled.OptimizeVariableNames(generatedCode)
	if err != nil {
		t.Fatalf("OptimizeVariableNames with disabled flag failed: %v", err)
	}
}

func TestCodeOptimizer_SimplifyExpressions(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	metrics := NewMetrics()
	optimizer := NewCodeOptimizer(config, logger, metrics).(*ConcreteCodeOptimizer)

	generatedCode := &GeneratedCode{
		Methods: []*MethodCode{
			{
				Name: "TestMethod",
				Body: "result := (variable)\nother := (anotherVar)",
			},
		},
	}

	err := optimizer.SimplifyExpressions(generatedCode)
	if err != nil {
		t.Fatalf("SimplifyExpressions failed: %v", err)
	}
}

func TestCodeOptimizer_RemoveRedundancy(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	metrics := NewMetrics()
	optimizer := NewCodeOptimizer(config, logger, metrics).(*ConcreteCodeOptimizer)

	generatedCode := &GeneratedCode{
		Methods: []*MethodCode{
			{
				Name: "TestMethod",
				Body: "line1\n\n\n\nline2",
			},
		},
	}

	err := optimizer.RemoveRedundancy(generatedCode)
	if err != nil {
		t.Fatalf("RemoveRedundancy failed: %v", err)
	}
}

func TestCodeOptimizer_GetMetrics(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	metrics := NewMetrics()
	optimizer := NewCodeOptimizer(config, logger, metrics).(*ConcreteCodeOptimizer)

	metricsResult := optimizer.GetMetrics()
	if metricsResult == nil {
		t.Error("GetMetrics should return metrics")
	}

	if metricsResult != nil && metricsResult.OptimizationsApplied == nil {
		t.Error("OptimizationsApplied should be initialized")
	}
}

func TestCodeOptimizer_Shutdown(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	metrics := NewMetrics()
	optimizer := NewCodeOptimizer(config, logger, metrics)

	ctx := context.Background()

	err := optimizer.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

func TestDeadCodeEliminator(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eliminator := NewDeadCodeEliminator(logger)

	// Test EliminateInCode
	code := "line1\n\n\nline2\n\nline3"

	optimized, eliminated, err := eliminator.EliminateInCode(code)
	if err != nil {
		t.Fatalf("EliminateInCode failed: %v", err)
	}

	if optimized == "" {
		t.Error("EliminateInCode should return optimized code")
	}

	if eliminated < 0 {
		t.Error("EliminateInCode should return non-negative elimination count")
	}

	t.Logf("Eliminated %d empty lines", eliminated)

	// Test FindUnusedVariables
	unused, err := eliminator.FindUnusedVariables("var x int\nx = 5")
	if err != nil {
		t.Fatalf("FindUnusedVariables failed: %v", err)
	}

	if unused == nil {
		t.Error("FindUnusedVariables should return slice")
	}

	// Test FindUnreachableCode
	unreachable, err := eliminator.FindUnreachableCode("return nil\ndeadCode := true")
	if err != nil {
		t.Fatalf("FindUnreachableCode failed: %v", err)
	}

	if unreachable == nil {
		t.Error("FindUnreachableCode should return slice")
	}
}

func TestVariableOptimizer(t *testing.T) {
	logger := zaptest.NewLogger(t)
	optimizer := NewVariableOptimizer(logger)

	// Test OptimizeNames
	code := "converted_result := src.Field\ntemporary_value := result_data"

	optimized, optimizations, err := optimizer.OptimizeNames(code)
	if err != nil {
		t.Fatalf("OptimizeNames failed: %v", err)
	}

	if optimized == "" {
		t.Error("OptimizeNames should return optimized code")
	}

	if optimizations < 0 {
		t.Error("OptimizeNames should return non-negative optimization count")
	}

	t.Logf("Applied %d variable optimizations", optimizations)

	// Test DetectConflicts
	conflicts, err := optimizer.DetectConflicts("var x int\nvar x string")
	if err != nil {
		t.Fatalf("DetectConflicts failed: %v", err)
	}

	if conflicts == nil {
		t.Error("DetectConflicts should return slice")
	}

	// Test ShortenNames
	shortened, err := optimizer.ShortenNames("veryLongVariableName := 42")
	if err != nil {
		t.Fatalf("ShortenNames failed: %v", err)
	}

	if shortened == "" {
		t.Error("ShortenNames should return code")
	}
}

func TestExpressionSimplifier(t *testing.T) {
	logger := zaptest.NewLogger(t)
	simplifier := NewExpressionSimplifier(logger)

	// Test SimplifyInCode
	code := "result := (variable)\nother := (anotherVar)"

	simplified, simplifications, err := simplifier.SimplifyInCode(code)
	if err != nil {
		t.Fatalf("SimplifyInCode failed: %v", err)
	}

	if simplified == "" {
		t.Error("SimplifyInCode should return simplified code")
	}

	if simplifications < 0 {
		t.Error("SimplifyInCode should return non-negative simplification count")
	}

	t.Logf("Applied %d expression simplifications", simplifications)

	// Test SimplifyBooleanExpressions
	boolSimplified, err := simplifier.SimplifyBooleanExpressions("if true && x { }")
	if err != nil {
		t.Fatalf("SimplifyBooleanExpressions failed: %v", err)
	}

	if boolSimplified == "" {
		t.Error("SimplifyBooleanExpressions should return code")
	}

	// Test SimplifyArithmeticExpressions
	arithSimplified, err := simplifier.SimplifyArithmeticExpressions("x := 1 + 2")
	if err != nil {
		t.Fatalf("SimplifyArithmeticExpressions failed: %v", err)
	}

	if arithSimplified == "" {
		t.Error("SimplifyArithmeticExpressions should return code")
	}
}

func TestRedundancyRemover(t *testing.T) {
	logger := zaptest.NewLogger(t)
	remover := NewRedundancyRemover(logger)

	// Test RemoveInCode
	code := "line1\n\n\n\nline2"

	optimized, removed, err := remover.RemoveInCode(code)
	if err != nil {
		t.Fatalf("RemoveInCode failed: %v", err)
	}

	if optimized == "" {
		t.Error("RemoveInCode should return optimized code")
	}

	if removed < 0 {
		t.Error("RemoveInCode should return non-negative removal count")
	}

	t.Logf("Removed %d redundant elements", removed)

	// Test FindRedundantAssignments
	assignments, err := remover.FindRedundantAssignments("x := 1\nx := 1")
	if err != nil {
		t.Fatalf("FindRedundantAssignments failed: %v", err)
	}

	if assignments == nil {
		t.Error("FindRedundantAssignments should return slice")
	}

	// Test FindDuplicateCode
	duplicates, err := remover.FindDuplicateCode("code\ncode")
	if err != nil {
		t.Fatalf("FindDuplicateCode failed: %v", err)
	}

	if duplicates == nil {
		t.Error("FindDuplicateCode should return slice")
	}
}

func TestASTAnalyzer(t *testing.T) {
	logger := zaptest.NewLogger(t)
	analyzer := NewASTAnalyzer(logger)

	// Test ParseCode
	code := "package main\nfunc main() {}"

	file, fset, err := analyzer.ParseCode(code)
	if err != nil {
		t.Fatalf("ParseCode failed: %v", err)
	}

	if file == nil {
		t.Error("ParseCode should return AST file")
	}

	if fset == nil {
		t.Error("ParseCode should return file set")
	}

	// Test AnalyzeUsage
	usage, err := analyzer.AnalyzeUsage(file)
	if err != nil {
		t.Fatalf("AnalyzeUsage failed: %v", err)
	}

	if usage == nil {
		t.Error("AnalyzeUsage should return usage analysis")
	}

	if usage != nil && usage.Variables == nil {
		t.Error("AnalyzeUsage should initialize Variables map")
	}

	// Test FindDefinitions
	definitions, err := analyzer.FindDefinitions(file)
	if err != nil {
		t.Fatalf("FindDefinitions failed: %v", err)
	}

	if definitions == nil {
		t.Error("FindDefinitions should return slice")
	}

	// Test with invalid code
	_, _, err = analyzer.ParseCode("invalid go code {")
	if err == nil {
		t.Error("ParseCode should fail with invalid code")
	}
}

func TestControlFlowAnalyzer(t *testing.T) {
	logger := zaptest.NewLogger(t)
	analyzer := NewControlFlowAnalyzer(logger)

	// Create a simple AST for testing
	astAnalyzer := NewASTAnalyzer(logger)
	code := "package main\nfunc main() { if true { return } }"

	file, _, err := astAnalyzer.ParseCode(code)
	if err != nil {
		t.Fatalf("Failed to parse code for CFG test: %v", err)
	}

	// Test AnalyzeControlFlow
	cfg, err := analyzer.AnalyzeControlFlow(file)
	if err != nil {
		t.Fatalf("AnalyzeControlFlow failed: %v", err)
	}

	if cfg == nil {
		t.Error("AnalyzeControlFlow should return control flow graph")
	}

	if cfg != nil && cfg.Nodes == nil {
		t.Error("AnalyzeControlFlow should initialize Nodes slice")
	}

	if cfg.Edges == nil {
		t.Error("AnalyzeControlFlow should initialize Edges slice")
	}

	// Test FindUnreachableBlocks
	unreachable, err := analyzer.FindUnreachableBlocks(cfg)
	if err != nil {
		t.Fatalf("FindUnreachableBlocks failed: %v", err)
	}

	if unreachable == nil {
		t.Error("FindUnreachableBlocks should return slice")
	}

	// Test CalculateComplexity
	complexity, err := analyzer.CalculateComplexity(cfg)
	if err != nil {
		t.Fatalf("CalculateComplexity failed: %v", err)
	}

	if complexity < 1 {
		t.Error("CalculateComplexity should return positive complexity")
	}

	t.Logf("Calculated complexity: %d", complexity)
}

func TestOptimizerMetrics(t *testing.T) {
	metrics := NewOptimizerMetrics()

	if metrics == nil {
		t.Error("NewOptimizerMetrics should return metrics")
		return
	}

	if metrics.OptimizationsApplied == nil {
		t.Error("NewOptimizerMetrics should initialize OptimizationsApplied")
		return
	}

	// Test metrics updates
	metrics.OptimizationsApplied["test"] = 5
	metrics.TotalOptimizationTime = time.Millisecond * 100
	metrics.DeadCodeEliminated = 10
	metrics.VariablesOptimized = 15
	metrics.ExpressionsSimplified = 8
	metrics.RedundancyRemoved = 3
	metrics.BytesSaved = 256
	metrics.PerformanceGain = 1.5

	if metrics.OptimizationsApplied["test"] != 5 {
		t.Error("Metrics should track optimizations applied")
	}

	if metrics.TotalOptimizationTime != time.Millisecond*100 {
		t.Error("Metrics should track optimization time")
	}
}

func TestOptimizer_EdgeCases(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	metrics := NewMetrics()
	optimizer := NewCodeOptimizer(config, logger, metrics).(*ConcreteCodeOptimizer)

	// Test with empty generated code
	emptyCode := &GeneratedCode{
		PackageName: "empty",
		Methods:     []*MethodCode{},
		Metrics:     NewGenerationMetrics(),
	}

	ctx := context.Background()

	optimized, err := optimizer.OptimizeCode(ctx, emptyCode)
	if err != nil {
		t.Fatalf("OptimizeCode with empty code failed: %v", err)
	}

	if optimized == nil {
		t.Error("OptimizeCode should handle empty code")
	}

	// Test with methods containing empty bodies
	codeWithEmptyMethods := &GeneratedCode{
		PackageName: "test",
		Methods: []*MethodCode{
			{
				Name:      "EmptyMethod",
				Signature: "func EmptyMethod()",
				Body:      "",
			},
		},
		Metrics: NewGenerationMetrics(),
	}

	optimized2, err := optimizer.OptimizeCode(ctx, codeWithEmptyMethods)
	if err != nil {
		t.Fatalf("OptimizeCode with empty method bodies failed: %v", err)
	}

	if optimized2 == nil {
		t.Error("OptimizeCode should handle empty method bodies")
	}

	// Test optimization components with edge cases
	deadCodeElim := NewDeadCodeEliminator(logger)

	_, _, err = deadCodeElim.EliminateInCode("")
	if err != nil {
		t.Fatalf("DeadCodeEliminator should handle empty code: %v", err)
	}

	varOpt := NewVariableOptimizer(logger)

	_, _, err = varOpt.OptimizeNames("")
	if err != nil {
		t.Fatalf("VariableOptimizer should handle empty code: %v", err)
	}

	exprSimp := NewExpressionSimplifier(logger)

	_, _, err = exprSimp.SimplifyInCode("")
	if err != nil {
		t.Fatalf("ExpressionSimplifier should handle empty code: %v", err)
	}

	redRem := NewRedundancyRemover(logger)

	_, _, err = redRem.RemoveInCode("")
	if err != nil {
		t.Fatalf("RedundancyRemover should handle empty code: %v", err)
	}
}
