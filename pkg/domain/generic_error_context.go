package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Error category constants to avoid goconst violations.
const (
	constraintViolation = "constraint_violation"
	typeInstantiation   = "type_instantiation"
	typeCompatibility   = "type_compatibility"
)

// GenericErrorEnhancer provides enhanced error messages with type parameter context.
type GenericErrorEnhancer struct {
	logger *zap.Logger

	// Configuration
	verboseMode              bool
	includeTypeHierarchy     bool
	includeSuggestions       bool
	includeConstraintDetails bool
	maxSuggestions           int
}

// EnhancedError represents an error with rich generic type context.
type EnhancedError struct {
	// Core error information
	OriginalError   error  `json:"original_error"`
	ErrorCode       string `json:"error_code"`
	ErrorMessage    string `json:"error_message"`
	EnhancedMessage string `json:"enhanced_message"`

	// Type parameter context
	GenericContext       *GenericErrorContext  `json:"generic_context,omitempty"`
	TypeParameterDetails []TypeParameterDetail `json:"type_parameter_details,omitempty"`
	ConstraintDetails    []ConstraintDetail    `json:"constraint_details,omitempty"`

	// Error resolution
	ErrorCategory string            `json:"error_category"`
	Suggestions   []ErrorSuggestion `json:"suggestions,omitempty"`
	RelatedTypes  []RelatedTypeInfo `json:"related_types,omitempty"`

	// Metadata
	Timestamp time.Time         `json:"timestamp"`
	Context   map[string]string `json:"context,omitempty"`
}

// GenericErrorContext provides context specific to generic type errors.
type GenericErrorContext struct {
	SourceTypeName          string                `json:"source_type_name"`
	TargetTypeName          string                `json:"target_type_name"`
	SourceTypeParameters    []string              `json:"source_type_parameters,omitempty"`
	TargetTypeParameters    []string              `json:"target_type_parameters,omitempty"`
	FailedTypeParameter     string                `json:"failed_type_parameter,omitempty"`
	ExpectedConstraint      string                `json:"expected_constraint,omitempty"`
	ActualType              string                `json:"actual_type,omitempty"`
	ConstraintViolationType string                `json:"constraint_violation_type,omitempty"`
	InstantiationContext    *InstantiationContext `json:"instantiation_context,omitempty"`
}

// TypeParameterDetail provides detailed information about a type parameter.
type TypeParameterDetail struct {
	ParameterName      string   `json:"parameter_name"`
	ParameterIndex     int      `json:"parameter_index"`
	ConstraintType     string   `json:"constraint_type"`
	ConstraintString   string   `json:"constraint_string"`
	ProvidedType       string   `json:"provided_type,omitempty"`
	CompatibilityLevel string   `json:"compatibility_level"`
	ViolationReason    string   `json:"violation_reason,omitempty"`
	AlternativeTypes   []string `json:"alternative_types,omitempty"`
}

// ConstraintDetail provides detailed constraint information.
type ConstraintDetail struct {
	ConstraintType        string   `json:"constraint_type"`
	ConstraintDescription string   `json:"constraint_description"`
	AllowedTypes          []string `json:"allowed_types,omitempty"`
	ForbiddenTypes        []string `json:"forbidden_types,omitempty"`
	ExampleUsage          string   `json:"example_usage,omitempty"`
	CommonMistakes        []string `json:"common_mistakes,omitempty"`
}

// ErrorSuggestion provides a suggested resolution for the error.
type ErrorSuggestion struct {
	SuggestionType     string  `json:"suggestion_type"`
	Description        string  `json:"description"`
	CodeExample        string  `json:"code_example,omitempty"`
	Confidence         float64 `json:"confidence"` // 0.0 to 1.0
	Priority           int     `json:"priority"`   // 1 (highest) to 10 (lowest)
	RequiresCodeChange bool    `json:"requires_code_change"`
}

// RelatedTypeInfo provides information about related types that might be relevant.
type RelatedTypeInfo struct {
	TypeName           string `json:"type_name"`
	Relationship       string `json:"relationship"`
	CompatibilityNotes string `json:"compatibility_notes,omitempty"`
	UsageExample       string `json:"usage_example,omitempty"`
}

// InstantiationContext provides context about generic type instantiation.
type InstantiationContext struct {
	GenericTypeName   string   `json:"generic_type_name"`
	TypeArguments     []string `json:"type_arguments"`
	InstantiationStep string   `json:"instantiation_step"`
	FailurePoint      string   `json:"failure_point,omitempty"`
}

// NewGenericErrorEnhancer creates a new error enhancer.
func NewGenericErrorEnhancer(logger *zap.Logger) *GenericErrorEnhancer {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &GenericErrorEnhancer{
		logger:                   logger,
		verboseMode:              true,
		includeTypeHierarchy:     true,
		includeSuggestions:       true,
		includeConstraintDetails: true,
		maxSuggestions:           5,
	}
}

// EnhanceError creates an enhanced error with rich generic type context.
func (gee *GenericErrorEnhancer) EnhanceError(
	originalError error,
	sourceType Type,
	targetType Type,
	context map[string]string,
) *EnhancedError {
	enhanced := &EnhancedError{
		OriginalError: originalError,
		ErrorMessage:  originalError.Error(),
		Timestamp:     time.Now(),
		Context:       context,
		Suggestions:   make([]ErrorSuggestion, 0),
		RelatedTypes:  make([]RelatedTypeInfo, 0),
	}

	// Generate error code based on error type
	enhanced.ErrorCode = gee.generateErrorCode(originalError, sourceType, targetType)
	enhanced.ErrorCategory = gee.categorizeError(originalError, targetType)

	// Create generic context
	if sourceType != nil || targetType != nil {
		enhanced.GenericContext = gee.createGenericContext(sourceType, targetType, originalError)
	}

	// Extract type parameter details
	enhanced.TypeParameterDetails = gee.extractTypeParameterDetails(sourceType, targetType)

	// Generate constraint details
	enhanced.ConstraintDetails = gee.generateConstraintDetails(sourceType, targetType)

	// Generate enhanced message
	enhanced.EnhancedMessage = gee.generateEnhancedMessage(enhanced)

	// Generate suggestions
	if gee.includeSuggestions {
		enhanced.Suggestions = gee.generateSuggestions(enhanced, sourceType, targetType)

		// Sort suggestions by priority and confidence
		sort.Slice(enhanced.Suggestions, func(i, j int) bool {
			if enhanced.Suggestions[i].Priority != enhanced.Suggestions[j].Priority {
				return enhanced.Suggestions[i].Priority < enhanced.Suggestions[j].Priority
			}
			return enhanced.Suggestions[i].Confidence > enhanced.Suggestions[j].Confidence
		})

		// Limit suggestions
		if len(enhanced.Suggestions) > gee.maxSuggestions {
			enhanced.Suggestions = enhanced.Suggestions[:gee.maxSuggestions]
		}
	}

	// Find related types
	enhanced.RelatedTypes = gee.findRelatedTypes(sourceType, targetType)

	gee.logger.Debug("enhanced error created",
		zap.String("error_code", enhanced.ErrorCode),
		zap.String("error_category", enhanced.ErrorCategory),
		zap.Int("suggestions", len(enhanced.Suggestions)),
		zap.Int("related_types", len(enhanced.RelatedTypes)))

	return enhanced
}

// generateErrorCode generates a specific error code based on the error characteristics.
func (gee *GenericErrorEnhancer) generateErrorCode(originalError error, sourceType, targetType Type) string {
	errorMessage := strings.ToLower(originalError.Error())

	// Generic-specific error codes
	if sourceType != nil && sourceType.Generic() || targetType != nil && targetType.Generic() {
		if strings.Contains(errorMessage, "constraint") {
			if strings.Contains(errorMessage, "violation") {
				return "GEN001" // Generic constraint violation
			}
			if strings.Contains(errorMessage, "comparable") {
				return "GEN002" // Comparable constraint violation
			}
			if strings.Contains(errorMessage, "union") {
				return "GEN003" // Union constraint violation
			}
			if strings.Contains(errorMessage, "underlying") {
				return "GEN004" // Underlying type constraint violation
			}
			return "GEN005" // General constraint error
		}

		if strings.Contains(errorMessage, "instantiation") {
			return "GEN010" // Type instantiation error
		}

		if strings.Contains(errorMessage, "substitution") {
			return "GEN020" // Type substitution error
		}

		if strings.Contains(errorMessage, "parameter") {
			return "GEN030" // Type parameter error
		}

		return "GEN100" // General generic error
	}

	// Compatibility-specific error codes
	if strings.Contains(errorMessage, "incompatible") {
		return "COMP001" // Type incompatibility
	}

	if strings.Contains(errorMessage, "assignment") {
		return "COMP010" // Assignment compatibility
	}

	return "ERR000" // General error
}

// categorizeError categorizes the error for better organization.
func (gee *GenericErrorEnhancer) categorizeError(originalError error, _targetType Type) string {
	errorMessage := strings.ToLower(originalError.Error())

	if strings.Contains(errorMessage, "constraint") {
		return constraintViolation
	}

	if strings.Contains(errorMessage, "instantiation") {
		return typeInstantiation
	}

	if strings.Contains(errorMessage, "substitution") {
		return "type_substitution"
	}

	if strings.Contains(errorMessage, "compatibility") || strings.Contains(errorMessage, "incompatible") {
		return typeCompatibility
	}

	if strings.Contains(errorMessage, "assignment") {
		return "field_assignment"
	}

	return "generic_error"
}

// createGenericContext creates context information for generic types.
func (gee *GenericErrorEnhancer) createGenericContext(sourceType, targetType Type, originalError error) *GenericErrorContext {
	context := &GenericErrorContext{}

	if sourceType != nil {
		context.SourceTypeName = sourceType.String()
		if sourceType.Generic() {
			params := sourceType.TypeParams()
			context.SourceTypeParameters = make([]string, len(params))
			for i, param := range params {
				context.SourceTypeParameters[i] = param.Name
			}
		}
	}

	if targetType != nil {
		context.TargetTypeName = targetType.String()
		if targetType.Generic() {
			params := targetType.TypeParams()
			context.TargetTypeParameters = make([]string, len(params))
			for i, param := range params {
				context.TargetTypeParameters[i] = param.Name
			}
		}
	}

	// Extract constraint violation details from error message
	errorMessage := originalError.Error()
	if strings.Contains(errorMessage, "constraint violation") {
		context.ConstraintViolationType = constraintViolation
		// Try to extract specific details from error message
		if strings.Contains(errorMessage, "comparable") {
			context.ExpectedConstraint = comparableConstraint
		} else if strings.Contains(errorMessage, "union") {
			context.ExpectedConstraint = unionConstraint
		} else if strings.Contains(errorMessage, "underlying") {
			context.ExpectedConstraint = underlyingConstraint
		}
	}

	return context
}

// extractTypeParameterDetails extracts detailed information about type parameters.
func (gee *GenericErrorEnhancer) extractTypeParameterDetails(sourceType, targetType Type) []TypeParameterDetail {
	details := make([]TypeParameterDetail, 0)

	// Extract from source type
	if sourceType != nil && sourceType.Generic() {
		params := sourceType.TypeParams()
		for i, param := range params {
			detail := TypeParameterDetail{
				ParameterName:      param.Name,
				ParameterIndex:     i,
				ConstraintType:     param.GetConstraintType(),
				ConstraintString:   gee.getConstraintString(param),
				CompatibilityLevel: "unknown",
			}

			// Add alternative types based on constraint
			detail.AlternativeTypes = gee.getAlternativeTypes(param)

			details = append(details, detail)
		}
	}

	// Extract from target type
	if targetType != nil && targetType.Generic() && sourceType != targetType {
		params := targetType.TypeParams()
		for i, param := range params {
			detail := TypeParameterDetail{
				ParameterName:      fmt.Sprintf("target_%s", param.Name),
				ParameterIndex:     i,
				ConstraintType:     param.GetConstraintType(),
				ConstraintString:   gee.getConstraintString(param),
				CompatibilityLevel: "unknown",
			}

			detail.AlternativeTypes = gee.getAlternativeTypes(param)

			details = append(details, detail)
		}
	}

	return details
}

// getConstraintString returns a human-readable constraint string.
func (gee *GenericErrorEnhancer) getConstraintString(param TypeParam) string {
	switch param.GetConstraintType() {
	case anyConstraint:
		return "any type"
	case comparableConstraint:
		return "comparable type (supports == and !=)"
	case unionConstraint:
		if 0 < len(param.UnionTypes) {
			types := make([]string, len(param.UnionTypes))
			for i, t := range param.UnionTypes {
				types[i] = t.String()
			}
			return fmt.Sprintf("one of: %s", strings.Join(types, " | "))
		}
		return "union type"
	case unionUnderlyingConstraint:
		if 0 < len(param.UnionTypes) {
			types := make([]string, len(param.UnionTypes))
			for i, t := range param.UnionTypes {
				types[i] = "~" + t.String()
			}
			return fmt.Sprintf("underlying type of: %s", strings.Join(types, " | "))
		}
		return "underlying union type"
	case underlyingConstraint:
		if param.Underlying != nil {
			return fmt.Sprintf("underlying type ~%s", param.Underlying.Type.String())
		}
		return "underlying type"
	case interfaceKeyword:
		if param.Constraint != nil {
			return fmt.Sprintf("implements %s", param.Constraint.String())
		}
		return "interface implementation"
	default:
		return param.GetConstraintType()
	}
}

// getAlternativeTypes suggests alternative types for a constraint.
func (gee *GenericErrorEnhancer) getAlternativeTypes(param TypeParam) []string {
	switch param.GetConstraintType() {
	case anyConstraint:
		return []string{"int", "string", "bool", "interface{}", "any"}
	case comparableConstraint:
		return []string{"int", "string", "bool", "float64", "int64", "uint64"}
	case unionConstraint:
		if 0 < len(param.UnionTypes) {
			types := make([]string, len(param.UnionTypes))
			for i, t := range param.UnionTypes {
				types[i] = t.String()
			}
			return types
		}
		return []string{}
	case underlyingConstraint:
		if param.Underlying != nil {
			baseType := param.Underlying.Type.String()
			return []string{baseType, "~" + baseType}
		}
		return []string{}
	default:
		return []string{}
	}
}

// generateConstraintDetails generates detailed constraint information.
func (gee *GenericErrorEnhancer) generateConstraintDetails(sourceType, targetType Type) []ConstraintDetail {
	details := make([]ConstraintDetail, 0)
	constraintsSeen := make(map[string]bool)

	// Collect unique constraints from both types
	if sourceType != nil && sourceType.Generic() {
		for _, param := range sourceType.TypeParams() {
			constraintType := param.GetConstraintType()
			if !constraintsSeen[constraintType] {
				detail := gee.createConstraintDetail(constraintType, param)
				details = append(details, detail)
				constraintsSeen[constraintType] = true
			}
		}
	}

	if targetType != nil && targetType.Generic() {
		for _, param := range targetType.TypeParams() {
			constraintType := param.GetConstraintType()
			if !constraintsSeen[constraintType] {
				detail := gee.createConstraintDetail(constraintType, param)
				details = append(details, detail)
				constraintsSeen[constraintType] = true
			}
		}
	}

	return details
}

// createConstraintDetail creates detailed information for a specific constraint type.
func (gee *GenericErrorEnhancer) createConstraintDetail(constraintType string, param TypeParam) ConstraintDetail {
	detail := ConstraintDetail{
		ConstraintType: constraintType,
	}

	switch constraintType {
	case anyConstraint:
		detail.ConstraintDescription = "Accepts any type - no restrictions"
		detail.AllowedTypes = []string{"any type"}
		detail.ExampleUsage = "func Process[T any](value T) { ... }"
		detail.CommonMistakes = []string{"No common mistakes - any type is accepted"}

	case comparableConstraint:
		detail.ConstraintDescription = "Requires types that support equality operators (== and !=)"
		detail.AllowedTypes = []string{"int", "string", "bool", "float64", "pointers", "arrays", "channels"}
		detail.ForbiddenTypes = []string{"slices", "maps", "functions"}
		detail.ExampleUsage = "func FindIndex[T comparable](slice []T, target T) int { ... }"
		detail.CommonMistakes = []string{
			"Using slice types ([]T) with comparable constraint",
			"Using map types (map[K]V) with comparable constraint",
			"Using function types with comparable constraint",
		}

	case unionConstraint:
		detail.ConstraintDescription = "Must be one of the specified types"
		if 0 < len(param.UnionTypes) {
			for _, t := range param.UnionTypes {
				detail.AllowedTypes = append(detail.AllowedTypes, t.String())
			}
		}
		detail.ExampleUsage = "type Number interface { int | float64 | string }"
		detail.CommonMistakes = []string{
			"Providing types not in the union",
			"Confusing union with underlying type union (~int | ~float64)",
		}

	case unionUnderlyingConstraint:
		detail.ConstraintDescription = "Must have underlying type that matches one of the specified types"
		if 0 < len(param.UnionTypes) {
			for _, t := range param.UnionTypes {
				detail.AllowedTypes = append(detail.AllowedTypes, "~"+t.String())
			}
		}
		detail.ExampleUsage = "type Integer interface { ~int | ~int64 | ~int32 }"
		detail.CommonMistakes = []string{
			"Using exact types instead of underlying types",
			"Forgetting the ~ prefix in constraint definitions",
		}

	case underlyingConstraint:
		detail.ConstraintDescription = "Must have the specified underlying type"
		if param.Underlying != nil {
			detail.AllowedTypes = []string{"~" + param.Underlying.Type.String()}
		}
		detail.ExampleUsage = "type StringLike interface { ~string }"
		detail.CommonMistakes = []string{
			"Using exact type instead of underlying type",
			"Not understanding the difference between T and ~T",
		}

	case interfaceKeyword:
		detail.ConstraintDescription = "Must implement the specified interface"
		if param.Constraint != nil {
			detail.AllowedTypes = []string{"types implementing " + param.Constraint.String()}
		}
		detail.ExampleUsage = "func Process[T io.Reader](r T) { ... }"
		detail.CommonMistakes = []string{
			"Providing types that don't implement the interface",
			"Confusing structural and nominal typing",
		}

	default:
		detail.ConstraintDescription = "Custom constraint type"
		detail.ExampleUsage = fmt.Sprintf("func Process[T %s](value T) { ... }", constraintType)
	}

	return detail
}

// generateEnhancedMessage creates an enhanced, human-readable error message.
func (gee *GenericErrorEnhancer) generateEnhancedMessage(enhanced *EnhancedError) string {
	var builder strings.Builder

	// Start with the error category
	switch enhanced.ErrorCategory {
	case constraintViolation:
		builder.WriteString("🚫 Generic Constraint Violation\n")
	case typeInstantiation:
		builder.WriteString("🔧 Type Instantiation Error\n")
	case "type_substitution":
		builder.WriteString("🔄 Type Substitution Error\n")
	case typeCompatibility:
		builder.WriteString("❌ Type Compatibility Error\n")
	case "field_assignment":
		builder.WriteString("📝 Field Assignment Error\n")
	default:
		builder.WriteString("⚠️ Generic Type Error\n")
	}

	builder.WriteString(fmt.Sprintf("Error Code: %s\n\n", enhanced.ErrorCode))

	// Add the original error message
	builder.WriteString(fmt.Sprintf("Original Error: %s\n\n", enhanced.ErrorMessage))

	// Add generic context if available
	if enhanced.GenericContext != nil {
		gee.addGenericContextToMessage(&builder, enhanced.GenericContext)
	}

	// Add type parameter details
	if 0 < len(enhanced.TypeParameterDetails) {
		builder.WriteString("📋 Type Parameter Details:\n")
		for _, detail := range enhanced.TypeParameterDetails {
			builder.WriteString(fmt.Sprintf("  • %s: %s\n", detail.ParameterName, detail.ConstraintString))
			if detail.ViolationReason != "" {
				builder.WriteString(fmt.Sprintf("    ❌ %s\n", detail.ViolationReason))
			}
			if 0 < len(detail.AlternativeTypes) {
				builder.WriteString(fmt.Sprintf("    💡 Try: %s\n", strings.Join(detail.AlternativeTypes[:min(3, len(detail.AlternativeTypes))], ", ")))
			}
		}
		builder.WriteString("\n")
	}

	// Add top suggestions if in verbose mode
	if gee.verboseMode && 0 < len(enhanced.Suggestions) {
		builder.WriteString("💡 Suggested Solutions:\n")
		for i, suggestion := range enhanced.Suggestions[:min(3, len(enhanced.Suggestions))] {
			builder.WriteString(fmt.Sprintf("%d. %s", i+1, suggestion.Description))
			if suggestion.CodeExample != "" {
				builder.WriteString(fmt.Sprintf("\n   Example: %s", suggestion.CodeExample))
			}
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// addGenericContextToMessage adds generic context information to the message.
func (gee *GenericErrorEnhancer) addGenericContextToMessage(builder *strings.Builder, context *GenericErrorContext) {
	gee.addTypeInformation(builder, context)
	gee.addFailedTypeParameter(builder, context)
}

// addTypeInformation adds source and target type information to the message.
func (gee *GenericErrorEnhancer) addTypeInformation(builder *strings.Builder, context *GenericErrorContext) {
	if context.SourceTypeName == "" && context.TargetTypeName == "" {
		return
	}

	builder.WriteString("🔍 Type Information:\n")
	gee.addSingleTypeInfo(builder, "  Source: %s", context.SourceTypeName, context.SourceTypeParameters)
	gee.addSingleTypeInfo(builder, "  Target: %s", context.TargetTypeName, context.TargetTypeParameters)
	builder.WriteString("\n")
}

// addSingleTypeInfo adds information for a single type (source or target).
func (gee *GenericErrorEnhancer) addSingleTypeInfo(builder *strings.Builder, format, typeName string, typeParameters []string) {
	if typeName == "" {
		return
	}

	builder.WriteString(fmt.Sprintf(format, typeName))
	if 0 < len(typeParameters) {
		builder.WriteString(fmt.Sprintf(" [%s]", strings.Join(typeParameters, ", ")))
	}
	builder.WriteString("\n")
}

// addFailedTypeParameter adds failed type parameter information to the message.
func (gee *GenericErrorEnhancer) addFailedTypeParameter(builder *strings.Builder, context *GenericErrorContext) {
	if context.FailedTypeParameter == "" {
		return
	}

	builder.WriteString(fmt.Sprintf("❌ Failed Type Parameter: %s\n", context.FailedTypeParameter))
	if context.ExpectedConstraint != "" && context.ActualType != "" {
		builder.WriteString(fmt.Sprintf("   Expected: %s, Got: %s\n", context.ExpectedConstraint, context.ActualType))
	}
	builder.WriteString("\n")
}

// generateSuggestions generates helpful suggestions for resolving the error.
func (gee *GenericErrorEnhancer) generateSuggestions(enhanced *EnhancedError, sourceType, targetType Type) []ErrorSuggestion {
	suggestions := make([]ErrorSuggestion, 0)

	// Generate suggestions based on error category
	switch enhanced.ErrorCategory {
	case constraintViolation:
		suggestions = append(suggestions, gee.generateConstraintSuggestions(enhanced, targetType)...)
	case typeInstantiation:
		suggestions = append(suggestions, gee.generateInstantiationSuggestions(sourceType, targetType)...)
	case typeCompatibility:
		suggestions = append(suggestions, gee.generateCompatibilitySuggestions(sourceType, targetType)...)
	}

	// Add general suggestions
	suggestions = append(suggestions, gee.generateGeneralSuggestions(enhanced)...)

	return suggestions
}

// generateConstraintSuggestions generates suggestions for constraint violations.
func (gee *GenericErrorEnhancer) generateConstraintSuggestions(enhanced *EnhancedError, _targetType Type) []ErrorSuggestion {
	suggestions := make([]ErrorSuggestion, 0)

	if enhanced.GenericContext != nil && enhanced.GenericContext.ExpectedConstraint != "" {
		constraint := enhanced.GenericContext.ExpectedConstraint

		switch constraint {
		case comparableConstraint:
			suggestions = append(suggestions, ErrorSuggestion{
				SuggestionType:     "constraint_fix",
				Description:        "Use a comparable type like int, string, or bool",
				CodeExample:        "func Process[T comparable](items []T, target T) bool",
				Confidence:         0.9,
				Priority:           1,
				RequiresCodeChange: true,
			})

		case unionConstraint:
			suggestions = append(suggestions, ErrorSuggestion{
				SuggestionType:     "constraint_fix",
				Description:        "Use one of the allowed union types",
				CodeExample:        "type Number interface { int | float64 | string }",
				Confidence:         0.8,
				Priority:           1,
				RequiresCodeChange: true,
			})
		}
	}

	return suggestions
}

// generateInstantiationSuggestions generates suggestions for instantiation errors.
func (gee *GenericErrorEnhancer) generateInstantiationSuggestions(_sourceType, _targetType Type) []ErrorSuggestion {
	suggestions := make([]ErrorSuggestion, 0)

	suggestions = append(suggestions, ErrorSuggestion{
		SuggestionType:     "instantiation_fix",
		Description:        "Check that all type arguments satisfy their constraints",
		Confidence:         0.7,
		Priority:           2,
		RequiresCodeChange: true,
	})

	return suggestions
}

// generateCompatibilitySuggestions generates suggestions for compatibility errors.
func (gee *GenericErrorEnhancer) generateCompatibilitySuggestions(_sourceType, _targetType Type) []ErrorSuggestion {
	suggestions := make([]ErrorSuggestion, 0)

	suggestions = append(suggestions, ErrorSuggestion{
		SuggestionType:     "compatibility_fix",
		Description:        "Consider using type conversion or type assertion",
		CodeExample:        "targetValue := TargetType(sourceValue)",
		Confidence:         0.6,
		Priority:           2,
		RequiresCodeChange: true,
	})

	return suggestions
}

// generateGeneralSuggestions generates general suggestions applicable to most errors.
func (gee *GenericErrorEnhancer) generateGeneralSuggestions(_enhanced *EnhancedError) []ErrorSuggestion {
	suggestions := make([]ErrorSuggestion, 0)

	suggestions = append(suggestions, ErrorSuggestion{
		SuggestionType:     "general",
		Description:        "Review the Go generics documentation for constraint syntax",
		Confidence:         0.5,
		Priority:           5,
		RequiresCodeChange: false,
	})

	return suggestions
}

// findRelatedTypes finds types that might be relevant to the error context.
func (gee *GenericErrorEnhancer) findRelatedTypes(sourceType, targetType Type) []RelatedTypeInfo {
	relatedTypes := make([]RelatedTypeInfo, 0)

	// This is a simplified implementation
	// In a real system, you would analyze the type hierarchy and find related types

	if sourceType != nil {
		if sourceType.Generic() {
			relatedTypes = append(relatedTypes, RelatedTypeInfo{
				TypeName:     sourceType.String(),
				Relationship: "source_generic_type",
				UsageExample: fmt.Sprintf("var x %s", sourceType.String()),
			})
		}
	}

	if targetType != nil {
		if targetType.Generic() {
			relatedTypes = append(relatedTypes, RelatedTypeInfo{
				TypeName:     targetType.String(),
				Relationship: "target_generic_type",
				UsageExample: fmt.Sprintf("var x %s", targetType.String()),
			})
		}
	}

	return relatedTypes
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// String returns a formatted string representation of the enhanced error.
func (ee *EnhancedError) String() string {
	return ee.EnhancedMessage
}

// Error implements the error interface.
func (ee *EnhancedError) Error() string {
	return ee.EnhancedMessage
}

// Unwrap returns the original error.
func (ee *EnhancedError) Unwrap() error {
	return ee.OriginalError
}

// SetVerboseMode enables or disables verbose error messages.
func (gee *GenericErrorEnhancer) SetVerboseMode(verbose bool) {
	gee.verboseMode = verbose
	gee.logger.Debug("verbose mode updated", zap.Bool("verbose", verbose))
}

// SetIncludeSuggestions enables or disables suggestion generation.
func (gee *GenericErrorEnhancer) SetIncludeSuggestions(include bool) {
	gee.includeSuggestions = include
	gee.logger.Debug("suggestions mode updated", zap.Bool("include_suggestions", include))
}

// SetMaxSuggestions sets the maximum number of suggestions to generate.
func (gee *GenericErrorEnhancer) SetMaxSuggestions(max int) {
	gee.maxSuggestions = max
	gee.logger.Debug("max suggestions updated", zap.Int("max_suggestions", max))
}
