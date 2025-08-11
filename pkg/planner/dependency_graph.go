package planner

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v9/pkg/domain"
)

// Static errors for err113 compliance.
var (
	ErrFieldMappingNil           = errors.New("field mapping cannot be nil")
	ErrFieldMappingIDEmpty       = errors.New("field mapping ID cannot be empty")
	ErrFieldMappingAlreadyExists = errors.New("field mapping already exists")
	ErrDependencyIDsEmpty        = errors.New("dependency IDs cannot be empty")
	ErrFieldSelfDependency       = errors.New("field cannot depend on itself")
	ErrSourceFieldNotFound       = errors.New("source field not found in graph")
	ErrTargetFieldNotFound       = errors.New("target field not found in graph")
	ErrFieldsNotFoundInGraph     = errors.New("one or both fields not found in graph")
	ErrDependencyCycleDetected   = errors.New("dependency cycle detected")
)

// DependencyGraph represents dependencies between field mappings.
type DependencyGraph interface {
	AddField(mapping *domain.FieldMapping) error
	AddDependency(from, to string) error
	RemoveDependency(from, to string) error
	TopologicalSort() ([][]*domain.FieldMapping, error)
	DetectCycles() ([][]string, error)
	GetExecutionOrder() ([]*ExecutionBatch, error)
	Size() int
	DependencyCount() int
	MethodCount() int
	Clear()
	GetField(id string) (*domain.FieldMapping, bool)
	GetDependencies(id string) []string
	GetDependents(id string) []string
}

// GraphNode represents a node in the dependency graph.
type GraphNode struct {
	ID           string               `json:"id"`
	Mapping      *domain.FieldMapping `json:"mapping"`
	Dependencies []string             `json:"dependencies"` // IDs this node depends on
	Dependents   []string             `json:"dependents"`   // IDs that depend on this node
	Visited      bool                 `json:"-"`            // For cycle detection
	InStack      bool                 `json:"-"`            // For cycle detection
	Order        int                  `json:"order"`        // Topological order
}

// ConcreteDependencyGraph implements DependencyGraph.
type ConcreteDependencyGraph struct {
	nodes       map[string]*GraphNode
	methodNodes map[string][]string // method name -> field IDs
	logger      *zap.Logger
	mutex       sync.RWMutex
}

// NewDependencyGraph creates a new dependency graph.
func NewDependencyGraph(logger *zap.Logger) DependencyGraph {
	return &ConcreteDependencyGraph{
		nodes:       make(map[string]*GraphNode),
		methodNodes: make(map[string][]string),
		logger:      logger,
	}
}

// AddField adds a field mapping to the dependency graph.
func (g *ConcreteDependencyGraph) AddField(mapping *domain.FieldMapping) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if mapping == nil {
		return ErrFieldMappingNil
	}

	if mapping.ID == "" {
		return ErrFieldMappingIDEmpty
	}

	// Check if field already exists
	if _, exists := g.nodes[mapping.ID]; exists {
		return fmt.Errorf("%w: %s", ErrFieldMappingAlreadyExists, mapping.ID)
	}

	// Create new node
	node := &GraphNode{
		ID:           mapping.ID,
		Mapping:      mapping,
		Dependencies: make([]string, 0),
		Dependents:   make([]string, 0),
		Visited:      false,
		InStack:      false,
		Order:        -1,
	}

	g.nodes[mapping.ID] = node

	// Track method associations (if we can determine the method)
	// This would be enhanced with proper method tracking

	g.logger.Debug("added field to dependency graph",
		zap.String("field_id", mapping.ID),
		zap.String("strategy", mapping.StrategyName))

	return nil
}

// AddDependency adds a dependency relationship between two fields.
func (g *ConcreteDependencyGraph) AddDependency(from, to string) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if from == "" || to == "" {
		return ErrDependencyIDsEmpty
	}

	if from == to {
		return fmt.Errorf("%w: %s", ErrFieldSelfDependency, from)
	}

	// Check that both nodes exist
	fromNode, fromExists := g.nodes[from]
	toNode, toExists := g.nodes[to]

	if !fromExists {
		return fmt.Errorf("%w: %s", ErrSourceFieldNotFound, from)
	}

	if !toExists {
		return fmt.Errorf("%w: %s", ErrTargetFieldNotFound, to)
	}

	// Check if dependency already exists
	for _, dep := range fromNode.Dependencies {
		if dep == to {
			return nil // Dependency already exists
		}
	}

	// Add dependency
	fromNode.Dependencies = append(fromNode.Dependencies, to)
	toNode.Dependents = append(toNode.Dependents, from)

	g.logger.Debug("added dependency",
		zap.String("from", from),
		zap.String("to", to))

	return nil
}

// RemoveDependency removes a dependency relationship.
func (g *ConcreteDependencyGraph) RemoveDependency(from, to string) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	fromNode, fromExists := g.nodes[from]
	toNode, toExists := g.nodes[to]

	if !fromExists || !toExists {
		return ErrFieldsNotFoundInGraph
	}

	// Remove from dependencies
	for i, dep := range fromNode.Dependencies {
		if dep == to {
			fromNode.Dependencies = append(fromNode.Dependencies[:i], fromNode.Dependencies[i+1:]...)
			break
		}
	}

	// Remove from dependents
	for i, dep := range toNode.Dependents {
		if dep == from {
			toNode.Dependents = append(toNode.Dependents[:i], toNode.Dependents[i+1:]...)
			break
		}
	}

	g.logger.Debug("removed dependency",
		zap.String("from", from),
		zap.String("to", to))

	return nil
}

// TopologicalSort returns field mappings in dependency order (batches).
func (g *ConcreteDependencyGraph) TopologicalSort() ([][]*domain.FieldMapping, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	// Kahn's algorithm for topological sorting
	inDegree := make(map[string]int)

	// Initialize in-degrees
	for id, node := range g.nodes {
		inDegree[id] = len(node.Dependencies)
	}

	var batches [][]*domain.FieldMapping

	queue := make([]string, 0)

	// Find nodes with no dependencies (in-degree 0)
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	processed := 0
	batchIndex := 0

	for len(queue) > 0 {
		// Process current batch (all nodes with in-degree 0)
		currentBatch := make([]*domain.FieldMapping, 0, len(queue))
		nextQueue := make([]string, 0)

		// Sort queue for deterministic output
		sort.Strings(queue)

		for _, nodeID := range queue {
			node := g.nodes[nodeID]
			currentBatch = append(currentBatch, node.Mapping)
			node.Order = batchIndex
			processed++

			// Reduce in-degree of dependent nodes
			for _, depID := range node.Dependents {
				inDegree[depID]--
				if inDegree[depID] == 0 {
					nextQueue = append(nextQueue, depID)
				}
			}
		}

		if len(currentBatch) > 0 {
			batches = append(batches, currentBatch)
		}

		queue = nextQueue
		batchIndex++
	}

	// Check for cycles (unprocessed nodes)
	if processed != len(g.nodes) {
		return nil, fmt.Errorf("%w: processed %d of %d nodes", ErrDependencyCycleDetected, processed, len(g.nodes))
	}

	g.logger.Info("topological sort completed",
		zap.Int("batches", len(batches)),
		zap.Int("total_fields", processed))

	return batches, nil
}

// DetectCycles detects circular dependencies using DFS.
func (g *ConcreteDependencyGraph) DetectCycles() ([][]string, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	// Reset visit state
	for _, node := range g.nodes {
		node.Visited = false
		node.InStack = false
	}

	var cycles [][]string

	var currentPath []string

	// DFS from each unvisited node
	for id, node := range g.nodes {
		if !node.Visited {
			if cycle := g.dfsDetectCycle(id, &currentPath); cycle != nil {
				cycles = append(cycles, cycle)
			}
		}
	}

	if len(cycles) > 0 {
		g.logger.Warn("cycles detected in dependency graph",
			zap.Int("cycle_count", len(cycles)))
	}

	return cycles, nil
}

// dfsDetectCycle performs DFS cycle detection.
func (g *ConcreteDependencyGraph) dfsDetectCycle(nodeID string, path *[]string) []string {
	node := g.nodes[nodeID]

	if node.InStack {
		// Found a cycle - extract it from the path
		cycleStart := -1

		for i, id := range *path {
			if id == nodeID {
				cycleStart = i
				break
			}
		}

		if 0 <= cycleStart {
			cycle := make([]string, len(*path)-cycleStart+1)
			copy(cycle, (*path)[cycleStart:])
			cycle[len(cycle)-1] = nodeID // Close the cycle

			return cycle
		}
	}

	if node.Visited {
		return nil
	}

	node.Visited = true
	node.InStack = true

	*path = append(*path, nodeID)

	// Visit dependencies
	for _, depID := range node.Dependencies {
		if cycle := g.dfsDetectCycle(depID, path); cycle != nil {
			return cycle
		}
	}

	node.InStack = false
	*path = (*path)[:len(*path)-1] // Remove from path

	return nil
}

// GetExecutionOrder returns execution batches in dependency order.
func (g *ConcreteDependencyGraph) GetExecutionOrder() ([]*ExecutionBatch, error) {
	batches, err := g.TopologicalSort()
	if err != nil {
		return nil, err
	}

	executionBatches := make([]*ExecutionBatch, len(batches))
	for i, batch := range batches {
		executionBatches[i] = &ExecutionBatch{
			ID:                  fmt.Sprintf("batch_%d", i),
			Mappings:            batch,
			EstimatedDurationMS: int64(len(batch) * 10), // Placeholder estimation
			ResourceRequirement: &ResourceRequirement{
				MemoryMB:     len(batch) * 2,
				CPUIntensive: false,
				IOOperations: 0,
			},
			DependsOn:        g.calculateBatchDependencies(i),
			ConcurrencyLevel: len(batch), // All fields in batch can run concurrently
		}
	}

	return executionBatches, nil
}

// calculateBatchDependencies determines which previous batches this batch depends on.
func (g *ConcreteDependencyGraph) calculateBatchDependencies(batchIndex int) []string {
	if batchIndex == 0 {
		return nil
	}
	// For simplicity, each batch depends on the previous one
	// In a more sophisticated implementation, we'd analyze actual dependencies
	return []string{fmt.Sprintf("batch_%d", batchIndex-1)}
}

// Size returns the number of fields in the graph.
func (g *ConcreteDependencyGraph) Size() int {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	return len(g.nodes)
}

// DependencyCount returns the total number of dependencies.
func (g *ConcreteDependencyGraph) DependencyCount() int {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	count := 0
	for _, node := range g.nodes {
		count += len(node.Dependencies)
	}

	return count
}

// MethodCount returns the number of methods represented in the graph.
func (g *ConcreteDependencyGraph) MethodCount() int {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	return len(g.methodNodes)
}

// Clear removes all nodes and dependencies from the graph.
func (g *ConcreteDependencyGraph) Clear() {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.nodes = make(map[string]*GraphNode)
	g.methodNodes = make(map[string][]string)

	g.logger.Debug("dependency graph cleared")
}

// GetField retrieves a field mapping by ID.
func (g *ConcreteDependencyGraph) GetField(id string) (*domain.FieldMapping, bool) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	if node, exists := g.nodes[id]; exists {
		return node.Mapping, true
	}

	return nil, false
}

// GetDependencies returns the IDs of fields that the given field depends on.
func (g *ConcreteDependencyGraph) GetDependencies(id string) []string {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	if node, exists := g.nodes[id]; exists {
		deps := make([]string, len(node.Dependencies))
		copy(deps, node.Dependencies)

		return deps
	}

	return nil
}

// GetDependents returns the IDs of fields that depend on the given field.
func (g *ConcreteDependencyGraph) GetDependents(id string) []string {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	if node, exists := g.nodes[id]; exists {
		deps := make([]string, len(node.Dependents))
		copy(deps, node.Dependents)

		return deps
	}

	return nil
}
