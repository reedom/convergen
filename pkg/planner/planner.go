package planner

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/reedom/convergen/v8/pkg/domain"
	"github.com/reedom/convergen/v8/pkg/internal/events"
	"go.uber.org/zap"
)

// ExecutionPlanner creates optimized execution plans for concurrent field processing
type ExecutionPlanner struct {
	logger      *zap.Logger
	eventBus    events.EventBus
	config      *PlannerConfig
	depGraph    DependencyGraph
	optimizer   PlanOptimizer
	metrics     *PlannerMetrics
	mutex       sync.RWMutex
}

// PlannerConfig configures the execution planner behavior
type PlannerConfig struct {
	MaxConcurrentWorkers int           `json:"max_concurrent_workers"`
	MaxMemoryMB          int           `json:"max_memory_mb"`
	PlanningTimeout      time.Duration `json:"planning_timeout"`
	EnableOptimizations  bool          `json:"enable_optimizations"`
	OptimizationLevel    int           `json:"optimization_level"` // 0=none, 1=basic, 2=aggressive
	MinBatchSize         int           `json:"min_batch_size"`
	MaxBatchSize         int           `json:"max_batch_size"`
	EnableMetrics        bool          `json:"enable_metrics"`
	DebugMode            bool          `json:"debug_mode"`
}

// NewExecutionPlanner creates a new execution planner
func NewExecutionPlanner(logger *zap.Logger, eventBus events.EventBus, config *PlannerConfig) *ExecutionPlanner {
	if config == nil {
		config = DefaultPlannerConfig()
	}

	depGraph := NewDependencyGraph(logger)
	optimizer := NewPlanOptimizer(config, logger)
	metrics := NewPlannerMetrics()

	return &ExecutionPlanner{
		logger:    logger,
		eventBus:  eventBus,
		config:    config,
		depGraph:  depGraph,
		optimizer: optimizer,
		metrics:   metrics,
	}
}

// DefaultPlannerConfig returns sensible default configuration
func DefaultPlannerConfig() *PlannerConfig {
	return &PlannerConfig{
		MaxConcurrentWorkers: runtime.NumCPU(),
		MaxMemoryMB:          512,
		PlanningTimeout:      30 * time.Second,
		EnableOptimizations:  true,
		OptimizationLevel:    1,
		MinBatchSize:         1,
		MaxBatchSize:         50,
		EnableMetrics:        true,
		DebugMode:            false,
	}
}

// CreateExecutionPlan creates an optimized execution plan for the given methods
func (ep *ExecutionPlanner) CreateExecutionPlan(ctx context.Context, methods []*domain.Method) (*domain.ExecutionPlan, error) {
	startTime := time.Now()
	
	// Emit planning started event
	planStartedEvent := events.NewPlanStartedEvent(ctx, methods)
	if err := ep.eventBus.Publish(ctx, planStartedEvent); err != nil {
		ep.logger.Warn("failed to publish plan started event", zap.Error(err))
	}

	defer func() {
		ep.logger.Info("execution planning completed",
			zap.Int("methods", len(methods)),
			zap.Duration("duration", time.Since(startTime)))
	}()

	// Build dependency graph for all field mappings
	if err := ep.buildDependencyGraph(ctx, methods); err != nil {
		return nil, fmt.Errorf("failed to build dependency graph: %w", err)
	}

	// Detect and resolve circular dependencies
	if err := ep.detectAndResolveCycles(ctx); err != nil {
		return nil, fmt.Errorf("failed to resolve dependency cycles: %w", err)
	}

	// Generate execution batches
	batches, err := ep.generateExecutionBatches(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate execution batches: %w", err)
	}

	// Calculate resource allocation
	resources, err := ep.calculateResourceAllocation(ctx, batches)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate resource allocation: %w", err)
	}

	// Create method-specific execution plans
	methodPlans, err := ep.createMethodPlans(ctx, methods, batches)
	if err != nil {
		return nil, fmt.Errorf("failed to create method plans: %w", err)
	}

	// Apply optimizations
	if ep.config.EnableOptimizations {
		if err := ep.optimizer.OptimizePlan(ctx, methodPlans, batches); err != nil {
			ep.logger.Warn("plan optimization failed", zap.Error(err))
		}
	}

	// Create execution plan
	plan := &domain.ExecutionPlan{
		Methods:      methodPlans,
		GlobalLimits: resources,
		Strategy:     ep.determineExecutionStrategy(batches),
		Metrics:      ep.generatePlanMetrics(startTime, batches),
	}

	// Emit planning completed event
	plannedEvent := events.NewPlannedEvent(ctx, plan)
	if err := ep.eventBus.Publish(ctx, plannedEvent); err != nil {
		ep.logger.Warn("failed to publish planned event", zap.Error(err))
	}

	return plan, nil
}

// buildDependencyGraph constructs the dependency graph from field mappings
func (ep *ExecutionPlanner) buildDependencyGraph(ctx context.Context, methods []*domain.Method) error {
	ep.mutex.Lock()
	defer ep.mutex.Unlock()

	// Clear previous graph
	ep.depGraph.Clear()

	for _, method := range methods {
		for _, mapping := range method.FieldMappings() {
			// Add field mapping to dependency graph
			if err := ep.depGraph.AddField(mapping); err != nil {
				return fmt.Errorf("failed to add field mapping %s: %w", mapping.ID, err)
			}

			// Add dependencies based on mapping configuration
			for _, depID := range mapping.Dependencies {
				if err := ep.depGraph.AddDependency(depID, mapping.ID); err != nil {
					return fmt.Errorf("failed to add dependency %s -> %s: %w", depID, mapping.ID, err)
				}
			}

			// Analyze converter dependencies
			if err := ep.analyzeConverterDependencies(mapping); err != nil {
				return fmt.Errorf("failed to analyze converter dependencies for %s: %w", mapping.ID, err)
			}
		}
	}

	ep.logger.Debug("dependency graph built",
		zap.Int("total_fields", ep.depGraph.Size()),
		zap.Int("total_dependencies", ep.depGraph.DependencyCount()))

	return nil
}

// detectAndResolveCycles detects circular dependencies and attempts resolution
func (ep *ExecutionPlanner) detectAndResolveCycles(ctx context.Context) error {
	cycles, err := ep.depGraph.DetectCycles()
	if err != nil {
		return fmt.Errorf("cycle detection failed: %w", err)
	}

	if len(cycles) == 0 {
		return nil // No cycles found
	}

	ep.logger.Warn("circular dependencies detected",
		zap.Int("cycle_count", len(cycles)))

	// Attempt to resolve cycles
	for i, cycle := range cycles {
		if err := ep.resolveCycle(ctx, cycle, i); err != nil {
			return fmt.Errorf("failed to resolve cycle %d: %w", i, err)
		}
	}

	// Verify cycles are resolved
	remainingCycles, err := ep.depGraph.DetectCycles()
	if err != nil {
		return fmt.Errorf("cycle verification failed: %w", err)
	}

	if len(remainingCycles) > 0 {
		return fmt.Errorf("unable to resolve %d circular dependencies", len(remainingCycles))
	}

	return nil
}

// generateExecutionBatches creates batches of independent field mappings
func (ep *ExecutionPlanner) generateExecutionBatches(ctx context.Context) ([]*ExecutionBatch, error) {
	// Get topologically sorted batches
	sortedBatches, err := ep.depGraph.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("topological sort failed: %w", err)
	}

	var executionBatches []*ExecutionBatch
	for i, mappings := range sortedBatches {
		batch := &ExecutionBatch{
			ID:                  fmt.Sprintf("batch_%d", i),
			Mappings:            mappings,
			EstimatedDurationMS: ep.estimateBatchDuration(mappings),
			ResourceRequirement: ep.calculateBatchResources(mappings),
			DependsOn:           ep.calculateBatchDependencies(i, executionBatches),
			ConcurrencyLevel:    ep.calculateOptimalConcurrency(mappings),
		}

		// Apply batch size limits
		if len(mappings) > ep.config.MaxBatchSize {
			subBatches := ep.splitLargeBatch(batch)
			executionBatches = append(executionBatches, subBatches...)
		} else {
			executionBatches = append(executionBatches, batch)
		}
	}

	ep.logger.Info("execution batches generated",
		zap.Int("batch_count", len(executionBatches)),
		zap.Int("total_mappings", ep.depGraph.Size()))

	return executionBatches, nil
}

// calculateResourceAllocation determines optimal resource limits
func (ep *ExecutionPlanner) calculateResourceAllocation(ctx context.Context, batches []*ExecutionBatch) (*domain.ResourceLimits, error) {
	maxConcurrency := 0
	totalMemoryMB := 0

	for _, batch := range batches {
		if batch.ConcurrencyLevel > maxConcurrency {
			maxConcurrency = batch.ConcurrencyLevel
		}
		totalMemoryMB += batch.ResourceRequirement.MemoryMB
	}

	// Apply configuration limits
	if maxConcurrency > ep.config.MaxConcurrentWorkers {
		maxConcurrency = ep.config.MaxConcurrentWorkers
	}

	if totalMemoryMB > ep.config.MaxMemoryMB {
		totalMemoryMB = ep.config.MaxMemoryMB
	}

	return &domain.ResourceLimits{
		MaxWorkers:          maxConcurrency,
		MaxMemoryMB:         totalMemoryMB,
		MaxDurationMS:       int64(ep.config.PlanningTimeout.Milliseconds()),
		MaxFieldsPerBatch:   ep.config.MaxBatchSize,
		EnableGoroutinePool: true,
		EnableMemoryPool:    true,
	}, nil
}

// createMethodPlans creates execution plans for individual methods
func (ep *ExecutionPlanner) createMethodPlans(ctx context.Context, methods []*domain.Method, batches []*ExecutionBatch) (map[string]*domain.MethodPlan, error) {
	methodPlans := make(map[string]*domain.MethodPlan)

	for _, method := range methods {
		// Find batches containing this method's field mappings
		methodBatches := ep.findMethodBatches(method, batches)

		plan := &domain.MethodPlan{
			MethodName:          method.Name,
			TotalFields:         len(method.FieldMappings()),
			Batches:             methodBatches,
			EstimatedDurationMS: ep.estimateMethodDuration(methodBatches),
			RequiredWorkers:     ep.calculateMethodWorkers(methodBatches),
			MemoryRequirementMB: ep.calculateMethodMemory(methodBatches),
			Strategy:            ep.selectMethodStrategy(method, methodBatches),
		}

		methodPlans[method.Name] = plan
	}

	return methodPlans, nil
}

// Helper methods for planner operations (implementations would follow)

func (ep *ExecutionPlanner) analyzeConverterDependencies(mapping *domain.FieldMapping) error {
	// Implementation depends on converter analysis
	return nil
}

func (ep *ExecutionPlanner) resolveCycle(ctx context.Context, cycle []string, cycleIndex int) error {
	// Implementation for cycle resolution strategies
	return nil
}

func (ep *ExecutionPlanner) estimateBatchDuration(mappings []*domain.FieldMapping) int64 {
	// Heuristic for estimating batch execution time
	return int64(len(mappings) * 10) // 10ms per field mapping (placeholder)
}

func (ep *ExecutionPlanner) calculateBatchResources(mappings []*domain.FieldMapping) *ResourceRequirement {
	return &ResourceRequirement{
		MemoryMB:     len(mappings) * 2, // 2MB per mapping (placeholder)
		CPUIntensive: false,
		IOOperations: 0,
	}
}

func (ep *ExecutionPlanner) calculateBatchDependencies(batchIndex int, previousBatches []*ExecutionBatch) []string {
	if batchIndex == 0 {
		return nil
	}
	return []string{previousBatches[batchIndex-1].ID}
}

func (ep *ExecutionPlanner) calculateOptimalConcurrency(mappings []*domain.FieldMapping) int {
	// Calculate optimal concurrency based on mapping characteristics
	concurrency := len(mappings)
	if concurrency > ep.config.MaxConcurrentWorkers {
		concurrency = ep.config.MaxConcurrentWorkers
	}
	if concurrency < 1 {
		concurrency = 1
	}
	return concurrency
}

func (ep *ExecutionPlanner) splitLargeBatch(batch *ExecutionBatch) []*ExecutionBatch {
	// Split batch if it exceeds maximum size
	var subBatches []*ExecutionBatch
	mappings := batch.Mappings
	
	for i := 0; i < len(mappings); i += ep.config.MaxBatchSize {
		end := i + ep.config.MaxBatchSize
		if end > len(mappings) {
			end = len(mappings)
		}
		
		subBatch := &ExecutionBatch{
			ID:                  fmt.Sprintf("%s_sub_%d", batch.ID, i/ep.config.MaxBatchSize),
			Mappings:            mappings[i:end],
			EstimatedDurationMS: ep.estimateBatchDuration(mappings[i:end]),
			ResourceRequirement: ep.calculateBatchResources(mappings[i:end]),
			ConcurrencyLevel:    ep.calculateOptimalConcurrency(mappings[i:end]),
		}
		subBatches = append(subBatches, subBatch)
	}
	
	return subBatches
}

func (ep *ExecutionPlanner) determineExecutionStrategy(batches []*ExecutionBatch) domain.ExecutionStrategy {
	// Determine the best execution strategy based on batch characteristics
	totalBatches := len(batches)
	if totalBatches <= 1 {
		return domain.StrategySequential
	}
	
	// Check if batches can run in parallel
	hasParallelizableBatches := false
	for _, batch := range batches {
		if len(batch.DependsOn) == 0 || batch.ConcurrencyLevel > 1 {
			hasParallelizableBatches = true
			break
		}
	}
	
	if hasParallelizableBatches {
		return domain.StrategyPipelined
	}
	
	return domain.StrategySequential
}

func (ep *ExecutionPlanner) generatePlanMetrics(startTime time.Time, batches []*ExecutionBatch) *domain.PlanMetrics {
	totalFields := 0
	for _, batch := range batches {
		totalFields += len(batch.Mappings)
	}

	return &domain.PlanMetrics{
		PlanningDurationMS:    time.Since(startTime).Milliseconds(),
		MethodsPlanned:        ep.depGraph.MethodCount(),
		TotalFields:           totalFields,
		ConcurrentBatches:     len(batches),
		ParallelizationRatio:  ep.calculateParallelizationRatio(batches),
		EstimatedSpeedupRatio: ep.calculateEstimatedSpeedup(batches),
	}
}

func (ep *ExecutionPlanner) findMethodBatches(method *domain.Method, batches []*ExecutionBatch) []*ExecutionBatch {
	var methodBatches []*ExecutionBatch
	methodMappings := make(map[string]bool)
	
	// Create mapping ID lookup
	for _, mapping := range method.FieldMappings() {
		methodMappings[mapping.ID] = true
	}
	
	// Find batches containing method's mappings
	for _, batch := range batches {
		containsMethodMapping := false
		for _, mapping := range batch.Mappings {
			if methodMappings[mapping.ID] {
				containsMethodMapping = true
				break
			}
		}
		if containsMethodMapping {
			methodBatches = append(methodBatches, batch)
		}
	}
	
	return methodBatches
}

func (ep *ExecutionPlanner) estimateMethodDuration(batches []*ExecutionBatch) int64 {
	var totalDuration int64
	for _, batch := range batches {
		totalDuration += batch.EstimatedDurationMS
	}
	return totalDuration
}

func (ep *ExecutionPlanner) calculateMethodWorkers(batches []*ExecutionBatch) int {
	maxWorkers := 0
	for _, batch := range batches {
		if batch.ConcurrencyLevel > maxWorkers {
			maxWorkers = batch.ConcurrencyLevel
		}
	}
	return maxWorkers
}

func (ep *ExecutionPlanner) calculateMethodMemory(batches []*ExecutionBatch) int {
	totalMemory := 0
	for _, batch := range batches {
		totalMemory += batch.ResourceRequirement.MemoryMB
	}
	return totalMemory
}

func (ep *ExecutionPlanner) selectMethodStrategy(method *domain.Method, batches []*ExecutionBatch) domain.MethodStrategy {
	if len(batches) <= 1 {
		return domain.MethodStrategyDirect
	}
	
	// Check for complex dependencies
	hasComplexDependencies := false
	for _, batch := range batches {
		if len(batch.DependsOn) > 1 {
			hasComplexDependencies = true
			break
		}
	}
	
	if hasComplexDependencies {
		return domain.MethodStrategyPipelined
	}
	
	return domain.MethodStrategyBatched
}

func (ep *ExecutionPlanner) calculateParallelizationRatio(batches []*ExecutionBatch) float64 {
	if len(batches) == 0 {
		return 0.0
	}
	
	parallelFields := 0
	totalFields := 0
	
	for _, batch := range batches {
		totalFields += len(batch.Mappings)
		if batch.ConcurrencyLevel > 1 {
			parallelFields += len(batch.Mappings)
		}
	}
	
	if totalFields == 0 {
		return 0.0
	}
	
	return float64(parallelFields) / float64(totalFields)
}

func (ep *ExecutionPlanner) calculateEstimatedSpeedup(batches []*ExecutionBatch) float64 {
	if len(batches) == 0 {
		return 1.0
	}
	
	sequentialTime := int64(0)
	parallelTime := int64(0)
	
	for _, batch := range batches {
		sequentialTime += batch.EstimatedDurationMS
		parallelTime += batch.EstimatedDurationMS / int64(batch.ConcurrencyLevel)
	}
	
	if parallelTime == 0 {
		return 1.0
	}
	
	return float64(sequentialTime) / float64(parallelTime)
}

// ResourceRequirement represents resource needs for a batch
type ResourceRequirement struct {
	MemoryMB     int  `json:"memory_mb"`
	CPUIntensive bool `json:"cpu_intensive"`
	IOOperations int  `json:"io_operations"`
}

// ExecutionBatch represents a group of field mappings that can be processed concurrently
type ExecutionBatch struct {
	ID                  string                   `json:"id"`
	Mappings            []*domain.FieldMapping   `json:"mappings"`
	EstimatedDurationMS int64                    `json:"estimated_duration_ms"`
	ResourceRequirement *ResourceRequirement     `json:"resource_requirement"`
	DependsOn           []string                 `json:"depends_on"`
	ConcurrencyLevel    int                      `json:"concurrency_level"`
}

// PlannerMetrics tracks planner performance and statistics
type PlannerMetrics struct {
	TotalPlansCreated    int64         `json:"total_plans_created"`
	AveragePlanningTime  time.Duration `json:"average_planning_time"`
	CyclesDetected       int64         `json:"cycles_detected"`
	CyclesResolved       int64         `json:"cycles_resolved"`
	OptimizationsApplied int64         `json:"optimizations_applied"`
	mutex                sync.RWMutex
}

// NewPlannerMetrics creates new planner metrics
func NewPlannerMetrics() *PlannerMetrics {
	return &PlannerMetrics{}
}

// RecordPlan records metrics for a completed plan
func (pm *PlannerMetrics) RecordPlan(duration time.Duration) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	
	pm.TotalPlansCreated++
	// Update average (simple moving average)
	pm.AveragePlanningTime = (pm.AveragePlanningTime*time.Duration(pm.TotalPlansCreated-1) + duration) / time.Duration(pm.TotalPlansCreated)
}