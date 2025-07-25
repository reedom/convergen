// Package parser provides an event-driven AST parsing system for Convergen.
//
// This package implements a modern, concurrent parser that analyzes Go source files
// to extract conversion method specifications and emits events throughout the parsing
// process for coordination with other pipeline components.
//
// Key features:
//   - Event-driven architecture with progress tracking
//   - Concurrent type resolution with worker pools
//   - Comprehensive caching for performance optimization
//   - Full generics support with type parameter analysis
//   - Rich annotation processing with validation
//   - Error recovery and detailed reporting
//   - Integration with the new domain models
//
// The parser operates in multiple phases:
//   1. Source analysis and AST construction
//   2. Interface discovery and annotation extraction
//   3. Method signature analysis with type resolution
//   4. Concurrent annotation processing and validation
//   5. Result assembly with event emission
//
// All operations are thread-safe and designed for high performance with large
// codebases containing complex generic types and extensive annotations.
package parser