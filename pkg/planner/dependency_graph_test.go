package planner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/reedom/convergen/v8/pkg/domain"
)

func TestDependencyGraph_BasicOperations(t *testing.T) {
	logger := zaptest.NewLogger(t)
	graph := NewDependencyGraph(logger)

	// Test empty graph
	assert.Equal(t, 0, graph.Size())
	assert.Equal(t, 0, graph.DependencyCount())

	// Add a field
	mapping1 := createTestFieldMapping(1, 1)
	err := graph.AddField(mapping1)
	require.NoError(t, err)

	assert.Equal(t, 1, graph.Size())
	assert.Equal(t, 0, graph.DependencyCount())

	// Retrieve the field
	retrieved, exists := graph.GetField(mapping1.ID)
	assert.True(t, exists)
	assert.Equal(t, mapping1, retrieved)

	// Add another field
	mapping2 := createTestFieldMapping(1, 2)
	err = graph.AddField(mapping2)
	require.NoError(t, err)

	assert.Equal(t, 2, graph.Size())
}

func TestDependencyGraph_Dependencies(t *testing.T) {
	logger := zaptest.NewLogger(t)
	graph := NewDependencyGraph(logger)

	// Add fields
	mapping1 := createTestFieldMapping(1, 1)
	mapping2 := createTestFieldMapping(1, 2)
	mapping3 := createTestFieldMapping(1, 3)

	err := graph.AddField(mapping1)
	require.NoError(t, err)
	err = graph.AddField(mapping2)
	require.NoError(t, err)
	err = graph.AddField(mapping3)
	require.NoError(t, err)

	// Add dependencies: mapping2 depends on mapping1, mapping3 depends on mapping2
	err = graph.AddDependency(mapping2.ID, mapping1.ID)
	require.NoError(t, err)
	err = graph.AddDependency(mapping3.ID, mapping2.ID)
	require.NoError(t, err)

	assert.Equal(t, 2, graph.DependencyCount())

	// Test dependency retrieval
	deps1 := graph.GetDependencies(mapping1.ID)
	assert.Empty(t, deps1)

	deps2 := graph.GetDependencies(mapping2.ID)
	assert.Contains(t, deps2, mapping1.ID)

	deps3 := graph.GetDependencies(mapping3.ID)
	assert.Contains(t, deps3, mapping2.ID)

	// Test dependents
	dependents1 := graph.GetDependents(mapping1.ID)
	assert.Contains(t, dependents1, mapping2.ID)

	dependents2 := graph.GetDependents(mapping2.ID)
	assert.Contains(t, dependents2, mapping3.ID)

	dependents3 := graph.GetDependents(mapping3.ID)
	assert.Empty(t, dependents3)
}

func TestDependencyGraph_TopologicalSort(t *testing.T) {
	tests := []struct {
		name            string
		setupGraph      func(DependencyGraph) error
		expectedBatches int
		expectedError   bool
		validateBatches func([]*domain.FieldMapping, [][]*domain.FieldMapping) bool
	}{
		{
			name: "no dependencies",
			setupGraph: func(g DependencyGraph) error {
				for i := 1; i <= 3; i++ {
					mapping := createTestFieldMapping(1, i)
					if err := g.AddField(mapping); err != nil {
						return err
					}
				}
				return nil
			},
			expectedBatches: 1,
			expectedError:   false,
			validateBatches: func(mappings []*domain.FieldMapping, batches [][]*domain.FieldMapping) bool {
				return len(batches[0]) == 3 // All in one batch
			},
		},
		{
			name: "linear dependencies",
			setupGraph: func(g DependencyGraph) error {
				mappings := make([]*domain.FieldMapping, 3)
				for i := 0; i < 3; i++ {
					mappings[i] = createTestFieldMapping(1, i+1)
					if err := g.AddField(mappings[i]); err != nil {
						return err
					}
				}
				// Chain dependencies: 0 <- 1 <- 2
				if err := g.AddDependency(mappings[1].ID, mappings[0].ID); err != nil {
					return err
				}
				return g.AddDependency(mappings[2].ID, mappings[1].ID)
			},
			expectedBatches: 3,
			expectedError:   false,
			validateBatches: func(mappings []*domain.FieldMapping, batches [][]*domain.FieldMapping) bool {
				// Each batch should have one item
				return len(batches[0]) == 1 && len(batches[1]) == 1 && len(batches[2]) == 1
			},
		},
		{
			name: "partial dependencies",
			setupGraph: func(g DependencyGraph) error {
				mappings := make([]*domain.FieldMapping, 4)
				for i := 0; i < 4; i++ {
					mappings[i] = createTestFieldMapping(1, i+1)
					if err := g.AddField(mappings[i]); err != nil {
						return err
					}
				}
				// Dependencies: 0 <- 1, 2 is independent, 2 <- 3
				if err := g.AddDependency(mappings[1].ID, mappings[0].ID); err != nil {
					return err
				}
				return g.AddDependency(mappings[3].ID, mappings[2].ID)
			},
			expectedBatches: 2,
			expectedError:   false,
			validateBatches: func(mappings []*domain.FieldMapping, batches [][]*domain.FieldMapping) bool {
				// First batch should have 2 items (0 and 2), second batch should have 2 items (1 and 3)
				return len(batches[0]) == 2 && len(batches[1]) == 2
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			graph := NewDependencyGraph(logger)

			err := tt.setupGraph(graph)
			require.NoError(t, err)

			batches, err := graph.TopologicalSort()

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, batches, tt.expectedBatches)

			// Validate batch structure if provided
			if tt.validateBatches != nil {
				var allMappings []*domain.FieldMapping
				for _, batch := range batches {
					allMappings = append(allMappings, batch...)
				}

				assert.True(t, tt.validateBatches(allMappings, batches))
			}

			// Verify all mappings are included
			totalMappings := 0
			for _, batch := range batches {
				totalMappings += len(batch)
			}

			assert.Equal(t, graph.Size(), totalMappings)
		})
	}
}

func TestDependencyGraph_CycleDetection(t *testing.T) {
	tests := []struct {
		name           string
		setupGraph     func(DependencyGraph) ([]*domain.FieldMapping, error)
		expectedCycles int
		hasCycles      bool
	}{
		{
			name: "no cycles",
			setupGraph: func(g DependencyGraph) ([]*domain.FieldMapping, error) {
				mappings := make([]*domain.FieldMapping, 3)
				for i := 0; i < 3; i++ {
					mappings[i] = createTestFieldMapping(1, i+1)
					if err := g.AddField(mappings[i]); err != nil {
						return nil, err
					}
				}
				// Linear chain: 0 <- 1 <- 2
				if err := g.AddDependency(mappings[1].ID, mappings[0].ID); err != nil {
					return nil, err
				}
				if err := g.AddDependency(mappings[2].ID, mappings[1].ID); err != nil {
					return nil, err
				}
				return mappings, nil
			},
			expectedCycles: 0,
			hasCycles:      false,
		},
		{
			name: "simple cycle",
			setupGraph: func(g DependencyGraph) ([]*domain.FieldMapping, error) {
				mappings := make([]*domain.FieldMapping, 2)
				for i := 0; i < 2; i++ {
					mappings[i] = createTestFieldMapping(1, i+1)
					if err := g.AddField(mappings[i]); err != nil {
						return nil, err
					}
				}
				// Circular dependency: 0 <- 1 <- 0
				if err := g.AddDependency(mappings[1].ID, mappings[0].ID); err != nil {
					return nil, err
				}
				if err := g.AddDependency(mappings[0].ID, mappings[1].ID); err != nil {
					return nil, err
				}
				return mappings, nil
			},
			expectedCycles: 1,
			hasCycles:      true,
		},
		{
			name: "complex cycle",
			setupGraph: func(g DependencyGraph) ([]*domain.FieldMapping, error) {
				mappings := make([]*domain.FieldMapping, 4)
				for i := 0; i < 4; i++ {
					mappings[i] = createTestFieldMapping(1, i+1)
					if err := g.AddField(mappings[i]); err != nil {
						return nil, err
					}
				}
				// Cycle: 0 <- 1 <- 2 <- 0, and 3 is independent
				if err := g.AddDependency(mappings[1].ID, mappings[0].ID); err != nil {
					return nil, err
				}
				if err := g.AddDependency(mappings[2].ID, mappings[1].ID); err != nil {
					return nil, err
				}
				if err := g.AddDependency(mappings[0].ID, mappings[2].ID); err != nil {
					return nil, err
				}
				return mappings, nil
			},
			expectedCycles: 1,
			hasCycles:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			graph := NewDependencyGraph(logger)

			mappings, err := tt.setupGraph(graph)
			require.NoError(t, err)
			require.NotEmpty(t, mappings)

			cycles, err := graph.DetectCycles()
			require.NoError(t, err)

			if tt.hasCycles {
				assert.GreaterOrEqual(t, len(cycles), tt.expectedCycles)

				// Verify cycles contain valid mapping IDs
				for _, cycle := range cycles {
					assert.GreaterOrEqual(t, len(cycle), 2) // A cycle must have at least 2 nodes

					for _, id := range cycle {
						_, exists := graph.GetField(id)
						assert.True(t, exists, "Cycle contains invalid mapping ID: %s", id)
					}
				}
			} else {
				assert.Empty(t, cycles)
			}
		})
	}
}

func TestDependencyGraph_RemoveDependency(t *testing.T) {
	logger := zaptest.NewLogger(t)
	graph := NewDependencyGraph(logger)

	// Setup graph with dependencies
	mapping1 := createTestFieldMapping(1, 1)
	mapping2 := createTestFieldMapping(1, 2)

	err := graph.AddField(mapping1)
	require.NoError(t, err)
	err = graph.AddField(mapping2)
	require.NoError(t, err)
	err = graph.AddDependency(mapping2.ID, mapping1.ID)
	require.NoError(t, err)

	assert.Equal(t, 1, graph.DependencyCount())

	// Remove dependency
	err = graph.RemoveDependency(mapping2.ID, mapping1.ID)
	require.NoError(t, err)

	assert.Equal(t, 0, graph.DependencyCount())

	// Verify dependency is removed
	deps := graph.GetDependencies(mapping2.ID)
	assert.NotContains(t, deps, mapping1.ID)

	dependents := graph.GetDependents(mapping1.ID)
	assert.NotContains(t, dependents, mapping2.ID)
}

func TestDependencyGraph_ErrorHandling(t *testing.T) {
	logger := zaptest.NewLogger(t)
	graph := NewDependencyGraph(logger)

	// Test adding nil field
	err := graph.AddField(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")

	// Test adding field with empty ID
	invalidMapping := &domain.FieldMapping{ID: ""}
	err = graph.AddField(invalidMapping)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")

	// Test adding duplicate field
	mapping := createTestFieldMapping(1, 1)
	err = graph.AddField(mapping)
	require.NoError(t, err)
	err = graph.AddField(mapping)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// Test adding dependency with empty IDs
	err = graph.AddDependency("", "something")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")

	err = graph.AddDependency("something", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")

	// Test self-dependency
	err = graph.AddDependency(mapping.ID, mapping.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot depend on itself")

	// Test dependency with non-existent field
	err = graph.AddDependency(mapping.ID, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDependencyGraph_Clear(t *testing.T) {
	logger := zaptest.NewLogger(t)
	graph := NewDependencyGraph(logger)

	// Add some fields and dependencies
	mapping1 := createTestFieldMapping(1, 1)
	mapping2 := createTestFieldMapping(1, 2)

	err := graph.AddField(mapping1)
	require.NoError(t, err)
	err = graph.AddField(mapping2)
	require.NoError(t, err)
	err = graph.AddDependency(mapping2.ID, mapping1.ID)
	require.NoError(t, err)

	assert.Equal(t, 2, graph.Size())
	assert.Equal(t, 1, graph.DependencyCount())

	// Clear the graph
	graph.Clear()

	assert.Equal(t, 0, graph.Size())
	assert.Equal(t, 0, graph.DependencyCount())

	// Verify fields are gone
	_, exists := graph.GetField(mapping1.ID)
	assert.False(t, exists)
	_, exists = graph.GetField(mapping2.ID)
	assert.False(t, exists)
}

func TestDependencyGraph_GetExecutionOrder(t *testing.T) {
	logger := zaptest.NewLogger(t)
	graph := NewDependencyGraph(logger)

	// Create a more complex dependency graph
	mappings := make([]*domain.FieldMapping, 5)
	for i := 0; i < 5; i++ {
		mappings[i] = createTestFieldMapping(1, i+1)
		err := graph.AddField(mappings[i])
		require.NoError(t, err)
	}

	// Dependencies: 0 <- 1, 0 <- 2, 3 <- 4
	err := graph.AddDependency(mappings[1].ID, mappings[0].ID)
	require.NoError(t, err)
	err = graph.AddDependency(mappings[2].ID, mappings[0].ID)
	require.NoError(t, err)
	err = graph.AddDependency(mappings[4].ID, mappings[3].ID)
	require.NoError(t, err)

	batches, err := graph.GetExecutionOrder()
	require.NoError(t, err)

	// Verify batch structure
	assert.GreaterOrEqual(t, len(batches), 2)

	// Verify all mappings are included
	totalMappings := 0
	for _, batch := range batches {
		totalMappings += len(batch.Mappings)
		assert.NotEmpty(t, batch.ID)
		assert.NotNil(t, batch.ResourceRequirement)
		assert.GreaterOrEqual(t, batch.ConcurrencyLevel, 1)
	}

	assert.Equal(t, graph.Size(), totalMappings)
}

func BenchmarkDependencyGraph_AddField(b *testing.B) {
	logger := zaptest.NewLogger(b)
	graph := NewDependencyGraph(logger)

	mappings := make([]*domain.FieldMapping, b.N)
	for i := 0; i < b.N; i++ {
		mappings[i] = createTestFieldMapping(1, i+1)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := graph.AddField(mappings[i])
		require.NoError(b, err)
	}
}

func BenchmarkDependencyGraph_TopologicalSort(b *testing.B) {
	logger := zaptest.NewLogger(b)

	// Create a graph with many fields and some dependencies
	graph := NewDependencyGraph(logger)
	numFields := 1000

	mappings := make([]*domain.FieldMapping, numFields)
	for i := 0; i < numFields; i++ {
		mappings[i] = createTestFieldMapping(1, i+1)
		err := graph.AddField(mappings[i])
		require.NoError(b, err)
	}

	// Add some dependencies (every 10th field depends on the previous one)
	for i := 10; i < numFields; i += 10 {
		err := graph.AddDependency(mappings[i].ID, mappings[i-10].ID)
		require.NoError(b, err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := graph.TopologicalSort()
		require.NoError(b, err)
	}
}
