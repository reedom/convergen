package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Static errors for err113 compliance.
var (
	ErrSubstitutionValidationFailed = errors.New("substitution validation failed")
	ErrInvalidSubstitutionResult    = errors.New("invalid substitution result")
	ErrSubstitutionInconsistency    = errors.New("substitution result inconsistency")
	ErrSubstitutionCorruption       = errors.New("substitution result corruption")
	ErrValidatorNil                 = errors.New("substitution validator cannot be nil")
)

// SubstitutionValidator validates the correctness of type substitution operations.
type SubstitutionValidator struct {
	logger               *zap.Logger
	compatibilityChecker *TypeCompatibilityChecker

	// Configuration
	strictValidation      bool
	checkConsistency      bool
	validateConstraints   bool
	performDeepValidation bool

	// Performance tracking
	validationsPerformed int64
	validationFailures   int64
	totalValidationTime  time.Duration
}

// SubstitutionValidationResult contains the results of substitution validation.
type SubstitutionValidationResult struct {
	Valid                 bool                          `json:"valid"`
	ValidationErrors      []SubstitutionValidationError `json:"validation_errors,omitempty"`
	ConsistencyCheck      *ConsistencyCheckResult       `json:"consistency_check,omitempty"`
	ConstraintValidation  *ConstraintValidationResult   `json:"constraint_validation,omitempty"`
	PerformanceValidation *PerformanceValidationResult  `json:"performance_validation,omitempty"`
	ValidationSummary     SubstitutionValidationSummary `json:"validation_summary"`
	ValidationDurationMS  int64                         `json:"validation_duration_ms"`
}

// SubstitutionValidationError represents a specific validation error.
type SubstitutionValidationError struct {
	ErrorType     string `json:"error_type"`
	ErrorMessage  string `json:"error_message"`
	Severity      string `json:"severity"`
	AffectedType  string `json:"affected_type,omitempty"`
	ExpectedValue string `json:"expected_value,omitempty"`
	ActualValue   string `json:"actual_value,omitempty"`
	SuggestedFix  string `json:"suggested_fix,omitempty"`
}

// ConsistencyCheckResult contains results of consistency validation.
type ConsistencyCheckResult struct {
	TypeMappingConsistent bool     `json:"type_mapping_consistent"`
	SubstitutionPathValid bool     `json:"substitution_path_valid"`
	OriginalTypePreserved bool     `json:"original_type_preserved"`
	InconsistentMappings  []string `json:"inconsistent_mappings,omitempty"`
	InvalidPathSteps      []string `json:"invalid_path_steps,omitempty"`
}

// ConstraintValidationResult contains results of constraint validation.
type ConstraintValidationResult struct {
	AllConstraintsSatisfied bool                      `json:"all_constraints_satisfied"`
	ConstraintViolations    []ConstraintViolation     `json:"constraint_violations,omitempty"`
	TypeParameterValidation []TypeParameterValidation `json:"type_parameter_validation,omitempty"`
}

// TypeParameterValidation contains validation results for a single type parameter.
type TypeParameterValidation struct {
	ParameterName     string `json:"parameter_name"`
	OriginalType      string `json:"original_type"`
	SubstitutedType   string `json:"substituted_type"`
	ConstraintType    string `json:"constraint_type"`
	Valid             bool   `json:"valid"`
	ValidationMessage string `json:"validation_message,omitempty"`
}

// PerformanceValidationResult contains performance-related validation.
type PerformanceValidationResult struct {
	WithinPerformanceLimits   bool    `json:"within_performance_limits"`
	RecursionDepthAcceptable  bool    `json:"recursion_depth_acceptable"`
	CacheUtilizationGood      bool    `json:"cache_utilization_good"`
	MemoryUsageReasonable     bool    `json:"memory_usage_reasonable"`
	MaxRecursionDepth         int     `json:"max_recursion_depth"`
	CacheHitRate              float64 `json:"cache_hit_rate"`
	EstimatedMemoryUsageBytes int64   `json:"estimated_memory_usage_bytes"`
}

// SubstitutionValidationSummary provides a summary of validation results.
type SubstitutionValidationSummary struct {
	TotalChecksPerformed int `json:"total_checks_performed"`
	SuccessfulChecks     int `json:"successful_checks"`
	FailedChecks         int `json:"failed_checks"`
	WarningsGenerated    int `json:"warnings_generated"`
	CriticalErrors       int `json:"critical_errors"`
}

// NewSubstitutionValidator creates a new substitution validator.
func NewSubstitutionValidator(logger *zap.Logger) *SubstitutionValidator {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &SubstitutionValidator{
		logger:                logger,
		strictValidation:      true,
		checkConsistency:      true,
		validateConstraints:   true,
		performDeepValidation: true,
	}
}

// SetCompatibilityChecker sets the compatibility checker for constraint validation.
func (sv *SubstitutionValidator) SetCompatibilityChecker(checker *TypeCompatibilityChecker) {
	sv.compatibilityChecker = checker
	sv.logger.Debug("compatibility checker set for substitution validator")
}

// ValidateSubstitutionResult validates the correctness of a substitution result.
func (sv *SubstitutionValidator) ValidateSubstitutionResult(
	ctx context.Context,
	result *SubstitutionResult,
) (*SubstitutionValidationResult, error) {
	startTime := time.Now()
	sv.validationsPerformed++

	sv.logger.Debug("validating substitution result",
		zap.String("original_type", result.OriginalType.String()),
		zap.String("substituted_type", result.SubstitutedType.String()),
		zap.Int("type_mappings", len(result.TypeMapping)))

	validationResult := &SubstitutionValidationResult{
		Valid:             true,
		ValidationErrors:  make([]SubstitutionValidationError, 0),
		ValidationSummary: SubstitutionValidationSummary{},
	}

	// Basic validation
	if err := sv.performBasicValidation(result, validationResult); err != nil {
		return nil, fmt.Errorf("basic validation failed: %w", err)
	}

	// Consistency check
	if sv.checkConsistency {
		consistencyResult := sv.performConsistencyCheck(result)
		validationResult.ConsistencyCheck = consistencyResult

		if !consistencyResult.TypeMappingConsistent ||
			!consistencyResult.SubstitutionPathValid ||
			!consistencyResult.OriginalTypePreserved {
			validationResult.Valid = false
			sv.addValidationError(validationResult, "consistency", "Substitution result inconsistency detected", "error")
		}
	}

	// Constraint validation
	if sv.validateConstraints {
		constraintResult := sv.performConstraintValidation(ctx, result)
		validationResult.ConstraintValidation = constraintResult

		if !constraintResult.AllConstraintsSatisfied {
			validationResult.Valid = false
			sv.addValidationError(validationResult, "constraint", "Constraint validation failed", "error")
		}
	}

	// Performance validation
	if sv.performDeepValidation {
		performanceResult := sv.performPerformanceValidation(result)
		validationResult.PerformanceValidation = performanceResult

		if !performanceResult.WithinPerformanceLimits {
			sv.addValidationError(validationResult, "performance", "Performance limits exceeded", "warning")
		}
	}

	// Update summary
	sv.updateValidationSummary(validationResult)

	// Set timing
	validationResult.ValidationDurationMS = time.Since(startTime).Milliseconds()
	sv.totalValidationTime += time.Since(startTime)

	if !validationResult.Valid {
		sv.validationFailures++
	}

	sv.logger.Debug("substitution validation completed",
		zap.Bool("valid", validationResult.Valid),
		zap.Int("validation_errors", len(validationResult.ValidationErrors)),
		zap.Int64("duration_ms", validationResult.ValidationDurationMS))

	return validationResult, nil
}

// performBasicValidation performs basic validation checks.
func (sv *SubstitutionValidator) performBasicValidation(
	result *SubstitutionResult,
	validationResult *SubstitutionValidationResult,
) error {
	// Check that result is not nil
	if result == nil {
		return fmt.Errorf("%w: substitution result is nil", ErrInvalidSubstitutionResult)
	}

	// Check that original type is not nil
	if result.OriginalType == nil {
		sv.addValidationError(validationResult, "basic", "Original type is nil", "error")
		validationResult.Valid = false
		return nil
	}

	// Check that substituted type is not nil
	if result.SubstitutedType == nil {
		sv.addValidationError(validationResult, "basic", "Substituted type is nil", "error")
		validationResult.Valid = false
		return nil
	}

	// Check type mapping is not nil
	if result.TypeMapping == nil {
		sv.addValidationError(validationResult, "basic", "Type mapping is nil", "error")
		validationResult.Valid = false
		return nil
	}

	// Check for nil values in type mapping
	for paramName, typeArg := range result.TypeMapping {
		if paramName == "" {
			sv.addValidationError(validationResult, "basic", "Empty type parameter name in mapping", "error")
			validationResult.Valid = false
		}
		if typeArg == nil {
			sv.addValidationError(validationResult, "basic",
				fmt.Sprintf("Nil type argument for parameter %s", paramName), "error")
			validationResult.Valid = false
		}
	}

	return nil
}

// performConsistencyCheck validates the consistency of substitution results.
func (sv *SubstitutionValidator) performConsistencyCheck(result *SubstitutionResult) *ConsistencyCheckResult {
	consistency := &ConsistencyCheckResult{
		TypeMappingConsistent: true,
		SubstitutionPathValid: true,
		OriginalTypePreserved: true,
		InconsistentMappings:  make([]string, 0),
		InvalidPathSteps:      make([]string, 0),
	}

	// Check type mapping consistency
	if result.OriginalType.Generic() {
		originalParams := result.OriginalType.TypeParams()

		for _, param := range originalParams {
			if mappedType, exists := result.TypeMapping[param.Name]; exists {
				// Verify the mapped type is valid
				if mappedType == nil {
					consistency.TypeMappingConsistent = false
					consistency.InconsistentMappings = append(consistency.InconsistentMappings,
						fmt.Sprintf("Parameter %s mapped to nil type", param.Name))
				}
			}
		}
	}

	// Validate substitution path
	for i, step := range result.SubstitutionPath {
		if step == "" {
			consistency.SubstitutionPathValid = false
			consistency.InvalidPathSteps = append(consistency.InvalidPathSteps,
				fmt.Sprintf("Empty step at index %d", i))
		}
	}

	// Check that original type information is preserved
	if result.OriginalType.String() == "" {
		consistency.OriginalTypePreserved = false
	}

	return consistency
}

// performConstraintValidation validates that substitutions satisfy type constraints.
func (sv *SubstitutionValidator) performConstraintValidation(
	_ctx context.Context,
	result *SubstitutionResult,
) *ConstraintValidationResult {
	constraintResult := &ConstraintValidationResult{
		AllConstraintsSatisfied: true,
		ConstraintViolations:    make([]ConstraintViolation, 0),
		TypeParameterValidation: make([]TypeParameterValidation, 0),
	}

	// Only validate constraints for generic types
	if !result.OriginalType.Generic() {
		return constraintResult
	}

	originalParams := result.OriginalType.TypeParams()

	for _, param := range originalParams {
		validation := TypeParameterValidation{
			ParameterName:  param.Name,
			OriginalType:   result.OriginalType.String(),
			ConstraintType: param.GetConstraintType(),
			Valid:          true,
		}

		// Get the substituted type for this parameter
		if substitutedType, exists := result.TypeMapping[param.Name]; exists {
			validation.SubstitutedType = substitutedType.String()

			// Check if substituted type satisfies the constraint
			if !param.SatisfiesConstraint(substitutedType) {
				validation.Valid = false
				validation.ValidationMessage = fmt.Sprintf(
					"Type %s does not satisfy constraint %s",
					substitutedType.String(), param.GetConstraintType())

				constraintResult.AllConstraintsSatisfied = false

				// Add constraint violation
				violation := ConstraintViolation{
					TypeParamName:      param.Name,
					ExpectedConstraint: param.GetConstraintType(),
					ActualType:         substitutedType.String(),
					ViolationMessage:   validation.ValidationMessage,
				}
				constraintResult.ConstraintViolations = append(constraintResult.ConstraintViolations, violation)
			} else {
				validation.ValidationMessage = "Constraint satisfied"
			}
		} else {
			validation.Valid = false
			validation.ValidationMessage = "No substituted type found for parameter"
			constraintResult.AllConstraintsSatisfied = false
		}

		constraintResult.TypeParameterValidation = append(constraintResult.TypeParameterValidation, validation)
	}

	return constraintResult
}

// performPerformanceValidation validates performance characteristics of the substitution.
func (sv *SubstitutionValidator) performPerformanceValidation(result *SubstitutionResult) *PerformanceValidationResult {
	performance := &PerformanceValidationResult{
		WithinPerformanceLimits:  true,
		RecursionDepthAcceptable: true,
		CacheUtilizationGood:     true,
		MemoryUsageReasonable:    true,
		MaxRecursionDepth:        result.RecursionDepth,
	}

	// Check recursion depth
	if result.RecursionDepth > DefaultMaxRecursionDepth {
		performance.RecursionDepthAcceptable = false
		performance.WithinPerformanceLimits = false
	}

	// Calculate cache hit rate
	if result.PerformanceStats != nil {
		totalCacheOps := result.PerformanceStats.TotalCacheHits + result.PerformanceStats.TotalCacheMisses
		if totalCacheOps > 0 {
			performance.CacheHitRate = float64(result.PerformanceStats.TotalCacheHits) / float64(totalCacheOps) * 100

			// Consider cache utilization poor if hit rate is below 30%
			if performance.CacheHitRate < 30.0 {
				performance.CacheUtilizationGood = false
			}
		}

		performance.EstimatedMemoryUsageBytes = result.PerformanceStats.MemoryUsageBytes

		// Consider memory usage unreasonable if over 100MB for a single substitution
		if performance.EstimatedMemoryUsageBytes > 100*1024*1024 {
			performance.MemoryUsageReasonable = false
			performance.WithinPerformanceLimits = false
		}
	}

	return performance
}

// addValidationError adds a validation error to the result.
func (sv *SubstitutionValidator) addValidationError(
	validationResult *SubstitutionValidationResult,
	errorType, message, severity string,
) {
	error := SubstitutionValidationError{
		ErrorType:    errorType,
		ErrorMessage: message,
		Severity:     severity,
	}

	validationResult.ValidationErrors = append(validationResult.ValidationErrors, error)
}

// updateValidationSummary updates the validation summary based on results.
func (sv *SubstitutionValidator) updateValidationSummary(validationResult *SubstitutionValidationResult) {
	summary := &validationResult.ValidationSummary

	// Count different types of checks
	summary.TotalChecksPerformed = 1 // Basic validation always performed

	if validationResult.ConsistencyCheck != nil {
		summary.TotalChecksPerformed++
		if validationResult.ConsistencyCheck.TypeMappingConsistent &&
			validationResult.ConsistencyCheck.SubstitutionPathValid &&
			validationResult.ConsistencyCheck.OriginalTypePreserved {
			summary.SuccessfulChecks++
		} else {
			summary.FailedChecks++
		}
	}

	if validationResult.ConstraintValidation != nil {
		summary.TotalChecksPerformed++
		if validationResult.ConstraintValidation.AllConstraintsSatisfied {
			summary.SuccessfulChecks++
		} else {
			summary.FailedChecks++
		}
	}

	if validationResult.PerformanceValidation != nil {
		summary.TotalChecksPerformed++
		if validationResult.PerformanceValidation.WithinPerformanceLimits {
			summary.SuccessfulChecks++
		} else {
			summary.FailedChecks++
		}
	}

	// Count errors by severity
	for _, err := range validationResult.ValidationErrors {
		switch strings.ToLower(err.Severity) {
		case "error":
			summary.CriticalErrors++
		case "warning":
			summary.WarningsGenerated++
		}
	}

	// If we haven't counted successful checks for basic validation
	if summary.SuccessfulChecks+summary.FailedChecks < summary.TotalChecksPerformed {
		if validationResult.Valid {
			summary.SuccessfulChecks++
		} else {
			summary.FailedChecks++
		}
	}
}

// ValidateSubstitutionCorrectness performs a comprehensive correctness check.
func (sv *SubstitutionValidator) ValidateSubstitutionCorrectness(
	ctx context.Context,
	originalType Type,
	typeParams []TypeParam,
	typeArgs []Type,
	result *SubstitutionResult,
) (*SubstitutionValidationResult, error) {
	sv.logger.Debug("validating substitution correctness",
		zap.String("original_type", originalType.String()),
		zap.Int("type_params", len(typeParams)),
		zap.Int("type_args", len(typeArgs)))

	// First validate the substitution result itself
	validationResult, err := sv.ValidateSubstitutionResult(ctx, result)
	if err != nil {
		return nil, fmt.Errorf("substitution result validation failed: %w", err)
	}

	// Additional correctness checks
	if len(typeParams) != len(typeArgs) {
		sv.addValidationError(validationResult, "correctness",
			fmt.Sprintf("Type parameter count (%d) does not match type argument count (%d)",
				len(typeParams), len(typeArgs)), "error")
		validationResult.Valid = false
	}

	// Check that the result mapping matches input parameters and arguments
	for i, param := range typeParams {
		if i < len(typeArgs) {
			expectedArg := typeArgs[i]
			if actualArg, exists := result.TypeMapping[param.Name]; exists {
				if actualArg.String() != expectedArg.String() {
					sv.addValidationError(validationResult, "correctness",
						fmt.Sprintf("Type mapping inconsistency for parameter %s: expected %s, got %s",
							param.Name, expectedArg.String(), actualArg.String()), "error")
					validationResult.Valid = false
				}
			} else {
				sv.addValidationError(validationResult, "correctness",
					fmt.Sprintf("Missing type mapping for parameter %s", param.Name), "error")
				validationResult.Valid = false
			}
		}
	}

	return validationResult, nil
}

// GetValidationStats returns statistics about validation operations.
func (sv *SubstitutionValidator) GetValidationStats() ValidationStats {
	totalValidations := sv.validationsPerformed
	successRate := float64(0)
	avgValidationTime := time.Duration(0)

	if totalValidations > 0 {
		successRate = float64(totalValidations-sv.validationFailures) / float64(totalValidations) * 100
		avgValidationTime = sv.totalValidationTime / time.Duration(totalValidations)
	}

	return ValidationStats{
		TotalValidations:      totalValidations,
		ValidationFailures:    sv.validationFailures,
		SuccessRate:           successRate,
		AverageValidationTime: avgValidationTime,
	}
}

// SetStrictValidation enables or disables strict validation mode.
func (sv *SubstitutionValidator) SetStrictValidation(strict bool) {
	sv.strictValidation = strict
	sv.logger.Debug("strict validation mode updated",
		zap.Bool("strict_validation", strict))
}

// SetConsistencyChecking enables or disables consistency checking.
func (sv *SubstitutionValidator) SetConsistencyChecking(check bool) {
	sv.checkConsistency = check
	sv.logger.Debug("consistency checking updated",
		zap.Bool("check_consistency", check))
}

// SetConstraintValidation enables or disables constraint validation.
func (sv *SubstitutionValidator) SetConstraintValidation(validate bool) {
	sv.validateConstraints = validate
	sv.logger.Debug("constraint validation updated",
		zap.Bool("validate_constraints", validate))
}

// SetDeepValidation enables or disables deep validation.
func (sv *SubstitutionValidator) SetDeepValidation(deep bool) {
	sv.performDeepValidation = deep
	sv.logger.Debug("deep validation updated",
		zap.Bool("perform_deep_validation", deep))
}
