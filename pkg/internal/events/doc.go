// Package events provides an event-driven coordination system for the Convergen pipeline.
//
// This package implements a thread-safe, in-memory event bus with middleware support
// for coordinating between different phases of code generation:
//   - Parsing events for AST analysis completion
//   - Planning events for execution plan generation
//   - Execution events for concurrent field processing
//   - Emission events for code generation completion
//   - Progress events for user feedback
//   - Error events for failure handling
//
// Key features:
//   - Concurrent event handling with goroutine safety
//   - Middleware chain for cross-cutting concerns (logging, timeouts)
//   - Event statistics and monitoring
//   - Context propagation for cancellation
//   - Rich event metadata
//
// The event system enables loose coupling between pipeline components while
// maintaining strong coordination and error handling capabilities.
package events