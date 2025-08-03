package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Static errors for err113 compliance.
var (
	ErrSubstitutionTypeNil            = errors.New("type for substitution cannot be nil")
	ErrSubstitutionTypeParamsNil      = errors.New("type parameters cannot be nil")
	ErrSubstitutionTypeArgsNil        = errors.New("type arguments cannot be nil")
	ErrSubstitutionParameterMismatch  = errors.New("type parameter count does not match type argument count")
	ErrSubstitutionRecursionLimit     = errors.New("recursion limit exceeded during type substitution")
	ErrSubstitutionCycleDetected      = errors.New("circular type dependency detected during substitution")
	ErrSubstitutionFailed             = errors.New("type substitution failed")
	ErrSubstitutionCacheKeyGeneration = errors.New("failed to generate substitution cache key")
	ErrSubstitutionUnsupportedType    = errors.New("unsupported type for substitution")
)

// Constants for substitution configuration.
const (
	// DefaultMaxRecursionDepth is the default maximum recursion depth for type substitution.
	DefaultMaxRecursionDepth = 100
	// DefaultCacheCapacity is the default cache capacity for substitution results.
	DefaultCacheCapacity = 10000
	// DefaultCycleLimitDepth is the default depth limit for cycle detection.
	DefaultCycleLimitDepth = 50
)

// SubstitutionResult represents the result of a type substitution operation.
type SubstitutionResult struct {
	// Core substitution data
	OriginalType     Type            `json:"original_type"`     // The original type before substitution
	SubstitutedType  Type            `json:"substituted_type"`  // The resulting type after substitution
	TypeMapping      map[string]Type `json:"type_mapping"`      // Map of type parameter names to concrete types
	SubstitutionPath []string        `json:"substitution_path"` // Path of type substitutions performed

	// Operation metadata
	CacheHit         bool      `json:"cache_hit"`         // Whether this result came from cache
	SubstitutedAt    time.Time `json:"substituted_at"`    // When the substitution was performed
	SubstitutionTime int64     `json:"substitution_time"` // Time taken in milliseconds

	// Performance tracking
	RecursionDepth   int                `json:"recursion_depth"`   // Maximum recursion depth reached
	CycleCheckCount  int                `json:"cycle_check_count"` // Number of cycle checks performed
	CacheOperations  int                `json:"cache_operations"`  // Number of cache operations performed
	TypesProcessed   int                `json:"types_processed"`   // Total number of types processed
	PerformanceStats *SubstitutionStats `json:"performance_stats"` // Detailed performance statistics
}

// SubstitutionStats provides detailed performance statistics for substitution operations.
type SubstitutionStats struct {
	BasicTypeSubstitutions     int   `json:"basic_type_substitutions"`
	SliceTypeSubstitutions     int   `json:"slice_type_substitutions"`
	PointerTypeSubstitutions   int   `json:"pointer_type_substitutions"`
	StructTypeSubstitutions    int   `json:"struct_type_substitutions"`
	GenericTypeSubstitutions   int   `json:"generic_type_substitutions"`
	MapTypeSubstitutions       int   `json:"map_type_substitutions"`
	InterfaceTypeSubstitutions int   `json:"interface_type_substitutions"`
	FunctionTypeSubstitutions  int   `json:"function_type_substitutions"`
	NamedTypeSubstitutions     int   `json:"named_type_substitutions"`
	TotalCacheHits             int   `json:"total_cache_hits"`
	TotalCacheMisses           int   `json:"total_cache_misses"`
	OptimizationApplied        bool  `json:"optimization_applied"`
	MemoryUsageBytes           int64 `json:"memory_usage_bytes"`
}

// NewSubstitutionResult creates a new substitution result with proper initialization.
func NewSubstitutionResult(originalType, substitutedType Type, typeMapping map[string]Type) *SubstitutionResult {
	return &SubstitutionResult{
		OriginalType:     originalType,
		SubstitutedType:  substitutedType,
		TypeMapping:      copyTypeMapping(typeMapping),
		SubstitutionPath: make([]string, 0),
		CacheHit:         false,
		SubstitutedAt:    time.Now(),
		SubstitutionTime: 0,
		RecursionDepth:   0,
		CycleCheckCount:  0,
		CacheOperations:  0,
		TypesProcessed:   1,
		PerformanceStats: &SubstitutionStats{},
	}
}

// copyTypeMapping creates a defensive copy of the type mapping.
func copyTypeMapping(mapping map[string]Type) map[string]Type {
	if mapping == nil {
		return make(map[string]Type)
	}
	result := make(map[string]Type, len(mapping))
	for k, v := range mapping {
		result[k] = v
	}
	return result
}

// TypeSubstitutionEngine is the core engine for performing type parameter substitution.
// It handles complex type hierarchies, cycle detection, and performance optimization.
type TypeSubstitutionEngine struct {
	// Core components
	logger      *zap.Logger
	typeBuilder *TypeBuilder

	// Performance optimization
	cache               map[string]*SubstitutionResult
	cacheMutex          sync.RWMutex
	enableCaching       bool
	cacheCapacity       int
	substitutionStats   *SubstitutionStats
	optimizePerformance bool

	// Safety mechanisms
	maxRecursionDepth int
	recursionStack    []string
	visitedTypes      map[string]bool
	cycleLimitDepth   int

	// Context and control
	ctx                  context.Context
	enableDetailedStats  bool
	enableMemoryTracking bool
}

// SubstitutionEngineConfig configures the behavior of TypeSubstitutionEngine.
type SubstitutionEngineConfig struct {
	MaxRecursionDepth    int         `json:"max_recursion_depth"`
	EnableCaching        bool        `json:"enable_caching"`
	CacheCapacity        int         `json:"cache_capacity"`
	CycleLimitDepth      int         `json:"cycle_limit_depth"`
	OptimizePerformance  bool        `json:"optimize_performance"`
	EnableDetailedStats  bool        `json:"enable_detailed_stats"`
	EnableMemoryTracking bool        `json:"enable_memory_tracking"`
	Logger               *zap.Logger `json:"-"`
}

// NewSubstitutionEngineConfig creates a default configuration for the substitution engine.
func NewSubstitutionEngineConfig() *SubstitutionEngineConfig {
	return &SubstitutionEngineConfig{
		MaxRecursionDepth:    DefaultMaxRecursionDepth,
		EnableCaching:        true,
		CacheCapacity:        DefaultCacheCapacity,
		CycleLimitDepth:      DefaultCycleLimitDepth,
		OptimizePerformance:  true,
		EnableDetailedStats:  true,
		EnableMemoryTracking: false,
		Logger:               nil,
	}
}

// NewTypeSubstitutionEngine creates a new type substitution engine with default configuration.
func NewTypeSubstitutionEngine(typeBuilder *TypeBuilder, logger *zap.Logger) *TypeSubstitutionEngine {
	return NewTypeSubstitutionEngineWithConfig(typeBuilder, NewSubstitutionEngineConfig(), logger)
}

// NewTypeSubstitutionEngineWithConfig creates a new type substitution engine with custom configuration.
func NewTypeSubstitutionEngineWithConfig(
	typeBuilder *TypeBuilder,
	config *SubstitutionEngineConfig,
	logger *zap.Logger,
) *TypeSubstitutionEngine {
	if typeBuilder == nil {
		typeBuilder = NewTypeBuilder()
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	if config.Logger != nil {
		logger = config.Logger
	}

	var cache map[string]*SubstitutionResult
	if config.EnableCaching {
		cache = make(map[string]*SubstitutionResult, config.CacheCapacity)
	}

	return &TypeSubstitutionEngine{
		logger:               logger,
		typeBuilder:          typeBuilder,
		cache:                cache,
		enableCaching:        config.EnableCaching,
		cacheCapacity:        config.CacheCapacity,
		substitutionStats:    &SubstitutionStats{},
		optimizePerformance:  config.OptimizePerformance,
		maxRecursionDepth:    config.MaxRecursionDepth,
		recursionStack:       make([]string, 0),
		visitedTypes:         make(map[string]bool),
		cycleLimitDepth:      config.CycleLimitDepth,
		ctx:                  context.Background(),
		enableDetailedStats:  config.EnableDetailedStats,
		enableMemoryTracking: config.EnableMemoryTracking,
	}
}

// SubstituteType is the main entry point for type substitution.
// It replaces type parameters in the given type with concrete type arguments.
func (tse *TypeSubstitutionEngine) SubstituteType(
	genericType Type,
	typeParams []TypeParam,
	typeArgs []Type,
) (*SubstitutionResult, error) {
	return tse.SubstituteTypeWithContext(context.Background(), genericType, typeParams, typeArgs)
}

// SubstituteTypeWithContext performs type substitution with context support.
func (tse *TypeSubstitutionEngine) SubstituteTypeWithContext(
	ctx context.Context,
	genericType Type,
	typeParams []TypeParam,
	typeArgs []Type,
) (*SubstitutionResult, error) {
	startTime := time.Now()
	tse.ctx = ctx

	// Input validation
	if err := tse.validateSubstitutionInputs(genericType, typeParams, typeArgs); err != nil {
		return nil, err
	}

	// Create type parameter mapping
	typeMapping := tse.createTypeMapping(typeParams, typeArgs)

	// Generate cache key for this substitution
	cacheKey := tse.generateCacheKey(genericType, typeMapping)

	// Check cache first if enabled
	if tse.enableCaching {
		if cached := tse.getCachedResult(cacheKey); cached != nil {
			tse.logger.Debug("cache hit for type substitution",
				zap.String("original_type", genericType.String()),
				zap.String("cache_key", cacheKey))

			// Create copy with updated cache hit status
			result := *cached
			result.CacheHit = true
			result.SubstitutedAt = time.Now()
			return &result, nil
		}
	}

	tse.logger.Debug("performing type substitution",
		zap.String("type", genericType.String()),
		zap.Int("type_params", len(typeParams)),
		zap.Int("type_args", len(typeArgs)),
		zap.String("cache_key", cacheKey))

	// Initialize substitution context
	tse.resetSubstitutionContext()

	// Perform the actual substitution
	substitutedType, err := tse.performSubstitution(genericType, typeMapping, 0)
	if err != nil {
		tse.logger.Error("type substitution failed",
			zap.String("type", genericType.String()),
			zap.Error(err))
		return nil, fmt.Errorf("%w: %s", ErrSubstitutionFailed, err.Error())
	}

	// Create the result
	result := NewSubstitutionResult(genericType, substitutedType, typeMapping)
	result.SubstitutionTime = time.Since(startTime).Milliseconds()
	result.RecursionDepth = len(tse.recursionStack)
	result.TypesProcessed = len(tse.visitedTypes)
	result.PerformanceStats = tse.substitutionStats

	// Update statistics
	if tse.enableDetailedStats {
		tse.updateSubstitutionStats(result)
	}

	// Cache the result if enabled
	if tse.enableCaching {
		tse.cacheResult(cacheKey, result)
	}

	tse.logger.Info("type substitution completed successfully",
		zap.String("original_type", genericType.String()),
		zap.String("substituted_type", substitutedType.String()),
		zap.Int64("duration_ms", result.SubstitutionTime),
		zap.Int("recursion_depth", result.RecursionDepth),
		zap.Int("types_processed", result.TypesProcessed))

	return result, nil
}

// validateSubstitutionInputs validates the inputs for type substitution.
func (tse *TypeSubstitutionEngine) validateSubstitutionInputs(
	genericType Type,
	typeParams []TypeParam,
	typeArgs []Type,
) error {
	if genericType == nil {
		return ErrSubstitutionTypeNil
	}

	if typeParams == nil {
		return ErrSubstitutionTypeParamsNil
	}

	if typeArgs == nil {
		return ErrSubstitutionTypeArgsNil
	}

	if len(typeParams) != len(typeArgs) {
		return fmt.Errorf("%w: expected %d type arguments, got %d",
			ErrSubstitutionParameterMismatch,
			len(typeParams),
			len(typeArgs))
	}

	// Validate type arguments are not nil
	for i, typeArg := range typeArgs {
		if typeArg == nil {
			return fmt.Errorf("type argument at index %d is nil", i)
		}
	}

	return nil
}

// createTypeMapping creates a mapping from type parameter names to concrete types.
func (tse *TypeSubstitutionEngine) createTypeMapping(typeParams []TypeParam, typeArgs []Type) map[string]Type {
	mapping := make(map[string]Type, len(typeParams))
	for i, param := range typeParams {
		mapping[param.Name] = typeArgs[i]
	}
	return mapping
}

// generateCacheKey generates a unique cache key for the substitution operation.
func (tse *TypeSubstitutionEngine) generateCacheKey(genericType Type, typeMapping map[string]Type) string {
	var builder strings.Builder

	// Include the original type
	builder.WriteString(genericType.String())
	builder.WriteString("|")

	// Include type mappings in a deterministic order
	keys := make([]string, 0, len(typeMapping))
	for key := range typeMapping {
		keys = append(keys, key)
	}

	// Simple sort for deterministic ordering
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	for i, key := range keys {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(key)
		builder.WriteString(":")
		builder.WriteString(typeMapping[key].String())
	}

	return builder.String()
}

// getCachedResult retrieves a cached substitution result.
func (tse *TypeSubstitutionEngine) getCachedResult(cacheKey string) *SubstitutionResult {
	if !tse.enableCaching {
		return nil
	}

	tse.cacheMutex.RLock()
	defer tse.cacheMutex.RUnlock()

	result, found := tse.cache[cacheKey]
	if found {
		tse.substitutionStats.TotalCacheHits++
		return result
	}

	tse.substitutionStats.TotalCacheMisses++
	return nil
}

// cacheResult stores a substitution result in the cache.
func (tse *TypeSubstitutionEngine) cacheResult(cacheKey string, result *SubstitutionResult) {
	if !tse.enableCaching {
		return
	}

	tse.cacheMutex.Lock()
	defer tse.cacheMutex.Unlock()

	// Check cache capacity and evict if necessary
	if len(tse.cache) >= tse.cacheCapacity {
		tse.evictOldestCacheEntry()
	}

	tse.cache[cacheKey] = result
}

// evictOldestCacheEntry removes the oldest entry from the cache.
func (tse *TypeSubstitutionEngine) evictOldestCacheEntry() {
	// Simple eviction strategy - remove first entry found
	// In a production system, this could be enhanced with LRU or other strategies
	for key := range tse.cache {
		delete(tse.cache, key)
		break
	}
}

// resetSubstitutionContext resets the context for a new substitution operation.
func (tse *TypeSubstitutionEngine) resetSubstitutionContext() {
	tse.recursionStack = tse.recursionStack[:0]
	tse.visitedTypes = make(map[string]bool)
	tse.substitutionStats = &SubstitutionStats{}
}

// performSubstitution performs the actual type substitution with recursion and cycle detection.
func (tse *TypeSubstitutionEngine) performSubstitution(
	typ Type,
	typeMapping map[string]Type,
	depth int,
) (Type, error) {
	// Check recursion limit
	if depth > tse.maxRecursionDepth {
		return nil, fmt.Errorf("%w: depth %d exceeds maximum %d",
			ErrSubstitutionRecursionLimit, depth, tse.maxRecursionDepth)
	}

	// Check for cycles
	if err := tse.checkForCycles(typ, depth); err != nil {
		return nil, err
	}

	// Add to recursion tracking
	typeKey := typ.String()
	tse.recursionStack = append(tse.recursionStack, typeKey)
	tse.visitedTypes[typeKey] = true

	defer func() {
		// Remove from recursion stack
		if len(tse.recursionStack) > 0 {
			tse.recursionStack = tse.recursionStack[:len(tse.recursionStack)-1]
		}
	}()

	// Perform substitution based on type kind
	result, err := tse.substituteByKind(typ, typeMapping, depth)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// checkForCycles detects circular type dependencies.
func (tse *TypeSubstitutionEngine) checkForCycles(typ Type, depth int) error {
	if depth > tse.cycleLimitDepth {
		return fmt.Errorf("%w: cycle limit depth %d exceeded", ErrSubstitutionCycleDetected, tse.cycleLimitDepth)
	}

	typeKey := typ.String()

	// Check if this type is already in the recursion stack
	for _, existing := range tse.recursionStack {
		if existing == typeKey {
			return fmt.Errorf("%w: circular dependency detected for type %s",
				ErrSubstitutionCycleDetected, typeKey)
		}
	}

	return nil
}

// substituteByKind performs substitution based on the type's kind.
func (tse *TypeSubstitutionEngine) substituteByKind(
	typ Type,
	typeMapping map[string]Type,
	depth int,
) (Type, error) {
	switch typ.Kind() {
	case KindGeneric:
		return tse.substituteGenericType(typ, typeMapping, depth)
	case KindSlice:
		return tse.substituteSliceType(typ, typeMapping, depth)
	case KindPointer:
		return tse.substitutePointerType(typ, typeMapping, depth)
	case KindStruct:
		return tse.substituteStructType(typ, typeMapping, depth)
	case KindMap:
		return tse.substituteMapType(typ, typeMapping, depth)
	case KindInterface:
		return tse.substituteInterfaceType(typ, typeMapping, depth)
	case KindFunction:
		return tse.substituteFunctionType(typ, typeMapping, depth)
	case KindNamed:
		return tse.substituteNamedType(typ, typeMapping, depth)
	case KindBasic:
		return tse.substituteBasicType(typ, typeMapping, depth)
	default:
		return nil, fmt.Errorf("%w: unsupported type kind %s", ErrSubstitutionUnsupportedType, typ.Kind().String())
	}
}

// substituteGenericType substitutes a generic type parameter with its concrete type.
func (tse *TypeSubstitutionEngine) substituteGenericType(
	typ Type,
	typeMapping map[string]Type,
	depth int,
) (Type, error) {
	if tse.enableDetailedStats {
		tse.substitutionStats.GenericTypeSubstitutions++
	}

	// Check if this generic type has a substitution
	if concreteType, found := typeMapping[typ.Name()]; found {
		tse.logger.Debug("substituting generic type parameter",
			zap.String("type_param", typ.Name()),
			zap.String("concrete_type", concreteType.String()))
		return concreteType, nil
	}

	// If no substitution found, return the original type
	// This handles cases where the generic type is not being substituted
	tse.logger.Debug("no substitution found for generic type parameter",
		zap.String("type_param", typ.Name()))
	return typ, nil
}

// substituteSliceType substitutes the element type of a slice.
func (tse *TypeSubstitutionEngine) substituteSliceType(
	typ Type,
	typeMapping map[string]Type,
	depth int,
) (Type, error) {
	if tse.enableDetailedStats {
		tse.substitutionStats.SliceTypeSubstitutions++
	}

	sliceType, ok := typ.(*SliceType)
	if !ok {
		return nil, fmt.Errorf("expected SliceType, got %T", typ)
	}

	// Substitute the element type
	substitutedElem, err := tse.performSubstitution(sliceType.Elem(), typeMapping, depth+1)
	if err != nil {
		return nil, fmt.Errorf("failed to substitute slice element type: %w", err)
	}

	// If element type didn't change, return original
	if substitutedElem == sliceType.Elem() {
		return typ, nil
	}

	// Create new slice type with substituted element
	return NewSliceType(substitutedElem, sliceType.Package()), nil
}

// substitutePointerType substitutes the element type of a pointer.
func (tse *TypeSubstitutionEngine) substitutePointerType(
	typ Type,
	typeMapping map[string]Type,
	depth int,
) (Type, error) {
	if tse.enableDetailedStats {
		tse.substitutionStats.PointerTypeSubstitutions++
	}

	pointerType, ok := typ.(*PointerType)
	if !ok {
		return nil, fmt.Errorf("expected PointerType, got %T", typ)
	}

	// Substitute the element type
	substitutedElem, err := tse.performSubstitution(pointerType.Elem(), typeMapping, depth+1)
	if err != nil {
		return nil, fmt.Errorf("failed to substitute pointer element type: %w", err)
	}

	// If element type didn't change, return original
	if substitutedElem == pointerType.Elem() {
		return typ, nil
	}

	// Create new pointer type with substituted element
	return NewPointerType(substitutedElem, pointerType.Package()), nil
}

// substituteStructType substitutes type parameters in struct fields.
func (tse *TypeSubstitutionEngine) substituteStructType(
	typ Type,
	typeMapping map[string]Type,
	depth int,
) (Type, error) {
	if tse.enableDetailedStats {
		tse.substitutionStats.StructTypeSubstitutions++
	}

	structType, ok := typ.(*StructType)
	if !ok {
		return nil, fmt.Errorf("expected StructType, got %T", typ)
	}

	// Get original fields
	originalFields := structType.Fields()
	hasChanges := false
	substitutedFields := make([]Field, len(originalFields))

	// Substitute each field type
	for i, field := range originalFields {
		substitutedFieldType, err := tse.performSubstitution(field.Type, typeMapping, depth+1)
		if err != nil {
			return nil, fmt.Errorf("failed to substitute field type for %s: %w", field.Name, err)
		}

		substitutedFields[i] = field
		if substitutedFieldType != field.Type {
			substitutedFields[i].Type = substitutedFieldType
			hasChanges = true
		}
	}

	// If no changes, return original
	if !hasChanges {
		return typ, nil
	}

	// Create new struct type with substituted field types
	return NewStructType(structType.Name(), substitutedFields, structType.Package()), nil
}

// substituteMapType substitutes key and value types of a map.
func (tse *TypeSubstitutionEngine) substituteMapType(
	typ Type,
	typeMapping map[string]Type,
	depth int,
) (Type, error) {
	if tse.enableDetailedStats {
		tse.substitutionStats.MapTypeSubstitutions++
	}

	// For now, return the original type since mapType is a simple implementation
	// In a full implementation, this would substitute both key and value types
	return typ, nil
}

// substituteInterfaceType substitutes type parameters in interface methods.
func (tse *TypeSubstitutionEngine) substituteInterfaceType(
	typ Type,
	typeMapping map[string]Type,
	depth int,
) (Type, error) {
	if tse.enableDetailedStats {
		tse.substitutionStats.InterfaceTypeSubstitutions++
	}

	// For basic interface types, no substitution needed
	// In a full implementation, this would handle method signatures
	return typ, nil
}

// substituteFunctionType substitutes type parameters in function signatures.
func (tse *TypeSubstitutionEngine) substituteFunctionType(
	typ Type,
	typeMapping map[string]Type,
	depth int,
) (Type, error) {
	if tse.enableDetailedStats {
		tse.substitutionStats.FunctionTypeSubstitutions++
	}

	// For basic function types, no substitution needed
	// In a full implementation, this would handle parameter and return types
	return typ, nil
}

// substituteNamedType substitutes type parameters in named types.
func (tse *TypeSubstitutionEngine) substituteNamedType(
	typ Type,
	typeMapping map[string]Type,
	depth int,
) (Type, error) {
	if tse.enableDetailedStats {
		tse.substitutionStats.NamedTypeSubstitutions++
	}

	// For named types, substitute the underlying type if it's generic
	if typ.Generic() {
		return tse.performSubstitution(typ.Underlying(), typeMapping, depth+1)
	}

	return typ, nil
}

// substituteBasicType handles basic types (no substitution needed).
func (tse *TypeSubstitutionEngine) substituteBasicType(
	typ Type,
	typeMapping map[string]Type,
	depth int,
) (Type, error) {
	if tse.enableDetailedStats {
		tse.substitutionStats.BasicTypeSubstitutions++
	}

	// Basic types don't need substitution
	return typ, nil
}

// updateSubstitutionStats updates the performance statistics.
func (tse *TypeSubstitutionEngine) updateSubstitutionStats(result *SubstitutionResult) {
	// Update optimization flag
	tse.substitutionStats.OptimizationApplied = tse.optimizePerformance

	// Memory tracking would be implemented here if enabled
	if tse.enableMemoryTracking {
		// This would involve runtime memory profiling
		tse.substitutionStats.MemoryUsageBytes = 0 // Placeholder
	}
}

// GetSubstitutionStats returns the current substitution statistics.
func (tse *TypeSubstitutionEngine) GetSubstitutionStats() *SubstitutionStats {
	return tse.substitutionStats
}

// ClearCache clears the substitution cache.
func (tse *TypeSubstitutionEngine) ClearCache() {
	if !tse.enableCaching {
		return
	}

	tse.cacheMutex.Lock()
	defer tse.cacheMutex.Unlock()

	tse.cache = make(map[string]*SubstitutionResult, tse.cacheCapacity)
	tse.logger.Debug("substitution cache cleared")
}

// GetCacheSize returns the current size of the substitution cache.
func (tse *TypeSubstitutionEngine) GetCacheSize() int {
	if !tse.enableCaching {
		return 0
	}

	tse.cacheMutex.RLock()
	defer tse.cacheMutex.RUnlock()

	return len(tse.cache)
}

// SetContext sets the context for substitution operations.
func (tse *TypeSubstitutionEngine) SetContext(ctx context.Context) {
	tse.ctx = ctx
}
