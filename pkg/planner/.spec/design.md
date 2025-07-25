# Planner Package Design

This document outlines the design of the `pkg/planner` package, which creates optimized execution plans for concurrent field processing while maintaining stable output ordering.

## Architecture Overview

The planner transforms parsed domain methods into detailed execution plans that balance concurrency with deterministic output. It analyzes field dependencies, creates processing batches, and optimizes resource utilization.

## Core Components

### Planner Coordinator

```go
// Planner orchestrates the planning pipeline
type Planner struct {
    config           *Config
    eventBus         events.EventBus
    dependencyGraph  *DependencyAnalyzer
    batchOptimizer   *BatchOptimizer
    resourceManager  *ResourceManager
    validator        *PlanValidator
    logger           *zap.Logger
}

// Plan creates execution plan from parsed methods
func (p *Planner) Plan(ctx context.Context, methods []*domain.Method) (*domain.ExecutionPlan, error) {
    plans := make(map[string]*domain.MethodPlan)
    
    for _, method := range methods {
        // 1. Analyze field dependencies
        dependencies, err := p.dependencyGraph.Analyze(ctx, method)
        if err != nil {
            return nil, fmt.Errorf("dependency analysis failed for %s: %w", method.Name, err)
        }
        
        // 2. Create field mappings
        mappings, err := p.createFieldMappings(ctx, method, dependencies)
        if err != nil {
            return nil, fmt.Errorf("field mapping failed for %s: %w", method.Name, err)
        }
        
        // 3. Generate concurrent batches
        batches, err := p.batchOptimizer.CreateBatches(ctx, mappings, dependencies)
        if err != nil {
            return nil, fmt.Errorf("batch creation failed for %s: %w", method.Name, err)
        }
        
        // 4. Optimize resource allocation
        resources, err := p.resourceManager.AllocateResources(ctx, batches)
        if err != nil {
            return nil, fmt.Errorf("resource allocation failed for %s: %w", method.Name, err)
        }
        
        // 5. Create method plan
        plans[method.Name] = &domain.MethodPlan{
            Method:    method,
            Mappings:  mappings,
            Batches:   batches,
            Resources: resources,
        }
    }
    
    // 6. Create overall execution plan
    executionPlan := &domain.ExecutionPlan{
        Methods:     plans,
        GlobalLimits: p.config.ResourceLimits,
        Strategy:    p.determineExecutionStrategy(ctx, plans),
    }
    
    // 7. Validate plan
    if err := p.validator.ValidatePlan(ctx, executionPlan); err != nil {
        return nil, fmt.Errorf("plan validation failed: %w", err)
    }
    
    // 8. Emit plan event
    event := &PlanEvent{
        Plan:    executionPlan,
        Metrics: p.calculatePlanMetrics(executionPlan),
        Context: ctx,
    }
    
    if err := p.eventBus.Publish(ctx, event); err != nil {
        return nil, fmt.Errorf("failed to emit plan event: %w", err)
    }
    
    return executionPlan, nil
}
```

### Dependency Analysis

```go
// DependencyAnalyzer builds field dependency graphs
type DependencyAnalyzer struct {
    typeAnalyzer *TypeDependencyAnalyzer
    cache        *DependencyCache
    logger       *zap.Logger
}

// Analyze creates dependency graph for method fields
func (a *DependencyAnalyzer) Analyze(ctx context.Context, method *domain.Method) (*DependencyGraph, error) {
    graph := NewDependencyGraph()
    
    // Extract source and destination fields
    srcFields, err := a.extractFields(ctx, method.SourceType)
    if err != nil {
        return nil, fmt.Errorf("failed to extract source fields: %w", err)
    }
    
    dstFields, err := a.extractFields(ctx, method.DestType)
    if err != nil {
        return nil, fmt.Errorf("failed to extract destination fields: %w", err)
    }
    
    // Create field mappings based on method configuration
    mappings := a.createMappings(ctx, method, srcFields, dstFields)
    
    // Build dependency relationships
    for _, mapping := range mappings {
        graph.AddField(mapping)
        
        // Analyze dependencies based on conversion strategy
        deps, err := a.analyzeMappingDependencies(ctx, mapping, method)
        if err != nil {
            return nil, fmt.Errorf("failed to analyze dependencies for %s: %w", mapping.ID, err)
        }
        
        for _, dep := range deps {
            graph.AddDependency(mapping.ID, dep)
        }
    }
    
    // Detect circular dependencies
    if cycles, err := graph.DetectCycles(); err != nil {
        return nil, fmt.Errorf("cycle detection failed: %w", err)
    } else if len(cycles) > 0 {
        return nil, fmt.Errorf("circular dependencies detected: %v", cycles)
    }
    
    return graph, nil
}

// DependencyGraph represents field processing dependencies
type DependencyGraph struct {
    fields       map[string]*domain.FieldMapping
    dependencies map[string][]string  // field ID -> dependent field IDs
    mutex        sync.RWMutex
}

// TopologicalSort returns batches of fields that can be processed concurrently
func (g *DependencyGraph) TopologicalSort() ([][]string, error) {
    g.mutex.RLock()
    defer g.mutex.RUnlock()
    
    // Kahn's algorithm for topological sorting
    inDegree := make(map[string]int)
    
    // Calculate in-degrees
    for fieldID := range g.fields {
        inDegree[fieldID] = 0
    }
    
    for _, deps := range g.dependencies {
        for _, dep := range deps {
            inDegree[dep]++
        }
    }
    
    // Create batches of independent fields
    var batches [][]string
    queue := make([]string, 0)
    
    // Find initial fields with no dependencies
    for fieldID, degree := range inDegree {
        if degree == 0 {
            queue = append(queue, fieldID)
        }
    }
    
    for len(queue) > 0 {
        currentBatch := make([]string, len(queue))
        copy(currentBatch, queue)
        batches = append(batches, currentBatch)
        
        // Process current batch
        nextQueue := make([]string, 0)
        for _, fieldID := range queue {
            for _, dependent := range g.dependencies[fieldID] {
                inDegree[dependent]--
                if inDegree[dependent] == 0 {
                    nextQueue = append(nextQueue, dependent)
                }
            }
        }
        
        queue = nextQueue
    }
    
    return batches, nil
}
```

### Batch Optimization

```go
// BatchOptimizer creates optimized concurrent processing batches
type BatchOptimizer struct {
    config    *OptimizationConfig
    profiler  *PerformanceProfiler
    logger    *zap.Logger
}

// CreateBatches generates optimized batches for concurrent processing
func (o *BatchOptimizer) CreateBatches(ctx context.Context, mappings []*domain.FieldMapping, graph *DependencyGraph) ([]*domain.ConcurrentBatch, error) {
    // Get topological ordering
    levelSets, err := graph.TopologicalSort()
    if err != nil {
        return nil, fmt.Errorf("topological sort failed: %w", err)
    }
    
    batches := make([]*domain.ConcurrentBatch, 0, len(levelSets))
    
    for i, levelSet := range levelSets {
        // Optimize batch size based on processing complexity
        optimizedBatches := o.optimizeBatchSize(ctx, levelSet, mappings)
        
        for j, fieldIDs := range optimizedBatches {
            batchMappings := make([]*domain.FieldMapping, 0, len(fieldIDs))
            for _, fieldID := range fieldIDs {
                for _, mapping := range mappings {
                    if mapping.ID == fieldID {
                        batchMappings = append(batchMappings, mapping)
                        break
                    }
                }
            }
            
            batch := &domain.ConcurrentBatch{
                ID:          fmt.Sprintf("batch_%d_%d", i, j),
                Fields:      batchMappings,
                DependsOn:   o.calculateBatchDependencies(i, levelSets),
                Complexity:  o.calculateBatchComplexity(batchMappings),
                EstimatedMS: o.estimateProcessingTime(batchMappings),
            }
            
            batches = append(batches, batch)
        }
    }
    
    return batches, nil
}

// optimizeBatchSize splits large batches for better load balancing
func (o *BatchOptimizer) optimizeBatchSize(ctx context.Context, fieldIDs []string, mappings []*domain.FieldMapping) [][]string {
    if len(fieldIDs) <= o.config.MaxBatchSize {
        return [][]string{fieldIDs}
    }
    
    // Sort by processing complexity for balanced distribution
    complexityMap := make(map[string]int)
    for _, fieldID := range fieldIDs {
        for _, mapping := range mappings {
            if mapping.ID == fieldID {
                complexityMap[fieldID] = o.calculateMappingComplexity(mapping)
                break
            }
        }
    }
    
    // Use greedy algorithm for balanced partitioning
    return o.balancedPartition(fieldIDs, complexityMap, o.config.MaxBatchSize)
}
```

### Resource Management

```go
// ResourceManager handles resource allocation and constraints
type ResourceManager struct {
    systemResources *SystemResourceMonitor
    config          *ResourceConfig
    logger          *zap.Logger
}

// AllocateResources determines optimal resource allocation for batches
func (r *ResourceManager) AllocateResources(ctx context.Context, batches []*domain.ConcurrentBatch) (*domain.ResourceAllocation, error) {
    systemLimits := r.systemResources.GetCurrentLimits()
    
    // Calculate total resource requirements
    totalComplexity := 0
    for _, batch := range batches {
        totalComplexity += batch.Complexity
    }
    
    // Determine optimal goroutine allocation
    maxGoroutines := min(systemLimits.MaxGoroutines, r.config.MaxConcurrentFields)
    
    // Allocate memory budgets
    memoryPerBatch := systemLimits.MaxMemoryMB / len(batches)
    if memoryPerBatch < r.config.MinMemoryPerBatchMB {
        return nil, fmt.Errorf("insufficient memory for concurrent processing")
    }
    
    // Create resource allocation
    allocation := &domain.ResourceAllocation{
        MaxConcurrentBatches: r.calculateOptimalConcurrency(batches, systemLimits),
        MemoryLimitMB:        systemLimits.MaxMemoryMB,
        TimeoutMS:            r.config.DefaultTimeoutMS,
        GoroutinePool:        r.createGoroutinePoolConfig(maxGoroutines),
    }
    
    return allocation, nil
}

// SystemResourceMonitor tracks available system resources
type SystemResourceMonitor struct {
    cpuCores     int
    memoryMB     int
    loadAverage  float64
    mutex        sync.RWMutex
}

func (m *SystemResourceMonitor) GetCurrentLimits() *SystemLimits {
    m.mutex.RLock()
    defer m.mutex.RUnlock()
    
    // Adjust limits based on current system load
    availableCores := max(1, int(float64(m.cpuCores)*(1.0-m.loadAverage)))
    availableMemory := int(float64(m.memoryMB) * 0.8) // Leave 20% buffer
    
    return &SystemLimits{
        MaxGoroutines: availableCores * 2, // 2 goroutines per core
        MaxMemoryMB:   availableMemory,
        MaxCPUPercent: 80.0,
    }
}
```

### Execution Strategy Selection

```go
// ExecutionStrategySelector chooses optimal execution strategy
type ExecutionStrategySelector struct {
    profiler *PerformanceProfiler
    config   *StrategyConfig
}

// SelectStrategy determines the best execution strategy for the plan
func (s *ExecutionStrategySelector) SelectStrategy(ctx context.Context, plan *domain.ExecutionPlan) domain.ExecutionStrategy {
    totalFields := s.countTotalFields(plan)
    avgComplexity := s.calculateAverageComplexity(plan)
    parallelizationRatio := s.calculateParallelizationRatio(plan)
    
    // Decision matrix based on characteristics
    switch {
    case totalFields < 5:
        return domain.StrategySequential
        
    case parallelizationRatio < 0.3:
        return domain.StrategySequential
        
    case totalFields < 20 && avgComplexity < 3:
        return domain.StrategyBatched
        
    case parallelizationRatio > 0.7:
        return domain.StrategyFullyConcurrent
        
    default:
        return domain.StrategyAdaptive
    }
}

// PerformanceProfiler estimates processing characteristics
type PerformanceProfiler struct {
    historyData *ProcessingHistory
    cache       *ProfileCache
}

func (p *PerformanceProfiler) EstimateProcessingTime(mapping *domain.FieldMapping) time.Duration {
    // Use historical data and complexity analysis to estimate time
    baseTime := p.getBaseProcessingTime(mapping.Strategy.Name())
    complexityMultiplier := p.calculateComplexityMultiplier(mapping)
    
    return time.Duration(float64(baseTime) * complexityMultiplier)
}
```

## Event Integration

### Plan Event Definition

```go
// PlanEvent represents successful planning completion
type PlanEvent struct {
    BaseEvent
    Plan     *domain.ExecutionPlan
    Metrics  PlanMetrics
}

// PlanMetrics track planning performance
type PlanMetrics struct {
    PlanningDurationMS    int64
    MethodsPlanned        int
    TotalFields           int
    ConcurrentBatches     int
    ParallelizationRatio  float64
    EstimatedSpeedupRatio float64
}
```

## Validation and Error Handling

### Plan Validation

```go
// PlanValidator ensures execution plans are correct and feasible
type PlanValidator struct {
    resourceValidator *ResourceValidator
    dependencyValidator *DependencyValidator
    logger *zap.Logger
}

func (v *PlanValidator) ValidatePlan(ctx context.Context, plan *domain.ExecutionPlan) error {
    var errors []error
    
    // Validate resource constraints
    if err := v.resourceValidator.ValidateResources(ctx, plan); err != nil {
        errors = append(errors, fmt.Errorf("resource validation failed: %w", err))
    }
    
    // Validate dependency consistency
    if err := v.dependencyValidator.ValidateDependencies(ctx, plan); err != nil {
        errors = append(errors, fmt.Errorf("dependency validation failed: %w", err))
    }
    
    // Validate batch ordering
    if err := v.validateBatchOrdering(ctx, plan); err != nil {
        errors = append(errors, fmt.Errorf("batch ordering validation failed: %w", err))
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("plan validation failed: %v", errors)
    }
    
    return nil
}
```

This design provides a comprehensive planning system that creates optimized execution plans for concurrent field processing while maintaining deterministic output ordering and respecting system resource constraints.