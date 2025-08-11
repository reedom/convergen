package emitter

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v9/pkg/domain"
	"github.com/reedom/convergen/v9/pkg/executor"
)

// CompositeLiteralStrategy generates code using composite literal initialization.
type CompositeLiteralStrategy struct {
	config *Config
	logger *zap.Logger
}

// AssignmentBlockStrategy generates code using assignment blocks.
type AssignmentBlockStrategy struct {
	config *Config
	logger *zap.Logger
}

// MixedApproachStrategy combines both composite literals and assignments.
type MixedApproachStrategy struct {
	config *Config
	logger *zap.Logger
}

// NewCompositeLiteralStrategy creates a new composite literal strategy.
func NewCompositeLiteralStrategy(config *Config, logger *zap.Logger) GenerationStrategy {
	return &CompositeLiteralStrategy{
		config: config,
		logger: logger,
	}
}

// NewAssignmentBlockStrategy creates a new assignment block strategy.
func NewAssignmentBlockStrategy(config *Config, logger *zap.Logger) GenerationStrategy {
	return &AssignmentBlockStrategy{
		config: config,
		logger: logger,
	}
}

// NewMixedApproachStrategy creates a new mixed approach strategy.
func NewMixedApproachStrategy(config *Config, logger *zap.Logger) GenerationStrategy {
	return &MixedApproachStrategy{
		config: config,
		logger: logger,
	}
}

// CompositeLiteralStrategy implementation

// Name returns the name of the strategy.
func (cls *CompositeLiteralStrategy) Name() string {
	return "composite_literal"
}

// CanHandle determines if this strategy can handle the given method.
func (cls *CompositeLiteralStrategy) CanHandle(method *domain.MethodResult) bool {
	if method == nil {
		return false
	}

	fieldCount := len(method.Metadata)
	if cls.config.MaxFieldsForComposite < fieldCount {
		return false
	}

	// Check for error handling requirements
	for _, fieldResult := range method.Metadata {
		if fr, ok := fieldResult.(*executor.FieldResult); ok {
			if !fr.Success || fr.Error != nil {
				return false // Cannot use composite literals with error handling
			}
		}
	}

	return true
}

// GenerateCode generates code for the given method using this strategy.
func (cls *CompositeLiteralStrategy) GenerateCode(ctx context.Context, method *domain.MethodResult, data *TemplateData) (string, error) {
	cls.logger.Debug("generating composite literal code",
		zap.String("method", method.Method.Name))

	var code strings.Builder

	// Generate method body with composite literal
	code.WriteString(fmt.Sprintf("%sreturn &DestType{\n", cls.config.IndentStyle))

	// Generate field assignments
	fieldCount := 0

	for fieldName, fieldResult := range method.Metadata {
		if fr, ok := fieldResult.(*executor.FieldResult); ok {
			if fr.Success && fr.Error == nil {
				assignment := cls.generateFieldAssignment(fieldName, fr)
				code.WriteString(fmt.Sprintf("%s%s%s,\n",
					cls.config.IndentStyle, cls.config.IndentStyle, assignment))

				fieldCount++
			}
		}
	}

	code.WriteString(fmt.Sprintf("%s}, nil\n", cls.config.IndentStyle))

	cls.logger.Debug("composite literal code generated",
		zap.String("method", method.Method.Name),
		zap.Int("fields", fieldCount))

	return code.String(), nil
}

// GetComplexity returns the complexity metrics for this strategy.
func (cls *CompositeLiteralStrategy) GetComplexity(method *domain.MethodResult) *ComplexityMetrics {
	metrics := NewComplexityMetrics()
	metrics.FieldCount = len(method.Metadata)
	metrics.ComplexityScore = float64(metrics.FieldCount) * 2.0 // Low complexity
	metrics.CyclomaticComplexity = 1                            // Simple linear flow
	metrics.LinesGenerated = metrics.FieldCount + 2             // Fields + return statement
	metrics.RecommendedStrategy = StrategyCompositeLiteral

	return metrics
}

// GetRequiredImports returns the imports required by this strategy.
func (cls *CompositeLiteralStrategy) GetRequiredImports(method *domain.MethodResult) []*Import {
	// Composite literals typically don't require additional imports
	// beyond what's already available
	return []*Import{}
}

func (cls *CompositeLiteralStrategy) generateFieldAssignment(fieldName string, field *executor.FieldResult) string {
	// Generate simple field assignment for composite literal
	switch field.StrategyUsed {
	case domain.DirectStrategyType:
		return fmt.Sprintf("%s: src.%s", fieldName, fieldName)
	case domain.ConverterStrategyType:
		return fmt.Sprintf("%s: converter.Convert(src.%s)", fieldName, fieldName)
	case domain.LiteralStrategyType:
		return fmt.Sprintf("%s: %v", fieldName, field.Result)
	default:
		return fmt.Sprintf("%s: src.%s", fieldName, fieldName)
	}
}

// AssignmentBlockStrategy implementation

// Name returns the name of the strategy.
func (abs *AssignmentBlockStrategy) Name() string {
	return "assignment_block"
}

// CanHandle determines if this strategy can handle the given method.
func (abs *AssignmentBlockStrategy) CanHandle(method *domain.MethodResult) bool {
	// Assignment block strategy can handle any method
	return method != nil
}

// GenerateCode generates code for the given method using this strategy.
func (abs *AssignmentBlockStrategy) GenerateCode(ctx context.Context, method *domain.MethodResult, data *TemplateData) (string, error) {
	abs.logger.Debug("generating assignment block code",
		zap.String("method", method.Method.Name))

	var code strings.Builder

	// Generate variable declaration
	code.WriteString(fmt.Sprintf("%svar dest DestType\n\n", abs.config.IndentStyle))

	// Generate field assignments with error handling
	fieldCount := 0

	for fieldName, fieldResult := range method.Metadata {
		if fr, ok := fieldResult.(*executor.FieldResult); ok {
			assignment, errorHandling := abs.generateFieldAssignment(fieldName, fr)

			code.WriteString(fmt.Sprintf("%s%s\n", abs.config.IndentStyle, assignment))

			if errorHandling != "" {
				code.WriteString(fmt.Sprintf("%s%s\n", abs.config.IndentStyle, errorHandling))
			}

			code.WriteString("\n")

			fieldCount++
		}
	}

	// Generate return statement
	code.WriteString(fmt.Sprintf("%sreturn &dest, nil\n", abs.config.IndentStyle))

	abs.logger.Debug("assignment block code generated",
		zap.String("method", method.Method.Name),
		zap.Int("fields", fieldCount))

	return code.String(), nil
}

// GetComplexity returns the complexity metrics for this strategy.
func (abs *AssignmentBlockStrategy) GetComplexity(method *domain.MethodResult) *ComplexityMetrics {
	metrics := NewComplexityMetrics()
	metrics.FieldCount = len(method.Metadata)

	// Calculate complexity based on error handling and field types
	errorFields := 0

	for _, fieldResult := range method.Metadata {
		if fr, ok := fieldResult.(*executor.FieldResult); ok {
			if fr.Error != nil || !fr.Success {
				errorFields++
			}
		}
	}

	metrics.ErrorFields = errorFields
	metrics.ComplexityScore = float64(metrics.FieldCount)*3.0 + float64(errorFields)*5.0
	metrics.CyclomaticComplexity = 1 + errorFields*2                  // Base + error handling paths
	metrics.LinesGenerated = metrics.FieldCount*2 + errorFields*3 + 3 // Assignments + error handling + structure
	metrics.RecommendedStrategy = StrategyAssignmentBlock

	return metrics
}

// GetRequiredImports returns the imports required by this strategy.
func (abs *AssignmentBlockStrategy) GetRequiredImports(method *domain.MethodResult) []*Import {
	var imports []*Import

	// Check if error handling is needed
	hasErrors := false

	for _, fieldResult := range method.Metadata {
		if fr, ok := fieldResult.(*executor.FieldResult); ok {
			if fr.Error != nil || !fr.Success {
				hasErrors = true
				break
			}
		}
	}

	if hasErrors {
		imports = append(imports, &Import{
			Path:     "fmt",
			Used:     true,
			Standard: true,
			Required: true,
		})
	}

	return imports
}

func (abs *AssignmentBlockStrategy) generateFieldAssignment(fieldName string, field *executor.FieldResult) (string, string) {
	var assignment, errorHandling string

	switch field.StrategyUsed {
	case domain.DirectStrategyType:
		assignment = fmt.Sprintf("dest.%s = src.%s", fieldName, fieldName)

	case domain.ConverterStrategyType:
		if field.Error != nil || !field.Success {
			assignment = fmt.Sprintf("converted_%s, err := converter.Convert(src.%s)", fieldName, fieldName)
			errorHandling = fmt.Sprintf("if err != nil {\n%s%sreturn nil, fmt.Errorf(\"converting %s: %%w\", err)\n%s}",
				abs.config.IndentStyle, abs.config.IndentStyle, fieldName, abs.config.IndentStyle)
			assignment += fmt.Sprintf("\ndest.%s = converted_%s", fieldName, fieldName)
		} else {
			assignment = fmt.Sprintf("dest.%s = converter.Convert(src.%s)", fieldName, fieldName)
		}

	case domain.LiteralStrategyType:
		assignment = fmt.Sprintf("dest.%s = %v", fieldName, field.Result)

	case domain.ExpressionStrategyType:
		if field.Error != nil || !field.Success {
			assignment = fmt.Sprintf("result_%s, err := expression.Evaluate(src.%s)", fieldName, fieldName)
			errorHandling = fmt.Sprintf("if err != nil {\n%s%sreturn nil, fmt.Errorf(\"evaluating expression for %s: %%w\", err)\n%s}",
				abs.config.IndentStyle, abs.config.IndentStyle, fieldName, abs.config.IndentStyle)
			assignment += fmt.Sprintf("\ndest.%s = result_%s", fieldName, fieldName)
		} else {
			assignment = fmt.Sprintf("dest.%s = expression.Evaluate(src.%s)", fieldName, fieldName)
		}

	default:
		assignment = fmt.Sprintf("dest.%s = src.%s", fieldName, fieldName)
	}

	return assignment, errorHandling
}

// MixedApproachStrategy implementation

// Name returns the name of the strategy.
func (mas *MixedApproachStrategy) Name() string {
	return "mixed_approach"
}

// CanHandle determines if this strategy can handle the given method.
func (mas *MixedApproachStrategy) CanHandle(method *domain.MethodResult) bool {
	if method == nil {
		return false
	}

	// Mixed approach is suitable when there are both simple and complex fields
	fieldCount := len(method.Metadata)
	if fieldCount < 3 {
		return false // Not worth the complexity for few fields
	}

	simpleFields := 0
	complexFields := 0

	for _, fieldResult := range method.Metadata {
		if fr, ok := fieldResult.(*executor.FieldResult); ok {
			if mas.isSimpleField(fr) {
				simpleFields++
			} else {
				complexFields++
			}
		}
	}

	// Mixed approach is beneficial when we have both simple and complex fields
	return 0 < simpleFields && 0 < complexFields
}

// GenerateCode generates code for the given method using this strategy.
func (mas *MixedApproachStrategy) GenerateCode(ctx context.Context, method *domain.MethodResult, data *TemplateData) (string, error) {
	mas.logger.Debug("generating mixed approach code",
		zap.String("method", method.Method.Name))

	var code strings.Builder

	// Separate simple and complex fields
	simpleFields := make(map[string]*executor.FieldResult)
	complexFields := make(map[string]*executor.FieldResult)

	for fieldName, fieldResult := range method.Metadata {
		if fr, ok := fieldResult.(*executor.FieldResult); ok {
			if mas.isSimpleField(fr) {
				simpleFields[fieldName] = fr
			} else {
				complexFields[fieldName] = fr
			}
		}
	}

	// Generate composite literal for simple fields
	if len(simpleFields) > 0 {
		code.WriteString(fmt.Sprintf("%sdest := &DestType{\n", mas.config.IndentStyle))

		for fieldName, field := range simpleFields {
			assignment := mas.generateSimpleAssignment(fieldName, field)
			code.WriteString(fmt.Sprintf("%s%s%s,\n",
				mas.config.IndentStyle, mas.config.IndentStyle, assignment))
		}

		code.WriteString(fmt.Sprintf("%s}\n\n", mas.config.IndentStyle))
	} else {
		code.WriteString(fmt.Sprintf("%svar dest DestType\n\n", mas.config.IndentStyle))
	}

	// Generate assignments for complex fields
	for fieldName, field := range complexFields {
		assignment, errorHandling := mas.generateComplexAssignment(fieldName, field)

		code.WriteString(fmt.Sprintf("%s%s\n", mas.config.IndentStyle, assignment))

		if errorHandling != "" {
			code.WriteString(fmt.Sprintf("%s%s\n", mas.config.IndentStyle, errorHandling))
		}

		code.WriteString("\n")
	}

	// Generate return statement
	if len(simpleFields) > 0 {
		code.WriteString(fmt.Sprintf("%sreturn dest, nil\n", mas.config.IndentStyle))
	} else {
		code.WriteString(fmt.Sprintf("%sreturn &dest, nil\n", mas.config.IndentStyle))
	}

	mas.logger.Debug("mixed approach code generated",
		zap.String("method", method.Method.Name),
		zap.Int("simple_fields", len(simpleFields)),
		zap.Int("complex_fields", len(complexFields)))

	return code.String(), nil
}

// GetComplexity returns the complexity metrics for this strategy.
func (mas *MixedApproachStrategy) GetComplexity(method *domain.MethodResult) *ComplexityMetrics {
	metrics := NewComplexityMetrics()
	metrics.FieldCount = len(method.Metadata)

	simpleFields := 0
	complexFields := 0
	errorFields := 0

	for _, fieldResult := range method.Metadata {
		if fr, ok := fieldResult.(*executor.FieldResult); ok {
			if mas.isSimpleField(fr) {
				simpleFields++
			} else {
				complexFields++
			}

			if fr.Error != nil || !fr.Success {
				errorFields++
			}
		}
	}

	metrics.ErrorFields = errorFields
	metrics.ComplexityScore = float64(simpleFields)*1.5 + float64(complexFields)*4.0
	metrics.CyclomaticComplexity = 1 + errorFields*2
	metrics.LinesGenerated = simpleFields + complexFields*2 + errorFields*3 + 4
	metrics.RecommendedStrategy = StrategyMixedApproach

	return metrics
}

// GetRequiredImports returns the imports required by this strategy.
func (mas *MixedApproachStrategy) GetRequiredImports(method *domain.MethodResult) []*Import {
	var imports []*Import

	// Check if error handling is needed
	hasErrors := false

	for _, fieldResult := range method.Metadata {
		if fr, ok := fieldResult.(*executor.FieldResult); ok {
			if fr.Error != nil || !fr.Success {
				hasErrors = true
				break
			}
		}
	}

	if hasErrors {
		imports = append(imports, &Import{
			Path:     "fmt",
			Used:     true,
			Standard: true,
			Required: true,
		})
	}

	return imports
}

func (mas *MixedApproachStrategy) isSimpleField(field *executor.FieldResult) bool {
	return field.Success &&
		field.Error == nil &&
		field.RetryCount == 0 &&
		(field.StrategyUsed == domain.DirectStrategyType || field.StrategyUsed == domain.LiteralStrategyType)
}

func (mas *MixedApproachStrategy) generateSimpleAssignment(fieldName string, field *executor.FieldResult) string {
	switch field.StrategyUsed {
	case domain.DirectStrategyType:
		return fmt.Sprintf("%s: src.%s", fieldName, fieldName)
	case domain.LiteralStrategyType:
		return fmt.Sprintf("%s: %v", fieldName, field.Result)
	default:
		return fmt.Sprintf("%s: src.%s", fieldName, fieldName)
	}
}

func (mas *MixedApproachStrategy) generateComplexAssignment(fieldName string, field *executor.FieldResult) (string, string) {
	var assignment, errorHandling string

	switch field.StrategyUsed {
	case domain.ConverterStrategyType:
		if field.Error != nil || !field.Success {
			assignment = fmt.Sprintf("converted_%s, err := converter.Convert(src.%s)", fieldName, fieldName)
			errorHandling = fmt.Sprintf("if err != nil {\n%s%sreturn nil, fmt.Errorf(\"converting %s: %%w\", err)\n%s}",
				mas.config.IndentStyle, mas.config.IndentStyle, fieldName, mas.config.IndentStyle)
			assignment += fmt.Sprintf("\ndest.%s = converted_%s", fieldName, fieldName)
		} else {
			assignment = fmt.Sprintf("dest.%s = converter.Convert(src.%s)", fieldName, fieldName)
		}

	case domain.ExpressionStrategyType:
		if field.Error != nil || !field.Success {
			assignment = fmt.Sprintf("result_%s, err := expression.Evaluate(src.%s)", fieldName, fieldName)
			errorHandling = fmt.Sprintf("if err != nil {\n%s%sreturn nil, fmt.Errorf(\"evaluating expression for %s: %%w\", err)\n%s}",
				mas.config.IndentStyle, mas.config.IndentStyle, fieldName, mas.config.IndentStyle)
			assignment += fmt.Sprintf("\ndest.%s = result_%s", fieldName, fieldName)
		} else {
			assignment = fmt.Sprintf("dest.%s = expression.Evaluate(src.%s)", fieldName, fieldName)
		}

	case domain.CustomStrategyType:
		assignment = fmt.Sprintf("dest.%s = custom.Transform(src.%s)", fieldName, fieldName)

	default:
		assignment = fmt.Sprintf("dest.%s = src.%s", fieldName, fieldName)
	}

	return assignment, errorHandling
}
