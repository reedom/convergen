package coordinator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v9/pkg/domain"
	"github.com/reedom/convergen/v9/pkg/internal/events"
)

// Static errors for err113 compliance.
var (
	ErrPipelineAlreadyStarted      = errors.New("pipeline already started")
	ErrPipelineCancelled           = errors.New("pipeline cancelled")
	ErrOrchestratorShutdown        = errors.New("orchestrator shutdown")
	ErrParseResultsNotFound        = errors.New("parse results not found in event data")
	ErrParsingStageFailed          = errors.New("parsing stage failed")
	ErrPlanResultsNotFound         = errors.New("plan results not found in event data")
	ErrPlanningStageFailed         = errors.New("planning stage failed")
	ErrExecuteResultsNotFound      = errors.New("execute results not found in event data")
	ErrExecutionStageFailed        = errors.New("execution stage failed")
	ErrEmitResultsNotFound         = errors.New("emit results not found in event data")
	ErrEmissionStageFailed         = errors.New("emission stage failed")
	ErrComponentNameNotFound       = errors.New("component name not found in status change event")
	ErrStatusNotFound              = errors.New("status not found in status change event")
	ErrEmissionResultsNotAvailable = errors.New("emission results not available")
	ErrInvalidEmissionResultType   = errors.New("invalid emission result type")
)

// EventOrchestrator manages event-driven coordination between components.
type EventOrchestrator interface {
	// Start pipeline execution
	StartPipeline(ctx context.Context, input *PipelineInput) error

	// Handle pipeline events
	HandleEvent(ctx context.Context, event events.Event) error

	// Get pipeline status
	GetStatus() *PipelineStatus

	// Cancel pipeline execution
	Cancel() error

	// Set component manager reference
	SetComponentManager(mgr ComponentManager)

	// Set error handler reference
	SetErrorHandler(handler ErrorHandler)
}

// ConcreteEventOrchestrator implements EventOrchestrator.
type ConcreteEventOrchestrator struct {
	logger       *zap.Logger
	config       *Config
	eventBus     events.EventBus
	componentMgr ComponentManager
	errorHandler ErrorHandler

	// Pipeline state
	mutex        sync.RWMutex
	status       *PipelineStatus
	currentInput *PipelineInput
	results      map[PipelineStage]interface{}
	cancelled    bool

	// Event handling
	eventHandlers map[string]events.EventHandler
	eventQueue    chan events.Event
	shutdown      chan struct{}
	started       bool
}

// NewEventOrchestrator creates a new event orchestrator.
func NewEventOrchestrator(logger *zap.Logger, eventBus events.EventBus, config *Config) EventOrchestrator {
	orchestrator := &ConcreteEventOrchestrator{
		logger:        logger,
		config:        config,
		eventBus:      eventBus,
		eventHandlers: make(map[string]events.EventHandler),
		eventQueue:    make(chan events.Event, config.EventBufferSize),
		shutdown:      make(chan struct{}),
		results:       make(map[PipelineStage]interface{}),
		status: &PipelineStatus{
			Stage:           StageInitializing,
			ComponentStatus: make(map[string]ComponentStatus),
			StartTime:       time.Now(),
		},
	}

	orchestrator.registerEventHandlers()

	return orchestrator
}

// StartPipeline begins pipeline execution.
func (e *ConcreteEventOrchestrator) StartPipeline(ctx context.Context, input *PipelineInput) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if e.started {
		return ErrPipelineAlreadyStarted
	}

	e.logger.Info("starting pipeline orchestration",
		zap.String("pipeline_id", input.Metadata["generation_id"].(string)))

	e.currentInput = input
	e.status.CurrentInput = input
	e.status.PipelineID = input.Metadata["generation_id"].(string)
	e.status.StartTime = time.Now()
	e.started = true

	// Start event processing goroutine
	go e.processEvents(ctx)

	// Start the pipeline with parsing stage
	parseEvent := events.NewEvent("pipeline.parse.start", map[string]interface{}{
		"sources":     input.Sources,
		"source_code": input.SourceCode,
		"config":      input.Config,
		"pipeline_id": input.Metadata["generation_id"].(string),
	})

	if err := e.eventBus.Publish(parseEvent); err != nil {
		return fmt.Errorf("failed to publish parse event: %w", err)
	}

	return nil
}

// HandleEvent processes pipeline events.
func (e *ConcreteEventOrchestrator) HandleEvent(ctx context.Context, event events.Event) error {
	if e.cancelled {
		return ErrPipelineCancelled
	}

	select {
	case e.eventQueue <- event:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	case <-e.shutdown:
		return ErrOrchestratorShutdown
	}
}

// GetStatus returns current pipeline status.
func (e *ConcreteEventOrchestrator) GetStatus() *PipelineStatus {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	// Create a copy to avoid race conditions
	status := *e.status
	status.ElapsedTime = time.Since(e.status.StartTime)

	// Update component status from component manager
	if e.componentMgr != nil {
		components := e.componentMgr.GetComponents()
		for name := range components {
			status.ComponentStatus[name] = e.componentMgr.GetComponentStatus(name)
		}
	}

	return &status
}

// Cancel stops pipeline execution.
func (e *ConcreteEventOrchestrator) Cancel() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if e.cancelled {
		return nil
	}

	e.logger.Info("cancelling pipeline execution")

	e.cancelled = true
	e.status.Stage = StageFailed

	// Signal shutdown
	close(e.shutdown)

	// Publish cancellation event
	cancelEvent := events.NewEvent("pipeline.cancelled", map[string]interface{}{
		"pipeline_id": e.status.PipelineID,
		"stage":       string(e.status.Stage),
	})

	if err := e.eventBus.Publish(cancelEvent); err != nil {
		return fmt.Errorf("failed to publish cancel event: %w", err)
	}

	return nil
}

// SetComponentManager sets the component manager reference.
func (e *ConcreteEventOrchestrator) SetComponentManager(mgr ComponentManager) {
	e.componentMgr = mgr
}

// SetErrorHandler sets the error handler reference.
func (e *ConcreteEventOrchestrator) SetErrorHandler(handler ErrorHandler) {
	e.errorHandler = handler
}

// Private methods

// funcEventHandler wraps a function to implement EventHandler interface.
type funcEventHandler struct {
	handlerFunc func(ctx context.Context, event events.Event) error
	eventType   string
}

func (f *funcEventHandler) Handle(ctx context.Context, event events.Event) error {
	return f.handlerFunc(ctx, event)
}

func (f *funcEventHandler) CanHandle(eventType string) bool {
	return f.eventType == eventType
}

func (e *ConcreteEventOrchestrator) registerEventHandlers() {
	// Parser events
	e.eventHandlers["parser.completed"] = &funcEventHandler{e.handleParseComplete, "parser.completed"}
	e.eventHandlers["parser.failed"] = &funcEventHandler{e.handleParseFailed, "parser.failed"}

	// Planner events
	e.eventHandlers["planner.completed"] = &funcEventHandler{e.handlePlanComplete, "planner.completed"}
	e.eventHandlers["planner.failed"] = &funcEventHandler{e.handlePlanFailed, "planner.failed"}

	// Executor events
	e.eventHandlers["executor.completed"] = &funcEventHandler{e.handleExecuteComplete, "executor.completed"}
	e.eventHandlers["executor.failed"] = &funcEventHandler{e.handleExecuteFailed, "executor.failed"}

	// Emitter events
	e.eventHandlers["emitter.completed"] = &funcEventHandler{e.handleEmitComplete, "emitter.completed"}
	e.eventHandlers["emitter.failed"] = &funcEventHandler{e.handleEmitFailed, "emitter.failed"}

	// Component status events
	e.eventHandlers["component.status.changed"] = &funcEventHandler{e.handleComponentStatusChanged, "component.status.changed"}
}

func (e *ConcreteEventOrchestrator) processEvents(ctx context.Context) {
	e.logger.Debug("starting event processing loop")

	for {
		select {
		case event := <-e.eventQueue:
			if err := e.handleEventInternal(ctx, event); err != nil {
				e.logger.Error("event handling failed",
					zap.String("event_type", event.Type()),
					zap.Error(err))

				if e.errorHandler != nil {
					e.errorHandler.CollectError("orchestrator", err)
				}
			}

		case <-ctx.Done():
			e.logger.Debug("event processing stopped due to context cancellation")
			return

		case <-e.shutdown:
			e.logger.Debug("event processing stopped due to shutdown")
			return
		}
	}
}

func (e *ConcreteEventOrchestrator) handleEventInternal(ctx context.Context, event events.Event) error {
	handler, exists := e.eventHandlers[event.Type()]
	if !exists {
		e.logger.Debug("no handler for event type", zap.String("event_type", event.Type()))
		return nil
	}

	e.logger.Debug("handling event",
		zap.String("event_type", event.Type()),
		zap.Any("event_data", event.Data()))

	if err := handler.Handle(ctx, event); err != nil {
		return fmt.Errorf("event handler failed: %w", err)
	}

	return nil
}

// Event handlers

func (e *ConcreteEventOrchestrator) handleParseComplete(ctx context.Context, event events.Event) error {
	e.updateStage(StagePlanning)

	data := event.Data()

	parseResults, ok := data["results"]
	if !ok {
		return ErrParseResultsNotFound
	}

	e.results[StageParsing] = parseResults

	// Start planning stage
	planEvent := events.NewEvent("pipeline.plan.start", map[string]interface{}{
		"parse_results": parseResults,
		"config":        e.currentInput.Config,
		"pipeline_id":   e.status.PipelineID,
	})

	if err := e.eventBus.Publish(planEvent); err != nil {
		return fmt.Errorf("failed to publish plan event: %w", err)
	}

	return nil
}

func (e *ConcreteEventOrchestrator) handleParseFailed(ctx context.Context, event events.Event) error {
	e.updateStage(StageFailed)

	data := event.Data()
	if errData, ok := data["error"]; ok {
		if e.errorHandler != nil {
			e.errorHandler.CollectError("parser", fmt.Errorf("%w: %v", ErrParsingStageFailed, errData))
		}
	}

	return ErrParsingStageFailed
}

func (e *ConcreteEventOrchestrator) handlePlanComplete(ctx context.Context, event events.Event) error {
	e.updateStage(StageExecuting)

	data := event.Data()

	planResults, ok := data["results"]
	if !ok {
		return ErrPlanResultsNotFound
	}

	e.results[StagePlanning] = planResults

	// Start execution stage
	executeEvent := events.NewEvent("pipeline.execute.start", map[string]interface{}{
		"plan_results": planResults,
		"config":       e.currentInput.Config,
		"pipeline_id":  e.status.PipelineID,
	})

	if err := e.eventBus.Publish(executeEvent); err != nil {
		return fmt.Errorf("failed to publish execute event: %w", err)
	}

	return nil
}

func (e *ConcreteEventOrchestrator) handlePlanFailed(ctx context.Context, event events.Event) error {
	e.updateStage(StageFailed)

	data := event.Data()
	if errData, ok := data["error"]; ok {
		if e.errorHandler != nil {
			e.errorHandler.CollectError("planner", fmt.Errorf("%w: %v", ErrPlanningStageFailed, errData))
		}
	}

	return ErrPlanningStageFailed
}

func (e *ConcreteEventOrchestrator) handleExecuteComplete(ctx context.Context, event events.Event) error {
	e.updateStage(StageEmitting)

	data := event.Data()

	executeResults, ok := data["results"]
	if !ok {
		return ErrExecuteResultsNotFound
	}

	e.results[StageExecuting] = executeResults

	// Start emission stage
	emitEvent := events.NewEvent("pipeline.emit.start", map[string]interface{}{
		"execute_results": executeResults,
		"config":          e.currentInput.Config,
		"pipeline_id":     e.status.PipelineID,
	})

	if err := e.eventBus.Publish(emitEvent); err != nil {
		return fmt.Errorf("failed to publish emit event: %w", err)
	}

	return nil
}

func (e *ConcreteEventOrchestrator) handleExecuteFailed(ctx context.Context, event events.Event) error {
	e.updateStage(StageFailed)

	data := event.Data()
	if errData, ok := data["error"]; ok {
		if e.errorHandler != nil {
			e.errorHandler.CollectError("executor", fmt.Errorf("%w: %v", ErrExecutionStageFailed, errData))
		}
	}

	return ErrExecutionStageFailed
}

func (e *ConcreteEventOrchestrator) handleEmitComplete(ctx context.Context, event events.Event) error {
	e.updateStage(StageCompleted)

	data := event.Data()

	emitResults, ok := data["results"]
	if !ok {
		return ErrEmitResultsNotFound
	}

	e.results[StageEmitting] = emitResults

	// Pipeline completed successfully
	completeEvent := events.NewEvent("pipeline.completed", map[string]interface{}{
		"results":     emitResults,
		"pipeline_id": e.status.PipelineID,
		"duration":    time.Since(e.status.StartTime),
	})

	if err := e.eventBus.Publish(completeEvent); err != nil {
		return fmt.Errorf("failed to publish complete event: %w", err)
	}

	return nil
}

func (e *ConcreteEventOrchestrator) handleEmitFailed(ctx context.Context, event events.Event) error {
	e.updateStage(StageFailed)

	data := event.Data()
	if errData, ok := data["error"]; ok {
		if e.errorHandler != nil {
			e.errorHandler.CollectError("emitter", fmt.Errorf("%w: %v", ErrEmissionStageFailed, errData))
		}
	}

	return ErrEmissionStageFailed
}

func (e *ConcreteEventOrchestrator) handleComponentStatusChanged(ctx context.Context, event events.Event) error {
	data := event.Data()

	component, ok := data["component"].(string)
	if !ok {
		return ErrComponentNameNotFound
	}

	status, ok := data["status"].(ComponentStatus)
	if !ok {
		return ErrStatusNotFound
	}

	e.mutex.Lock()
	e.status.ComponentStatus[component] = status
	e.mutex.Unlock()

	e.logger.Debug("component status updated",
		zap.String("component", component),
		zap.String("status", string(status)))

	return nil
}

func (e *ConcreteEventOrchestrator) updateStage(stage PipelineStage) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	oldStage := e.status.Stage
	e.status.Stage = stage
	e.status.ElapsedTime = time.Since(e.status.StartTime)

	e.logger.Info("pipeline stage updated",
		zap.String("old_stage", string(oldStage)),
		zap.String("new_stage", string(stage)),
		zap.Duration("elapsed", e.status.ElapsedTime))

	// Update progress based on stage
	switch stage {
	case StageInitializing:
		e.status.Progress = 0.0
	case StageParsing:
		e.status.Progress = 0.2
	case StagePlanning:
		e.status.Progress = 0.4
	case StageExecuting:
		e.status.Progress = 0.6
	case StageEmitting:
		e.status.Progress = 0.8
	case StageCompleted:
		e.status.Progress = 1.0
	case StageFailed:
		e.status.Progress = 1.0
	}
}

// GetPipelineResults returns the accumulated results from all stages.
func (e *ConcreteEventOrchestrator) GetPipelineResults() map[PipelineStage]interface{} {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	results := make(map[PipelineStage]interface{})
	for stage, result := range e.results {
		results[stage] = result
	}

	return results
}

// GetFinalResult extracts the final generation result from pipeline results.
func (e *ConcreteEventOrchestrator) GetFinalResult() (*domain.ExecutionResults, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	emitResult, exists := e.results[StageEmitting]
	if !exists {
		return nil, ErrEmissionResultsNotAvailable
	}

	executionResults, ok := emitResult.(*domain.ExecutionResults)
	if !ok {
		return nil, ErrInvalidEmissionResultType
	}

	return executionResults, nil
}
