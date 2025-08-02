package emitter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/reedom/convergen/v8/pkg/domain"
	"github.com/reedom/convergen/v8/pkg/executor"
	"github.com/reedom/convergen/v8/pkg/internal/events"
)

// Static errors for err113 compliance.
var (
	ErrMethodResultNil      = errors.New("method result cannot be nil")
	ErrNoGenerationStrategy = errors.New("no generation strategy available")
	ErrFieldResultNil       = errors.New("field result cannot be nil")
)

// EventBus interface for event publishing (simplified interface).
type EventBus interface {
	Publish(ctx context.Context, event events.Event) error
}

// CodeGenerator handles the core code generation logic.
type CodeGenerator interface {
	// GenerateMethodCode generates complete method implementation
	GenerateMethodCode(ctx context.Context, method *domain.MethodResult) (*MethodCode, error)

	// GenerateFieldCode generates code for individual field assignments
	GenerateFieldCode(ctx context.Context, field *executor.FieldResult) (*FieldCode, error)

	// GenerateErrorHandling generates error handling code
	GenerateErrorHandling(ctx context.Context, errors []domain.ExecutionError) (*ErrorCode, error)

	// GetMetrics returns code generation metrics
	GetMetrics() *CodeGenMetrics

	// Shutdown gracefully shuts down the generator
	Shutdown(ctx context.Context) error
}

// ConcreteCodeGenerator implements CodeGenerator.
type ConcreteCodeGenerator struct {
	config      *Config
	logger      *zap.Logger
	strategies  map[string]GenerationStrategy
	validator   CodeValidator
	metrics     *CodeGenMetrics
	outputStrat OutputStrategy
	eventBus    EventBus // For event publishing
}

// CodeGenMetrics tracks code generation performance.
type CodeGenMetrics struct {
	mu                     sync.RWMutex
	MethodsGenerated       int64            `json:"methods_generated"`
	FieldsGenerated        int64            `json:"fields_generated"`
	ErrorHandlersGenerated int64            `json:"error_handlers_generated"`
	TotalGenerationTime    time.Duration    `json:"total_generation_time"`
	AverageMethodTime      time.Duration    `json:"average_method_time"`
	StrategyUsage          map[string]int64 `json:"strategy_usage"`
	TemplateUsage          map[string]int64 `json:"template_usage"`
	ValidationTime         time.Duration    `json:"validation_time"`
	ErrorsEncountered      int64            `json:"errors_encountered"`
}

// GenerationStrategy defines how different types of code should be generated.
type GenerationStrategy interface {
	// Name returns the strategy name
	Name() string

	// CanHandle determines if this strategy can handle the given method
	CanHandle(method *domain.MethodResult) bool

	// GenerateCode generates code for the method
	GenerateCode(ctx context.Context, method *domain.MethodResult, data *TemplateData) (string, error)

	// GetComplexity estimates the complexity of the generated code
	GetComplexity(method *domain.MethodResult) *ComplexityMetrics

	// GetRequiredImports returns imports needed for this strategy
	GetRequiredImports(method *domain.MethodResult) []*Import
}

// NewCodeGenerator creates a new code generator.
func NewCodeGenerator(config *Config, logger *zap.Logger, metrics *Metrics) CodeGenerator {
	generator := &ConcreteCodeGenerator{
		config:      config,
		logger:      logger,
		strategies:  make(map[string]GenerationStrategy),
		metrics:     NewCodeGenMetrics(),
		outputStrat: NewOutputStrategy(config, logger),
	}

	// Register default generation strategies
	generator.registerDefaultStrategies()

	return generator
}

// GenerateMethodCode generates complete method implementation.
func (cg *ConcreteCodeGenerator) GenerateMethodCode(ctx context.Context, method *domain.MethodResult) (*MethodCode, error) {
	if method == nil {
		return nil, ErrMethodResultNil
	}

	startTime := time.Now()

	cg.logger.Debug("generating method code",
		zap.String("method", method.Method.Name),
		zap.Int("metadata_fields", len(method.Metadata)))

	// Analyze method complexity and select strategy
	strategy := cg.outputStrat.SelectStrategy(ctx, method)
	complexity := cg.outputStrat.AnalyzeFieldComplexity(cg.extractFieldResults(method))

	// Prepare template data
	templateData := &TemplateData{
		Method: method,
		Fields: cg.extractFieldResults(method),
		Config: cg.config,
		Metadata: map[string]interface{}{
			"strategy":   strategy.String(),
			"complexity": complexity,
		},
		HelperFunctions: cg.getHelperFunctions(),
	}

	// Generate method signature
	signature := cg.generateMethodSignature(method)

	// Generate method body based on strategy
	var body string

	var err error

	var generationStrategy GenerationStrategy

	switch strategy {
	case StrategyCompositeLiteral:
		generationStrategy = cg.strategies["composite_literal"]
	case StrategyAssignmentBlock:
		generationStrategy = cg.strategies["assignment_block"]
	case StrategyMixedApproach:
		generationStrategy = cg.strategies["mixed_approach"]
	default:
		generationStrategy = cg.strategies["assignment_block"] // fallback
	}

	if generationStrategy == nil {
		return nil, fmt.Errorf("%w for %s", ErrNoGenerationStrategy, strategy.String())
	}

	body, err = generationStrategy.GenerateCode(ctx, method, templateData)
	if err != nil {
		return nil, fmt.Errorf("code generation failed: %w", err)
	}

	// Generate error handling if needed
	var errorHandling string

	if cg.hasErrorHandling(method) {
		errorCode, err := cg.GenerateErrorHandling(ctx, cg.extractErrors(method))
		if err != nil {
			cg.logger.Warn("error handling generation failed", zap.Error(err))
		} else {
			errorHandling = errorCode.HandlingCode
		}
	}

	// Generate documentation
	documentation := cg.generateDocumentation(method)

	// Collect required imports
	imports := generationStrategy.GetRequiredImports(method)

	// Generate field codes for detailed analysis
	fields := make([]*FieldCode, 0, len(method.Metadata))

	for fieldName, fieldResult := range method.Metadata {
		if fr, ok := fieldResult.(*executor.FieldResult); ok {
			fieldCode, err := cg.GenerateFieldCode(ctx, fr)
			if err != nil {
				cg.logger.Warn("field code generation failed",
					zap.String("field", fieldName),
					zap.Error(err))

				continue
			}

			fields = append(fields, fieldCode)
		}
	}

	methodCode := &MethodCode{
		Name:          method.Method.Name,
		Signature:     signature,
		Body:          body,
		ErrorHandling: errorHandling,
		Documentation: documentation,
		Imports:       imports,
		Complexity:    complexity,
		Strategy:      strategy,
		Fields:        fields,
	}

	// Update metrics
	duration := time.Since(startTime)

	cg.metrics.IncrementMethods()
	cg.metrics.AddGenerationTime(duration)
	cg.metrics.IncrementStrategy(strategy.String())

	// Validate generated code if enabled
	if cg.config.EnableSyntaxValidation && cg.validator != nil {
		if err := cg.validator.ValidateMethodCode(methodCode); err != nil {
			cg.logger.Warn("method code validation failed",
				zap.String("method", method.Method.Name),
				zap.Error(err))
			cg.metrics.IncrementErrors()
		}
	}

	cg.logger.Debug("method code generated",
		zap.String("method", method.Method.Name),
		zap.Duration("duration", duration),
		zap.String("strategy", strategy.String()),
		zap.Int("lines", strings.Count(body, "\n")+1))

	return methodCode, nil
}

// GenerateFieldCode generates code for individual field assignments.
func (cg *ConcreteCodeGenerator) GenerateFieldCode(ctx context.Context, field *executor.FieldResult) (*FieldCode, error) {
	if field == nil {
		return nil, ErrFieldResultNil
	}

	cg.logger.Debug("generating field code",
		zap.String("field", field.FieldID))

	// Determine field assignment strategy based on field characteristics
	var assignment, declaration, errorCheck string

	var imports []*Import

	var dependencies []string

	// Simple field assignment (this would be more sophisticated in practice)
	if field.Error == nil {
		assignment = cg.generateSimpleAssignment(field)
	} else {
		// Complex assignment with error handling
		assignment, errorCheck = cg.generateComplexAssignment(field)
	}

	// Analyze dependencies (simplified)
	dependencies = cg.analyzeDependencies(field)

	// Determine required imports (simplified)
	imports = cg.analyzeFieldImports(field)

	// Determine strategy from FieldResult or default
	strategy := "default"
	if field.StrategyUsed != "" {
		strategy = field.StrategyUsed
	}

	fieldCode := &FieldCode{
		Name:         field.FieldID,
		Assignment:   assignment,
		Declaration:  declaration,
		ErrorCheck:   errorCheck,
		Imports:      imports,
		Dependencies: dependencies,
		Order:        0,        // This would be determined from source order
		Strategy:     strategy, // Use strategy from FieldResult
	}

	cg.metrics.IncrementFields()

	return fieldCode, nil
}

// GenerateErrorHandling generates error handling code.
func (cg *ConcreteCodeGenerator) GenerateErrorHandling(ctx context.Context, errors []domain.ExecutionError) (*ErrorCode, error) {
	if len(errors) == 0 {
		return &ErrorCode{}, nil
	}

	cg.logger.Debug("generating error handling code",
		zap.Int("errors", len(errors)))

	var checkCode, handlingCode, returnCode, wrapperCode strings.Builder

	var imports []*Import

	// Generate error checking patterns
	for _, err := range errors {
		checkCode.WriteString(cg.generateErrorCheck(err))
		checkCode.WriteString("\n")
	}

	// Generate error handling logic
	handlingCode.WriteString("if err != nil {\n")
	handlingCode.WriteString("\treturn nil, fmt.Errorf(\"field processing failed: %w\", err)\n")
	handlingCode.WriteString("}\n")

	// Generate return pattern
	returnCode.WriteString("return result, nil")

	// Add fmt import for error wrapping
	imports = append(imports, &Import{
		Path:     "fmt",
		Used:     true,
		Standard: true,
		Required: true,
	})

	errorCode := &ErrorCode{
		CheckCode:    checkCode.String(),
		HandlingCode: handlingCode.String(),
		ReturnCode:   returnCode.String(),
		WrapperCode:  wrapperCode.String(),
		Imports:      imports,
	}

	cg.metrics.IncrementErrorHandlers()

	return errorCode, nil
}

// GetMetrics returns code generation metrics.
func (cg *ConcreteCodeGenerator) GetMetrics() *CodeGenMetrics {
	return cg.metrics
}

// Shutdown gracefully shuts down the generator.
func (cg *ConcreteCodeGenerator) Shutdown(ctx context.Context) error {
	cg.logger.Info("shutting down code generator")
	return nil
}

// Helper methods

func (cg *ConcreteCodeGenerator) registerDefaultStrategies() {
	cg.strategies["composite_literal"] = NewCompositeLiteralStrategy(cg.config, cg.logger)
	cg.strategies["assignment_block"] = NewAssignmentBlockStrategy(cg.config, cg.logger)
	cg.strategies["mixed_approach"] = NewMixedApproachStrategy(cg.config, cg.logger)
}

func (cg *ConcreteCodeGenerator) generateMethodSignature(method *domain.MethodResult) string {
	// Simplified method signature generation
	return fmt.Sprintf("func %s(src *SourceType) (*DestType, error)", method.Method.Name)
}

func (cg *ConcreteCodeGenerator) hasErrorHandling(method *domain.MethodResult) bool {
	for _, fieldResult := range method.Metadata {
		if fr, ok := fieldResult.(*executor.FieldResult); ok {
			if fr.Error != nil {
				return true
			}
		}
	}

	return false
}

func (cg *ConcreteCodeGenerator) extractErrors(method *domain.MethodResult) []domain.ExecutionError {
	var errors []domain.ExecutionError

	for _, fieldResult := range method.Metadata {
		if fr, ok := fieldResult.(*executor.FieldResult); ok {
			if fr.Error != nil {
				// Convert executor.ExecutionError to domain.ExecutionError
				execError := domain.ExecutionError{
					Type:      fr.Error.ErrorType,
					Message:   fr.Error.Error,
					Component: "emitter",
					Method:    method.Method.Name,
					Field:     fr.Error.FieldID,
				}
				errors = append(errors, execError)
			}
		}
	}

	return errors
}

func (cg *ConcreteCodeGenerator) extractFieldResults(method *domain.MethodResult) []*executor.FieldResult {
	var results []*executor.FieldResult

	for _, fieldResult := range method.Metadata {
		if fr, ok := fieldResult.(*executor.FieldResult); ok {
			results = append(results, fr)
		}
	}

	return results
}

func (cg *ConcreteCodeGenerator) generateDocumentation(method *domain.MethodResult) string {
	return fmt.Sprintf("// %s converts SourceType to DestType\n", method.Method.Name)
}

func (cg *ConcreteCodeGenerator) generateSimpleAssignment(field *executor.FieldResult) string {
	return fmt.Sprintf("dest.%s = src.%s", field.FieldID, field.FieldID)
}

func (cg *ConcreteCodeGenerator) generateComplexAssignment(field *executor.FieldResult) (string, string) {
	assignment := fmt.Sprintf("converted, err := converter.Convert(src.%s)", field.FieldID)
	errorCheck := "if err != nil { return nil, fmt.Errorf(\"converting %s: %%w\", err) }"

	return assignment, fmt.Sprintf(errorCheck, field.FieldID)
}

func (cg *ConcreteCodeGenerator) analyzeDependencies(field *executor.FieldResult) []string {
	// Simplified dependency analysis
	return []string{}
}

func (cg *ConcreteCodeGenerator) analyzeFieldImports(field *executor.FieldResult) []*Import {
	// Simplified import analysis
	var imports []*Import

	// Add fmt import if error handling is present
	if field.Error != nil {
		imports = append(imports, &Import{
			Path:     "fmt",
			Used:     true,
			Standard: true,
			Required: true,
		})
	}

	return imports
}

func (cg *ConcreteCodeGenerator) generateErrorCheck(err domain.ExecutionError) string {
	return fmt.Sprintf("// Error check for %s: %s", err.Field, err.Message)
}

func (cg *ConcreteCodeGenerator) getHelperFunctions() map[string]interface{} {
	return map[string]interface{}{
		"capitalize": cases.Title(language.English).String,
		"camelCase":  cg.toCamelCase,
		"snakeCase":  cg.toSnakeCase,
		"indent":     cg.indent,
	}
}

// emitEvent function removed - was unused

// SetEventBus sets the event bus for this code generator.
func (cg *ConcreteCodeGenerator) SetEventBus(eventBus EventBus) {
	cg.eventBus = eventBus
}

func (cg *ConcreteCodeGenerator) toCamelCase(s string) string {
	// Simple camel case conversion
	words := strings.Split(s, "_")
	for i := 1; i < len(words); i++ {
		words[i] = cases.Title(language.English).String(words[i])
	}

	return strings.Join(words, "")
}

func (cg *ConcreteCodeGenerator) toSnakeCase(s string) string {
	// Simple snake case conversion
	return strings.ToLower(strings.ReplaceAll(s, " ", "_"))
}

func (cg *ConcreteCodeGenerator) indent(s string, level int) string {
	indent := strings.Repeat(cg.config.IndentStyle, level)

	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = indent + line
		}
	}

	return strings.Join(lines, "\n")
}

// NewCodeGenMetrics creates a new CodeGenMetrics instance.
func NewCodeGenMetrics() *CodeGenMetrics {
	return &CodeGenMetrics{
		StrategyUsage: make(map[string]int64),
		TemplateUsage: make(map[string]int64),
	}
}

// IncrementMethods safely increments the methods generated counter.
func (m *CodeGenMetrics) IncrementMethods() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.MethodsGenerated++
}

// AddGenerationTime safely adds generation time and updates average.
func (m *CodeGenMetrics) AddGenerationTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalGenerationTime += duration
	if 0 < m.MethodsGenerated {
		m.AverageMethodTime = m.TotalGenerationTime / time.Duration(m.MethodsGenerated)
	}
}

// IncrementStrategy safely increments strategy usage counter.
func (m *CodeGenMetrics) IncrementStrategy(strategy string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StrategyUsage[strategy]++
}

// IncrementFields safely increments the fields generated counter.
func (m *CodeGenMetrics) IncrementFields() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.FieldsGenerated++
}

// IncrementErrors safely increments the errors encountered counter.
func (m *CodeGenMetrics) IncrementErrors() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ErrorsEncountered++
}

// IncrementErrorHandlers safely increments the error handlers generated counter.
func (m *CodeGenMetrics) IncrementErrorHandlers() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ErrorHandlersGenerated++
}

// GetSnapshot returns a thread-safe copy of current metrics.
func (m *CodeGenMetrics) GetSnapshot() *CodeGenMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create deep copy of maps
	strategyUsage := make(map[string]int64)
	for k, v := range m.StrategyUsage {
		strategyUsage[k] = v
	}

	templateUsage := make(map[string]int64)
	for k, v := range m.TemplateUsage {
		templateUsage[k] = v
	}

	return &CodeGenMetrics{
		MethodsGenerated:       m.MethodsGenerated,
		FieldsGenerated:        m.FieldsGenerated,
		ErrorHandlersGenerated: m.ErrorHandlersGenerated,
		TotalGenerationTime:    m.TotalGenerationTime,
		AverageMethodTime:      m.AverageMethodTime,
		StrategyUsage:          strategyUsage,
		TemplateUsage:          templateUsage,
		ValidationTime:         m.ValidationTime,
		ErrorsEncountered:      m.ErrorsEncountered,
	}
}
