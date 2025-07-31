// Package coordinator provides pipeline orchestration and coordination functionality.
package coordinator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v8/pkg/domain"
	"github.com/reedom/convergen/v8/pkg/internal/events"
)

// Static errors for err113 compliance.
var (
	ErrNoSourceFilesProvided   = errors.New("no source files provided")
	ErrNoSourceCodeProvided    = errors.New("no source code provided")
	ErrCoordinatorShuttingDown = errors.New("coordinator shutting down")
	ErrPipelineFailed          = errors.New("pipeline failed")
	ErrPipelineTimeout         = errors.New("pipeline timeout")
	ErrShutdownErrors          = errors.New("shutdown errors")
)

// Coordinator orchestrates the entire Convergen pipeline.
type Coordinator interface {
	// Generate code from source files
	Generate(ctx context.Context, sources []string, config *Config) (*GenerationResult, error)

	// Generate from in-memory source code
	GenerateFromSource(ctx context.Context, source string, config *Config) (*GenerationResult, error)

	// Get coordinator metrics
	GetMetrics() *Metrics

	// Get current pipeline status
	GetStatus() *PipelineStatus

	// Shutdown gracefully
	Shutdown(ctx context.Context) error
}

// ConcreteCoordinator implements the Coordinator interface.
type ConcreteCoordinator struct {
	config            *Config
	logger            *zap.Logger
	eventBus          events.EventBus
	componentMgr      ComponentManager
	eventOrchestrator EventOrchestrator
	resourcePool      ResourcePool
	errorHandler      ErrorHandler
	metricsCollector  MetricsCollector
	contextMgr        ContextManager

	// State management
	mutex    sync.RWMutex
	status   *PipelineStatus
	shutdown chan struct{}
	running  bool
}

// New creates a new coordinator instance.
func New(logger *zap.Logger, config *Config) Coordinator {
	if config == nil {
		config = DefaultConfig()
	}

	coord := &ConcreteCoordinator{
		config:   config,
		logger:   logger,
		shutdown: make(chan struct{}),
		status: &PipelineStatus{
			Stage:           StageInitializing,
			ComponentStatus: make(map[string]ComponentStatus),
			StartTime:       time.Now(),
		},
	}

	// Initialize core subsystems
	coord.initializeSubsystems()

	logger.Info("coordinator initialized",
		zap.String("version", "2.0.0"),
		zap.Bool("metrics_enabled", config.EnableMetrics),
		zap.Int("max_concurrency", config.MaxConcurrency))

	return coord
}

// Generate processes source files through the complete pipeline.
func (c *ConcreteCoordinator) Generate(ctx context.Context, sources []string, config *Config) (*GenerationResult, error) {
	if len(sources) == 0 {
		return nil, ErrNoSourceFilesProvided
	}

	c.logger.Info("starting code generation",
		zap.Strings("sources", sources),
		zap.Int("source_count", len(sources)))

	startTime := time.Now()

	// Create pipeline input
	input := &PipelineInput{
		Sources: sources,
		Config:  config,
		Context: ctx,
		Metadata: map[string]interface{}{
			"generation_id": c.generatePipelineID(),
			"start_time":    startTime,
		},
	}

	// Execute pipeline
	result, err := c.executePipeline(ctx, input)
	if err != nil {
		c.logger.Error("pipeline execution failed",
			zap.Error(err),
			zap.Duration("duration", time.Since(startTime)))

		return result, err
	}

	// Record successful execution
	c.metricsCollector.RecordEvent("pipeline_completed", time.Since(startTime), map[string]interface{}{
		"sources":     len(sources),
		"success":     true,
		"duration_ms": time.Since(startTime).Milliseconds(),
	})

	c.logger.Info("code generation completed successfully",
		zap.Duration("duration", time.Since(startTime)),
		zap.Int("methods_generated", len(result.Methods)))

	return result, nil
}

// GenerateFromSource processes in-memory source code.
func (c *ConcreteCoordinator) GenerateFromSource(ctx context.Context, source string, config *Config) (*GenerationResult, error) {
	if source == "" {
		return nil, ErrNoSourceCodeProvided
	}

	c.logger.Debug("starting generation from source code",
		zap.Int("source_length", len(source)))

	startTime := time.Now()

	// Create pipeline input
	input := &PipelineInput{
		SourceCode: source,
		Config:     config,
		Context:    ctx,
		Metadata: map[string]interface{}{
			"generation_id": c.generatePipelineID(),
			"start_time":    startTime,
			"source_type":   "in_memory",
		},
	}

	// Execute pipeline
	result, err := c.executePipeline(ctx, input)
	if err != nil {
		c.logger.Error("pipeline execution failed for source code",
			zap.Error(err),
			zap.Duration("duration", time.Since(startTime)))

		return result, err
	}

	c.logger.Debug("generation from source code completed",
		zap.Duration("duration", time.Since(startTime)))

	return result, nil
}

// GetMetrics returns current coordinator metrics.
func (c *ConcreteCoordinator) GetMetrics() *Metrics {
	return c.metricsCollector.GetMetrics()
}

// GetStatus returns current pipeline status.
func (c *ConcreteCoordinator) GetStatus() *PipelineStatus {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Create a copy to avoid race conditions
	status := *c.status
	status.ElapsedTime = time.Since(c.status.StartTime)

	return &status
}

// Shutdown gracefully shuts down the coordinator.
func (c *ConcreteCoordinator) Shutdown(ctx context.Context) error {
	c.mutex.Lock()
	if !c.running {
		c.mutex.Unlock()
		return nil
	}

	c.running = false
	c.mutex.Unlock()

	c.logger.Info("shutting down coordinator")

	// Signal shutdown
	close(c.shutdown)

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, c.config.ComponentTimeout)
	defer cancel()

	// Shutdown components in reverse order
	var shutdownErrors []error

	if err := c.eventOrchestrator.Cancel(); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("event orchestrator shutdown: %w", err))
	}

	if err := c.componentMgr.Shutdown(shutdownCtx); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("component manager shutdown: %w", err))
	}

	if err := c.resourcePool.Release(shutdownCtx); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("resource pool shutdown: %w", err))
	}

	if err := c.eventBus.Close(); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("event bus shutdown: %w", err))
	}

	// Update status
	c.mutex.Lock()

	c.status.Stage = StageCompleted
	for component := range c.status.ComponentStatus {
		c.status.ComponentStatus[component] = StatusShutdown
	}
	c.mutex.Unlock()

	if len(shutdownErrors) > 0 {
		c.logger.Warn("coordinator shutdown completed with errors",
			zap.Int("error_count", len(shutdownErrors)))
		return fmt.Errorf("%w: %v", ErrShutdownErrors, shutdownErrors)
	}

	c.logger.Info("coordinator shutdown completed successfully")

	return nil
}

// Private methods

func (c *ConcreteCoordinator) initializeSubsystems() {
	// Initialize event bus
	c.eventBus = events.NewInMemoryEventBus(c.logger)

	// Initialize subsystems
	c.componentMgr = NewComponentManager(c.logger, c.config)
	c.eventOrchestrator = NewEventOrchestrator(c.logger, c.eventBus, c.config)
	c.resourcePool = NewResourcePool(c.logger, c.config)
	c.errorHandler = NewErrorHandler(c.logger, c.config)
	c.metricsCollector = NewMetricsCollector(c.logger, c.config)
	c.contextMgr = NewContextManager(c.logger, c.config)

	// Set up cross-references between subsystems
	c.eventOrchestrator.SetComponentManager(c.componentMgr)
	c.eventOrchestrator.SetErrorHandler(c.errorHandler)

	c.running = true
}

func (c *ConcreteCoordinator) executePipeline(ctx context.Context, input *PipelineInput) (*GenerationResult, error) {
	// Create pipeline context with timeout
	pipelineCtx, cancel := c.contextMgr.CreatePipelineContext(ctx, c.config.ComponentTimeout)
	defer cancel()

	// Update pipeline status
	c.updateStatus(StageParsing, 0.0)

	// Initialize components if not already done
	if err := c.componentMgr.Initialize(pipelineCtx, c.config); err != nil {
		return nil, fmt.Errorf("component initialization failed: %w", err)
	}

	// Reset error handler for new pipeline
	c.errorHandler.Reset()

	// Start pipeline orchestration
	if err := c.eventOrchestrator.StartPipeline(pipelineCtx, input); err != nil {
		return nil, fmt.Errorf("pipeline orchestration failed: %w", err)
	}

	// Wait for pipeline completion
	result, err := c.waitForCompletion(pipelineCtx, input)
	if err != nil {
		c.updateStatus(StageFailed, 1.0)
		return result, err
	}

	c.updateStatus(StageCompleted, 1.0)

	return result, nil
}

func (c *ConcreteCoordinator) waitForCompletion(ctx context.Context, input *PipelineInput) (*GenerationResult, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled: %w", ctx.Err())

		case <-c.shutdown:
			return nil, ErrCoordinatorShuttingDown

		case <-ticker.C:
			status := c.eventOrchestrator.GetStatus()

			// Update progress
			progress := c.calculateProgress(status.Stage)
			c.updateStatus(status.Stage, progress)

			// Check for completion
			if status.Stage == StageCompleted {
				return c.assembleResult(startTime, input, status)
			}

			// Check for failure
			if status.Stage == StageFailed {
				errors := c.errorHandler.GetErrors()
				return nil, fmt.Errorf("%w: %s", ErrPipelineFailed, c.formatErrors(errors))
			}

			// Check for timeout
			if time.Since(startTime) > c.config.ComponentTimeout {
				return nil, fmt.Errorf("%w after %v", ErrPipelineTimeout, c.config.ComponentTimeout)
			}
		}
	}
}

func (c *ConcreteCoordinator) assembleResult(startTime time.Time, input *PipelineInput, status *PipelineStatus) (*GenerationResult, error) {
	duration := time.Since(startTime)

	// Get component metrics
	componentMetrics := make(map[string]interface{})
	for name, component := range c.componentMgr.GetComponents() {
		componentMetrics[name] = component.GetMetrics()
	}

	// Create metadata
	metadata := &GenerationMetadata{
		Timestamp:          startTime,
		CoordinatorVersion: "2.0.0",
		PipelineID:         input.Metadata["generation_id"].(string),
		InputSources:       input.Sources,
		ComponentVersions:  c.getComponentVersions(),
		ProcessingStages:   []StageMetadata{}, // TODO: collect from orchestrator
	}

	// Get final result from event orchestrator
	orchestrator := c.eventOrchestrator.(*ConcreteEventOrchestrator)

	executionResults, err := orchestrator.GetFinalResult()
	if err != nil {
		c.logger.Warn("failed to get final result from orchestrator", zap.Error(err))
		// Continue with empty result rather than failing
		executionResults = &domain.ExecutionResults{}
	}

	result := &GenerationResult{
		Code:       "",         // Will be populated by emitter result
		Imports:    []string{}, // Will be populated by emitter result
		Methods:    executionResults.Methods,
		Metadata:   metadata,
		Duration:   duration,
		Metrics:    c.metricsCollector.GetMetrics(),
		Status:     StageCompleted,
		Components: status.ComponentStatus,
	}

	// Add error report if there were non-critical errors
	if errors := c.errorHandler.GetErrors(); errors.TotalCount > 0 {
		result.Errors = errors

		if len(errors.Warnings) > 0 {
			for _, warning := range errors.Warnings {
				result.Warnings = append(result.Warnings, warning.Error())
			}
		}
	}

	return result, nil
}

func (c *ConcreteCoordinator) updateStatus(stage PipelineStage, progress float64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.status.Stage = stage
	c.status.Progress = progress
	c.status.ElapsedTime = time.Since(c.status.StartTime)
}

func (c *ConcreteCoordinator) calculateProgress(stage PipelineStage) float64 {
	switch stage {
	case StageInitializing:
		return 0.0
	case StageParsing:
		return 0.2
	case StagePlanning:
		return 0.4
	case StageExecuting:
		return 0.6
	case StageEmitting:
		return 0.8
	case StageCompleted:
		return 1.0
	case StageFailed:
		return 1.0
	default:
		return 0.0
	}
}

func (c *ConcreteCoordinator) generatePipelineID() string {
	return fmt.Sprintf("pipeline_%d_%d", time.Now().Unix(), time.Now().Nanosecond())
}

func (c *ConcreteCoordinator) getComponentVersions() map[string]string {
	// TODO: Get actual versions from components
	return map[string]string{
		"parser":   "2.0.0",
		"planner":  "2.0.0",
		"executor": "2.0.0",
		"emitter":  "2.0.0",
	}
}

func (c *ConcreteCoordinator) formatErrors(errors *ErrorReport) string {
	if errors.TotalCount == 0 {
		return "no errors"
	}

	if errors.CriticalCount > 0 {
		return fmt.Sprintf("%d critical errors, %d total errors",
			errors.CriticalCount, errors.TotalCount)
	}

	return fmt.Sprintf("%d total errors", errors.TotalCount)
}
