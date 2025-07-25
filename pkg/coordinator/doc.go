// Package coordinator provides the central orchestration system for the Convergen
// pipeline. It manages the event-driven flow between parser, planner, executor,
// and emitter components while handling resource management, error aggregation,
// and performance monitoring.
//
// The coordinator implements a layered architecture with clear separation between
// orchestration logic and component management:
//
//   - Pipeline Orchestrator: Controls the overall flow and sequencing
//   - Event Bus Management: Handles inter-component communication
//   - Resource Pool: Manages shared resources like goroutines and memory
//   - Error Handler: Aggregates and reports errors from all components
//   - Metrics Collector: Tracks performance and execution statistics
//   - Context Manager: Handles cancellation and timeout propagation
//
// The coordinator supports both synchronous and asynchronous pipeline execution,
// with comprehensive error handling, resource cleanup, and extensibility through
// custom components and middleware.
//
// Example usage:
//
//	config := coordinator.DefaultConfig()
//	coord := coordinator.New(logger, config)
//	
//	result, err := coord.Generate(ctx, []string{"input.go"}, config)
//	if err != nil {
//		log.Fatal(err)
//	}
//	
//	fmt.Println("Generated:", result.Code)
//
// The coordinator provides thread-safe operations and handles concurrent access
// to shared resources through proper synchronization mechanisms.
package coordinator