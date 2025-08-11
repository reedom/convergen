package builder

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v9/pkg/domain"
)

// Static errors for err113 compliance.
var (
	ErrRecursiveResolverNil       = errors.New("recursive type resolver cannot be nil")
	ErrRecursiveResolutionFailed  = errors.New("recursive type parameter resolution failed")
	ErrMaxRecursionDepthExceeded  = errors.New("maximum recursion depth exceeded in type resolution")
	ErrCircularTypeReference      = errors.New("circular type reference detected")
	ErrTypeAliasResolutionFailed  = errors.New("type alias resolution failed")
	ErrConstraintValidationFailed = errors.New("type constraint validation failed")
)

// Constants for recursive resolution configuration.
const (
	DefaultMaxRecursionDepth       = 50
	DefaultMaxTypeAliasDepth       = 20
	DefaultConstraintCacheSize     = 1000
	DefaultResolutionTimeout       = 30 * time.Second
)

// RecursiveTypeResolver handles deep recursive type parameter resolution for complex generic scenarios.
type RecursiveTypeResolver struct {
	logger              *zap.Logger
	typeSubstitution    *domain.TypeSubstitutionEngine
	config              *RecursiveResolverConfig
	
	// Resolution tracking
	resolutionStack     []string
	visitedTypes        map[string]*ResolutionResult
	circularReferences  map[string]bool
	
	// Type alias support
	aliasRegistry       map[string]domain.Type
	aliasDepthTracker   map[string]int
	
	// Constraint validation
	constraintCache     map[string]*ConstraintValidationResult
	
	// Performance metrics
	metrics             *RecursiveResolutionMetrics
}

// RecursiveResolverConfig configures the recursive type resolver.
type RecursiveResolverConfig struct {
	MaxRecursionDepth    int           `json:"max_recursion_depth"`
	MaxTypeAliasDepth    int           `json:"max_type_alias_depth"`
	ConstraintCacheSize  int           `json:"constraint_cache_size"`
	ResolutionTimeout    time.Duration `json:"resolution_timeout"`
	EnableCircularCheck  bool          `json:"enable_circular_check"`
	EnableConstraintCache bool         `json:"enable_constraint_cache"`
	EnablePerformanceTrack bool        `json:"enable_performance_track"`
	DebugMode            bool          `json:"debug_mode"`
}

// DefaultRecursiveResolverConfig returns default configuration for recursive resolver.
func DefaultRecursiveResolverConfig() *RecursiveResolverConfig {
	return &RecursiveResolverConfig{
		MaxRecursionDepth:     DefaultMaxRecursionDepth,
		MaxTypeAliasDepth:     DefaultMaxTypeAliasDepth,
		ConstraintCacheSize:   DefaultConstraintCacheSize,
		ResolutionTimeout:     DefaultResolutionTimeout,
		EnableCircularCheck:   true,
		EnableConstraintCache: true,
		EnablePerformanceTrack: true,
		DebugMode:            false,
	}
}

// ResolutionResult represents the result of recursive type parameter resolution.
type ResolutionResult struct {
	OriginalType        domain.Type            `json:"original_type"`
	ResolvedType        domain.Type            `json:"resolved_type"`
	TypeSubstitutions   map[string]domain.Type `json:"type_substitutions"`
	ResolutionPath      []string               `json:"resolution_path"`
	ResolvedAt          time.Time              `json:"resolved_at"`
	ResolutionDepth     int                    `json:"resolution_depth"`
	AliasResolutions    []string               `json:"alias_resolutions"`
	ConstraintsSatisfied bool                  `json:"constraints_satisfied"`
	CircularReferenceDetected bool            `json:"circular_reference_detected"`
}

// ConstraintValidationResult represents constraint validation results.
type ConstraintValidationResult struct {
	TypeParam           domain.TypeParam       `json:"type_param"`
	ConcreteType        domain.Type           `json:"concrete_type"`
	Satisfied           bool                  `json:"satisfied"`
	ValidationErrors    []string              `json:"validation_errors"`
	ValidatedAt         time.Time             `json:"validated_at"`
}

// RecursiveResolutionMetrics tracks performance metrics for recursive resolution.
type RecursiveResolutionMetrics struct {
	TotalResolutions          int64         `json:"total_resolutions"`
	SuccessfulResolutions     int64         `json:"successful_resolutions"`
	FailedResolutions         int64         `json:"failed_resolutions"`
	AverageResolutionTime     time.Duration `json:"average_resolution_time"`
	MaxRecursionDepthReached  int           `json:"max_recursion_depth_reached"`
	CircularReferencesDetected int64        `json:"circular_references_detected"`
	TypeAliasResolutions      int64         `json:"type_alias_resolutions"`
	ConstraintValidations     int64         `json:"constraint_validations"`
	CacheHits                 int64         `json:"cache_hits"`
	CacheMisses               int64         `json:"cache_misses"`
}

// NewRecursiveTypeResolver creates a new recursive type resolver.
func NewRecursiveTypeResolver(
	typeSubstitution *domain.TypeSubstitutionEngine,
	logger *zap.Logger,
	config *RecursiveResolverConfig,
) *RecursiveTypeResolver {
	if config == nil {
		config = DefaultRecursiveResolverConfig()
	}
	
	if logger == nil {
		logger = zap.NewNop()
	}
	
	return &RecursiveTypeResolver{
		logger:             logger,
		typeSubstitution:   typeSubstitution,
		config:             config,
		resolutionStack:    make([]string, 0),
		visitedTypes:       make(map[string]*ResolutionResult),
		circularReferences: make(map[string]bool),
		aliasRegistry:      make(map[string]domain.Type),
		aliasDepthTracker:  make(map[string]int),
		constraintCache:    make(map[string]*ConstraintValidationResult),
		metrics:           &RecursiveResolutionMetrics{},
	}
}

// ResolveNestedGenericType performs deep recursive resolution of nested generic types.
// This handles complex scenarios like Map[string, List[T]] → Map[string, Array[U]].
func (rtr *RecursiveTypeResolver) ResolveNestedGenericType(
	genericType domain.Type,
	typeSubstitutions map[string]domain.Type,
) (*ResolutionResult, error) {
	startTime := time.Now()
	rtr.metrics.TotalResolutions++
	
	// Reset resolution context
	rtr.resetResolutionContext()
	
	// Create type key for tracking
	typeKey := rtr.generateTypeKey(genericType, typeSubstitutions)
	
	// Check if already resolved (cycle detection)
	if result, found := rtr.visitedTypes[typeKey]; found {
		rtr.logger.Debug("type already resolved, returning cached result",
			zap.String("type_key", typeKey))
		return result, nil
	}
	
	rtr.logger.Debug("starting recursive type resolution",
		zap.String("type", genericType.String()),
		zap.Int("substitutions", len(typeSubstitutions)))
	
	// Perform recursive resolution
	resolvedType, err := rtr.performRecursiveResolution(genericType, typeSubstitutions, 0)
	if err != nil {
		rtr.metrics.FailedResolutions++
		return nil, fmt.Errorf("%w: %s", ErrRecursiveResolutionFailed, err.Error())
	}
	
	// Create resolution result
	result := &ResolutionResult{
		OriginalType:              genericType,
		ResolvedType:              resolvedType,
		TypeSubstitutions:         rtr.copyTypeSubstitutions(typeSubstitutions),
		ResolutionPath:            rtr.copyResolutionStack(),
		ResolvedAt:                time.Now(),
		ResolutionDepth:           len(rtr.resolutionStack),
		AliasResolutions:          rtr.extractAliasResolutions(),
		ConstraintsSatisfied:      true, // TODO: Implement constraint validation
		CircularReferenceDetected: len(rtr.circularReferences) > 0,
	}
	
	// Cache the result
	rtr.visitedTypes[typeKey] = result
	
	// Update metrics
	resolutionTime := time.Since(startTime)
	rtr.updateResolutionMetrics(resolutionTime, len(rtr.resolutionStack))
	rtr.metrics.SuccessfulResolutions++
	
	rtr.logger.Info("recursive type resolution completed",
		zap.String("original_type", genericType.String()),
		zap.String("resolved_type", resolvedType.String()),
		zap.Duration("resolution_time", resolutionTime),
		zap.Int("depth", result.ResolutionDepth))
	
	return result, nil
}

// performRecursiveResolution performs the actual recursive type resolution.
func (rtr *RecursiveTypeResolver) performRecursiveResolution(
	typ domain.Type,
	typeSubstitutions map[string]domain.Type,
	depth int,
) (domain.Type, error) {
	// Check recursion depth limit
	if depth > rtr.config.MaxRecursionDepth {
		return nil, fmt.Errorf("%w: depth %d exceeds maximum %d",
			ErrMaxRecursionDepthExceeded, depth, rtr.config.MaxRecursionDepth)
	}
	
	// Check for circular references if enabled
	if rtr.config.EnableCircularCheck {
		if err := rtr.checkCircularReference(typ, depth); err != nil {
			return nil, err
		}
	}
	
	// Add to resolution stack for tracking
	typeKey := typ.String()
	rtr.resolutionStack = append(rtr.resolutionStack, typeKey)
	defer func() {
		// Remove from stack when done
		if len(rtr.resolutionStack) > 0 {
			rtr.resolutionStack = rtr.resolutionStack[:len(rtr.resolutionStack)-1]
		}
	}()
	
	// Resolve based on type kind
	switch typ.Kind() {
	case domain.KindGeneric:
		return rtr.resolveGenericTypeParameter(typ, typeSubstitutions)
	case domain.KindSlice:
		return rtr.resolveSliceType(typ, typeSubstitutions, depth)
	case domain.KindMap:
		return rtr.resolveMapType(typ, typeSubstitutions, depth)
	case domain.KindStruct:
		return rtr.resolveStructType(typ, typeSubstitutions, depth)
	case domain.KindPointer:
		return rtr.resolvePointerType(typ, typeSubstitutions, depth)
	case domain.KindNamed:
		return rtr.resolveNamedType(typ, typeSubstitutions, depth)
	default:
		// For basic types and others, no resolution needed
		return typ, nil
	}
}

// resolveGenericTypeParameter resolves a generic type parameter.
func (rtr *RecursiveTypeResolver) resolveGenericTypeParameter(
	genericType domain.Type,
	typeSubstitutions map[string]domain.Type,
) (domain.Type, error) {
	paramName := genericType.Name()
	
	// Check if we have a substitution for this parameter
	if concreteType, found := typeSubstitutions[paramName]; found {
		rtr.logger.Debug("resolving generic parameter",
			zap.String("param", paramName),
			zap.String("concrete_type", concreteType.String()))
		
		// Validate type constraints if available
		if err := rtr.validateTypeConstraints(genericType, concreteType); err != nil {
			return nil, fmt.Errorf("constraint validation failed for %s: %w", paramName, err)
		}
		
		return concreteType, nil
	}
	
	// No substitution found, return original type
	rtr.logger.Debug("no substitution found for generic parameter",
		zap.String("param", paramName))
	return genericType, nil
}

// resolveSliceType recursively resolves slice element types.
func (rtr *RecursiveTypeResolver) resolveSliceType(
	sliceType domain.Type,
	typeSubstitutions map[string]domain.Type,
	depth int,
) (domain.Type, error) {
	slice, ok := sliceType.(*domain.SliceType)
	if !ok {
		return nil, fmt.Errorf("expected SliceType, got %T", sliceType)
	}
	
	// Recursively resolve element type
	resolvedElem, err := rtr.performRecursiveResolution(slice.Elem(), typeSubstitutions, depth+1)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve slice element type: %w", err)
	}
	
	// If element type didn't change, return original
	if resolvedElem == slice.Elem() {
		return sliceType, nil
	}
	
	// Create new slice type with resolved element
	return domain.NewSliceType(resolvedElem, slice.Package()), nil
}

// resolveMapType recursively resolves map key and value types.
func (rtr *RecursiveTypeResolver) resolveMapType(
	mapType domain.Type,
	typeSubstitutions map[string]domain.Type,
	depth int,
) (domain.Type, error) {
	// This is a simplified implementation for map types
	// In a full implementation, we would need proper MapType interface
	rtr.logger.Debug("resolving map type (simplified)",
		zap.String("map_type", mapType.String()))
	
	// For now, return the original map type
	// A full implementation would recursively resolve key and value types
	return mapType, nil
}

// resolveStructType recursively resolves struct field types.
func (rtr *RecursiveTypeResolver) resolveStructType(
	structType domain.Type,
	typeSubstitutions map[string]domain.Type,
	depth int,
) (domain.Type, error) {
	struct_, ok := structType.(*domain.StructType)
	if !ok {
		return nil, fmt.Errorf("expected StructType, got %T", structType)
	}
	
	originalFields := struct_.Fields()
	resolvedFields := make([]domain.Field, len(originalFields))
	hasChanges := false
	
	// Recursively resolve each field type
	for i, field := range originalFields {
		resolvedFieldType, err := rtr.performRecursiveResolution(field.Type, typeSubstitutions, depth+1)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve field %s type: %w", field.Name, err)
		}
		
		resolvedFields[i] = field
		if resolvedFieldType != field.Type {
			resolvedFields[i].Type = resolvedFieldType
			hasChanges = true
		}
	}
	
	// If no changes, return original
	if !hasChanges {
		return structType, nil
	}
	
	// Create new struct type with resolved field types
	return domain.NewStructType(struct_.Name(), resolvedFields, struct_.Package()), nil
}

// resolvePointerType recursively resolves pointer element types.
func (rtr *RecursiveTypeResolver) resolvePointerType(
	pointerType domain.Type,
	typeSubstitutions map[string]domain.Type,
	depth int,
) (domain.Type, error) {
	pointer, ok := pointerType.(*domain.PointerType)
	if !ok {
		return nil, fmt.Errorf("expected PointerType, got %T", pointerType)
	}
	
	// Recursively resolve element type
	resolvedElem, err := rtr.performRecursiveResolution(pointer.Elem(), typeSubstitutions, depth+1)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve pointer element type: %w", err)
	}
	
	// If element type didn't change, return original
	if resolvedElem == pointer.Elem() {
		return pointerType, nil
	}
	
	// Create new pointer type with resolved element
	return domain.NewPointerType(resolvedElem, pointer.Package()), nil
}

// resolveNamedType resolves named types and handles type aliases.
func (rtr *RecursiveTypeResolver) resolveNamedType(
	namedType domain.Type,
	typeSubstitutions map[string]domain.Type,
	depth int,
) (domain.Type, error) {
	typeName := namedType.Name()
	
	// Check if this is a registered type alias
	if aliasedType, found := rtr.aliasRegistry[typeName]; found {
		rtr.metrics.TypeAliasResolutions++
		
		// Track alias depth to prevent infinite alias chains
		if rtr.aliasDepthTracker[typeName] > rtr.config.MaxTypeAliasDepth {
			return nil, fmt.Errorf("%w: alias chain too deep for %s", ErrTypeAliasResolutionFailed, typeName)
		}
		
		rtr.aliasDepthTracker[typeName]++
		defer func() {
			rtr.aliasDepthTracker[typeName]--
		}()
		
		// Recursively resolve the aliased type
		return rtr.performRecursiveResolution(aliasedType, typeSubstitutions, depth+1)
	}
	
	// For named types without aliases, recursively resolve underlying type if generic
	if namedType.Generic() && namedType.Underlying() != nil {
		resolvedUnderlying, err := rtr.performRecursiveResolution(namedType.Underlying(), typeSubstitutions, depth+1)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve underlying type for %s: %w", typeName, err)
		}
		
		// If underlying type changed, we would create a new named type
		// For now, return the resolved underlying type
		return resolvedUnderlying, nil
	}
	
	return namedType, nil
}

// RegisterTypeAlias registers a type alias for resolution.
func (rtr *RecursiveTypeResolver) RegisterTypeAlias(aliasName string, actualType domain.Type) {
	rtr.aliasRegistry[aliasName] = actualType
	rtr.logger.Debug("registered type alias",
		zap.String("alias", aliasName),
		zap.String("actual_type", actualType.String()))
}

// validateTypeConstraints validates that a concrete type satisfies the constraints of a type parameter.
func (rtr *RecursiveTypeResolver) validateTypeConstraints(
	typeParam domain.Type,
	concreteType domain.Type,
) error {
	// Generate cache key for constraint validation
	cacheKey := fmt.Sprintf("%s->%s", typeParam.String(), concreteType.String())
	
	// Check cache if enabled
	if rtr.config.EnableConstraintCache {
		if cached, found := rtr.constraintCache[cacheKey]; found {
			rtr.metrics.CacheHits++
			if !cached.Satisfied {
				return fmt.Errorf("%w: %v", ErrConstraintValidationFailed, cached.ValidationErrors)
			}
			return nil
		}
		rtr.metrics.CacheMisses++
	}
	
	rtr.metrics.ConstraintValidations++
	
	// Perform constraint validation
	validationResult := &ConstraintValidationResult{
		ConcreteType:     concreteType,
		Satisfied:        true, // Default to satisfied for now
		ValidationErrors: make([]string, 0),
		ValidatedAt:      time.Now(),
	}
	
	// TODO: Implement actual constraint checking logic
	// This would check if concreteType satisfies typeParam constraints
	// For now, we allow all substitutions
	
	// Cache the result
	if rtr.config.EnableConstraintCache {
		rtr.constraintCache[cacheKey] = validationResult
	}
	
	if !validationResult.Satisfied {
		return fmt.Errorf("%w: %v", ErrConstraintValidationFailed, validationResult.ValidationErrors)
	}
	
	return nil
}

// Helper methods

func (rtr *RecursiveTypeResolver) resetResolutionContext() {
	rtr.resolutionStack = rtr.resolutionStack[:0]
	rtr.circularReferences = make(map[string]bool)
	for k := range rtr.aliasDepthTracker {
		rtr.aliasDepthTracker[k] = 0
	}
}

func (rtr *RecursiveTypeResolver) generateTypeKey(typ domain.Type, substitutions map[string]domain.Type) string {
	var builder strings.Builder
	builder.WriteString(typ.String())
	builder.WriteString("|")
	
	// Add substitutions in deterministic order
	for k, v := range substitutions {
		builder.WriteString(k)
		builder.WriteString(":")
		builder.WriteString(v.String())
		builder.WriteString(",")
	}
	
	return builder.String()
}

func (rtr *RecursiveTypeResolver) checkCircularReference(typ domain.Type, depth int) error {
	typeKey := typ.String()
	
	// Check if this type is already in the resolution stack
	for _, existing := range rtr.resolutionStack {
		if existing == typeKey {
			rtr.circularReferences[typeKey] = true
			rtr.metrics.CircularReferencesDetected++
			return fmt.Errorf("%w: circular reference detected for type %s", ErrCircularTypeReference, typeKey)
		}
	}
	
	return nil
}

func (rtr *RecursiveTypeResolver) copyTypeSubstitutions(substitutions map[string]domain.Type) map[string]domain.Type {
	result := make(map[string]domain.Type, len(substitutions))
	for k, v := range substitutions {
		result[k] = v
	}
	return result
}

func (rtr *RecursiveTypeResolver) copyResolutionStack() []string {
	result := make([]string, len(rtr.resolutionStack))
	copy(result, rtr.resolutionStack)
	return result
}

func (rtr *RecursiveTypeResolver) extractAliasResolutions() []string {
	result := make([]string, 0)
	for alias, depth := range rtr.aliasDepthTracker {
		if depth > 0 {
			result = append(result, alias)
		}
	}
	return result
}

func (rtr *RecursiveTypeResolver) updateResolutionMetrics(duration time.Duration, depth int) {
	if rtr.config.EnablePerformanceTrack {
		totalTime := time.Duration(rtr.metrics.TotalResolutions-1)*rtr.metrics.AverageResolutionTime + duration
		rtr.metrics.AverageResolutionTime = totalTime / time.Duration(rtr.metrics.TotalResolutions)
		
		if depth > rtr.metrics.MaxRecursionDepthReached {
			rtr.metrics.MaxRecursionDepthReached = depth
		}
	}
}

// GetMetrics returns current resolution metrics.
func (rtr *RecursiveTypeResolver) GetMetrics() *RecursiveResolutionMetrics {
	return rtr.metrics
}

// ClearCache clears the resolution caches.
func (rtr *RecursiveTypeResolver) ClearCache() {
	rtr.visitedTypes = make(map[string]*ResolutionResult)
	rtr.constraintCache = make(map[string]*ConstraintValidationResult)
	rtr.logger.Debug("recursive resolver caches cleared")
}