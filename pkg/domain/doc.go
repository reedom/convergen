// Package domain contains the core business entities and type system for Convergen.
//
// This package provides immutable, thread-safe domain models that represent:
//   - Type system with full generics support
//   - Field mappings and conversion strategies
//   - Method configurations and execution plans
//   - Error handling and result models
//
// The domain models are designed to be the single source of truth for all
// generation operations, ensuring consistency across the pipeline.
//
// Key types:
//   - Type: Represents Go types with generics support
//   - FieldMapping: Represents field conversion specifications
//   - Method: Represents a complete conversion method
//   - ExecutionPlan: Represents concurrent execution strategy
//   - GenerationError: Rich error context for failures
//
// All types are immutable after creation and safe for concurrent access.
package domain