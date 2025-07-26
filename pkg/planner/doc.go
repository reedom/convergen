// Package planner provides intelligent execution planning for code generation.
//
// This package implements a sophisticated planner that analyzes field mappings,
// resolves dependencies, and creates optimized execution plans for concurrent
// field processing. The planner is designed to maximize performance while
// ensuring correctness and deterministic output.
//
// Key features:
//   - Dependency graph analysis with cycle detection
//   - Topological sorting for execution order
//   - Concurrent batch generation for field-level parallelism
//   - Resource allocation and limit enforcement
//   - Performance heuristics and optimization strategies
//   - Event-driven coordination with other pipeline components
//   - Comprehensive metrics and monitoring
//
// The planner operates in multiple phases:
//  1. Dependency Analysis: Build dependency graphs between field mappings
//  2. Cycle Detection: Identify and resolve circular dependencies
//  3. Batch Generation: Group independent fields for concurrent processing
//  4. Resource Planning: Calculate optimal worker allocation and limits
//  5. Plan Optimization: Apply heuristics for performance improvement
//  6. Event Emission: Notify other components of the execution plan
//
// All operations maintain deterministic output ordering while maximizing
// concurrency opportunities for improved performance on multi-core systems.
package planner
