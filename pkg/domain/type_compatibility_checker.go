package domain

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Static errors for err113 compliance.
var (
	ErrTypesIncompatible       = errors.New("types are incompatible")
	ErrAssignmentNotAllowed    = errors.New("assignment not allowed")
	ErrGenericConstraintFailed = errors.New("generic constraint validation failed")
	ErrCircularTypeReference   = errors.New("circular type reference detected")
	ErrUnsupportedConversion   = errors.New("unsupported type conversion")
	ErrNilTypeInComparison     = errors.New("nil type in compatibility check")
)

// CompatibilityLevel represents the level of compatibility between types.
type CompatibilityLevel int

const (
	// CompatibilityNone indicates types are incompatible.
	CompatibilityNone CompatibilityLevel = iota
	// CompatibilityWithConversion indicates types are compatible with explicit conversion.
	CompatibilityWithConversion
	// CompatibilityDirect indicates types are directly assignable.
	CompatibilityDirect
	// CompatibilityIdentical indicates types are identical.
	CompatibilityIdentical
)

func (c CompatibilityLevel) String() string {
	switch c {
	case CompatibilityNone:
		return "none"
	case CompatibilityWithConversion:
		return "with_conversion"
	case CompatibilityDirect:
		return "direct"
	case CompatibilityIdentical:
		return "identical"
	default:
		return "unknown"
	}
}

// CompatibilityResult represents the result of type compatibility analysis.
type CompatibilityResult struct {
	Level              CompatibilityLevel `json:"level"`
	Compatible         bool               `json:"compatible"`
	RequiresConversion bool               `json:"requires_conversion"`
	ConversionFunction string             `json:"conversion_function,omitempty"`
	ErrorMessage       string             `json:"error_message,omitempty"`
	TypeContext        *TypeContext       `json:"type_context,omitempty"`

	// Generic-specific information
	GenericContext       *GenericContext       `json:"generic_context,omitempty"`
	ConstraintViolations []ConstraintViolation `json:"constraint_violations,omitempty"`

	// Performance tracking
	CheckDurationMS int64 `json:"check_duration_ms"`
}

// TypeContext provides detailed context about the types being compared.
type TypeContext struct {
	SourceType     Type   `json:"source_type"`
	TargetType     Type   `json:"target_type"`
	SourceTypeInfo string `json:"source_type_info"`
	TargetTypeInfo string `json:"target_type_info"`
	ConversionPath string `json:"conversion_path,omitempty"`
}

// GenericContext provides context specific to generic type compatibility.
type GenericContext struct {
	SourceIsGeneric       bool                    `json:"source_is_generic"`
	TargetIsGeneric       bool                    `json:"target_is_generic"`
	TypeParameterMappings map[string]Type         `json:"type_parameter_mappings,omitempty"`
	InstantiatedTypes     map[string]Type         `json:"instantiated_types,omitempty"`
	ConstraintChecks      []ConstraintCheckResult `json:"constraint_checks,omitempty"`
}

// ConstraintCheckResult represents the result of checking a single constraint.
type ConstraintCheckResult struct {
	TypeParameterName string `json:"type_parameter_name"`
	ConstraintType    string `json:"constraint_type"`
	TypeArgument      string `json:"type_argument"`
	Satisfied         bool   `json:"satisfied"`
	Details           string `json:"details,omitempty"`
}

// TypeCompatibilityChecker provides comprehensive type compatibility analysis.
// It handles basic types, generics, constraints, and conversion requirements.
type TypeCompatibilityChecker struct {
	logger           *zap.Logger
	typeInstantiator *TypeInstantiator

	// Configuration
	allowUnsafeConversions bool
	enableDetailedContext  bool
	maxRecursionDepth      int

	// State tracking for circular reference detection
	currentDepth  int
	checkingStack []string // Stack of type signatures being checked
}

// TypeCompatibilityConfig configures the behavior of TypeCompatibilityChecker.
type TypeCompatibilityConfig struct {
	AllowUnsafeConversions bool              `json:"allow_unsafe_conversions"`
	EnableDetailedContext  bool              `json:"enable_detailed_context"`
	MaxRecursionDepth      int               `json:"max_recursion_depth"`
	TypeInstantiator       *TypeInstantiator `json:"-"` // Cannot serialize interfaces
}

// NewTypeCompatibilityConfig creates a default configuration.
func NewTypeCompatibilityConfig() *TypeCompatibilityConfig {
	return &TypeCompatibilityConfig{
		AllowUnsafeConversions: false,
		EnableDetailedContext:  true,
		MaxRecursionDepth:      10,
		TypeInstantiator:       nil, // Must be set explicitly
	}
}

// NewTypeCompatibilityChecker creates a new type compatibility checker.
func NewTypeCompatibilityChecker(logger *zap.Logger) *TypeCompatibilityChecker {
	return NewTypeCompatibilityCheckerWithConfig(logger, NewTypeCompatibilityConfig())
}

// NewTypeCompatibilityCheckerWithConfig creates a new checker with custom configuration.
func NewTypeCompatibilityCheckerWithConfig(
	logger *zap.Logger,
	config *TypeCompatibilityConfig,
) *TypeCompatibilityChecker {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &TypeCompatibilityChecker{
		logger:                 logger,
		typeInstantiator:       config.TypeInstantiator,
		allowUnsafeConversions: config.AllowUnsafeConversions,
		enableDetailedContext:  config.EnableDetailedContext,
		maxRecursionDepth:      config.MaxRecursionDepth,
		checkingStack:          make([]string, 0),
	}
}

// CheckAssignability determines if sourceType can be assigned to targetType.
// This is the main entry point for type compatibility checking.
func (tcc *TypeCompatibilityChecker) CheckAssignability(
	sourceType, targetType Type,
) (*CompatibilityResult, error) {
	return tcc.CheckAssignabilityWithContext(context.Background(), sourceType, targetType)
}

// CheckAssignabilityWithContext performs compatibility checking with context support.
func (tcc *TypeCompatibilityChecker) CheckAssignabilityWithContext(
	ctx context.Context,
	sourceType, targetType Type,
) (*CompatibilityResult, error) {
	startTime := time.Now()

	// Validate input types
	if sourceType == nil {
		return &CompatibilityResult{
			Level:           CompatibilityNone,
			Compatible:      false,
			ErrorMessage:    "source type is nil",
			CheckDurationMS: time.Since(startTime).Milliseconds(),
		}, fmt.Errorf("%w: source type is nil", ErrNilTypeInComparison)
	}

	if targetType == nil {
		return &CompatibilityResult{
			Level:           CompatibilityNone,
			Compatible:      false,
			ErrorMessage:    "target type is nil",
			CheckDurationMS: time.Since(startTime).Milliseconds(),
		}, fmt.Errorf("%w: target type is nil", ErrNilTypeInComparison)
	}

	tcc.logger.Debug("checking type assignability",
		zap.String("source_type", sourceType.String()),
		zap.String("target_type", targetType.String()),
		zap.String("source_kind", sourceType.Kind().String()),
		zap.String("target_kind", targetType.Kind().String()))

	// Check for circular references
	if err := tcc.checkCircularReference(sourceType, targetType); err != nil {
		return &CompatibilityResult{
			Level:           CompatibilityNone,
			Compatible:      false,
			ErrorMessage:    err.Error(),
			CheckDurationMS: time.Since(startTime).Milliseconds(),
		}, err
	}

	// Perform the compatibility analysis
	result, err := tcc.performCompatibilityCheck(ctx, sourceType, targetType)
	if err != nil {
		return nil, err
	}

	// Set timing information
	result.CheckDurationMS = time.Since(startTime).Milliseconds()

	// Add detailed context if enabled
	if tcc.enableDetailedContext {
		result.TypeContext = &TypeContext{
			SourceType:     sourceType,
			TargetType:     targetType,
			SourceTypeInfo: tcc.getTypeInfo(sourceType),
			TargetTypeInfo: tcc.getTypeInfo(targetType),
		}
	}

	tcc.logger.Debug("compatibility check completed",
		zap.String("source_type", sourceType.String()),
		zap.String("target_type", targetType.String()),
		zap.String("compatibility_level", result.Level.String()),
		zap.Bool("compatible", result.Compatible),
		zap.Int64("duration_ms", result.CheckDurationMS))

	return result, nil
}

// performCompatibilityCheck performs the actual compatibility analysis.
func (tcc *TypeCompatibilityChecker) performCompatibilityCheck(
	ctx context.Context,
	sourceType, targetType Type,
) (*CompatibilityResult, error) {
	// 1. Check for identical types
	if tcc.areIdenticalTypes(sourceType, targetType) {
		return &CompatibilityResult{
			Level:      CompatibilityIdentical,
			Compatible: true,
		}, nil
	}

	// 2. Check for direct assignability (Go language rules)
	if sourceType.AssignableTo(targetType) {
		return &CompatibilityResult{
			Level:      CompatibilityDirect,
			Compatible: true,
		}, nil
	}

	// 3. Handle generic types specially
	if sourceType.Generic() || targetType.Generic() {
		return tcc.checkGenericCompatibility(ctx, sourceType, targetType)
	}

	// 4. Check for conversion-based compatibility
	conversionResult := tcc.checkConversionCompatibility(ctx, sourceType, targetType)
	if conversionResult.Compatible {
		return conversionResult, nil
	}

	// 5. Check for interface implementations
	if targetType.Kind() == KindInterface {
		if sourceType.Implements(targetType) {
			return &CompatibilityResult{
				Level:      CompatibilityDirect,
				Compatible: true,
			}, nil
		}
	}

	// 6. Types are incompatible
	return &CompatibilityResult{
		Level:        CompatibilityNone,
		Compatible:   false,
		ErrorMessage: fmt.Sprintf("cannot assign %s to %s", sourceType.String(), targetType.String()),
	}, nil
}

// areIdenticalTypes checks if two types are identical.
func (tcc *TypeCompatibilityChecker) areIdenticalTypes(sourceType, targetType Type) bool {
	// Basic identity check
	if sourceType.String() == targetType.String() {
		return true
	}

	// Kind and name comparison
	if sourceType.Kind() != targetType.Kind() {
		return false
	}

	if sourceType.Name() != targetType.Name() {
		return false
	}

	// Package comparison
	if sourceType.Package() != targetType.Package() {
		return false
	}

	return true
}

// checkGenericCompatibility handles compatibility checking for generic types.
func (tcc *TypeCompatibilityChecker) checkGenericCompatibility(
	ctx context.Context,
	sourceType, targetType Type,
) (*CompatibilityResult, error) {
	tcc.logger.Debug("checking generic type compatibility",
		zap.String("source_type", sourceType.String()),
		zap.String("target_type", targetType.String()),
		zap.Bool("source_generic", sourceType.Generic()),
		zap.Bool("target_generic", targetType.Generic()))

	result := &CompatibilityResult{
		GenericContext: &GenericContext{
			SourceIsGeneric: sourceType.Generic(),
			TargetIsGeneric: targetType.Generic(),
		},
	}

	// Case 1: Both types are generic
	if sourceType.Generic() && targetType.Generic() {
		return tcc.checkBothGenericCompatibility(ctx, sourceType, targetType, result)
	}

	// Case 2: Source is generic, target is concrete
	if sourceType.Generic() && !targetType.Generic() {
		return tcc.checkGenericToConcreteCompatibility(ctx, sourceType, targetType, result)
	}

	// Case 3: Source is concrete, target is generic
	if !sourceType.Generic() && targetType.Generic() {
		return tcc.checkConcreteToGenericCompatibility(ctx, sourceType, targetType, result)
	}

	// This should not be reached, but handle as incompatible
	result.Level = CompatibilityNone
	result.Compatible = false
	result.ErrorMessage = "unexpected generic compatibility case"
	return result, nil
}

// checkBothGenericCompatibility handles the case where both types are generic.
func (tcc *TypeCompatibilityChecker) checkBothGenericCompatibility(
	_ctx context.Context,
	sourceType, targetType Type,
	result *CompatibilityResult,
) (*CompatibilityResult, error) {
	// Check if the generic types have the same structure
	sourceParams := sourceType.TypeParams()
	targetParams := targetType.TypeParams()

	if len(sourceParams) != len(targetParams) {
		result.Level = CompatibilityNone
		result.Compatible = false
		result.ErrorMessage = fmt.Sprintf("type parameter count mismatch: %d vs %d", len(sourceParams), len(targetParams))
		return result, nil
	}

	// Check constraint compatibility
	constraintChecks := make([]ConstraintCheckResult, 0, len(sourceParams))
	allConstraintsSatisfied := true

	for i, sourceParam := range sourceParams {
		targetParam := targetParams[i]

		checkResult := ConstraintCheckResult{
			TypeParameterName: sourceParam.Name,
			ConstraintType:    sourceParam.GetConstraintType(),
			TypeArgument:      targetParam.GetConstraintType(),
		}

		// Check if constraints are compatible
		if tcc.areConstraintsCompatible(sourceParam, targetParam) {
			checkResult.Satisfied = true
			checkResult.Details = "constraints are compatible"
		} else {
			checkResult.Satisfied = false
			checkResult.Details = fmt.Sprintf("constraint mismatch: %s vs %s",
				sourceParam.GetConstraintType(), targetParam.GetConstraintType())
			allConstraintsSatisfied = false
		}

		constraintChecks = append(constraintChecks, checkResult)
	}

	result.GenericContext.ConstraintChecks = constraintChecks

	if allConstraintsSatisfied {
		result.Level = CompatibilityDirect
		result.Compatible = true
	} else {
		result.Level = CompatibilityNone
		result.Compatible = false
		result.ErrorMessage = "generic constraint compatibility failed"
	}

	return result, nil
}

// checkGenericToConcreteCompatibility handles generic source to concrete target.
func (tcc *TypeCompatibilityChecker) checkGenericToConcreteCompatibility(
	_ctx context.Context,
	sourceType, targetType Type,
	result *CompatibilityResult,
) (*CompatibilityResult, error) {
	// For generic to concrete, we need to check if the concrete type
	// can satisfy all the generic constraints
	sourceParams := sourceType.TypeParams()

	constraintChecks := make([]ConstraintCheckResult, 0, len(sourceParams))
	allConstraintsSatisfied := true

	for _, param := range sourceParams {
		checkResult := ConstraintCheckResult{
			TypeParameterName: param.Name,
			ConstraintType:    param.GetConstraintType(),
			TypeArgument:      targetType.String(),
		}

		if param.SatisfiesConstraint(targetType) {
			checkResult.Satisfied = true
			checkResult.Details = "concrete type satisfies constraint"
		} else {
			checkResult.Satisfied = false
			checkResult.Details = fmt.Sprintf("concrete type %s does not satisfy constraint %s",
				targetType.String(), param.GetConstraintType())
			allConstraintsSatisfied = false
		}

		constraintChecks = append(constraintChecks, checkResult)
	}

	result.GenericContext.ConstraintChecks = constraintChecks

	if allConstraintsSatisfied {
		result.Level = CompatibilityWithConversion
		result.Compatible = true
		result.RequiresConversion = true
		result.ConversionFunction = "type instantiation"
	} else {
		result.Level = CompatibilityNone
		result.Compatible = false
		result.ErrorMessage = "concrete type does not satisfy generic constraints"

		// Add constraint violations for detailed error reporting
		violations := make([]ConstraintViolation, 0)
		for _, check := range constraintChecks {
			if !check.Satisfied {
				violations = append(violations, ConstraintViolation{
					TypeParamName:      check.TypeParameterName,
					ExpectedConstraint: check.ConstraintType,
					ActualType:         check.TypeArgument,
					ViolationMessage:   check.Details,
				})
			}
		}
		result.ConstraintViolations = violations
	}

	return result, nil
}

// checkConcreteToGenericCompatibility handles concrete source to generic target.
func (tcc *TypeCompatibilityChecker) checkConcreteToGenericCompatibility(
	_ctx context.Context,
	sourceType, targetType Type,
	result *CompatibilityResult,
) (*CompatibilityResult, error) {
	// This case is less common but can occur in some scenarios
	// Generally, a concrete type can be assigned to a generic type parameter
	// if it satisfies all constraints

	targetParams := targetType.TypeParams()

	constraintChecks := make([]ConstraintCheckResult, 0, len(targetParams))
	allConstraintsSatisfied := true

	for _, param := range targetParams {
		checkResult := ConstraintCheckResult{
			TypeParameterName: param.Name,
			ConstraintType:    param.GetConstraintType(),
			TypeArgument:      sourceType.String(),
		}

		if param.SatisfiesConstraint(sourceType) {
			checkResult.Satisfied = true
			checkResult.Details = "concrete type satisfies constraint"
		} else {
			checkResult.Satisfied = false
			checkResult.Details = fmt.Sprintf("concrete type %s does not satisfy constraint %s",
				sourceType.String(), param.GetConstraintType())
			allConstraintsSatisfied = false
		}

		constraintChecks = append(constraintChecks, checkResult)
	}

	result.GenericContext.ConstraintChecks = constraintChecks

	if allConstraintsSatisfied {
		result.Level = CompatibilityWithConversion
		result.Compatible = true
		result.RequiresConversion = true
		result.ConversionFunction = "generic instantiation"
	} else {
		result.Level = CompatibilityNone
		result.Compatible = false
		result.ErrorMessage = "concrete type does not satisfy all generic constraints"
	}

	return result, nil
}

// areConstraintsCompatible checks if two type parameter constraints are compatible.
func (tcc *TypeCompatibilityChecker) areConstraintsCompatible(param1, param2 TypeParam) bool {
	// Same constraint type is always compatible
	constraint1 := param1.GetConstraintType()
	constraint2 := param2.GetConstraintType()

	if constraint1 == constraint2 {
		return true
	}

	// Special compatibility rules
	switch constraint1 {
	case "any":
		// 'any' is compatible with everything
		return true
	case comparableConstraint:
		// 'comparable' is compatible with basic types and specific interfaces
		return constraint2 == comparableConstraint || tcc.isBasicComparableType(constraint2)
	default:
		// For custom constraints, they need to be identical for now
		// In a more sophisticated implementation, we could check constraint relationships
		return constraint1 == constraint2
	}
}

// isBasicComparableType checks if a constraint represents a basic comparable type.
func (tcc *TypeCompatibilityChecker) isBasicComparableType(constraint string) bool {
	comparableTypes := []string{
		"bool", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "string",
	}

	for _, comparableType := range comparableTypes {
		if constraint == comparableType {
			return true
		}
	}

	return false
}

// checkConversionCompatibility checks if types are compatible through conversion.
func (tcc *TypeCompatibilityChecker) checkConversionCompatibility(
	_ctx context.Context,
	sourceType, targetType Type,
) *CompatibilityResult {
	result := &CompatibilityResult{
		Level:      CompatibilityNone,
		Compatible: false,
	}

	// Check for numeric conversions
	if tcc.areNumericTypes(sourceType, targetType) {
		result.Level = CompatibilityWithConversion
		result.Compatible = true
		result.RequiresConversion = true
		result.ConversionFunction = tcc.getNumericConversionFunction(sourceType, targetType)
		return result
	}

	// Check for string conversions
	if tcc.areStringCompatibleTypes(sourceType, targetType) {
		result.Level = CompatibilityWithConversion
		result.Compatible = true
		result.RequiresConversion = true
		result.ConversionFunction = tcc.getStringConversionFunction(sourceType, targetType)
		return result
	}

	// Check for pointer/value conversions
	if tcc.arePointerCompatibleTypes(sourceType, targetType) {
		result.Level = CompatibilityWithConversion
		result.Compatible = true
		result.RequiresConversion = true
		result.ConversionFunction = tcc.getPointerConversionFunction(sourceType, targetType)
		return result
	}

	// No conversion available
	result.ErrorMessage = fmt.Sprintf("no conversion available from %s to %s",
		sourceType.String(), targetType.String())
	return result
}

// areNumericTypes checks if both types are numeric and can be converted.
func (tcc *TypeCompatibilityChecker) areNumericTypes(sourceType, targetType Type) bool {
	numericKinds := []reflect.Kind{
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
	}

	sourceNumeric := false
	targetNumeric := false

	// This is simplified - in a real implementation, you'd check the actual reflect.Kind
	// For now, check by name
	sourceName := strings.ToLower(sourceType.Name())
	targetName := strings.ToLower(targetType.Name())

	for _, kind := range numericKinds {
		kindName := strings.ToLower(kind.String())
		if sourceName == kindName {
			sourceNumeric = true
		}
		if targetName == kindName {
			targetNumeric = true
		}
	}

	return sourceNumeric && targetNumeric
}

// areStringCompatibleTypes checks if types are string-compatible.
func (tcc *TypeCompatibilityChecker) areStringCompatibleTypes(sourceType, targetType Type) bool {
	sourceName := strings.ToLower(sourceType.Name())
	targetName := strings.ToLower(targetType.Name())

	stringTypes := []string{"string", "[]byte", "[]rune"}

	sourceIsString := false
	targetIsString := false

	for _, strType := range stringTypes {
		if sourceName == strType {
			sourceIsString = true
		}
		if targetName == strType {
			targetIsString = true
		}
	}

	return sourceIsString && targetIsString
}

// arePointerCompatibleTypes checks if types are pointer-compatible.
func (tcc *TypeCompatibilityChecker) arePointerCompatibleTypes(sourceType, targetType Type) bool {
	// Simplified pointer compatibility check
	sourceIsPtr := sourceType.Kind() == KindPointer || strings.HasPrefix(sourceType.String(), "*")
	targetIsPtr := targetType.Kind() == KindPointer || strings.HasPrefix(targetType.String(), "*")

	// Allow pointer to value and value to pointer conversions in some cases
	if sourceIsPtr != targetIsPtr {
		// Get underlying types
		sourceUnderlying := sourceType.Underlying()
		targetUnderlying := targetType.Underlying()

		if sourceUnderlying != nil && targetUnderlying != nil {
			return sourceUnderlying.String() == targetUnderlying.String()
		}
	}

	return false
}

// getNumericConversionFunction returns the conversion function name for numeric types.
func (tcc *TypeCompatibilityChecker) getNumericConversionFunction(_sourceType, targetType Type) string {
	return fmt.Sprintf("%s(%s)", targetType.Name(), "value")
}

// getStringConversionFunction returns the conversion function name for string types.
func (tcc *TypeCompatibilityChecker) getStringConversionFunction(sourceType, targetType Type) string {
	sourceName := sourceType.Name()
	targetName := targetType.Name()

	if sourceName == stringTypeName && targetName == "[]byte" {
		return "[]byte(value)"
	}
	if sourceName == "[]byte" && targetName == stringTypeName {
		return "string(value)"
	}
	if sourceName == stringTypeName && targetName == "[]rune" {
		return "[]rune(value)"
	}
	if sourceName == "[]rune" && targetName == stringTypeName {
		return "string(value)"
	}

	return fmt.Sprintf("%s(value)", targetName)
}

// getPointerConversionFunction returns the conversion function name for pointer types.
func (tcc *TypeCompatibilityChecker) getPointerConversionFunction(sourceType, targetType Type) string {
	sourceIsPtr := strings.HasPrefix(sourceType.String(), "*")
	targetIsPtr := strings.HasPrefix(targetType.String(), "*")

	if sourceIsPtr && !targetIsPtr {
		return "*value"
	}
	if !sourceIsPtr && targetIsPtr {
		return "&value"
	}

	return "value"
}

// checkCircularReference checks for circular references to prevent infinite recursion.
func (tcc *TypeCompatibilityChecker) checkCircularReference(sourceType, targetType Type) error {
	if tcc.maxRecursionDepth <= tcc.currentDepth {
		return fmt.Errorf("%w: maximum recursion depth %d exceeded",
			ErrCircularTypeReference, tcc.maxRecursionDepth)
	}

	// Create signature for this comparison
	signature := fmt.Sprintf("%s->%s", sourceType.String(), targetType.String())

	// Check if we're already checking this signature
	for _, existing := range tcc.checkingStack {
		if existing == signature {
			return fmt.Errorf("%w: circular reference detected for %s",
				ErrCircularTypeReference, signature)
		}
	}

	// Add to stack and increment depth
	tcc.checkingStack = append(tcc.checkingStack, signature)
	tcc.currentDepth++

	return nil
}

// getTypeInfo returns detailed information about a type.
func (tcc *TypeCompatibilityChecker) getTypeInfo(t Type) string {
	var info strings.Builder

	info.WriteString(fmt.Sprintf("Kind: %s", t.Kind().String()))

	if t.Generic() {
		info.WriteString(", Generic: true")
		params := t.TypeParams()
		if 0 < len(params) {
			info.WriteString(fmt.Sprintf(", TypeParams: %d", len(params)))
		}
	}

	if t.Package() != "" {
		info.WriteString(fmt.Sprintf(", Package: %s", t.Package()))
	}

	if underlying := t.Underlying(); underlying != nil && underlying != t {
		info.WriteString(fmt.Sprintf(", Underlying: %s", underlying.String()))
	}

	return info.String()
}

// SetTypeInstantiator sets the type instantiator for advanced generic operations.
func (tcc *TypeCompatibilityChecker) SetTypeInstantiator(instantiator *TypeInstantiator) {
	tcc.typeInstantiator = instantiator
	tcc.logger.Debug("type instantiator configured for compatibility checker")
}

// GetTypeInstantiator returns the configured type instantiator.
func (tcc *TypeCompatibilityChecker) GetTypeInstantiator() *TypeInstantiator {
	return tcc.typeInstantiator
}

// HasTypeInstantiator returns true if a type instantiator is configured.
func (tcc *TypeCompatibilityChecker) HasTypeInstantiator() bool {
	return tcc.typeInstantiator != nil
}

// ValidateFieldMapping validates that a field mapping is type-safe.
// This is specifically for validating field mappings in code generation.
func (tcc *TypeCompatibilityChecker) ValidateFieldMapping(
	ctx context.Context,
	sourceFieldType, targetFieldType Type,
	fieldName string,
) (*CompatibilityResult, error) {
	tcc.logger.Debug("validating field mapping",
		zap.String("field_name", fieldName),
		zap.String("source_field_type", sourceFieldType.String()),
		zap.String("target_field_type", targetFieldType.String()))

	result, err := tcc.CheckAssignabilityWithContext(ctx, sourceFieldType, targetFieldType)
	if err != nil {
		return nil, fmt.Errorf("field mapping validation failed for %s: %w", fieldName, err)
	}

	// Add field-specific context
	if result.TypeContext != nil {
		result.TypeContext.ConversionPath = fmt.Sprintf("field %s: %s -> %s",
			fieldName, sourceFieldType.String(), targetFieldType.String())
	}

	// Enhance error message with field context
	if !result.Compatible {
		result.ErrorMessage = fmt.Sprintf("field %s: %s", fieldName, result.ErrorMessage)
	}

	return result, nil
}

// ValidateGenericInstantiation validates that a generic type instantiation is correct.
func (tcc *TypeCompatibilityChecker) ValidateGenericInstantiation(
	ctx context.Context,
	genericType Type,
	typeArguments []Type,
) (*CompatibilityResult, error) {
	if !genericType.Generic() {
		return &CompatibilityResult{
			Level:        CompatibilityNone,
			Compatible:   false,
			ErrorMessage: "type is not generic",
		}, fmt.Errorf("%w: type %s is not generic", ErrInvalidTypeArgument, genericType.String())
	}

	typeParams := genericType.TypeParams()
	if len(typeArguments) != len(typeParams) {
		return &CompatibilityResult{
			Level:      CompatibilityNone,
			Compatible: false,
			ErrorMessage: fmt.Sprintf("type argument count mismatch: expected %d, got %d",
				len(typeParams), len(typeArguments)),
		}, fmt.Errorf("%w: type argument count mismatch", ErrTypeArgumentCountMismatch)
	}

	result := &CompatibilityResult{
		Level:      CompatibilityIdentical,
		Compatible: true,
		GenericContext: &GenericContext{
			SourceIsGeneric:       true,
			TypeParameterMappings: make(map[string]Type),
			ConstraintChecks:      make([]ConstraintCheckResult, 0, len(typeParams)),
		},
	}

	// Validate each type argument against its constraint
	allValid := true
	constraintViolations := make([]ConstraintViolation, 0)

	for i, param := range typeParams {
		typeArg := typeArguments[i]

		checkResult := ConstraintCheckResult{
			TypeParameterName: param.Name,
			ConstraintType:    param.GetConstraintType(),
			TypeArgument:      typeArg.String(),
		}

		if param.SatisfiesConstraint(typeArg) {
			checkResult.Satisfied = true
			checkResult.Details = "constraint satisfied"
			result.GenericContext.TypeParameterMappings[param.Name] = typeArg
		} else {
			checkResult.Satisfied = false
			checkResult.Details = fmt.Sprintf("type %s does not satisfy constraint %s",
				typeArg.String(), param.GetConstraintType())
			allValid = false

			constraintViolations = append(constraintViolations, ConstraintViolation{
				TypeParamName:      param.Name,
				ExpectedConstraint: param.GetConstraintType(),
				ActualType:         typeArg.String(),
				ViolationMessage:   checkResult.Details,
			})
		}

		result.GenericContext.ConstraintChecks = append(result.GenericContext.ConstraintChecks, checkResult)
	}

	if !allValid {
		result.Level = CompatibilityNone
		result.Compatible = false
		result.ErrorMessage = fmt.Sprintf("constraint validation failed: %d violations", len(constraintViolations))
		result.ConstraintViolations = constraintViolations
	}

	return result, nil
}
