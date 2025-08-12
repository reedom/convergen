package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Static errors for err113 compliance.
var (
	ErrFieldMappingValidationFailed = errors.New("field mapping validation failed")
	ErrInvalidFieldMapping          = errors.New("invalid field mapping")
	ErrFieldTypeIncompatible        = errors.New("field types are incompatible")
	ErrMissingFieldMapping          = errors.New("missing field mapping")
	ErrGenericFieldValidationFailed = errors.New("generic field validation failed")
)

// FieldMappingValidator validates field mappings for type safety, especially for generic types.
type FieldMappingValidator struct {
	compatibilityChecker *TypeCompatibilityChecker
	compatibilityMatrix  *GenericCompatibilityMatrix
	logger               *zap.Logger

	// Configuration
	strictValidation    bool
	allowUnsafeMappings bool
	requireAllFields    bool

	// Performance tracking
	validationCount    int64
	validationFailures int64
	validationTime     time.Duration
}

// ValidatorFieldMapping represents a mapping between source and destination fields for validation.
type ValidatorFieldMapping struct {
	SourceFieldName    string `json:"source_field_name"`
	TargetFieldName    string `json:"target_field_name"`
	SourceType         Type   `json:"source_type"`
	TargetType         Type   `json:"target_type"`
	ConversionRequired bool   `json:"conversion_required"`
	ConversionFunction string `json:"conversion_function,omitempty"`
	MappingStrategy    string `json:"mapping_strategy"`
}

// FieldMappingValidationResult contains the results of field mapping validation.
type FieldMappingValidationResult struct {
	Valid                bool                              `json:"valid"`
	FieldResults         map[string]*FieldValidationResult `json:"field_results"`
	IncompatibleMappings []IncompatibleMapping             `json:"incompatible_mappings,omitempty"`
	MissingMappings      []MissingMapping                  `json:"missing_mappings,omitempty"`
	GenericTypeIssues    []GenericTypeIssue                `json:"generic_type_issues,omitempty"`
	OverallCompatibility *CompatibilityResult              `json:"overall_compatibility,omitempty"`
	ValidationSummary    ValidationSummary                 `json:"validation_summary"`
	ValidationDurationMS int64                             `json:"validation_duration_ms"`
}

// FieldValidationResult contains the result of validating a single field mapping.
type FieldValidationResult struct {
	SourceField         string               `json:"source_field"`
	TargetField         string               `json:"target_field"`
	Compatible          bool                 `json:"compatible"`
	CompatibilityResult *CompatibilityResult `json:"compatibility_result,omitempty"`
	RequiredConversion  string               `json:"required_conversion,omitempty"`
	ValidationErrors    []string             `json:"validation_errors,omitempty"`
	GenericContext      *GenericFieldContext `json:"generic_context,omitempty"`
}

// IncompatibleMapping represents an incompatible field mapping.
type IncompatibleMapping struct {
	SourceField           string `json:"source_field"`
	TargetField           string `json:"target_field"`
	SourceType            string `json:"source_type"`
	TargetType            string `json:"target_type"`
	IncompatibilityReason string `json:"incompatibility_reason"`
	SuggestedFix          string `json:"suggested_fix,omitempty"`
}

// MissingMapping represents a missing field mapping.
type MissingMapping struct {
	FieldName        string `json:"field_name"`
	FieldType        string `json:"field_type"`
	InSourceType     bool   `json:"in_source_type"`
	InTargetType     bool   `json:"in_target_type"`
	SuggestedMapping string `json:"suggested_mapping,omitempty"`
}

// GenericTypeIssue represents issues specific to generic types in field mapping.
type GenericTypeIssue struct {
	IssueType            string   `json:"issue_type"`
	Description          string   `json:"description"`
	AffectedFields       []string `json:"affected_fields,omitempty"`
	AffectedTypeParams   []string `json:"affected_type_params,omitempty"`
	ResolutionSuggestion string   `json:"resolution_suggestion,omitempty"`
}

// GenericFieldContext provides context for generic field validation.
type GenericFieldContext struct {
	SourceIsGeneric       bool                         `json:"source_is_generic"`
	TargetIsGeneric       bool                         `json:"target_is_generic"`
	TypeParameterMappings map[string]Type              `json:"type_parameter_mappings,omitempty"`
	InstantiationRequired bool                         `json:"instantiation_required"`
	ConstraintValidation  *ConstraintValidationContext `json:"constraint_validation,omitempty"`
}

// ConstraintValidationContext provides constraint validation context.
type ConstraintValidationContext struct {
	ConstraintsChecked   []string              `json:"constraints_checked"`
	ConstraintViolations []ConstraintViolation `json:"constraint_violations,omitempty"`
	ValidationPassed     bool                  `json:"validation_passed"`
}

// ValidationSummary provides a summary of the validation results.
type ValidationSummary struct {
	TotalFields         int `json:"total_fields"`
	CompatibleFields    int `json:"compatible_fields"`
	IncompatibleFields  int `json:"incompatible_fields"`
	MissingFields       int `json:"missing_fields"`
	GenericIssues       int `json:"generic_issues"`
	RequiresConversions int `json:"requires_conversions"`
}

// NewFieldMappingValidator creates a new field mapping validator.
func NewFieldMappingValidator(logger *zap.Logger) *FieldMappingValidator {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &FieldMappingValidator{
		logger:              logger,
		strictValidation:    true,
		allowUnsafeMappings: false,
		requireAllFields:    true,
	}
}

// SetCompatibilityChecker sets the compatibility checker for the validator.
func (fmv *FieldMappingValidator) SetCompatibilityChecker(checker *TypeCompatibilityChecker) {
	fmv.compatibilityChecker = checker
	fmv.logger.Debug("compatibility checker set for field mapping validator")
}

// SetCompatibilityMatrix sets the compatibility matrix for advanced generic validation.
func (fmv *FieldMappingValidator) SetCompatibilityMatrix(matrix *GenericCompatibilityMatrix) {
	fmv.compatibilityMatrix = matrix
	fmv.logger.Debug("compatibility matrix set for field mapping validator")
}

// ValidateFieldMappings validates a set of field mappings for type safety.
func (fmv *FieldMappingValidator) ValidateFieldMappings(
	ctx context.Context,
	mappings []ValidatorFieldMapping,
) (*FieldMappingValidationResult, error) {
	startTime := time.Now()
	fmv.validationCount++

	fmv.logger.Debug("validating field mappings",
		zap.Int("mapping_count", len(mappings)))

	result := &FieldMappingValidationResult{
		Valid:                true,
		FieldResults:         make(map[string]*FieldValidationResult),
		IncompatibleMappings: make([]IncompatibleMapping, 0),
		MissingMappings:      make([]MissingMapping, 0),
		GenericTypeIssues:    make([]GenericTypeIssue, 0),
		ValidationSummary: ValidationSummary{
			TotalFields: len(mappings),
		},
	}

	// Validate each field mapping
	for _, mapping := range mappings {
		fieldResult := fmv.validateSingleFieldMapping(ctx, mapping)
		result.FieldResults[mapping.SourceFieldName] = fieldResult

		// Update summary statistics
		if fieldResult.Compatible {
			result.ValidationSummary.CompatibleFields++
			if fieldResult.RequiredConversion != "" {
				result.ValidationSummary.RequiresConversions++
			}
		} else {
			result.ValidationSummary.IncompatibleFields++
			result.Valid = false

			// Add to incompatible mappings
			incompatible := IncompatibleMapping{
				SourceField: mapping.SourceFieldName,
				TargetField: mapping.TargetFieldName,
				SourceType:  mapping.SourceType.String(),
				TargetType:  mapping.TargetType.String(),
			}

			if fieldResult.CompatibilityResult != nil {
				incompatible.IncompatibilityReason = fieldResult.CompatibilityResult.ErrorMessage
			}

			incompatible.SuggestedFix = fmv.generateFixSuggestion(mapping, fieldResult)
			result.IncompatibleMappings = append(result.IncompatibleMappings, incompatible)
		}

		// Handle generic-specific issues
		if fieldResult.GenericContext != nil {
			if !fieldResult.GenericContext.ConstraintValidation.ValidationPassed {
				issue := GenericTypeIssue{
					IssueType:      "constraint_violation",
					Description:    "Generic constraint validation failed",
					AffectedFields: []string{mapping.SourceFieldName, mapping.TargetFieldName},
				}

				if 0 < len(fieldResult.GenericContext.ConstraintValidation.ConstraintViolations) {
					violation := fieldResult.GenericContext.ConstraintValidation.ConstraintViolations[0]
					issue.ResolutionSuggestion = fmt.Sprintf("Use type %s that satisfies constraint %s",
						violation.ActualType, violation.ExpectedConstraint)
				}

				result.GenericTypeIssues = append(result.GenericTypeIssues, issue)
				result.ValidationSummary.GenericIssues++
			}
		}
	}

	// Set timing
	result.ValidationDurationMS = time.Since(startTime).Milliseconds()
	fmv.validationTime += time.Since(startTime)

	if !result.Valid {
		fmv.validationFailures++
	}

	fmv.logger.Debug("field mapping validation completed",
		zap.Bool("valid", result.Valid),
		zap.Int("compatible_fields", result.ValidationSummary.CompatibleFields),
		zap.Int("incompatible_fields", result.ValidationSummary.IncompatibleFields),
		zap.Int("generic_issues", result.ValidationSummary.GenericIssues),
		zap.Int64("duration_ms", result.ValidationDurationMS))

	return result, nil
}

// validateSingleFieldMapping validates a single field mapping.
func (fmv *FieldMappingValidator) validateSingleFieldMapping(
	ctx context.Context,
	mapping ValidatorFieldMapping,
) *FieldValidationResult {
	result := &FieldValidationResult{
		SourceField:      mapping.SourceFieldName,
		TargetField:      mapping.TargetFieldName,
		Compatible:       false,
		ValidationErrors: make([]string, 0),
	}

	// Basic validation
	if mapping.SourceType == nil {
		result.ValidationErrors = append(result.ValidationErrors, "source type is nil")
		return result
	}

	if mapping.TargetType == nil {
		result.ValidationErrors = append(result.ValidationErrors, "target type is nil")
		return result
	}

	// Perform compatibility check
	var compatibilityResult *CompatibilityResult
	var err error

	if fmv.compatibilityChecker != nil {
		compatibilityResult, err = fmv.compatibilityChecker.CheckAssignabilityWithContext(
			ctx, mapping.SourceType, mapping.TargetType)
		if err != nil {
			result.ValidationErrors = append(result.ValidationErrors,
				fmt.Sprintf("compatibility check failed: %s", err.Error()))
			return result
		}
	} else {
		// Fallback basic compatibility check
		compatibilityResult = fmv.performBasicCompatibilityCheck(mapping.SourceType, mapping.TargetType)
	}

	result.CompatibilityResult = compatibilityResult
	result.Compatible = compatibilityResult.Compatible

	if compatibilityResult.RequiresConversion {
		result.RequiredConversion = compatibilityResult.ConversionFunction
	}

	// Handle generic types
	if mapping.SourceType.Generic() || mapping.TargetType.Generic() {
		result.GenericContext = fmv.createGenericFieldContext(mapping, compatibilityResult)
	}

	return result
}

// createGenericFieldContext creates context information for generic field validation.
func (fmv *FieldMappingValidator) createGenericFieldContext(
	mapping ValidatorFieldMapping,
	compatibilityResult *CompatibilityResult,
) *GenericFieldContext {
	context := &GenericFieldContext{
		SourceIsGeneric:       mapping.SourceType.Generic(),
		TargetIsGeneric:       mapping.TargetType.Generic(),
		TypeParameterMappings: make(map[string]Type),
		InstantiationRequired: false,
		ConstraintValidation: &ConstraintValidationContext{
			ValidationPassed:     true,
			ConstraintsChecked:   make([]string, 0),
			ConstraintViolations: make([]ConstraintViolation, 0),
		},
	}

	// Extract constraint violations from compatibility result
	if compatibilityResult.ConstraintViolations != nil {
		context.ConstraintValidation.ConstraintViolations = compatibilityResult.ConstraintViolations
		context.ConstraintValidation.ValidationPassed = len(compatibilityResult.ConstraintViolations) == 0
	}

	// Extract type parameter mappings from generic context
	if compatibilityResult.GenericContext != nil {
		for name, t := range compatibilityResult.GenericContext.TypeParameterMappings {
			context.TypeParameterMappings[name] = t
		}
		context.InstantiationRequired = compatibilityResult.GenericContext.SourceIsGeneric ||
			compatibilityResult.GenericContext.TargetIsGeneric
	}

	// Record checked constraints
	if mapping.SourceType.Generic() {
		for _, param := range mapping.SourceType.TypeParams() {
			context.ConstraintValidation.ConstraintsChecked = append(
				context.ConstraintValidation.ConstraintsChecked, param.GetConstraintType())
		}
	}

	if mapping.TargetType.Generic() {
		for _, param := range mapping.TargetType.TypeParams() {
			context.ConstraintValidation.ConstraintsChecked = append(
				context.ConstraintValidation.ConstraintsChecked, param.GetConstraintType())
		}
	}

	return context
}

// performBasicCompatibilityCheck provides fallback compatibility checking.
func (fmv *FieldMappingValidator) performBasicCompatibilityCheck(sourceType, targetType Type) *CompatibilityResult {
	result := &CompatibilityResult{
		Level:      CompatibilityNone,
		Compatible: false,
	}

	// Basic type name comparison
	if sourceType.String() == targetType.String() {
		result.Level = CompatibilityIdentical
		result.Compatible = true
		return result
	}

	// Kind-based compatibility
	if sourceType.Kind() == targetType.Kind() {
		result.Level = CompatibilityWithConversion
		result.Compatible = true
		result.RequiresConversion = true
		result.ConversionFunction = fmt.Sprintf("%s(value)", targetType.Name())
		return result
	}

	// Basic assignability check
	if sourceType.AssignableTo(targetType) {
		result.Level = CompatibilityDirect
		result.Compatible = true
		return result
	}

	result.ErrorMessage = fmt.Sprintf("types %s and %s are not compatible",
		sourceType.String(), targetType.String())
	return result
}

// generateFixSuggestion generates a suggestion for fixing incompatible mappings.
func (fmv *FieldMappingValidator) generateFixSuggestion(
	mapping ValidatorFieldMapping,
	result *FieldValidationResult,
) string {
	if result.CompatibilityResult == nil {
		return "Unable to determine fix suggestion"
	}

	compatResult := result.CompatibilityResult

	if compatResult.RequiresConversion && compatResult.ConversionFunction != "" {
		return fmt.Sprintf("Use conversion: %s", compatResult.ConversionFunction)
	}

	// Handle generic-specific suggestions
	if result.GenericContext != nil && !result.GenericContext.ConstraintValidation.ValidationPassed {
		if 0 < len(result.GenericContext.ConstraintValidation.ConstraintViolations) {
			violation := result.GenericContext.ConstraintValidation.ConstraintViolations[0]
			return fmt.Sprintf("Change type argument to satisfy constraint %s", violation.ExpectedConstraint)
		}
	}

	// Basic suggestions based on types
	sourceKind := mapping.SourceType.Kind().String()
	targetKind := mapping.TargetType.Kind().String()

	if sourceKind != targetKind {
		return fmt.Sprintf("Consider converting %s to %s type", sourceKind, targetKind)
	}

	if mapping.SourceType.Generic() || mapping.TargetType.Generic() {
		return "Consider using generic type instantiation or type assertions"
	}

	return "Consider using explicit type conversion or refactoring the types"
}

// ValidateStructMapping validates field mappings for entire struct types.
func (fmv *FieldMappingValidator) ValidateStructMapping(
	ctx context.Context,
	sourceType, targetType Type,
) (*FieldMappingValidationResult, error) {
	fmv.logger.Debug("validating struct mapping",
		zap.String("source_type", sourceType.String()),
		zap.String("target_type", targetType.String()))

	// This is a simplified implementation. In a full system, you would:
	// 1. Extract all fields from both struct types
	// 2. Create field mappings based on field names and types
	// 3. Validate each mapping

	// For now, create a basic compatibility check
	mappings := []ValidatorFieldMapping{
		{
			SourceFieldName: "struct_" + sourceType.Name(),
			TargetFieldName: "struct_" + targetType.Name(),
			SourceType:      sourceType,
			TargetType:      targetType,
			MappingStrategy: "direct",
		},
	}

	return fmv.ValidateFieldMappings(ctx, mappings)
}

// GetValidationStats returns validation statistics.
func (fmv *FieldMappingValidator) GetValidationStats() ValidationStats {
	totalValidations := fmv.validationCount
	successRate := float64(0)
	avgValidationTime := time.Duration(0)

	if totalValidations > 0 {
		successRate = float64(totalValidations-fmv.validationFailures) / float64(totalValidations) * 100
		avgValidationTime = fmv.validationTime / time.Duration(totalValidations)
	}

	return ValidationStats{
		TotalValidations:      fmv.validationCount,
		ValidationFailures:    fmv.validationFailures,
		SuccessRate:           successRate,
		AverageValidationTime: avgValidationTime,
	}
}

// ValidationStats provides statistics about field mapping validation.
type ValidationStats struct {
	TotalValidations      int64         `json:"total_validations"`
	ValidationFailures    int64         `json:"validation_failures"`
	SuccessRate           float64       `json:"success_rate"`
	AverageValidationTime time.Duration `json:"average_validation_time"`
}

// SetStrictValidation enables or disables strict validation mode.
func (fmv *FieldMappingValidator) SetStrictValidation(strict bool) {
	fmv.strictValidation = strict
	fmv.logger.Debug("strict validation mode updated",
		zap.Bool("strict_validation", strict))
}

// SetAllowUnsafeMappings enables or disables unsafe mappings.
func (fmv *FieldMappingValidator) SetAllowUnsafeMappings(allow bool) {
	fmv.allowUnsafeMappings = allow
	fmv.logger.Debug("unsafe mappings setting updated",
		zap.Bool("allow_unsafe_mappings", allow))
}

// SetRequireAllFields enables or disables the requirement for all fields to be mapped.
func (fmv *FieldMappingValidator) SetRequireAllFields(require bool) {
	fmv.requireAllFields = require
	fmv.logger.Debug("require all fields setting updated",
		zap.Bool("require_all_fields", require))
}
