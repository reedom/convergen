package planner

import (
	"context"
	"sort"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v8/pkg/domain"
)

// PlanOptimizer applies optimization strategies to execution plans.
type PlanOptimizer interface {
	OptimizePlan(ctx context.Context, methodPlans map[string]*domain.MethodPlan, batches []*ExecutionBatch) error
	ApplyBatchOptimizations(batches []*ExecutionBatch) error
	OptimizeConcurrency(methodPlans map[string]*domain.MethodPlan) error
	OptimizeResourceUsage(methodPlans map[string]*domain.MethodPlan) error
}

// ConcretePlanOptimizer implements PlanOptimizer.
type ConcretePlanOptimizer struct {
	config *Config
	logger *zap.Logger
}

// NewPlanOptimizer creates a new plan optimizer.
func NewPlanOptimizer(config *Config, logger *zap.Logger) PlanOptimizer {
	return &ConcretePlanOptimizer{
		config: config,
		logger: logger,
	}
}

// OptimizePlan applies all optimization strategies to the execution plan.
func (po *ConcretePlanOptimizer) OptimizePlan(ctx context.Context, methodPlans map[string]*domain.MethodPlan, batches []*ExecutionBatch) error {
	if !po.config.EnableOptimizations {
		return nil
	}

	po.logger.Info("starting plan optimization",
		zap.Int("optimization_level", po.config.OptimizationLevel),
		zap.Int("methods", len(methodPlans)),
		zap.Int("batches", len(batches)))

	// Apply optimizations based on level
	switch po.config.OptimizationLevel {
	case 0:
		// No optimizations
		return nil
	case 1:
		// Basic optimizations
		return po.applyBasicOptimizations(ctx, methodPlans, batches)
	case 2:
		// Aggressive optimizations
		return po.applyAggressiveOptimizations(ctx, methodPlans, batches)
	default:
		return po.applyBasicOptimizations(ctx, methodPlans, batches)
	}
}

// applyBasicOptimizations applies conservative optimization strategies.
func (po *ConcretePlanOptimizer) applyBasicOptimizations(_ context.Context, methodPlans map[string]*domain.MethodPlan, batches []*ExecutionBatch) error {
	// Optimize batch sizes
	po.optimizeBatchSizes(batches)

	// Balance worker allocation
	po.balanceWorkerAllocation(methodPlans)

	// Optimize memory usage
	po.optimizeMemoryUsage(methodPlans, batches)

	po.logger.Info("basic optimizations applied successfully")

	return nil
}

// applyAggressiveOptimizations applies more aggressive optimization strategies.
func (po *ConcretePlanOptimizer) applyAggressiveOptimizations(ctx context.Context, methodPlans map[string]*domain.MethodPlan, batches []*ExecutionBatch) error {
	// Apply basic optimizations first
	if err := po.applyBasicOptimizations(ctx, methodPlans, batches); err != nil {
		return err
	}

	// Advanced batch merging
	po.mergeBatches(batches)

	// Pipeline optimization
	po.optimizePipeline(methodPlans, batches)

	// Resource pooling optimization
	po.optimizeResourcePooling(methodPlans)

	po.logger.Info("aggressive optimizations applied successfully")

	return nil
}

// ApplyBatchOptimizations optimizes execution batches.
func (po *ConcretePlanOptimizer) ApplyBatchOptimizations(batches []*ExecutionBatch) error {
	// Sort batches by estimated duration for better load balancing
	po.sortBatchesByDuration(batches)

	// Optimize batch concurrency levels
	for _, batch := range batches {
		po.optimizeBatchConcurrency(batch)
	}

	// Merge small batches if beneficial
	if po.config.OptimizationLevel >= 1 {
		po.mergeSmallBatches(batches)
	}

	po.logger.Debug("batch optimizations applied",
		zap.Int("batch_count", len(batches)))

	return nil
}

// OptimizeConcurrency optimizes concurrency settings across methods.
func (po *ConcretePlanOptimizer) OptimizeConcurrency(methodPlans map[string]*domain.MethodPlan) error {
	totalWorkers := 0

	// Calculate total worker requirements
	for _, plan := range methodPlans {
		totalWorkers += plan.RequiredWorkers
	}

	// If total exceeds limits, redistribute
	if totalWorkers > po.config.MaxConcurrentWorkers {
		po.redistributeWorkers(methodPlans, totalWorkers)
	}

	po.logger.Debug("concurrency optimization completed",
		zap.Int("total_workers", totalWorkers),
		zap.Int("max_workers", po.config.MaxConcurrentWorkers))

	return nil
}

// OptimizeResourceUsage optimizes memory and CPU resource usage.
func (po *ConcretePlanOptimizer) OptimizeResourceUsage(methodPlans map[string]*domain.MethodPlan) error {
	totalMemory := 0

	// Calculate total memory requirements
	for _, plan := range methodPlans {
		totalMemory += plan.MemoryRequirementMB
	}

	// If memory exceeds limits, apply memory optimization
	if totalMemory > po.config.MaxMemoryMB {
		po.optimizeMemoryAllocation(methodPlans, totalMemory)
	}

	po.logger.Debug("resource optimization completed",
		zap.Int("total_memory_mb", totalMemory),
		zap.Int("max_memory_mb", po.config.MaxMemoryMB))

	return nil
}

// Helper methods for optimization strategies

func (po *ConcretePlanOptimizer) optimizeBatchSizes(batches []*ExecutionBatch) {
	for _, batch := range batches {
		mappingCount := len(batch.Mappings)

		// Adjust concurrency level based on batch size
		if mappingCount < po.config.MinBatchSize {
			// Small batch - reduce concurrency to avoid overhead
			batch.ConcurrencyLevel = 1
		} else if mappingCount > po.config.MaxBatchSize {
			// Large batch - limit concurrency to prevent resource exhaustion
			batch.ConcurrencyLevel = po.config.MaxConcurrentWorkers / 2
		} else {
			// Optimal size - use calculated concurrency
			batch.ConcurrencyLevel = min(mappingCount, po.config.MaxConcurrentWorkers)
		}
	}
}

func (po *ConcretePlanOptimizer) balanceWorkerAllocation(methodPlans map[string]*domain.MethodPlan) {
	// Calculate priority scores for methods
	priorities := po.calculateMethodPriorities(methodPlans)

	// Sort methods by priority
	sortedMethods := make([]string, 0, len(methodPlans))
	for methodName := range methodPlans {
		sortedMethods = append(sortedMethods, methodName)
	}

	sort.Slice(sortedMethods, func(i, j int) bool {
		return priorities[sortedMethods[i]] > priorities[sortedMethods[j]]
	})

	// Allocate workers based on priority
	remainingWorkers := po.config.MaxConcurrentWorkers

	for _, methodName := range sortedMethods {
		plan := methodPlans[methodName]
		if remainingWorkers <= 0 {
			plan.RequiredWorkers = 1 // Minimum allocation
		} else {
			allocated := min(plan.RequiredWorkers, remainingWorkers)
			plan.RequiredWorkers = allocated
			remainingWorkers -= allocated
		}
	}
}

func (po *ConcretePlanOptimizer) optimizeMemoryUsage(methodPlans map[string]*domain.MethodPlan, batches []*ExecutionBatch) {
	// Calculate memory usage patterns
	for _, plan := range methodPlans {
		// Optimize memory based on field complexity
		baseMemory := len(plan.Batches) * 10 // 10MB per batch baseline
		plan.MemoryRequirementMB = baseMemory

		// Adjust based on batch characteristics
		for _, batch := range batches {
			if po.batchBelongsToMethod(batch, plan) {
				// Add memory for complex field mappings
				for _, mapping := range batch.Mappings {
					if po.isComplexMapping(mapping) {
						plan.MemoryRequirementMB += 5 // Additional 5MB for complex mappings
					}
				}
			}
		}
	}
}

func (po *ConcretePlanOptimizer) mergeBatches(batches []*ExecutionBatch) {
	// Find batches that can be merged without violating dependencies
	i := 0
	for i < len(batches)-1 {
		current := batches[i]
		next := batches[i+1]

		// Check if batches can be merged
		if po.canMergeBatches(current, next) {
			// Merge next into current
			current.Mappings = append(current.Mappings, next.Mappings...)
			current.EstimatedDurationMS += next.EstimatedDurationMS
			current.ResourceRequirement.MemoryMB += next.ResourceRequirement.MemoryMB
			current.ConcurrencyLevel = max(current.ConcurrencyLevel, next.ConcurrencyLevel)

			// Remove next batch
			copy(batches[i+1:], batches[i+2:])
			batches = batches[:len(batches)-1]

			po.logger.Debug("merged batches",
				zap.String("batch1", current.ID),
				zap.String("batch2", next.ID))
		} else {
			i++
		}
	}
}

func (po *ConcretePlanOptimizer) optimizePipeline(methodPlans map[string]*domain.MethodPlan, _ []*ExecutionBatch) {
	// Implement pipeline optimization strategies
	// This is a placeholder for more sophisticated pipeline optimization
	for _, plan := range methodPlans {
		if len(plan.Batches) > 2 {
			plan.Strategy = domain.MethodStrategyPipelined
		}
	}
}

func (po *ConcretePlanOptimizer) optimizeResourcePooling(methodPlans map[string]*domain.MethodPlan) {
	// Enable resource pooling for methods with high resource usage
	for _, plan := range methodPlans {
		if plan.MemoryRequirementMB > 100 || plan.RequiredWorkers > 4 {
			// Enable advanced resource pooling strategies
			// This would be implemented based on specific pooling mechanisms
			// TODO: Implement resource pooling optimization
			_ = plan // Mark as used until implementation
		}
	}
}

func (po *ConcretePlanOptimizer) sortBatchesByDuration(batches []*ExecutionBatch) {
	sort.Slice(batches, func(i, j int) bool {
		return batches[i].EstimatedDurationMS < batches[j].EstimatedDurationMS
	})
}

func (po *ConcretePlanOptimizer) optimizeBatchConcurrency(batch *ExecutionBatch) {
	// Optimize concurrency based on batch characteristics
	mappingCount := len(batch.Mappings)

	if mappingCount <= 2 {
		batch.ConcurrencyLevel = 1 // No benefit from concurrency
	} else if mappingCount <= 10 {
		batch.ConcurrencyLevel = min(mappingCount, 4) // Limited concurrency
	} else {
		batch.ConcurrencyLevel = min(mappingCount, po.config.MaxConcurrentWorkers)
	}
}

func (po *ConcretePlanOptimizer) mergeSmallBatches(batches []*ExecutionBatch) {
	// Merge batches smaller than minimum size
	for i := 0; i < len(batches)-1; i++ {
		if len(batches[i].Mappings) < po.config.MinBatchSize {
			// Try to merge with next batch if it's also small
			if len(batches[i+1].Mappings) < po.config.MinBatchSize {
				po.mergeTwoBatches(batches[i], batches[i+1])
			}
		}
	}
}

func (po *ConcretePlanOptimizer) redistributeWorkers(methodPlans map[string]*domain.MethodPlan, totalWorkers int) {
	// Calculate reduction factor
	factor := float64(po.config.MaxConcurrentWorkers) / float64(totalWorkers)

	for _, plan := range methodPlans {
		newWorkers := int(float64(plan.RequiredWorkers) * factor)
		if newWorkers < 1 {
			newWorkers = 1
		}

		plan.RequiredWorkers = newWorkers
	}
}

func (po *ConcretePlanOptimizer) optimizeMemoryAllocation(methodPlans map[string]*domain.MethodPlan, totalMemory int) {
	// Calculate reduction factor
	factor := float64(po.config.MaxMemoryMB) / float64(totalMemory)

	for _, plan := range methodPlans {
		newMemory := int(float64(plan.MemoryRequirementMB) * factor)
		if newMemory < 10 {
			newMemory = 10 // Minimum 10MB per method
		}

		plan.MemoryRequirementMB = newMemory
	}
}

func (po *ConcretePlanOptimizer) calculateMethodPriorities(methodPlans map[string]*domain.MethodPlan) map[string]float64 {
	priorities := make(map[string]float64)

	for methodName, plan := range methodPlans {
		// Priority based on estimated duration and field count
		priority := float64(plan.EstimatedDurationMS) * float64(plan.TotalFields)
		priorities[methodName] = priority
	}

	return priorities
}

func (po *ConcretePlanOptimizer) batchBelongsToMethod(batch *ExecutionBatch, plan *domain.MethodPlan) bool {
	// Check if batch belongs to method (simplified implementation)
	for _, planBatch := range plan.Batches {
		if planBatch.ID == batch.ID {
			return true
		}
	}

	return false
}

func (po *ConcretePlanOptimizer) isComplexMapping(mapping *domain.FieldMapping) bool {
	// Determine if mapping is complex based on strategy
	switch mapping.StrategyName {
	case "converter", "literal":
		return true
	case "direct":
		return false
	default:
		return true
	}
}

func (po *ConcretePlanOptimizer) canMergeBatches(batch1, batch2 *ExecutionBatch) bool {
	// Check if batches can be merged without violating dependencies
	// Simplified check - in practice, this would analyze actual dependencies
	totalMappings := len(batch1.Mappings) + len(batch2.Mappings)
	return totalMappings <= po.config.MaxBatchSize
}

func (po *ConcretePlanOptimizer) mergeTwoBatches(batch1, batch2 *ExecutionBatch) {
	batch1.Mappings = append(batch1.Mappings, batch2.Mappings...)
	batch1.EstimatedDurationMS += batch2.EstimatedDurationMS
	batch1.ResourceRequirement.MemoryMB += batch2.ResourceRequirement.MemoryMB
	batch1.ConcurrencyLevel = max(batch1.ConcurrencyLevel, batch2.ConcurrencyLevel)
}

// Utility functions.
func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}
