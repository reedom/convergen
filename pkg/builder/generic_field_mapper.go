// Package builder provides enhanced type conversion and field mapping functionality
// for Convergen's code generation pipeline, with specialized support for generic types
// and complex nested data structures.
package builder

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v9/pkg/domain"
)

// Static errors for err113 compliance.
var (
	ErrGenericFieldMapperNil     = errors.New("generic field mapper cannot be nil")
	ErrTypeSubstitutionEngineNil = errors.New("type substitution engine cannot be nil")
	ErrGenericMappingContextNil  = errors.New("generic mapping context cannot be nil")
	ErrGenericFieldMappingFailed = errors.New("generic field mapping failed")
	ErrTypeSubstitutionInMapping = errors.New("type substitution failed in field mapping")
	ErrGenericTypeNotSupported   = errors.New("generic type not supported in mapping")
	ErrFieldMappingOptionsNil    = errors.New("field mapping options cannot be nil")
	ErrGenericAssignmentFailed   = errors.New("generic assignment generation failed")
)

// FieldMapper defines the interface for basic field mapping operations.
type FieldMapper interface {
	MapFields(sourceType, destType domain.Type, options map[string]string) ([]*BasicFieldMapping, error)
}

// BasicFieldMapping represents a basic field mapping.
type BasicFieldMapping struct {
	SourceField string
	DestField   string
	SourceType  domain.Type
	DestType    domain.Type
}

// basicFieldMapper provides a default implementation of FieldMapper.
type basicFieldMapper struct{}

// MapFields provides a basic field mapping implementation.
func (bfm *basicFieldMapper) MapFields(_, _ domain.Type, _ map[string]string) ([]*BasicFieldMapping, error) {
	// Simple implementation that returns empty mappings
	// In a real implementation, this would analyze the types and create appropriate mappings
	return []*BasicFieldMapping{}, nil
}

// GenericFieldMapper handles field mapping for generic types with type substitution support.
type GenericFieldMapper struct {
	baseMapper       FieldMapper
	typeSubstitution *domain.TypeSubstitutionEngine
	logger           *zap.Logger

	// Configuration
	config *GenericFieldMapperConfig

	// Performance tracking
	metrics    *GenericFieldMappingMetrics
	metricsMux sync.RWMutex // Protects metrics for thread safety

	// Cache for field mapping strategies
	strategyCache map[string]domain.ConversionStrategy

	// Built-in conversion strategies
	strategies []domain.ConversionStrategy

	// Enhanced: Recursive type resolver for deeply nested generics
	recursiveResolver *RecursiveTypeResolver

	// Performance optimization engine
	performanceOptimizer *PerformanceOptimizer
}

// GenericFieldMapperConfig configures the generic field mapper.
type GenericFieldMapperConfig struct {
	EnableCaching        bool          `json:"enable_caching"`
	MaxCacheSize         int           `json:"max_cache_size"`
	EnableOptimization   bool          `json:"enable_optimization"`
	MappingTimeout       time.Duration `json:"mapping_timeout"`
	EnableTypeValidation bool          `json:"enable_type_validation"`
	DebugMode            bool          `json:"debug_mode"`
	PerformanceMode      bool          `json:"performance_mode"`
}

// DefaultGenericFieldMapperConfig returns default configuration.
func DefaultGenericFieldMapperConfig() *GenericFieldMapperConfig {
	return &GenericFieldMapperConfig{
		EnableCaching:        true,
		MaxCacheSize:         1000,
		EnableOptimization:   true,
		MappingTimeout:       30 * time.Second,
		EnableTypeValidation: true,
		DebugMode:            false,
		PerformanceMode:      false,
	}
}

// GenericFieldMappingMetrics tracks performance for generic field mapping.
type GenericFieldMappingMetrics struct {
	TotalMappings        int64         `json:"total_mappings"`
	SuccessfulMappings   int64         `json:"successful_mappings"`
	FailedMappings       int64         `json:"failed_mappings"`
	TypeSubstitutions    int64         `json:"type_substitutions"`
	CacheHits            int64         `json:"cache_hits"`
	CacheMisses          int64         `json:"cache_misses"`
	OptimizationsApplied int64         `json:"optimizations_applied"`
	AverageMappingTime   time.Duration `json:"average_mapping_time"`
	TotalMappingTime     time.Duration `json:"total_mapping_time"`
}

// NewGenericFieldMappingMetrics creates new metrics instance.
func NewGenericFieldMappingMetrics() *GenericFieldMappingMetrics {
	return &GenericFieldMappingMetrics{}
}

// FieldMappingOptions provides options for field mapping operations.
type FieldMappingOptions struct {
	IncludePrivateFields bool                   `json:"include_private_fields"`
	UseTypeConversion    bool                   `json:"use_type_conversion"`
	ValidateTypes        bool                   `json:"validate_types"`
	IgnoreUnmatched      bool                   `json:"ignore_unmatched"`
	CustomMappings       map[string]string      `json:"custom_mappings"`
	Annotations          map[string]*Annotation `json:"annotations"`
	ErrorHandling        ErrorHandlingStrategy  `json:"error_handling"`
}

// DefaultFieldMappingOptions returns default field mapping options.
func DefaultFieldMappingOptions() *FieldMappingOptions {
	return &FieldMappingOptions{
		IncludePrivateFields: false,
		UseTypeConversion:    true,
		ValidateTypes:        true,
		IgnoreUnmatched:      false,
		CustomMappings:       make(map[string]string),
		Annotations:          make(map[string]*Annotation),
		ErrorHandling:        domain.ErrorPropagate,
	}
}

// Annotation represents field mapping annotations.
type Annotation struct {
	Skip       bool              `json:"skip"`
	Map        string            `json:"map"`
	Converter  string            `json:"converter"`
	Validation string            `json:"validation"`
	Literal    string            `json:"literal"`
	Custom     map[string]string `json:"custom"`
}

// ErrorHandlingStrategy defines how to handle mapping errors.
type ErrorHandlingStrategy = domain.ErrorHandlingStrategy

// GenericFieldMapping represents a field mapping for generic types
// This mirrors the interface expected by the emitter package
type GenericFieldMapping struct {
	SourceField string            `json:"source_field"`
	DestField   string            `json:"dest_field"`
	SourceType  domain.Type       `json:"source_type"`
	DestType    domain.Type       `json:"dest_type"`
	Converter   string            `json:"converter,omitempty"`
	Validation  string            `json:"validation,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// NewGenericFieldMapper creates a new generic field mapper.
func NewGenericFieldMapper(
	baseMapper FieldMapper,
	typeSubstitution *domain.TypeSubstitutionEngine,
	logger *zap.Logger,
	config *GenericFieldMapperConfig,
) *GenericFieldMapper {
	if baseMapper == nil {
		baseMapper = &basicFieldMapper{} // Create a basic field mapper if none provided
	}

	if typeSubstitution == nil {
		// Create a default type substitution engine
		typeBuilder := domain.NewTypeBuilder()
		typeSubstitution = domain.NewTypeSubstitutionEngine(typeBuilder, logger)
	}

	if config == nil {
		config = DefaultGenericFieldMapperConfig()
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	// Create recursive resolver for enhanced generic support
	recursiveResolver := NewRecursiveTypeResolver(
		typeSubstitution,
		logger,
		DefaultRecursiveResolverConfig(),
	)

	// Create performance optimizer
	performanceOptimizer := NewPerformanceOptimizer(DefaultPerformanceConfig())

	mapper := &GenericFieldMapper{
		baseMapper:           baseMapper,
		typeSubstitution:     typeSubstitution,
		logger:               logger,
		config:               config,
		metrics:              NewGenericFieldMappingMetrics(),
		strategyCache:        make(map[string]domain.ConversionStrategy),
		strategies:           domain.DefaultConversionStrategies(),
		recursiveResolver:    recursiveResolver,
		performanceOptimizer: performanceOptimizer,
	}

	logger.Info("generic field mapper initialized",
		zap.Bool("caching_enabled", config.EnableCaching),
		zap.Bool("optimization_enabled", config.EnableOptimization),
		zap.Duration("timeout", config.MappingTimeout))

	return mapper
}

// Enhanced performance structures
type (
	// CacheKey represents a unique key for field mapping cache
	CacheKey struct {
		SourceType    string
		DestType      string
		Substitutions string
		Options       string
	}

	// CacheEntry holds cached field mapping results with metadata
	CacheEntry struct {
		Result    *FieldMapping
		Timestamp time.Time
		HitCount  int64
		Size      int64
	}

	// PerformanceOptimizer handles advanced performance optimizations
	PerformanceOptimizer struct {
		fieldMappingCache  sync.Map   // CacheKey -> *CacheEntry
		substitutionCache  sync.Map   // string -> domain.Type
		parallelWorkerPool *sync.Pool // Worker pool for parallel processing
		memoryPool         *sync.Pool // Memory pool for allocations
		metrics            *PerformanceMetrics
		config             *PerformanceConfig
		mu                 sync.RWMutex
	}

	// PerformanceMetrics tracks detailed performance data
	PerformanceMetrics struct {
		CacheHits          int64
		CacheMisses        int64
		CacheEvictions     int64
		ParallelOperations int64
		MemoryPoolHits     int64
		MemoryPoolMisses   int64
		MemoryAllocated    int64
		MemoryFreed        int64
		ProcessingTime     int64 // nanoseconds
		ParallelSpeedup    float64
		MemoryEfficiency   float64
		CacheEfficiency    float64
	}

	// PerformanceConfig configures performance optimizations
	PerformanceConfig struct {
		EnableFieldMappingCache bool
		MaxCacheSize            int
		CacheTTL                time.Duration
		EnableParallelMapping   bool
		MaxParallelWorkers      int
		MemoryPoolSize          int
		EnableMemoryPooling     bool
		PerformanceProfile      string // "speed", "memory", "balanced"
		AutoTune                bool
	}

	// ParallelFieldMappingJob represents a parallel field mapping task
	ParallelFieldMappingJob struct {
		Context   *GenericMappingContext
		DstField  *domain.Field
		SrcFields []*domain.Field
		Result    chan *FieldAssignment
		Error     chan error
	}
)

// NewPerformanceOptimizer creates a new performance optimizer
func NewPerformanceOptimizer(config *PerformanceConfig) *PerformanceOptimizer {
	if config == nil {
		config = DefaultPerformanceConfig()
	}

	optimizer := &PerformanceOptimizer{
		metrics: &PerformanceMetrics{},
		config:  config,
	}

	// Initialize worker pool for parallel operations
	if config.EnableParallelMapping {
		optimizer.parallelWorkerPool = &sync.Pool{
			New: func() interface{} {
				return &ParallelFieldMappingJob{
					Result: make(chan *FieldAssignment, 1),
					Error:  make(chan error, 1),
				}
			},
		}
	}

	// Initialize memory pool for efficient allocations
	if config.EnableMemoryPooling {
		optimizer.memoryPool = &sync.Pool{
			New: func() interface{} {
				return make([]*FieldAssignment, 0, 16) // Pre-allocate slice capacity
			},
		}
	}

	return optimizer
}

// DefaultPerformanceConfig returns optimized default configuration
func DefaultPerformanceConfig() *PerformanceConfig {
	return &PerformanceConfig{
		EnableFieldMappingCache: true,
		MaxCacheSize:            10000,
		CacheTTL:                1 * time.Hour,
		EnableParallelMapping:   true,
		MaxParallelWorkers:      runtime.NumCPU(),
		MemoryPoolSize:          1000,
		EnableMemoryPooling:     true,
		PerformanceProfile:      "balanced",
		AutoTune:                true,
	}
}

// generateCacheKey creates a unique cache key for field mapping
func (po *PerformanceOptimizer) generateCacheKey(srcType, dstType domain.Type,
	substitutions map[string]domain.Type, options *FieldMappingOptions) CacheKey {

	// Create deterministic string representations
	var substitutionsStr strings.Builder
	if len(substitutions) > 0 {
		keys := make([]string, 0, len(substitutions))
		for k := range substitutions {
			keys = append(keys, k)
		}
		sort.Strings(keys) // Ensure deterministic ordering

		for _, k := range keys {
			substitutionsStr.WriteString(k)
			substitutionsStr.WriteString(":")
			substitutionsStr.WriteString(substitutions[k].String())
			substitutionsStr.WriteString(";")
		}
	}

	var optionsStr strings.Builder
	if options != nil {
		optionsStr.WriteString(fmt.Sprintf("priv:%t,conv:%t,val:%t,ignore:%t",
			options.IncludePrivateFields, options.UseTypeConversion,
			options.ValidateTypes, options.IgnoreUnmatched))
	}

	return CacheKey{
		SourceType:    srcType.String(),
		DestType:      dstType.String(),
		Substitutions: substitutionsStr.String(),
		Options:       optionsStr.String(),
	}
}

// getCachedFieldMapping retrieves cached field mapping if available
func (po *PerformanceOptimizer) getCachedFieldMapping(key CacheKey) (*FieldMapping, bool) {
	if !po.config.EnableFieldMappingCache {
		return nil, false
	}

	if entry, ok := po.fieldMappingCache.Load(key); ok {
		cacheEntry := entry.(*CacheEntry)

		// Check TTL
		if time.Since(cacheEntry.Timestamp) > po.config.CacheTTL {
			po.fieldMappingCache.Delete(key)
			atomic.AddInt64(&po.metrics.CacheEvictions, 1)
			return nil, false
		}

		// Update hit count and metrics
		atomic.AddInt64(&cacheEntry.HitCount, 1)
		atomic.AddInt64(&po.metrics.CacheHits, 1)

		return cacheEntry.Result, true
	}

	atomic.AddInt64(&po.metrics.CacheMisses, 1)
	return nil, false
}

// cacheFieldMapping stores field mapping result in cache
func (po *PerformanceOptimizer) cacheFieldMapping(key CacheKey, result *FieldMapping) {
	if !po.config.EnableFieldMappingCache {
		return
	}

	// Calculate entry size for memory tracking
	size := int64(len(result.Assignments) * 64) // Rough estimate

	entry := &CacheEntry{
		Result:    result,
		Timestamp: time.Now(),
		HitCount:  0,
		Size:      size,
	}

	po.fieldMappingCache.Store(key, entry)
	atomic.AddInt64(&po.metrics.MemoryAllocated, size)

	// Check cache size and evict if necessary
	go po.maintainCacheSize()
}

// maintainCacheSize ensures cache doesn't exceed size limits
func (po *PerformanceOptimizer) maintainCacheSize() {
	po.mu.Lock()
	defer po.mu.Unlock()

	var count int
	po.fieldMappingCache.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	if count > po.config.MaxCacheSize {
		// Simple LRU-like eviction based on timestamp and hit count
		type evictionCandidate struct {
			key   interface{}
			entry *CacheEntry
			score float64
		}

		var candidates []evictionCandidate
		po.fieldMappingCache.Range(func(key, value interface{}) bool {
			entry := value.(*CacheEntry)
			// Score based on age and hit count (lower score = more likely to evict)
			age := time.Since(entry.Timestamp).Hours()
			score := float64(entry.HitCount) / (age + 1)
			candidates = append(candidates, evictionCandidate{key, entry, score})
			return true
		})

		// Sort by score and evict lowest scoring entries
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].score < candidates[j].score
		})

		toEvict := count - po.config.MaxCacheSize + 100 // Evict extra to avoid frequent evictions
		if toEvict > len(candidates) {
			toEvict = len(candidates)
		}

		for i := 0; i < toEvict; i++ {
			po.fieldMappingCache.Delete(candidates[i].key)
			atomic.AddInt64(&po.metrics.MemoryFreed, candidates[i].entry.Size)
			atomic.AddInt64(&po.metrics.CacheEvictions, 1)
		}
	}
}

// processFieldMappingsParallel processes field mappings in parallel for independent operations
func (po *PerformanceOptimizer) processFieldMappingsParallel(
	gfm *GenericFieldMapper,
	dstFields []*domain.Field,
	srcFields []*domain.Field,
	context *GenericMappingContext,
) ([]*FieldAssignment, error) {

	if !po.config.EnableParallelMapping || len(dstFields) < 4 {
		// Fall back to sequential processing for small sets
		return po.processFieldMappingsSequential(gfm, dstFields, srcFields, context)
	}

	atomic.AddInt64(&po.metrics.ParallelOperations, 1)

	// Calculate optimal number of workers
	numWorkers := po.config.MaxParallelWorkers
	if len(dstFields) < numWorkers {
		numWorkers = len(dstFields)
	}

	// Create jobs channel and results collection
	jobs := make(chan *ParallelFieldMappingJob, len(dstFields))
	var wg sync.WaitGroup

	// Get assignment slice from memory pool
	var assignments []*FieldAssignment
	if po.config.EnableMemoryPooling {
		if poolSlice := po.memoryPool.Get(); poolSlice != nil {
			assignments = poolSlice.([]*FieldAssignment)[:0] // Reset length but keep capacity
			atomic.AddInt64(&po.metrics.MemoryPoolHits, 1)
		} else {
			assignments = make([]*FieldAssignment, 0, len(dstFields))
			atomic.AddInt64(&po.metrics.MemoryPoolMisses, 1)
		}
	} else {
		assignments = make([]*FieldAssignment, 0, len(dstFields))
	}

	results := make([]*FieldAssignment, len(dstFields))
	errors := make([]error, len(dstFields))

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				assignment, err := gfm.generateFieldAssignment(job.DstField, job.SrcFields, job.Context)
				if err != nil {
					job.Error <- err
				} else {
					job.Result <- assignment
				}

				// Return job to pool
				if po.parallelWorkerPool != nil {
					po.parallelWorkerPool.Put(job)
				}
			}
		}()
	}

	// Submit jobs
	startTime := time.Now()
	for i, dstField := range dstFields {
		var job *ParallelFieldMappingJob
		if po.parallelWorkerPool != nil {
			job = po.parallelWorkerPool.Get().(*ParallelFieldMappingJob)
		} else {
			job = &ParallelFieldMappingJob{
				Result: make(chan *FieldAssignment, 1),
				Error:  make(chan error, 1),
			}
		}

		job.Context = context
		job.DstField = dstField
		job.SrcFields = srcFields

		jobs <- job

		// Collect result immediately in goroutine to avoid blocking
		go func(index int, j *ParallelFieldMappingJob) {
			select {
			case result := <-j.Result:
				results[index] = result
			case err := <-j.Error:
				errors[index] = err
			case <-time.After(30 * time.Second): // Timeout protection
				errors[index] = fmt.Errorf("field mapping timeout for field %s", j.DstField.Name)
			}
		}(i, job)
	}

	// Close jobs channel and wait for workers
	close(jobs)
	wg.Wait()

	// Calculate parallel speedup
	processingTime := time.Since(startTime)
	atomic.AddInt64(&po.metrics.ProcessingTime, processingTime.Nanoseconds())

	// Collect final results and check for errors
	for i, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("parallel field mapping failed for field %d: %w", i, err)
		}
		if results[i] != nil {
			assignments = append(assignments, results[i])
		}
	}

	return assignments, nil
}

// processFieldMappingsSequential processes field mappings sequentially (fallback)
func (po *PerformanceOptimizer) processFieldMappingsSequential(
	gfm *GenericFieldMapper,
	dstFields []*domain.Field,
	srcFields []*domain.Field,
	context *GenericMappingContext,
) ([]*FieldAssignment, error) {

	assignments := make([]*FieldAssignment, 0, len(dstFields))

	for _, dstField := range dstFields {
		assignment, err := gfm.generateFieldAssignment(dstField, srcFields, context)
		if err != nil {
			return nil, fmt.Errorf("field mapping failed for field %s: %w", dstField.Name, err)
		}
		if assignment != nil {
			assignments = append(assignments, assignment)
		}
	}

	return assignments, nil
}

// GetMetrics returns current performance metrics
func (po *PerformanceOptimizer) GetMetrics() *PerformanceMetrics {
	return &PerformanceMetrics{
		CacheHits:          atomic.LoadInt64(&po.metrics.CacheHits),
		CacheMisses:        atomic.LoadInt64(&po.metrics.CacheMisses),
		CacheEvictions:     atomic.LoadInt64(&po.metrics.CacheEvictions),
		ParallelOperations: atomic.LoadInt64(&po.metrics.ParallelOperations),
		MemoryPoolHits:     atomic.LoadInt64(&po.metrics.MemoryPoolHits),
		MemoryPoolMisses:   atomic.LoadInt64(&po.metrics.MemoryPoolMisses),
		MemoryAllocated:    atomic.LoadInt64(&po.metrics.MemoryAllocated),
		MemoryFreed:        atomic.LoadInt64(&po.metrics.MemoryFreed),
		ProcessingTime:     atomic.LoadInt64(&po.metrics.ProcessingTime),
		ParallelSpeedup:    po.calculateParallelSpeedup(),
		MemoryEfficiency:   po.calculateMemoryEfficiency(),
		CacheEfficiency:    po.calculateCacheEfficiency(),
	}
}

// calculateParallelSpeedup estimates speedup from parallel processing
func (po *PerformanceOptimizer) calculateParallelSpeedup() float64 {
	parallelOps := atomic.LoadInt64(&po.metrics.ParallelOperations)
	if parallelOps == 0 {
		return 1.0
	}

	// Estimate based on number of workers and typical efficiency
	workers := float64(po.config.MaxParallelWorkers)
	efficiency := 0.8 // Typical parallel efficiency
	return workers * efficiency
}

// calculateMemoryEfficiency calculates memory pool efficiency
func (po *PerformanceOptimizer) calculateMemoryEfficiency() float64 {
	hits := atomic.LoadInt64(&po.metrics.MemoryPoolHits)
	misses := atomic.LoadInt64(&po.metrics.MemoryPoolMisses)
	total := hits + misses

	if total == 0 {
		return 1.0
	}

	return float64(hits) / float64(total)
}

// calculateCacheEfficiency calculates cache hit rate
func (po *PerformanceOptimizer) calculateCacheEfficiency() float64 {
	hits := atomic.LoadInt64(&po.metrics.CacheHits)
	misses := atomic.LoadInt64(&po.metrics.CacheMisses)
	total := hits + misses

	if total == 0 {
		return 0.0
	}

	return float64(hits) / float64(total)
}

// ResetMetrics resets all performance metrics
func (po *PerformanceOptimizer) ResetMetrics() {
	atomic.StoreInt64(&po.metrics.CacheHits, 0)
	atomic.StoreInt64(&po.metrics.CacheMisses, 0)
	atomic.StoreInt64(&po.metrics.CacheEvictions, 0)
	atomic.StoreInt64(&po.metrics.ParallelOperations, 0)
	atomic.StoreInt64(&po.metrics.MemoryPoolHits, 0)
	atomic.StoreInt64(&po.metrics.MemoryPoolMisses, 0)
	atomic.StoreInt64(&po.metrics.MemoryAllocated, 0)
	atomic.StoreInt64(&po.metrics.MemoryFreed, 0)
	atomic.StoreInt64(&po.metrics.ProcessingTime, 0)
}

// MapGenericFields maps fields between generic source and destination types.
func (gfm *GenericFieldMapper) MapGenericFields(
	srcType domain.Type,
	dstType domain.Type,
	typeSubstitutions map[string]domain.Type,
	options *FieldMappingOptions,
) (*FieldMapping, error) {
	if options == nil {
		return nil, ErrFieldMappingOptionsNil
	}

	startTime := time.Now()
	gfm.metricsMux.Lock()
	gfm.metrics.TotalMappings++
	gfm.metricsMux.Unlock()

	gfm.logger.Debug("starting generic field mapping",
		zap.String("source_type", srcType.String()),
		zap.String("destination_type", dstType.String()),
		zap.Int("substitutions", len(typeSubstitutions)))

	// Create mapping context
	context := &GenericMappingContext{
		SourceType:          srcType,
		DestinationType:     dstType,
		TypeSubstitutions:   typeSubstitutions,
		AnnotationOverrides: options.Annotations,
		MappingStrategy:     SelectOptimalMappingStrategy,
		Options:             options,
	}

	// Perform type substitutions
	substitutedSrcType, err := gfm.substituteTypeIfNeeded(srcType, typeSubstitutions)
	if err != nil {
		gfm.metricsMux.Lock()
		gfm.metrics.FailedMappings++
		gfm.metricsMux.Unlock()
		return nil, fmt.Errorf("%w: source type substitution: %s", ErrTypeSubstitutionInMapping, err.Error())
	}

	substitutedDstType, err := gfm.substituteTypeIfNeeded(dstType, typeSubstitutions)
	if err != nil {
		gfm.metricsMux.Lock()
		gfm.metrics.FailedMappings++
		gfm.metricsMux.Unlock()
		return nil, fmt.Errorf("%w: destination type substitution: %s", ErrTypeSubstitutionInMapping, err.Error())
	}

	// Update context with substituted types
	context.SubstitutedSourceType = substitutedSrcType
	context.SubstitutedDestType = substitutedDstType

	// Generate field mappings
	fieldMapping, err := gfm.generateFieldMapping(context)
	if err != nil {
		gfm.metricsMux.Lock()
		gfm.metrics.FailedMappings++
		gfm.metricsMux.Unlock()
		return nil, fmt.Errorf("%w: %s", ErrGenericFieldMappingFailed, err.Error())
	}

	// Update metrics
	mappingTime := time.Since(startTime)
	gfm.metricsMux.Lock()
	gfm.metrics.SuccessfulMappings++
	gfm.metrics.TotalMappingTime += mappingTime
	gfm.metrics.AverageMappingTime = gfm.metrics.TotalMappingTime / time.Duration(gfm.metrics.TotalMappings)
	gfm.metricsMux.Unlock()

	gfm.logger.Info("generic field mapping completed",
		zap.String("source_type", srcType.String()),
		zap.String("destination_type", dstType.String()),
		zap.Duration("mapping_time", mappingTime),
		zap.Int("field_assignments", len(fieldMapping.Assignments)))

	return fieldMapping, nil
}

// MapFields implements the GenericFieldMapper interface from the emitter package.
// This method provides compatibility with the emitter's expected interface.
func (gfm *GenericFieldMapper) MapFields(sourceType, destType domain.Type, annotations map[string]string) ([]*GenericFieldMapping, error) {
	// Convert annotations to FieldMappingOptions
	options := DefaultFieldMappingOptions()

	// Parse annotations into custom mappings and field-specific annotations
	fieldAnnotations := make(map[string]*Annotation)
	for key, value := range annotations {
		// Handle common annotation patterns
		if strings.HasPrefix(key, "map:") {
			fieldName := strings.TrimPrefix(key, "map:")
			if fieldAnnotations[fieldName] == nil {
				fieldAnnotations[fieldName] = &Annotation{}
			}
			fieldAnnotations[fieldName].Map = value
		} else if strings.HasPrefix(key, "skip:") {
			fieldName := strings.TrimPrefix(key, "skip:")
			if fieldAnnotations[fieldName] == nil {
				fieldAnnotations[fieldName] = &Annotation{}
			}
			fieldAnnotations[fieldName].Skip = value == "true"
		} else if strings.HasPrefix(key, "converter:") {
			fieldName := strings.TrimPrefix(key, "converter:")
			if fieldAnnotations[fieldName] == nil {
				fieldAnnotations[fieldName] = &Annotation{}
			}
			fieldAnnotations[fieldName].Converter = value
		} else {
			// Add to custom mappings
			options.CustomMappings[key] = value
		}
	}

	options.Annotations = fieldAnnotations

	// Check cache first using performance optimizer
	cacheKey := gfm.performanceOptimizer.generateCacheKey(sourceType, destType, nil, options)
	if cachedMapping, found := gfm.performanceOptimizer.getCachedFieldMapping(cacheKey); found {
		return gfm.convertFieldMappingToGenericMappings(cachedMapping), nil
	}

	// Use MapGenericFields for the actual implementation
	fieldMapping, err := gfm.MapGenericFields(sourceType, destType, nil, options)
	if err != nil {
		return nil, fmt.Errorf("field mapping failed: %w", err)
	}

	// Cache the result
	gfm.performanceOptimizer.cacheFieldMapping(cacheKey, fieldMapping)

	// Convert to expected return type
	return gfm.convertFieldMappingToGenericMappings(fieldMapping), nil
}

// convertFieldMappingToGenericMappings converts internal FieldMapping to GenericFieldMapping slice
func (gfm *GenericFieldMapper) convertFieldMappingToGenericMappings(mapping *FieldMapping) []*GenericFieldMapping {
	if mapping == nil || len(mapping.Assignments) == 0 {
		return []*GenericFieldMapping{}
	}

	genericMappings := make([]*GenericFieldMapping, 0, len(mapping.Assignments))

	for _, assignment := range mapping.Assignments {
		if assignment == nil {
			continue
		}

		genericMapping := &GenericFieldMapping{
			SourceField: assignment.SourcePath,
			DestField:   assignment.DestField.Name,
			SourceType:  assignment.SourceField.Type,
			DestType:    assignment.DestField.Type,
			Converter:   assignment.Converter,
			Validation:  assignment.Validation,
			Annotations: make(map[string]string),
		}

		// Add any additional metadata from the assignment
		if assignment.Code != "" {
			genericMapping.Annotations["code"] = assignment.Code
		}
		if assignment.ErrorHandling != "" {
			genericMapping.Annotations["error_handling"] = string(assignment.ErrorHandling)
		}

		genericMappings = append(genericMappings, genericMapping)
	}

	return genericMappings
}

// ValidateMapping validates a GenericFieldMapping (satisfies the interface)
func (gfm *GenericFieldMapper) ValidateMapping(mapping *GenericFieldMapping) error {
	if mapping == nil {
		return errors.New("mapping cannot be nil")
	}

	if mapping.SourceField == "" {
		return errors.New("source field cannot be empty")
	}

	if mapping.DestField == "" {
		return errors.New("destination field cannot be empty")
	}

	if mapping.SourceType == nil {
		return errors.New("source type cannot be nil")
	}

	if mapping.DestType == nil {
		return errors.New("destination type cannot be nil")
	}

	// Validate type compatibility
	if !gfm.typesCompatible(mapping.SourceType, mapping.DestType, nil) {
		return fmt.Errorf("types are not compatible: %s -> %s",
			mapping.SourceType.String(), mapping.DestType.String())
	}

	return nil
}

// substituteTypeIfNeeded applies type substitutions if the type is generic, with enhanced recursive support.
func (gfm *GenericFieldMapper) substituteTypeIfNeeded(
	typ domain.Type,
	typeSubstitutions map[string]domain.Type,
) (domain.Type, error) {
	if !typ.Generic() || len(typeSubstitutions) == 0 {
		return typ, nil
	}

	gfm.metricsMux.Lock()
	gfm.metrics.TypeSubstitutions++
	gfm.metricsMux.Unlock()

	// Enhanced: Use recursive resolver for deeply nested generic types
	if gfm.isDeeplyNestedGeneric(typ) {
		result, err := gfm.recursiveResolver.ResolveNestedGenericType(typ, typeSubstitutions)
		if err != nil {
			return nil, fmt.Errorf("recursive type resolution failed: %w", err)
		}
		return result.ResolvedType, nil
	}

	// Fallback to standard type substitution for simpler cases
	// Convert the substitution map to the format expected by TypeSubstitutionEngine
	typeParams := make([]domain.TypeParam, 0, len(typeSubstitutions))
	typeArgs := make([]domain.Type, 0, len(typeSubstitutions))

	for paramName, concreteType := range typeSubstitutions {
		typeParam := domain.TypeParam{
			Name:       paramName,
			Constraint: domain.NewBasicType("any", reflect.Invalid), // Default constraint as Type
		}
		typeParams = append(typeParams, typeParam)
		typeArgs = append(typeArgs, concreteType)
	}

	// Perform type substitution
	result, err := gfm.typeSubstitution.SubstituteType(typ, typeParams, typeArgs)
	if err != nil {
		return nil, fmt.Errorf("type substitution failed: %w", err)
	}

	return result.SubstitutedType, nil
}

// generateFieldMapping generates the actual field mapping.
func (gfm *GenericFieldMapper) generateFieldMapping(context *GenericMappingContext) (*FieldMapping, error) {
	srcType := context.SubstitutedSourceType
	dstType := context.SubstitutedDestType

	// Extract fields from source and destination types
	srcFields, err := gfm.extractTypeFields(srcType)
	if err != nil {
		return nil, fmt.Errorf("failed to extract source fields: %w", err)
	}

	dstFields, err := gfm.extractTypeFields(dstType)
	if err != nil {
		return nil, fmt.Errorf("failed to extract destination fields: %w", err)
	}

	// Generate field assignments using performance optimizer
	assignments, err := gfm.performanceOptimizer.processFieldMappingsParallel(
		gfm, dstFields, srcFields, context)
	if err != nil {
		// Fall back to sequential processing if parallel fails
		gfm.logger.Warn("parallel processing failed, falling back to sequential",
			zap.Error(err))

		assignments = make([]*FieldAssignment, 0)
		for _, dstField := range dstFields {
			assignment, assignErr := gfm.generateFieldAssignment(dstField, srcFields, context)
			if assignErr != nil {
				if context.Options.IgnoreUnmatched {
					gfm.logger.Warn("skipping unmatched field",
						zap.String("field", dstField.Name),
						zap.Error(assignErr))
					continue
				}
				return nil, fmt.Errorf("failed to map field %s: %w", dstField.Name, assignErr)
			}

			if assignment != nil {
				assignments = append(assignments, assignment)
			}
		}
	}

	// Apply optimizations if enabled
	if gfm.config.EnableOptimization {
		assignments = gfm.optimizeAssignments(assignments, context)
		gfm.metricsMux.Lock()
		gfm.metrics.OptimizationsApplied++
		gfm.metricsMux.Unlock()
	}

	return &FieldMapping{
		SourceType:      context.SourceType,
		DestinationType: context.DestinationType,
		Assignments:     assignments,
		Context:         context,
		GeneratedAt:     time.Now(),
	}, nil
}

// extractTypeFields extracts fields from a type.
func (gfm *GenericFieldMapper) extractTypeFields(typ domain.Type) ([]*domain.Field, error) {
	switch typ.Kind() {
	case domain.KindStruct:
		if structType, ok := typ.(*domain.StructType); ok {
			fields := structType.Fields()
			result := make([]*domain.Field, len(fields))
			for i, field := range fields {
				result[i] = &domain.Field{
					Name:     field.Name,
					Type:     field.Type,
					Position: field.Position,
					Exported: field.Exported,
				}
			}
			return result, nil
		}
		return nil, fmt.Errorf("expected StructType, got %T", typ)
	default:
		return nil, fmt.Errorf("%w: type kind %s", ErrGenericTypeNotSupported, typ.Kind().String())
	}
}

// generateFieldAssignment generates an assignment for a destination field.
func (gfm *GenericFieldMapper) generateFieldAssignment(
	dstField *domain.Field,
	srcFields []*domain.Field,
	context *GenericMappingContext,
) (*FieldAssignment, error) {
	// Check for annotation overrides
	if annotation, found := context.AnnotationOverrides[dstField.Name]; found {
		if annotation.Skip {
			return &FieldAssignment{
				DestField:      dstField,
				AssignmentType: SkipAssignment,
				Code:           fmt.Sprintf("// Skipping field %s", dstField.Name),
			}, nil
		}

		if annotation.Map != "" {
			return gfm.generateMappedAssignment(dstField, srcFields, annotation.Map, context)
		}

		if annotation.Converter != "" {
			return gfm.generateConverterAssignment(dstField, srcFields, annotation.Converter, context)
		}

		if annotation.Literal != "" {
			return gfm.generateLiteralAssignment(dstField, annotation.Literal, context)
		}
	}

	// Try to find matching source field
	for _, srcField := range srcFields {
		if gfm.fieldsMatch(srcField, dstField, context) {
			return gfm.generateDirectAssignment(srcField, dstField, context)
		}
	}

	// Try to generate nested field assignment
	if nestedAssignment := gfm.generateNestedFieldAssignment(dstField, srcFields, context); nestedAssignment != nil {
		return nestedAssignment, nil
	}

	// No match found
	if context.Options.IgnoreUnmatched {
		return nil, nil
	}

	return nil, fmt.Errorf("no matching source field for destination field %s", dstField.Name)
}

// fieldsMatch checks if two fields can be mapped to each other.
func (gfm *GenericFieldMapper) fieldsMatch(srcField, dstField *domain.Field, context *GenericMappingContext) bool {
	// Check exact name match first
	if srcField.Name == dstField.Name {
		if !context.Options.ValidateTypes {
			return true
		}
		return gfm.typesCompatible(srcField.Type, dstField.Type, context)
	}

	// Check for common field mapping patterns
	if gfm.fieldsMatchByPattern(srcField, dstField, context) {
		return true
	}

	return false
}

// typesCompatible checks if two types are compatible for assignment.
func (gfm *GenericFieldMapper) typesCompatible(srcType, dstType domain.Type, context *GenericMappingContext) bool {
	if srcType == nil || dstType == nil {
		return false
	}

	// Apply type substitutions before comparing
	substitutedSrcType := gfm.applyTypeSubstitution(srcType, context)
	substitutedDstType := gfm.applyTypeSubstitution(dstType, context)

	// Direct assignability
	if substitutedSrcType.AssignableTo(substitutedDstType) {
		return true
	}

	// Type conversion allowed
	if context != nil && context.Options.UseTypeConversion {
		// Check if types are convertible
		if gfm.typesConvertible(substitutedSrcType, substitutedDstType) {
			return true
		}
	}

	return false
}

// applyTypeSubstitution applies type parameter substitutions to a type.
func (gfm *GenericFieldMapper) applyTypeSubstitution(typ domain.Type, context *GenericMappingContext) domain.Type {
	if typ == nil || context == nil || len(context.TypeSubstitutions) == 0 {
		return typ
	}

	// Handle generic types by substitution
	if genericType, ok := typ.(*domain.GenericType); ok {
		if substitution, exists := context.TypeSubstitutions[genericType.Name()]; exists {
			return substitution
		}
		return typ
	}

	// Handle other types recursively if needed
	switch typ.Kind() {
	case domain.KindSlice:
		if sliceType, ok := typ.(*domain.SliceType); ok {
			elemType := gfm.applyTypeSubstitution(sliceType.Elem(), context)
			return domain.NewSliceType(elemType, sliceType.Package())
		}
	case domain.KindPointer:
		if pointerType, ok := typ.(*domain.PointerType); ok {
			elemType := gfm.applyTypeSubstitution(pointerType.Elem(), context)
			return domain.NewPointerType(elemType, pointerType.Package())
		}
		// Note: Map type substitution is not fully implemented in the domain package yet
		// case domain.KindMap: ... would go here when available
	}

	return typ
}

// typesConvertible checks if types can be converted, with enhanced support for nested generics.
func (gfm *GenericFieldMapper) typesConvertible(srcType, dstType domain.Type) bool {
	// Basic type conversions
	if srcType.Kind() == domain.KindBasic && dstType.Kind() == domain.KindBasic {
		return true
	}

	// Pointer conversions - enhanced to handle pointer/value combinations
	if srcType.Kind() == domain.KindPointer && dstType.Kind() == domain.KindPointer {
		srcElem := srcType.(*domain.PointerType).Elem()
		dstElem := dstType.(*domain.PointerType).Elem()
		return gfm.typesConvertible(srcElem, dstElem)
	}

	// Pointer to value conversion (*T → T)
	if srcType.Kind() == domain.KindPointer && dstType.Kind() != domain.KindPointer {
		srcElem := srcType.(*domain.PointerType).Elem()
		return gfm.typesConvertible(srcElem, dstType)
	}

	// Value to pointer conversion (T → *T)
	if srcType.Kind() != domain.KindPointer && dstType.Kind() == domain.KindPointer {
		dstElem := dstType.(*domain.PointerType).Elem()
		return gfm.typesConvertible(srcType, dstElem)
	}

	// Slice conversions
	if srcType.Kind() == domain.KindSlice && dstType.Kind() == domain.KindSlice {
		srcElem := srcType.(*domain.SliceType).Elem()
		dstElem := dstType.(*domain.SliceType).Elem()
		return gfm.typesConvertible(srcElem, dstElem)
	}

	// Enhanced: Map conversions for nested generics
	if srcType.Kind() == domain.KindMap && dstType.Kind() == domain.KindMap {
		return gfm.typesConvertibleForMaps(srcType, dstType)
	}

	// Enhanced: Generic type conversions
	if srcType.Kind() == domain.KindGeneric || dstType.Kind() == domain.KindGeneric {
		return gfm.typesConvertibleForGenerics(srcType, dstType)
	}

	// Enhanced: Named type conversions with generic support
	if srcType.Kind() == domain.KindNamed || dstType.Kind() == domain.KindNamed {
		return gfm.typesConvertibleForNamedTypes(srcType, dstType)
	}

	// Interface{} conversions
	if gfm.isInterfaceEmptyType(srcType) || gfm.isInterfaceEmptyType(dstType) {
		return true // interface{} is convertible to/from any type
	}

	return false
}

// generateDirectAssignment generates a direct field assignment.
func (gfm *GenericFieldMapper) generateDirectAssignment(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) (*FieldAssignment, error) {
	// Start with default direct assignment
	assignmentCode := fmt.Sprintf("dst.%s = src.%s", dstField.Name, srcField.Name)
	assignmentType := DirectAssignment

	// Check if types are directly assignable
	if srcField.Type.AssignableTo(dstField.Type) {
		return &FieldAssignment{
			SourceField:    srcField,
			DestField:      dstField,
			AssignmentType: assignmentType,
			Code:           assignmentCode,
		}, nil
	}

	// Apply advanced conversion scenarios
	conversionCode := gfm.generateAdvancedConversion(srcField, dstField, context)
	if conversionCode != "" {
		// Determine appropriate assignment type based on the types being converted
		assignmentType := ConversionAssignment
		srcKind := srcField.Type.Kind()
		dstKind := dstField.Type.Kind()

		// Use specific assignment types for certain conversions
		if srcKind == domain.KindMap && dstKind == domain.KindMap {
			assignmentType = MapAssignment
		} else if srcKind == domain.KindSlice && dstKind == domain.KindSlice {
			assignmentType = SliceAssignment
		}

		return &FieldAssignment{
			SourceField:    srcField,
			DestField:      dstField,
			AssignmentType: assignmentType,
			Code:           conversionCode,
		}, nil
	}

	// Fall back to basic type conversion if enabled
	if context.Options.UseTypeConversion {
		dstTypeName := gfm.getTypeName(dstField.Type)
		assignmentCode = fmt.Sprintf("dst.%s = %s(src.%s)", dstField.Name, dstTypeName, srcField.Name)
		assignmentType = ConversionAssignment
	}

	return &FieldAssignment{
		SourceField:    srcField,
		DestField:      dstField,
		AssignmentType: assignmentType,
		Code:           assignmentCode,
	}, nil
}

// generateAdvancedConversion handles advanced conversion scenarios
func (gfm *GenericFieldMapper) generateAdvancedConversion(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) string {
	srcKind := srcField.Type.Kind()
	dstKind := dstField.Type.Kind()

	// Handle slice-to-slice conversions
	if srcKind == domain.KindSlice && dstKind == domain.KindSlice {
		return gfm.generateSliceConversionCode(srcField, dstField, context)
	}

	// Handle map-to-map conversions
	if srcKind == domain.KindMap && dstKind == domain.KindMap {
		return gfm.generateMapConversionCode(srcField, dstField, context)
	}

	// Handle interface{} to concrete type conversions
	if gfm.isInterfaceToConcreteConversion(srcField.Type, dstField.Type) {
		return gfm.generateInterfaceToConcreteConversion(srcField, dstField, context)
	}

	// Handle concrete to interface{} conversions
	if gfm.isConcreteToInterfaceConversion(srcField.Type, dstField.Type) {
		return gfm.generateConcreteToInterfaceConversion(srcField, dstField, context)
	}

	// Handle generic type conversions
	if srcField.Type.Generic() || dstField.Type.Generic() {
		return gfm.generateGenericTypeConversion(srcField, dstField, context)
	}

	// Handle channel conversions
	if gfm.isChannelType(srcField.Type) && gfm.isChannelType(dstField.Type) {
		return gfm.generateChannelConversionCode(srcField, dstField, context)
	}

	// Handle function conversions
	if srcKind == domain.KindFunction && dstKind == domain.KindFunction {
		return gfm.generateFunctionConversionCode(srcField, dstField, context)
	}

	// Handle pointer conversions
	if gfm.isPointerConversion(srcField.Type, dstField.Type) {
		return gfm.generatePointerConversion(srcField, dstField, context)
	}

	// No advanced conversion applicable
	return ""
}

// isInterfaceToConcreteConversion checks if conversion is from interface{} to concrete type
func (gfm *GenericFieldMapper) isInterfaceToConcreteConversion(srcType, dstType domain.Type) bool {
	return gfm.isInterfaceEmptyType(srcType) && !gfm.isInterfaceEmptyType(dstType)
}

// isConcreteToInterfaceConversion checks if conversion is from concrete type to interface{}
func (gfm *GenericFieldMapper) isConcreteToInterfaceConversion(srcType, dstType domain.Type) bool {
	return !gfm.isInterfaceEmptyType(srcType) && gfm.isInterfaceEmptyType(dstType)
}

// isChannelType checks if a type is a channel type
func (gfm *GenericFieldMapper) isChannelType(typ domain.Type) bool {
	// Check for channel types based on string representation since domain might not have full channel support
	typeStr := typ.String()
	return strings.Contains(typeStr, "chan ") || strings.HasPrefix(typeStr, "<-chan") || strings.HasSuffix(typeStr, "chan<-")
}

// isPointerConversion checks if conversion involves pointer types
func (gfm *GenericFieldMapper) isPointerConversion(srcType, dstType domain.Type) bool {
	return (srcType.Kind() == domain.KindPointer) != (dstType.Kind() == domain.KindPointer)
}

// generateInterfaceToConcreteConversion handles interface{} to concrete type conversion
func (gfm *GenericFieldMapper) generateInterfaceToConcreteConversion(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) string {
	dstTypeName := gfm.getTypeName(dstField.Type)
	return fmt.Sprintf(`// Interface{} to concrete type conversion: %s -> %s
	if typedValue, ok := src.%s.(%s); ok {
		dst.%s = typedValue
	} else {
		// TODO: Handle type assertion failure - could set zero value or return error
		// var zero %s
		// dst.%s = zero
	}`,
		srcField.Type.String(), dstField.Type.String(),
		srcField.Name, dstTypeName, dstField.Name,
		dstTypeName, dstField.Name)
}

// generateConcreteToInterfaceConversion handles concrete type to interface{} conversion
func (gfm *GenericFieldMapper) generateConcreteToInterfaceConversion(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) string {
	return fmt.Sprintf(`// Concrete to interface{} conversion: %s -> %s
	dst.%s = src.%s`,
		srcField.Type.String(), dstField.Type.String(),
		dstField.Name, srcField.Name)
}

// generateGenericTypeConversion handles generic type conversions
func (gfm *GenericFieldMapper) generateGenericTypeConversion(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) string {
	// Apply type substitution
	substitutedSrcType := gfm.applyTypeSubstitution(srcField.Type, context)
	substitutedDstType := gfm.applyTypeSubstitution(dstField.Type, context)

	// Check if substituted types are compatible
	if gfm.typesCompatible(substitutedSrcType, substitutedDstType, context) {
		conversion := gfm.generateTypeConversion(substitutedSrcType, substitutedDstType)
		return fmt.Sprintf(`// Generic type conversion with substitution: %s -> %s
		dst.%s = %s`,
			srcField.Type.String(), dstField.Type.String(),
			dstField.Name, gfm.applyConversionToValue(conversion, fmt.Sprintf("src.%s", srcField.Name)))
	}

	// Handle complex generic conversions
	return fmt.Sprintf(`// Complex generic conversion: %s -> %s
	// TODO: Implement complex generic type conversion logic
	// This may require custom converter functions or additional type constraints
	dst.%s = convertGenericType(src.%s)`,
		srcField.Type.String(), dstField.Type.String(),
		dstField.Name, srcField.Name)
}

// generatePointerConversion handles pointer/value conversions
func (gfm *GenericFieldMapper) generatePointerConversion(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) string {
	srcIsPtr := srcField.Type.Kind() == domain.KindPointer
	dstIsPtr := dstField.Type.Kind() == domain.KindPointer

	if srcIsPtr && !dstIsPtr {
		// Pointer to value conversion
		return fmt.Sprintf(`// Pointer to value conversion: %s -> %s
		if src.%s != nil {
			dst.%s = *src.%s
		} else {
			// TODO: Handle nil pointer - could set zero value or return error
			// var zero %s
			// dst.%s = zero
		}`,
			srcField.Type.String(), dstField.Type.String(),
			srcField.Name, dstField.Name, srcField.Name,
			gfm.getTypeName(dstField.Type), dstField.Name)
	}

	if !srcIsPtr && dstIsPtr {
		// Value to pointer conversion
		return fmt.Sprintf(`// Value to pointer conversion: %s -> %s
		dst.%s = &src.%s`,
			srcField.Type.String(), dstField.Type.String(),
			dstField.Name, srcField.Name)
	}

	// Should not reach here with current logic, but handle gracefully
	return fmt.Sprintf(`// Fallback pointer conversion: %s -> %s
	dst.%s = src.%s`,
		srcField.Type.String(), dstField.Type.String(),
		dstField.Name, srcField.Name)
}

// generateMappedAssignment generates an assignment using custom field mapping.
func (gfm *GenericFieldMapper) generateMappedAssignment(
	dstField *domain.Field,
	srcFields []*domain.Field,
	sourcePath string,
	context *GenericMappingContext,
) (*FieldAssignment, error) {
	// Find the mapped source field
	var sourceField *domain.Field
	for _, field := range srcFields {
		if field.Name == sourcePath {
			sourceField = field
			break
		}
	}

	if sourceField == nil {
		return nil, fmt.Errorf("mapped source field %s not found", sourcePath)
	}

	assignmentCode := fmt.Sprintf("dst.%s = src.%s", dstField.Name, sourcePath)

	// Add type conversion if needed
	if !sourceField.Type.AssignableTo(dstField.Type) && context.Options.UseTypeConversion {
		dstTypeName := gfm.getTypeName(dstField.Type)
		assignmentCode = fmt.Sprintf("dst.%s = %s(src.%s)", dstField.Name, dstTypeName, sourcePath)
	}

	return &FieldAssignment{
		SourceField:    sourceField,
		DestField:      dstField,
		AssignmentType: MappedAssignment,
		Code:           assignmentCode,
		SourcePath:     sourcePath,
	}, nil
}

// generateConverterAssignment generates an assignment using a converter function.
func (gfm *GenericFieldMapper) generateConverterAssignment(
	dstField *domain.Field,
	srcFields []*domain.Field,
	converter string,
	context *GenericMappingContext,
) (*FieldAssignment, error) {
	// Find matching source field (use field name as default)
	var sourceField *domain.Field
	for _, field := range srcFields {
		if field.Name == dstField.Name {
			sourceField = field
			break
		}
	}

	if sourceField == nil {
		return nil, fmt.Errorf("source field %s not found for converter assignment", dstField.Name)
	}

	var assignmentCode string
	if context.Options.ErrorHandling == domain.ErrorPropagate {
		assignmentCode = fmt.Sprintf(`convertedValue, err := %s(src.%s)
if err != nil {
	return dst, fmt.Errorf("conversion failed for field %s: %%w", err)
}
dst.%s = convertedValue`, converter, sourceField.Name, dstField.Name, dstField.Name)
	} else {
		assignmentCode = fmt.Sprintf("dst.%s = %s(src.%s)", dstField.Name, converter, sourceField.Name)
	}

	return &FieldAssignment{
		SourceField:    sourceField,
		DestField:      dstField,
		AssignmentType: ConverterAssignment,
		Code:           assignmentCode,
		Converter:      converter,
	}, nil
}

// generateLiteralAssignment generates an assignment using a literal value.
func (gfm *GenericFieldMapper) generateLiteralAssignment(
	dstField *domain.Field,
	literal string,
	context *GenericMappingContext,
) (*FieldAssignment, error) {
	assignmentCode := fmt.Sprintf("dst.%s = %s", dstField.Name, literal)

	return &FieldAssignment{
		DestField:      dstField,
		AssignmentType: LiteralAssignment,
		Code:           assignmentCode,
		Literal:        literal,
	}, nil
}

// getTypeName returns the string representation of a type for code generation.
func (gfm *GenericFieldMapper) getTypeName(typ domain.Type) string {
	if typ.Package() != "" {
		return typ.Package() + "." + typ.Name()
	}
	return typ.Name()
}

// fieldsMatchByPattern checks if fields can be mapped using common patterns.
func (gfm *GenericFieldMapper) fieldsMatchByPattern(srcField, dstField *domain.Field, context *GenericMappingContext) bool {
	// Common field name mapping patterns
	mappingPatterns := map[string][]string{
		"Value": {"Name", "Value", "Data", "Content", "Result"}, // Value can accept from multiple sources
		"Name":  {"Name", "Title", "Label", "Value"},            // Name can accept from multiple sources
		"Data":  {"Data", "Content", "Value", "Payload"},        // Data can accept from multiple sources
		"Inner": {"Inner", "Value", "Data", "Content"},          // Inner can accept from multiple sources
	}

	// Check if destination field can accept from source field
	if acceptableSources, exists := mappingPatterns[dstField.Name]; exists {
		for _, acceptableSource := range acceptableSources {
			if srcField.Name == acceptableSource {
				// Check type compatibility if validation is enabled
				if !context.Options.ValidateTypes {
					return true
				}
				return gfm.typesCompatible(srcField.Type, dstField.Type, context)
			}
		}
	}

	// Check if source and destination have compatible patterns (bidirectional)
	for pattern, sources := range mappingPatterns {
		// Check if source field matches this pattern
		for _, source := range sources {
			if srcField.Name == source {
				// Check if dest field also matches a compatible pattern
				for _, compatibleSource := range sources {
					if dstField.Name == compatibleSource {
						if !context.Options.ValidateTypes {
							return true
						}
						return gfm.typesCompatible(srcField.Type, dstField.Type, context)
					}
				}
				break
			}
		}
		if srcField.Name == pattern {
			// Source field is a pattern name, check if dest field is compatible
			for _, compatibleSource := range sources {
				if dstField.Name == compatibleSource {
					if !context.Options.ValidateTypes {
						return true
					}
					return gfm.typesCompatible(srcField.Type, dstField.Type, context)
				}
			}
		}
	}

	return false
}

// optimizeAssignments applies optimizations to field assignments.
func (gfm *GenericFieldMapper) optimizeAssignments(
	assignments []*FieldAssignment,
	context *GenericMappingContext,
) []*FieldAssignment {
	if !gfm.config.EnableOptimization {
		return assignments
	}

	// Group similar assignments
	optimized := make([]*FieldAssignment, 0, len(assignments))

	// Remove redundant type conversions
	for _, assignment := range assignments {
		if assignment.AssignmentType == DirectAssignment {
			// Check if type conversion is actually needed
			if assignment.SourceField != nil &&
				assignment.SourceField.Type.AssignableTo(assignment.DestField.Type) {
				// Remove unnecessary type conversion
				assignment.Code = fmt.Sprintf("dst.%s = src.%s",
					assignment.DestField.Name, assignment.SourceField.Name)
			}
		}
		optimized = append(optimized, assignment)
	}

	return optimized
}

// GetMetrics returns the current mapping metrics.
func (gfm *GenericFieldMapper) GetMetrics() *GenericFieldMappingMetrics {
	gfm.metricsMux.RLock()
	defer gfm.metricsMux.RUnlock()

	// Return a copy to prevent race conditions on the returned value
	metricsCopy := *gfm.metrics
	return &metricsCopy
}

// ClearMetrics resets all metrics.
func (gfm *GenericFieldMapper) ClearMetrics() {
	gfm.metricsMux.Lock()
	defer gfm.metricsMux.Unlock()
	gfm.metrics = NewGenericFieldMappingMetrics()
}

// Enhanced Performance and Memory Management Methods

// GetEnhancedMetrics returns comprehensive performance metrics including optimization data
func (gfm *GenericFieldMapper) GetEnhancedMetrics() map[string]interface{} {
	baseMetrics := gfm.GetMetrics()
	perfMetrics := gfm.performanceOptimizer.GetMetrics()

	return map[string]interface{}{
		"basic_metrics": map[string]interface{}{
			"total_mappings":        baseMetrics.TotalMappings,
			"successful_mappings":   baseMetrics.SuccessfulMappings,
			"failed_mappings":       baseMetrics.FailedMappings,
			"type_substitutions":    baseMetrics.TypeSubstitutions,
			"optimizations_applied": baseMetrics.OptimizationsApplied,
			"average_mapping_time":  baseMetrics.AverageMappingTime.String(),
			"total_mapping_time":    baseMetrics.TotalMappingTime.String(),
		},
		"performance_metrics": map[string]interface{}{
			"cache_hits":          perfMetrics.CacheHits,
			"cache_misses":        perfMetrics.CacheMisses,
			"cache_evictions":     perfMetrics.CacheEvictions,
			"cache_efficiency":    fmt.Sprintf("%.2f%%", perfMetrics.CacheEfficiency*100),
			"parallel_operations": perfMetrics.ParallelOperations,
			"parallel_speedup":    fmt.Sprintf("%.2fx", perfMetrics.ParallelSpeedup),
			"memory_pool_hits":    perfMetrics.MemoryPoolHits,
			"memory_pool_misses":  perfMetrics.MemoryPoolMisses,
			"memory_efficiency":   fmt.Sprintf("%.2f%%", perfMetrics.MemoryEfficiency*100),
			"memory_allocated":    fmt.Sprintf("%d bytes", perfMetrics.MemoryAllocated),
			"memory_freed":        fmt.Sprintf("%d bytes", perfMetrics.MemoryFreed),
			"processing_time":     fmt.Sprintf("%d ns", perfMetrics.ProcessingTime),
		},
		"configuration": map[string]interface{}{
			"cache_enabled":        gfm.performanceOptimizer.config.EnableFieldMappingCache,
			"parallel_enabled":     gfm.performanceOptimizer.config.EnableParallelMapping,
			"memory_pooling":       gfm.performanceOptimizer.config.EnableMemoryPooling,
			"max_cache_size":       gfm.performanceOptimizer.config.MaxCacheSize,
			"max_parallel_workers": gfm.performanceOptimizer.config.MaxParallelWorkers,
			"performance_profile":  gfm.performanceOptimizer.config.PerformanceProfile,
		},
	}
}

// OptimizeMemoryUsage performs immediate memory optimization
func (gfm *GenericFieldMapper) OptimizeMemoryUsage() {
	// Clear old cache entries
	gfm.performanceOptimizer.maintainCacheSize()

	// Reset memory pools to release unused capacity
	if gfm.performanceOptimizer.config.EnableMemoryPooling && gfm.performanceOptimizer.memoryPool != nil {
		// Create a new memory pool to release old allocations
		gfm.performanceOptimizer.memoryPool = &sync.Pool{
			New: func() interface{} {
				return make([]*FieldAssignment, 0, 16)
			},
		}
	}

	// Force garbage collection hint
	runtime.GC()

	gfm.logger.Info("memory optimization completed",
		zap.String("optimization_type", "manual_cleanup"))
}

// ConfigurePerformance allows runtime configuration of performance settings
func (gfm *GenericFieldMapper) ConfigurePerformance(config *PerformanceConfig) {
	if config == nil {
		return
	}

	gfm.performanceOptimizer.config = config

	// Reinitialize pools if needed
	if config.EnableParallelMapping && gfm.performanceOptimizer.parallelWorkerPool == nil {
		gfm.performanceOptimizer.parallelWorkerPool = &sync.Pool{
			New: func() interface{} {
				return &ParallelFieldMappingJob{
					Result: make(chan *FieldAssignment, 1),
					Error:  make(chan error, 1),
				}
			},
		}
	}

	if config.EnableMemoryPooling && gfm.performanceOptimizer.memoryPool == nil {
		gfm.performanceOptimizer.memoryPool = &sync.Pool{
			New: func() interface{} {
				return make([]*FieldAssignment, 0, 16)
			},
		}
	}

	gfm.logger.Info("performance configuration updated",
		zap.Bool("cache_enabled", config.EnableFieldMappingCache),
		zap.Bool("parallel_enabled", config.EnableParallelMapping),
		zap.String("profile", config.PerformanceProfile))
}

// GetPerformanceProfile returns current performance configuration
func (gfm *GenericFieldMapper) GetPerformanceProfile() *PerformanceConfig {
	return gfm.performanceOptimizer.config
}

// ResetPerformanceMetrics resets all performance counters
func (gfm *GenericFieldMapper) ResetPerformanceMetrics() {
	gfm.ClearMetrics()
	gfm.performanceOptimizer.ResetMetrics()
}

// OptimizeForProfile configures performance settings for specific use cases
func (gfm *GenericFieldMapper) OptimizeForProfile(profile string) {
	var config *PerformanceConfig

	switch profile {
	case "speed":
		config = &PerformanceConfig{
			EnableFieldMappingCache: true,
			MaxCacheSize:            20000,
			CacheTTL:                2 * time.Hour,
			EnableParallelMapping:   true,
			MaxParallelWorkers:      runtime.NumCPU() * 2,
			MemoryPoolSize:          2000,
			EnableMemoryPooling:     true,
			PerformanceProfile:      "speed",
			AutoTune:                false,
		}
	case "memory":
		config = &PerformanceConfig{
			EnableFieldMappingCache: true,
			MaxCacheSize:            1000,
			CacheTTL:                30 * time.Minute,
			EnableParallelMapping:   false,
			MaxParallelWorkers:      1,
			MemoryPoolSize:          100,
			EnableMemoryPooling:     true,
			PerformanceProfile:      "memory",
			AutoTune:                false,
		}
	case "balanced":
		config = DefaultPerformanceConfig()
	default:
		gfm.logger.Warn("unknown performance profile, using balanced", zap.String("profile", profile))
		config = DefaultPerformanceConfig()
	}

	gfm.ConfigurePerformance(config)
}

// generateNestedFieldAssignment attempts to generate a nested field assignment with enhanced generic support.
func (gfm *GenericFieldMapper) generateNestedFieldAssignment(
	dstField *domain.Field,
	srcFields []*domain.Field,
	context *GenericMappingContext,
) *FieldAssignment {
	// Handle nested struct field mappings
	if dstField.Type.Kind() == domain.KindStruct {
		return gfm.generateNestedStructFieldAssignment(dstField, srcFields, context)
	}

	// Enhanced: Handle nested slice field mappings (e.g., []List[T] -> []Array[U])
	if dstField.Type.Kind() == domain.KindSlice {
		return gfm.generateNestedSliceFieldAssignment(dstField, srcFields, context)
	}

	// Enhanced: Handle nested map field mappings (e.g., Map[string, List[T]] -> Map[string, Array[U]])
	if dstField.Type.Kind() == domain.KindMap {
		return gfm.generateNestedMapFieldAssignment(dstField, srcFields, context)
	}

	// Enhanced: Handle nested generic field mappings
	if dstField.Type.Kind() == domain.KindGeneric || dstField.Type.Generic() {
		return gfm.generateNestedGenericFieldAssignment(dstField, srcFields, context)
	}

	return nil
}

// SetConfiguration updates the mapper configuration.
func (gfm *GenericFieldMapper) SetConfiguration(config *GenericFieldMapperConfig) {
	if config != nil {
		gfm.config = config
	}
}

// Enhanced methods for nested generic type handling

// typesConvertibleForMaps checks convertibility between map types with potential generic arguments.
func (gfm *GenericFieldMapper) typesConvertibleForMaps(srcType, dstType domain.Type) bool {
	// For map types, we need to check both key and value type compatibility
	// This is a simplified implementation - in production, would need proper MapType interface
	gfm.logger.Debug("checking map type convertibility",
		zap.String("src_type", srcType.String()),
		zap.String("dst_type", dstType.String()))

	// For now, allow conversion between map types if they have compatible structure
	// A full implementation would examine key/value types recursively
	return srcType.Name() != "" && dstType.Name() != ""
}

// typesConvertibleForGenerics checks convertibility between generic types.
func (gfm *GenericFieldMapper) typesConvertibleForGenerics(srcType, dstType domain.Type) bool {
	// If one type is generic and the other is concrete, check if substitution can work
	if srcType.Kind() == domain.KindGeneric {
		// Source is generic parameter, check if destination can accept it
		return gfm.canAcceptGenericType(srcType, dstType)
	}

	if dstType.Kind() == domain.KindGeneric {
		// Destination is generic parameter, check if source can be assigned
		return gfm.canAssignToGenericType(srcType, dstType)
	}

	// Both types might have generic parameters in their structure
	if srcType.Generic() && dstType.Generic() {
		return gfm.compatibleGenericStructures(srcType, dstType)
	}

	return false
}

// typesConvertibleForNamedTypes checks convertibility between named types with generic support.
func (gfm *GenericFieldMapper) typesConvertibleForNamedTypes(srcType, dstType domain.Type) bool {
	// Named types can have generic parameters, need to check underlying types
	if srcType.Kind() == domain.KindNamed {
		srcUnderlying := srcType.Underlying()
		if srcUnderlying != nil && srcUnderlying != srcType {
			return gfm.typesConvertible(srcUnderlying, dstType)
		}
	}

	if dstType.Kind() == domain.KindNamed {
		dstUnderlying := dstType.Underlying()
		if dstUnderlying != nil && dstUnderlying != dstType {
			return gfm.typesConvertible(srcType, dstUnderlying)
		}
	}

	// Check if both are named types with compatible names/packages
	return gfm.compatibleNamedTypes(srcType, dstType)
}

// canAcceptGenericType checks if a concrete type can accept a generic type parameter.
func (gfm *GenericFieldMapper) canAcceptGenericType(genericType, concreteType domain.Type) bool {
	// This would check type constraints in a full implementation
	// For now, be permissive for any concrete type
	return concreteType.Kind() != domain.KindGeneric
}

// canAssignToGenericType checks if a concrete type can be assigned to a generic parameter.
func (gfm *GenericFieldMapper) canAssignToGenericType(concreteType, genericType domain.Type) bool {
	// This would check type constraints in a full implementation
	// For now, be permissive for any concrete type
	return concreteType.Kind() != domain.KindGeneric
}

// compatibleGenericStructures checks if two generic structures are compatible.
func (gfm *GenericFieldMapper) compatibleGenericStructures(srcType, dstType domain.Type) bool {
	// Check if the base structure is similar even if type parameters differ
	// This is a simplified compatibility check
	srcName := gfm.extractBaseTypeName(srcType.String())
	dstName := gfm.extractBaseTypeName(dstType.String())

	return srcName == dstName || gfm.structurallyCompatible(srcType, dstType)
}

// compatibleNamedTypes checks if two named types are compatible.
func (gfm *GenericFieldMapper) compatibleNamedTypes(srcType, dstType domain.Type) bool {
	// Check name compatibility and package compatibility
	if srcType.Name() == dstType.Name() {
		// Same name, check if packages are compatible
		return gfm.packagesCompatible(srcType.Package(), dstType.Package())
	}

	// Different names, check if they represent similar concepts
	return gfm.semanticallyCompatibleNames(srcType.Name(), dstType.Name())
}

// extractBaseTypeName extracts the base type name from a generic type string.
func (gfm *GenericFieldMapper) extractBaseTypeName(typeStr string) string {
	// Extract base name before any generic parameters
	if idx := strings.Index(typeStr, "["); idx != -1 {
		return typeStr[:idx]
	}
	return typeStr
}

// structurallyCompatible checks if two types have compatible structure.
func (gfm *GenericFieldMapper) structurallyCompatible(srcType, dstType domain.Type) bool {
	// This would perform deeper structural analysis
	// For now, use a heuristic based on type kinds
	return srcType.Kind() == dstType.Kind()
}

// packagesCompatible checks if two packages are compatible for type conversion.
func (gfm *GenericFieldMapper) packagesCompatible(srcPkg, dstPkg string) bool {
	// Same package is always compatible
	if srcPkg == dstPkg {
		return true
	}

	// Different packages might still be compatible if they're related
	return gfm.relatedPackages(srcPkg, dstPkg)
}

// semanticallyCompatibleNames checks if two type names represent compatible concepts.
func (gfm *GenericFieldMapper) semanticallyCompatibleNames(srcName, dstName string) bool {
	// Common type name mappings for generic collections
	compatibilityMap := map[string][]string{
		"List":  {"Array", "Slice", "Vector", "Collection"},
		"Array": {"List", "Slice", "Vector", "Collection"},
		"Map":   {"Dict", "HashMap", "Dictionary", "Table"},
		"Dict":  {"Map", "HashMap", "Dictionary", "Table"},
		"Set":   {"HashSet", "Collection"},
	}

	if compatibleNames, exists := compatibilityMap[srcName]; exists {
		for _, name := range compatibleNames {
			if name == dstName {
				return true
			}
		}
	}

	return false
}

// relatedPackages checks if two packages are related for type compatibility.
func (gfm *GenericFieldMapper) relatedPackages(srcPkg, dstPkg string) bool {
	// This could check for version differences, alias packages, etc.
	// For now, be conservative
	return false
}

// generateNestedStructFieldAssignment generates assignment for nested struct fields.
func (gfm *GenericFieldMapper) generateNestedStructFieldAssignment(
	dstField *domain.Field,
	srcFields []*domain.Field,
	context *GenericMappingContext,
) *FieldAssignment {
	// Look for a source field with the same name that's also a struct
	for _, srcField := range srcFields {
		if srcField.Name == dstField.Name && srcField.Type.Kind() == domain.KindStruct {
			// Generate nested struct assignment with generic support
			nestedCode := gfm.generateEnhancedNestedStructAssignment(srcField, dstField, context)
			if nestedCode != "" {
				return &FieldAssignment{
					SourceField:    srcField,
					DestField:      dstField,
					AssignmentType: DirectAssignment,
					Code:           nestedCode,
				}
			}
		}
	}
	return nil
}

// generateNestedSliceFieldAssignment generates assignment for nested slice fields.
func (gfm *GenericFieldMapper) generateNestedSliceFieldAssignment(
	dstField *domain.Field,
	srcFields []*domain.Field,
	context *GenericMappingContext,
) *FieldAssignment {
	// Look for compatible source slice fields
	for _, srcField := range srcFields {
		if gfm.fieldsMatch(srcField, dstField, context) && srcField.Type.Kind() == domain.KindSlice {
			// Generate slice conversion code
			sliceCode := gfm.generateSliceConversionCode(srcField, dstField, context)
			if sliceCode != "" {
				return &FieldAssignment{
					SourceField:    srcField,
					DestField:      dstField,
					AssignmentType: SliceAssignment,
					Code:           sliceCode,
				}
			}
		}
	}
	return nil
}

// generateNestedMapFieldAssignment generates assignment for nested map fields.
func (gfm *GenericFieldMapper) generateNestedMapFieldAssignment(
	dstField *domain.Field,
	srcFields []*domain.Field,
	context *GenericMappingContext,
) *FieldAssignment {
	// Look for compatible source map fields
	for _, srcField := range srcFields {
		if gfm.fieldsMatch(srcField, dstField, context) && srcField.Type.Kind() == domain.KindMap {
			// Generate map conversion code
			mapCode := gfm.generateMapConversionCode(srcField, dstField, context)
			if mapCode != "" {
				return &FieldAssignment{
					SourceField:    srcField,
					DestField:      dstField,
					AssignmentType: MapAssignment,
					Code:           mapCode,
				}
			}
		}
	}
	return nil
}

// generateNestedGenericFieldAssignment generates assignment for nested generic fields.
func (gfm *GenericFieldMapper) generateNestedGenericFieldAssignment(
	dstField *domain.Field,
	srcFields []*domain.Field,
	context *GenericMappingContext,
) *FieldAssignment {
	// Look for source fields that can be converted to the generic destination
	for _, srcField := range srcFields {
		if gfm.canConvertToGenericField(srcField, dstField, context) {
			// Generate generic conversion code
			genericCode := gfm.generateGenericConversionCode(srcField, dstField, context)
			if genericCode != "" {
				return &FieldAssignment{
					SourceField:    srcField,
					DestField:      dstField,
					AssignmentType: ConversionAssignment,
					Code:           genericCode,
				}
			}
		}
	}
	return nil
}

// generateEnhancedNestedStructAssignment generates enhanced code for nested struct assignments with generic support.
func (gfm *GenericFieldMapper) generateEnhancedNestedStructAssignment(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) string {
	// Extract fields from both structs with type substitution awareness
	srcStructFields, err := gfm.extractTypeFieldsWithSubstitution(srcField.Type, context)
	if err != nil {
		return ""
	}

	dstStructFields, err := gfm.extractTypeFieldsWithSubstitution(dstField.Type, context)
	if err != nil {
		return ""
	}

	// Generate field-by-field assignments for the nested struct with enhanced matching
	assignments := make([]string, 0)

	for _, dstNestedField := range dstStructFields {
		for _, srcNestedField := range srcStructFields {
			if gfm.enhancedFieldsCanMapNested(srcNestedField, dstNestedField, context) {
				// Generate the nested assignment with type conversion if needed
				assignment := gfm.generateNestedAssignmentCode(srcField, dstField, srcNestedField, dstNestedField, context)
				assignments = append(assignments, assignment)
				break
			}
		}
	}

	if len(assignments) == 0 {
		return ""
	}

	// Join all assignments with newlines
	return strings.Join(assignments, "\n\t")
}

// extractTypeFieldsWithSubstitution extracts fields from a type with type substitution applied.
func (gfm *GenericFieldMapper) extractTypeFieldsWithSubstitution(
	typ domain.Type,
	context *GenericMappingContext,
) ([]*domain.Field, error) {
	// Apply type substitutions first if needed
	if typ.Generic() && context.RequiresTypeSubstitution() {
		substitutedType, err := gfm.substituteTypeIfNeeded(typ, context.TypeSubstitutions)
		if err != nil {
			return nil, err
		}
		typ = substitutedType
	}

	return gfm.extractTypeFields(typ)
}

// enhancedFieldsCanMapNested checks if nested fields can be mapped with enhanced generic support.
func (gfm *GenericFieldMapper) enhancedFieldsCanMapNested(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) bool {
	// Enhanced field matching with generic type awareness
	if srcField.Name == dstField.Name {
		// Names match, check type compatibility with generic support
		return gfm.typesCompatible(srcField.Type, dstField.Type, context)
	}

	// Allow common transformations for nested fields with enhanced patterns
	return gfm.fieldsMatchByPattern(srcField, dstField, context)
}

// generateNestedAssignmentCode generates assignment code for nested fields.
func (gfm *GenericFieldMapper) generateNestedAssignmentCode(
	srcField, dstField *domain.Field,
	srcNestedField, dstNestedField *domain.Field,
	context *GenericMappingContext,
) string {
	basicAssignment := fmt.Sprintf("dst.%s.%s = src.%s.%s",
		dstField.Name, dstNestedField.Name,
		srcField.Name, srcNestedField.Name)

	// Add type conversion if needed
	if !srcNestedField.Type.AssignableTo(dstNestedField.Type) && context.Options.UseTypeConversion {
		dstTypeName := gfm.getTypeName(dstNestedField.Type)
		return fmt.Sprintf("dst.%s.%s = %s(src.%s.%s)",
			dstField.Name, dstNestedField.Name, dstTypeName,
			srcField.Name, srcNestedField.Name)
	}

	return basicAssignment
}

// generateSliceConversionCode generates conversion code for slice fields.
func (gfm *GenericFieldMapper) generateSliceConversionCode(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) string {
	srcSliceType, srcOk := srcField.Type.(*domain.SliceType)
	dstSliceType, dstOk := dstField.Type.(*domain.SliceType)

	if !srcOk || !dstOk {
		gfm.logger.Debug("slice conversion requires both types to be slice types",
			zap.String("src_type", srcField.Type.String()),
			zap.String("dst_type", dstField.Type.String()))
		return gfm.generateFallbackSliceConversion(srcField, dstField)
	}

	srcElemType := srcSliceType.Elem()
	dstElemType := dstSliceType.Elem()

	// Check if element types need conversion
	if gfm.typesCompatible(srcElemType, dstElemType, context) {
		// Direct element assignment - most efficient
		return gfm.generateDirectSliceConversion(srcField, dstField)
	}

	// Check for generic element conversion
	if srcElemType.Generic() || dstElemType.Generic() {
		return gfm.generateGenericSliceConversion(srcField, dstField, context)
	}

	// Check for interface{} to concrete type conversion
	if gfm.isInterfaceEmptyType(srcElemType) && !gfm.isInterfaceEmptyType(dstElemType) {
		return gfm.generateInterfaceToConcreteSliceConversion(srcField, dstField, dstElemType)
	}

	// Check for nested struct conversion
	if srcElemType.Kind() == domain.KindStruct && dstElemType.Kind() == domain.KindStruct {
		return gfm.generateNestedStructSliceConversion(srcField, dstField, context)
	}

	// Complex element transformation with custom converter
	return gfm.generateCustomSliceConversion(srcField, dstField, context)
}

// generateDirectSliceConversion handles slice conversion when element types are directly compatible
func (gfm *GenericFieldMapper) generateDirectSliceConversion(srcField, dstField *domain.Field) string {
	return fmt.Sprintf(`// Direct slice conversion: %s -> %s
	if src.%s != nil {
		dst.%s = make(%s, len(src.%s))
		copy(dst.%s, src.%s)
	}`,
		srcField.Type.String(), dstField.Type.String(),
		srcField.Name, dstField.Name, gfm.getTypeName(dstField.Type),
		srcField.Name, dstField.Name, srcField.Name)
}

// generateGenericSliceConversion handles slice conversion with generic element types
func (gfm *GenericFieldMapper) generateGenericSliceConversion(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) string {
	srcSliceType := srcField.Type.(*domain.SliceType)
	dstSliceType := dstField.Type.(*domain.SliceType)

	srcElemType := srcSliceType.Elem()
	dstElemType := dstSliceType.Elem()

	// Apply type substitution to element types
	substitutedSrcElem := gfm.applyTypeSubstitution(srcElemType, context)
	substitutedDstElem := gfm.applyTypeSubstitution(dstElemType, context)

	// Check if substituted types are compatible
	if gfm.typesCompatible(substitutedSrcElem, substitutedDstElem, context) {
		return fmt.Sprintf(`// Generic slice conversion with type substitution: %s -> %s
		if src.%s != nil {
			dst.%s = make(%s, len(src.%s))
			for i, item := range src.%s {
				dst.%s[i] = %s(item)
			}
		}`,
			srcField.Type.String(), dstField.Type.String(),
			srcField.Name, dstField.Name, gfm.getTypeName(dstField.Type),
			srcField.Name, srcField.Name, dstField.Name,
			gfm.generateTypeConversion(substitutedSrcElem, substitutedDstElem))
	}

	// Require nested conversion for complex generic types
	return gfm.generateComplexGenericSliceConversion(srcField, dstField, context)
}

// generateInterfaceToConcreteSliceConversion handles interface{} to concrete type slice conversion
func (gfm *GenericFieldMapper) generateInterfaceToConcreteSliceConversion(
	srcField, dstField *domain.Field,
	targetElemType domain.Type,
) string {
	return fmt.Sprintf(`// Interface{} to concrete slice conversion: %s -> %s
	if src.%s != nil {
		dst.%s = make(%s, 0, len(src.%s))
		for _, item := range src.%s {
			if typedItem, ok := item.(%s); ok {
				dst.%s = append(dst.%s, typedItem)
			}
		}
	}`,
		srcField.Type.String(), dstField.Type.String(),
		srcField.Name, dstField.Name, gfm.getTypeName(dstField.Type),
		srcField.Name, srcField.Name, gfm.getTypeName(targetElemType),
		dstField.Name, dstField.Name)
}

// generateNestedStructSliceConversion handles slice conversion with nested struct element conversion
func (gfm *GenericFieldMapper) generateNestedStructSliceConversion(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) string {
	srcSliceType := srcField.Type.(*domain.SliceType)
	dstSliceType := dstField.Type.(*domain.SliceType)

	srcElemType := srcSliceType.Elem()
	dstElemType := dstSliceType.Elem()

	return fmt.Sprintf(`// Nested struct slice conversion: %s -> %s
	if src.%s != nil {
		dst.%s = make(%s, len(src.%s))
		for i, item := range src.%s {
			%s
		}
	}`,
		srcField.Type.String(), dstField.Type.String(),
		srcField.Name, dstField.Name, gfm.getTypeName(dstField.Type),
		srcField.Name, srcField.Name,
		gfm.generateNestedElementConversion("item", fmt.Sprintf("dst.%s[i]", dstField.Name), srcElemType, dstElemType, context))
}

// generateCustomSliceConversion handles slice conversion requiring custom conversion logic
func (gfm *GenericFieldMapper) generateCustomSliceConversion(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) string {
	return fmt.Sprintf(`// Custom slice conversion with element transformation: %s -> %s
	if src.%s != nil {
		dst.%s = make(%s, len(src.%s))
		for i, item := range src.%s {
			// TODO: Add custom element conversion logic
			// This may require converter functions for complex transformations
			dst.%s[i] = item // Placeholder - replace with actual conversion
		}
	}`,
		srcField.Type.String(), dstField.Type.String(),
		srcField.Name, dstField.Name, gfm.getTypeName(dstField.Type),
		srcField.Name, srcField.Name, dstField.Name)
}

// generateFallbackSliceConversion provides a fallback for non-slice types
func (gfm *GenericFieldMapper) generateFallbackSliceConversion(srcField, dstField *domain.Field) string {
	return fmt.Sprintf(`// Fallback slice conversion: %s -> %s
	dst.%s = make(%s, len(src.%s))
	for i, item := range src.%s {
		dst.%s[i] = item // Direct assignment fallback
	}`,
		srcField.Type.String(), dstField.Type.String(),
		dstField.Name, gfm.getTypeName(dstField.Type),
		srcField.Name, srcField.Name, dstField.Name)
}

// generateMapConversionCode generates conversion code for map fields.
func (gfm *GenericFieldMapper) generateMapConversionCode(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) string {
	// Extract map key and value types
	srcMapInfo := gfm.extractMapTypeInfo(srcField.Type)
	dstMapInfo := gfm.extractMapTypeInfo(dstField.Type)

	if srcMapInfo == nil || dstMapInfo == nil {
		gfm.logger.Debug("map conversion requires both types to be map types",
			zap.String("src_type", srcField.Type.String()),
			zap.String("dst_type", dstField.Type.String()))
		return gfm.generateFallbackMapConversion(srcField, dstField)
	}

	// Check key and value type compatibility
	keyCompatible := gfm.typesCompatible(srcMapInfo.KeyType, dstMapInfo.KeyType, context)
	valueCompatible := gfm.typesCompatible(srcMapInfo.ValueType, dstMapInfo.ValueType, context)

	if keyCompatible && valueCompatible {
		// Direct map conversion - most efficient
		return gfm.generateDirectMapConversion(srcField, dstField)
	}

	// Handle generic key/value transformations
	if srcMapInfo.KeyType.Generic() || dstMapInfo.KeyType.Generic() ||
		srcMapInfo.ValueType.Generic() || dstMapInfo.ValueType.Generic() {
		return gfm.generateGenericMapConversion(srcField, dstField, srcMapInfo, dstMapInfo, context)
	}

	// Handle interface{} conversions
	if gfm.requiresInterfaceConversion(srcMapInfo, dstMapInfo) {
		return gfm.generateInterfaceMapConversion(srcField, dstField, srcMapInfo, dstMapInfo)
	}

	// Handle nested struct conversions
	if gfm.requiresNestedStructConversion(srcMapInfo, dstMapInfo) {
		return gfm.generateNestedStructMapConversion(srcField, dstField, srcMapInfo, dstMapInfo, context)
	}

	// Complex key/value transformation
	return gfm.generateCustomMapConversion(srcField, dstField, srcMapInfo, dstMapInfo, context)
}

// MapTypeInfo holds information about map key and value types
type MapTypeInfo struct {
	KeyType   domain.Type
	ValueType domain.Type
}

// extractMapTypeInfo extracts key and value type information from a map type
func (gfm *GenericFieldMapper) extractMapTypeInfo(mapType domain.Type) *MapTypeInfo {
	// Try to access map type through interface assertion with domain package's mapType
	if mt, ok := mapType.(interface {
		Key() domain.Type
		Value() domain.Type
	}); ok {
		return &MapTypeInfo{
			KeyType:   mt.Key(),
			ValueType: mt.Value(),
		}
	}

	// For types that don't implement our mapType interface, try to parse from string
	typeStr := mapType.String()
	if strings.HasPrefix(typeStr, "map[") {
		return gfm.parseMapTypeFromString(typeStr)
	}
	return nil
}

// parseMapTypeFromString attempts to parse map type information from a type string
func (gfm *GenericFieldMapper) parseMapTypeFromString(typeStr string) *MapTypeInfo {
	// This is a simplified parser - in production would use go/types for proper parsing
	if !strings.HasPrefix(typeStr, "map[") {
		return nil
	}

	// Extract key and value type strings (simplified)
	content := typeStr[4:] // Remove "map["
	bracketCount := 0
	keyEnd := -1

	for i, r := range content {
		switch r {
		case '[':
			bracketCount++
		case ']':
			if bracketCount == 0 && keyEnd == -1 {
				keyEnd = i
				break
			}
			bracketCount--
		}
	}

	if keyEnd == -1 || keyEnd+1 >= len(content) {
		return nil
	}

	keyTypeStr := content[:keyEnd]
	valueTypeStr := content[keyEnd+1:]

	// Create basic types for key and value (simplified)
	keyType := domain.NewBasicType(keyTypeStr, 0)
	valueType := domain.NewBasicType(valueTypeStr, 0)

	return &MapTypeInfo{
		KeyType:   keyType,
		ValueType: valueType,
	}
}

// generateDirectMapConversion handles map conversion when key and value types are compatible
func (gfm *GenericFieldMapper) generateDirectMapConversion(srcField, dstField *domain.Field) string {
	return fmt.Sprintf(`// Direct map conversion: %s -> %s
	if src.%s != nil {
		dst.%s = make(%s, len(src.%s))
		for k, v := range src.%s {
			dst.%s[k] = v
		}
	}`,
		srcField.Type.String(), dstField.Type.String(),
		srcField.Name, dstField.Name, gfm.getTypeName(dstField.Type),
		srcField.Name, srcField.Name, dstField.Name)
}

// generateGenericMapConversion handles map conversion with generic key/value types
func (gfm *GenericFieldMapper) generateGenericMapConversion(
	srcField, dstField *domain.Field,
	srcMapInfo, dstMapInfo *MapTypeInfo,
	context *GenericMappingContext,
) string {
	// Apply type substitution to key and value types
	substitutedSrcKey := gfm.applyTypeSubstitution(srcMapInfo.KeyType, context)
	substitutedDstKey := gfm.applyTypeSubstitution(dstMapInfo.KeyType, context)
	substitutedSrcValue := gfm.applyTypeSubstitution(srcMapInfo.ValueType, context)
	substitutedDstValue := gfm.applyTypeSubstitution(dstMapInfo.ValueType, context)

	// Generate key and value conversion expressions
	keyConversion := gfm.generateTypeConversion(substitutedSrcKey, substitutedDstKey)
	valueConversion := gfm.generateTypeConversion(substitutedSrcValue, substitutedDstValue)

	return fmt.Sprintf(`// Generic map conversion with type substitution: %s -> %s
	if src.%s != nil {
		dst.%s = make(%s, len(src.%s))
		for k, v := range src.%s {
			convertedKey := %s
			convertedValue := %s
			dst.%s[convertedKey] = convertedValue
		}
	}`,
		srcField.Type.String(), dstField.Type.String(),
		srcField.Name, dstField.Name, gfm.getTypeName(dstField.Type),
		srcField.Name, srcField.Name,
		gfm.applyConversionToValue(keyConversion, "k"),
		gfm.applyConversionToValue(valueConversion, "v"),
		dstField.Name)
}

// generateInterfaceMapConversion handles interface{} to concrete type map conversion
func (gfm *GenericFieldMapper) generateInterfaceMapConversion(
	srcField, dstField *domain.Field,
	srcMapInfo, dstMapInfo *MapTypeInfo,
) string {
	keyConversion := ""
	valueConversion := ""

	if gfm.isInterfaceEmptyType(srcMapInfo.KeyType) && !gfm.isInterfaceEmptyType(dstMapInfo.KeyType) {
		keyConversion = fmt.Sprintf("typedKey, keyOk := k.(%s); if !keyOk { continue }", gfm.getTypeName(dstMapInfo.KeyType))
	} else {
		keyConversion = "typedKey := k"
	}

	if gfm.isInterfaceEmptyType(srcMapInfo.ValueType) && !gfm.isInterfaceEmptyType(dstMapInfo.ValueType) {
		valueConversion = fmt.Sprintf("typedValue, valueOk := v.(%s); if !valueOk { continue }", gfm.getTypeName(dstMapInfo.ValueType))
	} else {
		valueConversion = "typedValue := v"
	}

	return fmt.Sprintf(`// Interface{} to concrete map conversion: %s -> %s
	if src.%s != nil {
		dst.%s = make(%s)
		for k, v := range src.%s {
			%s
			%s
			dst.%s[typedKey] = typedValue
		}
	}`,
		srcField.Type.String(), dstField.Type.String(),
		srcField.Name, dstField.Name, gfm.getTypeName(dstField.Type),
		srcField.Name, keyConversion, valueConversion, dstField.Name)
}

// generateNestedStructMapConversion handles map conversion with nested struct values
func (gfm *GenericFieldMapper) generateNestedStructMapConversion(
	srcField, dstField *domain.Field,
	srcMapInfo, dstMapInfo *MapTypeInfo,
	context *GenericMappingContext,
) string {
	return fmt.Sprintf(`// Nested struct map conversion: %s -> %s
	if src.%s != nil {
		dst.%s = make(%s, len(src.%s))
		for k, v := range src.%s {
			%s
		}
	}`,
		srcField.Type.String(), dstField.Type.String(),
		srcField.Name, dstField.Name, gfm.getTypeName(dstField.Type),
		srcField.Name, srcField.Name,
		gfm.generateNestedMapValueConversion("k", "v", dstField.Name, srcMapInfo, dstMapInfo, context))
}

// generateCustomMapConversion handles map conversion requiring custom conversion logic
func (gfm *GenericFieldMapper) generateCustomMapConversion(
	srcField, dstField *domain.Field,
	srcMapInfo, dstMapInfo *MapTypeInfo,
	context *GenericMappingContext,
) string {
	return fmt.Sprintf(`// Custom map conversion with key/value transformation: %s -> %s
	if src.%s != nil {
		dst.%s = make(%s, len(src.%s))
		for k, v := range src.%s {
			// TODO: Add custom key/value conversion logic
			// This may require converter functions for complex transformations
			dst.%s[k] = v // Placeholder - replace with actual conversion
		}
	}`,
		srcField.Type.String(), dstField.Type.String(),
		srcField.Name, dstField.Name, gfm.getTypeName(dstField.Type),
		srcField.Name, srcField.Name, dstField.Name)
}

// generateFallbackMapConversion provides a fallback for non-map types
func (gfm *GenericFieldMapper) generateFallbackMapConversion(srcField, dstField *domain.Field) string {
	return fmt.Sprintf(`// Fallback map conversion: %s -> %s
	dst.%s = make(%s)
	for k, v := range src.%s {
		dst.%s[k] = v // Direct assignment fallback
	}`,
		srcField.Type.String(), dstField.Type.String(),
		dstField.Name, gfm.getTypeName(dstField.Type),
		srcField.Name, dstField.Name)
}

// isInterfaceEmptyType checks if a type is interface{}
func (gfm *GenericFieldMapper) isInterfaceEmptyType(typ domain.Type) bool {
	if typ.Kind() == domain.KindInterface {
		return typ.String() == "interface{}" || typ.String() == "any"
	}
	return false
}

// requiresInterfaceConversion checks if map conversion requires interface{} handling
func (gfm *GenericFieldMapper) requiresInterfaceConversion(srcMapInfo, dstMapInfo *MapTypeInfo) bool {
	return (gfm.isInterfaceEmptyType(srcMapInfo.KeyType) && !gfm.isInterfaceEmptyType(dstMapInfo.KeyType)) ||
		(gfm.isInterfaceEmptyType(srcMapInfo.ValueType) && !gfm.isInterfaceEmptyType(dstMapInfo.ValueType))
}

// requiresNestedStructConversion checks if map conversion requires nested struct handling
func (gfm *GenericFieldMapper) requiresNestedStructConversion(srcMapInfo, dstMapInfo *MapTypeInfo) bool {
	return (srcMapInfo.ValueType.Kind() == domain.KindStruct && dstMapInfo.ValueType.Kind() == domain.KindStruct) ||
		(srcMapInfo.KeyType.Kind() == domain.KindStruct && dstMapInfo.KeyType.Kind() == domain.KindStruct)
}

// generateTypeConversion generates a type conversion expression between two types
func (gfm *GenericFieldMapper) generateTypeConversion(srcType, dstType domain.Type) string {
	if srcType.String() == dstType.String() {
		return "directAssignment"
	}

	// Handle basic type conversions
	if srcType.Kind() == domain.KindBasic && dstType.Kind() == domain.KindBasic {
		return fmt.Sprintf("%s(%s)", gfm.getTypeName(dstType), "VALUE_PLACEHOLDER")
	}

	// Handle pointer conversions
	if srcType.Kind() == domain.KindPointer || dstType.Kind() == domain.KindPointer {
		return "pointerConversion"
	}

	// Handle generic type conversions
	if srcType.Generic() || dstType.Generic() {
		return "genericConversion"
	}

	return "customConversion"
}

// applyConversionToValue applies a conversion expression to a specific value
func (gfm *GenericFieldMapper) applyConversionToValue(conversionExpr, valueName string) string {
	switch conversionExpr {
	case "directAssignment":
		return valueName
	case "pointerConversion":
		return fmt.Sprintf("convertPointer(%s)", valueName)
	case "genericConversion":
		return fmt.Sprintf("convertGeneric(%s)", valueName)
	default:
		return strings.ReplaceAll(conversionExpr, "VALUE_PLACEHOLDER", valueName)
	}
}

// generateNestedElementConversion generates conversion code for nested slice elements
func (gfm *GenericFieldMapper) generateNestedElementConversion(
	srcVarName, dstVarName string,
	srcElemType, dstElemType domain.Type,
	context *GenericMappingContext,
) string {
	if srcElemType.Kind() == domain.KindStruct && dstElemType.Kind() == domain.KindStruct {
		return fmt.Sprintf(`// Convert nested struct element
		%s = %s{
			// TODO: Add struct field conversions
		}`, dstVarName, gfm.getTypeName(dstElemType))
	}

	return fmt.Sprintf("%s = %s(%s)", dstVarName, gfm.getTypeName(dstElemType), srcVarName)
}

// generateNestedMapValueConversion generates conversion code for nested map values
func (gfm *GenericFieldMapper) generateNestedMapValueConversion(
	keyVarName, valueVarName, dstMapName string,
	srcMapInfo, dstMapInfo *MapTypeInfo,
	context *GenericMappingContext,
) string {
	if srcMapInfo.ValueType.Kind() == domain.KindStruct && dstMapInfo.ValueType.Kind() == domain.KindStruct {
		return fmt.Sprintf(`// Convert nested struct map value
		convertedValue := %s{
			// TODO: Add struct field conversions
		}
		%s[%s] = convertedValue`, gfm.getTypeName(dstMapInfo.ValueType), dstMapName, keyVarName)
	}

	return fmt.Sprintf("%s[%s] = %s(%s)", dstMapName, keyVarName, gfm.getTypeName(dstMapInfo.ValueType), valueVarName)
}

// generateComplexGenericSliceConversion handles complex generic slice conversions
func (gfm *GenericFieldMapper) generateComplexGenericSliceConversion(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) string {
	return fmt.Sprintf(`// Complex generic slice conversion: %s -> %s
	if src.%s != nil {
		dst.%s = make(%s, len(src.%s))
		for i, item := range src.%s {
			// TODO: Implement complex generic element conversion
			dst.%s[i] = convertGenericElement(item)
		}
	}`,
		srcField.Type.String(), dstField.Type.String(),
		srcField.Name, dstField.Name, gfm.getTypeName(dstField.Type),
		srcField.Name, srcField.Name, dstField.Name)
}

// generateChannelConversionCode handles channel type conversions where applicable
func (gfm *GenericFieldMapper) generateChannelConversionCode(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) string {
	// Extract channel element types and directions
	srcChanInfo := gfm.extractChannelTypeInfo(srcField.Type)
	dstChanInfo := gfm.extractChannelTypeInfo(dstField.Type)

	if srcChanInfo == nil || dstChanInfo == nil {
		return gfm.generateFallbackChannelConversion(srcField, dstField)
	}

	// Check element type compatibility
	if !gfm.typesCompatible(srcChanInfo.ElemType, dstChanInfo.ElemType, context) {
		gfm.logger.Debug("channel element types are not compatible",
			zap.String("src_elem", srcChanInfo.ElemType.String()),
			zap.String("dst_elem", dstChanInfo.ElemType.String()))
		return gfm.generateUnsupportedChannelConversion(srcField, dstField)
	}

	// Check direction compatibility
	if !gfm.channelDirectionsCompatible(srcChanInfo.Direction, dstChanInfo.Direction) {
		return gfm.generateChannelDirectionConversion(srcField, dstField, srcChanInfo, dstChanInfo)
	}

	// Direct channel assignment for compatible channels
	return gfm.generateDirectChannelConversion(srcField, dstField)
}

// generateFunctionConversionCode handles function type conversions where applicable
func (gfm *GenericFieldMapper) generateFunctionConversionCode(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) string {
	// Extract function signatures
	srcFuncInfo := gfm.extractFunctionTypeInfo(srcField.Type)
	dstFuncInfo := gfm.extractFunctionTypeInfo(dstField.Type)

	if srcFuncInfo == nil || dstFuncInfo == nil {
		return gfm.generateFallbackFunctionConversion(srcField, dstField)
	}

	// Check signature compatibility
	if !gfm.functionSignaturesCompatible(srcFuncInfo, dstFuncInfo, context) {
		gfm.logger.Debug("function signatures are not compatible",
			zap.String("src_func", srcField.Type.String()),
			zap.String("dst_func", dstField.Type.String()))
		return gfm.generateUnsupportedFunctionConversion(srcField, dstField)
	}

	// Direct function assignment for compatible signatures
	return gfm.generateDirectFunctionConversion(srcField, dstField)
}

// ChannelTypeInfo holds information about channel types
type ChannelTypeInfo struct {
	ElemType  domain.Type
	Direction domain.ChannelDirection
}

// FunctionTypeInfo holds information about function types
type FunctionTypeInfo struct {
	Params   []domain.Type
	Returns  []domain.Type
	Variadic bool
}

// extractChannelTypeInfo extracts channel type information
func (gfm *GenericFieldMapper) extractChannelTypeInfo(chanType domain.Type) *ChannelTypeInfo {
	if chanType.Kind() != domain.KindFunction { // Channel types might be represented differently
		// Try to parse from type string
		typeStr := chanType.String()
		if strings.HasPrefix(typeStr, "chan ") || strings.Contains(typeStr, "chan<") {
			return gfm.parseChannelTypeFromString(typeStr)
		}
		return nil
	}

	// TODO: Implement proper channel type extraction once channel types are properly supported in domain
	return nil
}

// extractFunctionTypeInfo extracts function type information
func (gfm *GenericFieldMapper) extractFunctionTypeInfo(funcType domain.Type) *FunctionTypeInfo {
	// Try to access function type through interface assertion
	if ft, ok := funcType.(interface {
		Params() []domain.Type
		Returns() []domain.Type
		Variadic() bool
	}); ok {
		return &FunctionTypeInfo{
			Params:   ft.Params(),
			Returns:  ft.Returns(),
			Variadic: ft.Variadic(),
		}
	}

	// Try to parse from type string for basic cases
	typeStr := funcType.String()
	if strings.HasPrefix(typeStr, "func") {
		return gfm.parseFunctionTypeFromString(typeStr)
	}
	return nil
}

// parseChannelTypeFromString attempts to parse channel type from string
func (gfm *GenericFieldMapper) parseChannelTypeFromString(typeStr string) *ChannelTypeInfo {
	// Simplified parser for channel types
	if strings.HasPrefix(typeStr, "chan ") {
		elemTypeStr := typeStr[5:] // Remove "chan "
		elemType := domain.NewBasicType(elemTypeStr, 0)
		return &ChannelTypeInfo{
			ElemType:  elemType,
			Direction: domain.ChannelBidirectional,
		}
	}

	if strings.HasPrefix(typeStr, "<-chan ") {
		elemTypeStr := typeStr[7:] // Remove "<-chan "
		elemType := domain.NewBasicType(elemTypeStr, 0)
		return &ChannelTypeInfo{
			ElemType:  elemType,
			Direction: domain.ChannelSendOnly, // Receive-only from perspective
		}
	}

	return nil
}

// parseFunctionTypeFromString attempts to parse function type from string
func (gfm *GenericFieldMapper) parseFunctionTypeFromString(typeStr string) *FunctionTypeInfo {
	// Simplified parser for function types
	// This would need proper implementation using go/types for production use
	return &FunctionTypeInfo{
		Params:   []domain.Type{},
		Returns:  []domain.Type{},
		Variadic: false,
	}
}

// channelDirectionsCompatible checks if channel directions are compatible
func (gfm *GenericFieldMapper) channelDirectionsCompatible(srcDir, dstDir domain.ChannelDirection) bool {
	// Bidirectional channels are compatible with any direction
	if srcDir == domain.ChannelBidirectional || dstDir == domain.ChannelBidirectional {
		return true
	}

	// Same directions are compatible
	return srcDir == dstDir
}

// functionSignaturesCompatible checks if function signatures are compatible
func (gfm *GenericFieldMapper) functionSignaturesCompatible(
	srcFunc, dstFunc *FunctionTypeInfo,
	context *GenericMappingContext,
) bool {
	// Check parameter count
	if len(srcFunc.Params) != len(dstFunc.Params) {
		return false
	}

	// Check return count
	if len(srcFunc.Returns) != len(dstFunc.Returns) {
		return false
	}

	// Check variadic compatibility
	if srcFunc.Variadic != dstFunc.Variadic {
		return false
	}

	// Check parameter types
	for i, srcParam := range srcFunc.Params {
		if !gfm.typesCompatible(srcParam, dstFunc.Params[i], context) {
			return false
		}
	}

	// Check return types
	for i, srcReturn := range srcFunc.Returns {
		if !gfm.typesCompatible(srcReturn, dstFunc.Returns[i], context) {
			return false
		}
	}

	return true
}

// generateDirectChannelConversion generates direct channel assignment
func (gfm *GenericFieldMapper) generateDirectChannelConversion(srcField, dstField *domain.Field) string {
	return fmt.Sprintf(`// Direct channel conversion: %s -> %s
	dst.%s = src.%s`,
		srcField.Type.String(), dstField.Type.String(),
		dstField.Name, srcField.Name)
}

// generateChannelDirectionConversion handles channel direction conversions
func (gfm *GenericFieldMapper) generateChannelDirectionConversion(
	srcField, dstField *domain.Field,
	srcChanInfo, dstChanInfo *ChannelTypeInfo,
) string {
	return fmt.Sprintf(`// Channel direction conversion: %s -> %s
	// Note: Channel direction conversions may require careful handling
	dst.%s = (%s)(src.%s)`,
		srcField.Type.String(), dstField.Type.String(),
		dstField.Name, gfm.getTypeName(dstField.Type), srcField.Name)
}

// generateDirectFunctionConversion generates direct function assignment
func (gfm *GenericFieldMapper) generateDirectFunctionConversion(srcField, dstField *domain.Field) string {
	return fmt.Sprintf(`// Direct function conversion: %s -> %s
	dst.%s = src.%s`,
		srcField.Type.String(), dstField.Type.String(),
		dstField.Name, srcField.Name)
}

// generateFallbackChannelConversion provides fallback for unsupported channel types
func (gfm *GenericFieldMapper) generateFallbackChannelConversion(srcField, dstField *domain.Field) string {
	return fmt.Sprintf(`// Fallback channel conversion: %s -> %s
	// TODO: Implement proper channel type conversion
	dst.%s = src.%s // Direct assignment fallback`,
		srcField.Type.String(), dstField.Type.String(),
		dstField.Name, srcField.Name)
}

// generateFallbackFunctionConversion provides fallback for unsupported function types
func (gfm *GenericFieldMapper) generateFallbackFunctionConversion(srcField, dstField *domain.Field) string {
	return fmt.Sprintf(`// Fallback function conversion: %s -> %s
	// TODO: Implement proper function type conversion
	dst.%s = src.%s // Direct assignment fallback`,
		srcField.Type.String(), dstField.Type.String(),
		dstField.Name, srcField.Name)
}

// generateUnsupportedChannelConversion handles unsupported channel conversions
func (gfm *GenericFieldMapper) generateUnsupportedChannelConversion(srcField, dstField *domain.Field) string {
	return fmt.Sprintf(`// Unsupported channel conversion: %s -> %s
	// Channel element types are incompatible - skipping conversion
	// dst.%s = nil // Uncomment if nil assignment is desired`,
		srcField.Type.String(), dstField.Type.String(),
		dstField.Name)
}

// generateUnsupportedFunctionConversion handles unsupported function conversions
func (gfm *GenericFieldMapper) generateUnsupportedFunctionConversion(srcField, dstField *domain.Field) string {
	return fmt.Sprintf(`// Unsupported function conversion: %s -> %s
	// Function signatures are incompatible - skipping conversion
	// dst.%s = nil // Uncomment if nil assignment is desired`,
		srcField.Type.String(), dstField.Type.String(),
		dstField.Name)
}

// canConvertToGenericField checks if a source field can be converted to a generic destination field.
func (gfm *GenericFieldMapper) canConvertToGenericField(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) bool {
	// Check if field names are compatible
	if !gfm.fieldsMatch(srcField, dstField, context) {
		return false
	}

	// Check if source type can be converted to the generic destination type
	return gfm.typesConvertibleForGenerics(srcField.Type, dstField.Type)
}

// generateGenericConversionCode generates conversion code for generic fields.
func (gfm *GenericFieldMapper) generateGenericConversionCode(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) string {
	// For generic conversions, we might need to apply type substitutions
	assignmentCode := fmt.Sprintf("dst.%s = src.%s", dstField.Name, srcField.Name)

	// Add type conversion if the destination field requires it
	if dstField.Type.Kind() == domain.KindGeneric {
		// Check if we have a concrete type substitution for this generic parameter
		if concreteType, found := context.TypeSubstitutions[dstField.Type.Name()]; found {
			concreteTypeName := gfm.getTypeName(concreteType)
			assignmentCode = fmt.Sprintf("dst.%s = %s(src.%s)",
				dstField.Name, concreteTypeName, srcField.Name)
		}
	}

	return assignmentCode
}

// Enhanced helper methods for deeply nested generic support

// isDeeplyNestedGeneric checks if a type contains deeply nested generic structures.
func (gfm *GenericFieldMapper) isDeeplyNestedGeneric(typ domain.Type) bool {
	// Check for nested generic patterns
	if !typ.Generic() {
		return false
	}

	// Heuristic: Check if type string contains nested generic patterns
	typeStr := typ.String()

	// Count bracket depth to detect nested generics like Map[K, List[V]]
	bracketDepth := 0
	maxDepth := 0

	for _, char := range typeStr {
		switch char {
		case '[':
			bracketDepth++
			if bracketDepth > maxDepth {
				maxDepth = bracketDepth
			}
		case ']':
			bracketDepth--
		}
	}

	// Consider deeply nested if bracket depth > 1 or contains known complex patterns
	return maxDepth > 1 || gfm.containsComplexGenericPatterns(typeStr)
}

// containsComplexGenericPatterns checks for known complex generic patterns.
func (gfm *GenericFieldMapper) containsComplexGenericPatterns(typeStr string) bool {
	complexPatterns := []string{
		"Map[",
		"List[",
		"Array[",
		"Set[",
		"Optional[",
		"Future[",
		"Result[",
		"Either[",
	}

	patternCount := 0
	for _, pattern := range complexPatterns {
		if strings.Contains(typeStr, pattern) {
			patternCount++
			if patternCount > 1 {
				return true // Multiple generic patterns indicate complexity
			}
		}
	}

	return false
}

// RegisterTypeAlias registers a type alias for use in generic field mapping.
func (gfm *GenericFieldMapper) RegisterTypeAlias(aliasName string, actualType domain.Type) {
	if gfm.recursiveResolver != nil {
		gfm.recursiveResolver.RegisterTypeAlias(aliasName, actualType)
		gfm.logger.Debug("registered type alias for field mapping",
			zap.String("alias", aliasName),
			zap.String("actual_type", actualType.String()))
	}
}

// SupportsGenericTypeAlias checks if the mapper supports generic type aliases.
func (gfm *GenericFieldMapper) SupportsGenericTypeAlias() bool {
	return gfm.recursiveResolver != nil
}

// GetRecursiveResolutionMetrics returns metrics from the recursive resolver.
func (gfm *GenericFieldMapper) GetRecursiveResolutionMetrics() *RecursiveResolutionMetrics {
	if gfm.recursiveResolver != nil {
		return gfm.recursiveResolver.GetMetrics()
	}
	return nil
}

// ClearRecursiveResolutionCache clears the recursive resolver's cache.
func (gfm *GenericFieldMapper) ClearRecursiveResolutionCache() {
	if gfm.recursiveResolver != nil {
		gfm.recursiveResolver.ClearCache()
	}
}
