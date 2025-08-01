package emitter

import (
	"context"
	"math"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v8/pkg/domain"
	"github.com/reedom/convergen/v8/pkg/executor"
)

// OutputStrategy determines the optimal code generation approach.
type OutputStrategy interface {
	// SelectStrategy determines the best construction strategy for a method
	SelectStrategy(ctx context.Context, method *domain.MethodResult) ConstructionStrategy

	// AnalyzeFieldComplexity analyzes the complexity of fields for strategy selection
	AnalyzeFieldComplexity(fields []*executor.FieldResult) *ComplexityMetrics

	// ShouldUseCompositeLiteral determines if composite literal approach is optimal
	ShouldUseCompositeLiteral(method *domain.MethodResult) bool

	// EstimatePerformance estimates the performance characteristics of different strategies
	EstimatePerformance(method *domain.MethodResult) *PerformanceEstimate
}

// ConcreteOutputStrategy implements OutputStrategy.
type ConcreteOutputStrategy struct {
	config *Config
	logger *zap.Logger

	// Strategy selection weights
	complexityWeight      float64
	performanceWeight     float64
	readabilityWeight     float64
	maintainabilityWeight float64
}

// PerformanceEstimate contains performance estimates for different strategies.
type PerformanceEstimate struct {
	CompositeLiteral *StrategyEstimate    `json:"composite_literal"`
	AssignmentBlock  *StrategyEstimate    `json:"assignment_block"`
	MixedApproach    *StrategyEstimate    `json:"mixed_approach"`
	Recommended      ConstructionStrategy `json:"recommended"`
}

// StrategyEstimate contains estimates for a specific strategy.
type StrategyEstimate struct {
	GenerationTime   float64 `json:"generation_time"`   // Estimated generation time in ms
	ExecutionTime    float64 `json:"execution_time"`    // Estimated runtime execution time in ns
	MemoryUsage      int     `json:"memory_usage"`      // Estimated memory usage in bytes
	ReadabilityScore float64 `json:"readability_score"` // Readability score (0-100)
	Complexity       float64 `json:"complexity"`        // Complexity score (0-100)
	LinesOfCode      int     `json:"lines_of_code"`     // Estimated lines of generated code
}

// NewOutputStrategy creates a new output strategy analyzer.
func NewOutputStrategy(config *Config, logger *zap.Logger) OutputStrategy {
	return &ConcreteOutputStrategy{
		config:                config,
		logger:                logger,
		complexityWeight:      0.3,
		performanceWeight:     0.25,
		readabilityWeight:     0.25,
		maintainabilityWeight: 0.2,
	}
}

// SelectStrategy determines the best construction strategy for a method.
func (os *ConcreteOutputStrategy) SelectStrategy(ctx context.Context, method *domain.MethodResult) ConstructionStrategy {
	if method == nil {
		return StrategyAssignmentBlock // Safe default
	}

	os.logger.Debug("selecting output strategy",
		zap.String("method", method.Method.Name))

	// Extract field results for analysis
	fields := os.extractFieldResults(method)

	// Analyze complexity
	complexity := os.AnalyzeFieldComplexity(fields)

	// Get performance estimates
	performance := os.EstimatePerformance(method)

	// Apply decision logic
	strategy := os.selectOptimalStrategy(complexity, performance, len(fields))

	os.logger.Debug("strategy selected",
		zap.String("method", method.Method.Name),
		zap.String("strategy", strategy.String()),
		zap.Float64("complexity_score", complexity.ComplexityScore),
		zap.Int("field_count", len(fields)))

	return strategy
}

// AnalyzeFieldComplexity analyzes the complexity of fields for strategy selection.
func (os *ConcreteOutputStrategy) AnalyzeFieldComplexity(fields []*executor.FieldResult) *ComplexityMetrics {
	metrics := NewComplexityMetrics()

	if len(fields) == 0 {
		return metrics
	}

	metrics.FieldCount = len(fields)

	var totalComplexity float64

	var cyclomaticComplexity int

	for _, field := range fields {
		fieldComplexity := os.calculateFieldComplexity(field)
		totalComplexity += fieldComplexity

		// Count error fields
		if field.Error != nil || !field.Success {
			metrics.ErrorFields++
			cyclomaticComplexity += 2 // Error handling adds cyclomatic complexity
		}

		// Count converter fields (complex transformations)
		if os.isConverterField(field) {
			metrics.ConverterFields++
			cyclomaticComplexity++
		}

		// Count nested fields (complex types)
		if os.isNestedField(field) {
			metrics.NestedFields++
			cyclomaticComplexity++
		}
	}

	// Calculate average complexity
	metrics.ComplexityScore = totalComplexity / float64(len(fields))
	metrics.CyclomaticComplexity = cyclomaticComplexity

	// Determine recommended strategy based on complexity
	metrics.RecommendedStrategy = os.determineRecommendedStrategy(metrics)

	return metrics
}

// ShouldUseCompositeLiteral determines if composite literal approach is optimal.
func (os *ConcreteOutputStrategy) ShouldUseCompositeLiteral(method *domain.MethodResult) bool {
	if method == nil {
		return false
	}

	fields := os.extractFieldResults(method)

	// Basic criteria for composite literal usage
	fieldCount := len(fields)
	if fieldCount > os.config.MaxFieldsForComposite {
		return false
	}

	// Check for complex scenarios that preclude composite literals
	complexity := os.AnalyzeFieldComplexity(fields)

	return complexity.ErrorFields == 0 &&
		complexity.ComplexityScore < 30.0 &&
		complexity.CyclomaticComplexity <= 2
}

// EstimatePerformance estimates the performance characteristics of different strategies.
func (os *ConcreteOutputStrategy) EstimatePerformance(method *domain.MethodResult) *PerformanceEstimate {
	fields := os.extractFieldResults(method)
	fieldCount := len(fields)
	complexity := os.AnalyzeFieldComplexity(fields)

	estimate := &PerformanceEstimate{
		CompositeLiteral: os.estimateCompositeLiteral(fieldCount, complexity),
		AssignmentBlock:  os.estimateAssignmentBlock(fieldCount, complexity),
		MixedApproach:    os.estimateMixedApproach(fieldCount, complexity),
	}

	// Determine recommended strategy based on estimates
	estimate.Recommended = os.selectBestPerformingStrategy(estimate)

	return estimate
}

// Helper methods for strategy selection

func (os *ConcreteOutputStrategy) selectOptimalStrategy(complexity *ComplexityMetrics, performance *PerformanceEstimate, fieldCount int) ConstructionStrategy {
	// Weight-based scoring system
	scores := make(map[ConstructionStrategy]float64)

	// Composite literal scoring
	if os.canUseCompositeLiteral(complexity, fieldCount) {
		scores[StrategyCompositeLiteral] = os.calculateStrategyScore(
			performance.CompositeLiteral, complexity, os.getCompositeLiteralWeights())
	}

	// Assignment block scoring
	scores[StrategyAssignmentBlock] = os.calculateStrategyScore(
		performance.AssignmentBlock, complexity, os.getAssignmentBlockWeights())

	// Mixed approach scoring
	if fieldCount > 3 {
		scores[StrategyMixedApproach] = os.calculateStrategyScore(
			performance.MixedApproach, complexity, os.getMixedApproachWeights())
	}

	// Select strategy with highest score
	bestStrategy := StrategyAssignmentBlock // Safe default
	bestScore := 0.0

	for strategy, score := range scores {
		if score > bestScore {
			bestScore = score
			bestStrategy = strategy
		}
	}

	return bestStrategy
}

func (os *ConcreteOutputStrategy) calculateFieldComplexity(field *executor.FieldResult) float64 {
	complexity := 1.0 // Base complexity

	// Add complexity for errors
	if field.Error != nil || !field.Success {
		complexity += 3.0
	}

	// Add complexity for retry attempts
	complexity += float64(field.RetryCount) * 0.5

	// Add complexity based on execution time (slower = more complex)
	if field.Duration.Milliseconds() > 10 {
		complexity += math.Log10(float64(field.Duration.Milliseconds())) * 0.5
	}

	// Add complexity for certain strategies
	switch field.StrategyUsed {
	case "converter", "expression":
		complexity += 2.0
	case "custom":
		complexity += 1.5
	}

	return complexity
}

func (os *ConcreteOutputStrategy) isConverterField(field *executor.FieldResult) bool {
	return field.StrategyUsed == "converter" || field.StrategyUsed == "expression"
}

func (os *ConcreteOutputStrategy) isNestedField(field *executor.FieldResult) bool {
	// Simplified check - in practice would analyze field types
	return field.StrategyUsed == "custom" || field.RetryCount > 0
}

func (os *ConcreteOutputStrategy) determineRecommendedStrategy(metrics *ComplexityMetrics) ConstructionStrategy {
	// Simple heuristic-based recommendation
	if metrics.FieldCount <= os.config.MaxFieldsForComposite &&
		metrics.ErrorFields == 0 &&
		metrics.ComplexityScore < 20.0 {
		return StrategyCompositeLiteral
	}

	if metrics.FieldCount > 10 &&
		metrics.ErrorFields > 0 &&
		metrics.ConverterFields > 2 {
		return StrategyMixedApproach
	}

	return StrategyAssignmentBlock
}

func (os *ConcreteOutputStrategy) estimateCompositeLiteral(fieldCount int, complexity *ComplexityMetrics) *StrategyEstimate {
	return &StrategyEstimate{
		GenerationTime:   float64(fieldCount) * 0.5,                  // Fast generation
		ExecutionTime:    float64(fieldCount) * 100,                  // Fast execution
		MemoryUsage:      fieldCount * 32,                            // Minimal memory
		ReadabilityScore: math.Max(0, 90-complexity.ComplexityScore), // High readability for simple cases
		Complexity:       complexity.ComplexityScore * 0.7,           // Lower perceived complexity
		LinesOfCode:      fieldCount + 3,                             // Compact code
	}
}

func (os *ConcreteOutputStrategy) estimateAssignmentBlock(fieldCount int, complexity *ComplexityMetrics) *StrategyEstimate {
	return &StrategyEstimate{
		GenerationTime:   float64(fieldCount) * 1.2,                      // Moderate generation time
		ExecutionTime:    float64(fieldCount) * 150,                      // Moderate execution time
		MemoryUsage:      fieldCount * 48,                                // Moderate memory usage
		ReadabilityScore: math.Max(0, 80-complexity.ComplexityScore*0.5), // Good readability
		Complexity:       complexity.ComplexityScore,                     // Direct complexity mapping
		LinesOfCode:      fieldCount*2 + 5,                               // More verbose
	}
}

func (os *ConcreteOutputStrategy) estimateMixedApproach(fieldCount int, complexity *ComplexityMetrics) *StrategyEstimate {
	// Mixed approach balances benefits of both strategies
	compositePart := float64(fieldCount) * 0.4  // 40% of fields use composite
	assignmentPart := float64(fieldCount) * 0.6 // 60% use assignments

	return &StrategyEstimate{
		GenerationTime:   compositePart*0.5 + assignmentPart*1.2,
		ExecutionTime:    compositePart*100 + assignmentPart*150,
		MemoryUsage:      int(compositePart*32 + assignmentPart*48),
		ReadabilityScore: math.Max(0, 85-complexity.ComplexityScore*0.3),
		Complexity:       complexity.ComplexityScore * 0.8,
		LinesOfCode:      int(compositePart*1.5+assignmentPart*2) + 4,
	}
}

func (os *ConcreteOutputStrategy) selectBestPerformingStrategy(estimate *PerformanceEstimate) ConstructionStrategy {
	strategies := []struct {
		strategy ConstructionStrategy
		estimate *StrategyEstimate
	}{
		{StrategyCompositeLiteral, estimate.CompositeLiteral},
		{StrategyAssignmentBlock, estimate.AssignmentBlock},
		{StrategyMixedApproach, estimate.MixedApproach},
	}

	bestStrategy := StrategyAssignmentBlock
	bestScore := 0.0

	for _, s := range strategies {
		if s.estimate == nil {
			continue
		}

		// Calculate composite score based on multiple factors
		score := os.calculateCompositeScore(s.estimate)
		if score > bestScore {
			bestScore = score
			bestStrategy = s.strategy
		}
	}

	return bestStrategy
}

func (os *ConcreteOutputStrategy) calculateCompositeScore(estimate *StrategyEstimate) float64 {
	// Normalize metrics to 0-1 scale and apply weights
	normalizedGenerationTime := math.Max(0, 1.0-estimate.GenerationTime/100.0)
	normalizedExecutionTime := math.Max(0, 1.0-estimate.ExecutionTime/10000.0)
	normalizedMemory := math.Max(0, 1.0-float64(estimate.MemoryUsage)/10000.0)
	normalizedReadability := estimate.ReadabilityScore / 100.0
	normalizedComplexity := math.Max(0, 1.0-estimate.Complexity/100.0)

	score := normalizedGenerationTime*0.15 +
		normalizedExecutionTime*0.25 +
		normalizedMemory*0.15 +
		normalizedReadability*0.25 +
		normalizedComplexity*0.20

	return score
}

func (os *ConcreteOutputStrategy) canUseCompositeLiteral(complexity *ComplexityMetrics, fieldCount int) bool {
	return fieldCount <= os.config.MaxFieldsForComposite &&
		complexity.ErrorFields == 0 &&
		complexity.ComplexityScore < 30.0
}

func (os *ConcreteOutputStrategy) calculateStrategyScore(estimate *StrategyEstimate, _ *ComplexityMetrics, weights map[string]float64) float64 {
	if estimate == nil {
		return 0.0
	}

	score := 0.0
	score += (1.0 - estimate.GenerationTime/100.0) * weights["generation_time"]
	score += (1.0 - estimate.ExecutionTime/10000.0) * weights["execution_time"]
	score += (1.0 - float64(estimate.MemoryUsage)/10000.0) * weights["memory"]
	score += (estimate.ReadabilityScore / 100.0) * weights["readability"]
	score += (1.0 - estimate.Complexity/100.0) * weights["complexity"]

	return math.Max(0, math.Min(1, score))
}

func (os *ConcreteOutputStrategy) getCompositeLiteralWeights() map[string]float64 {
	return map[string]float64{
		"generation_time": 0.2,
		"execution_time":  0.3,
		"memory":          0.15,
		"readability":     0.25,
		"complexity":      0.1,
	}
}

func (os *ConcreteOutputStrategy) getAssignmentBlockWeights() map[string]float64 {
	return map[string]float64{
		"generation_time": 0.15,
		"execution_time":  0.2,
		"memory":          0.15,
		"readability":     0.3,
		"complexity":      0.2,
	}
}

func (os *ConcreteOutputStrategy) getMixedApproachWeights() map[string]float64 {
	return map[string]float64{
		"generation_time": 0.2,
		"execution_time":  0.25,
		"memory":          0.15,
		"readability":     0.25,
		"complexity":      0.15,
	}
}

func (os *ConcreteOutputStrategy) extractFieldResults(method *domain.MethodResult) []*executor.FieldResult {
	var results []*executor.FieldResult

	for _, fieldResult := range method.Metadata {
		if fr, ok := fieldResult.(*executor.FieldResult); ok {
			results = append(results, fr)
		}
	}

	return results
}
