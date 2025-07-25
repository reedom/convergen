package coordinator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/reedom/convergen/v8/pkg/domain"
	"github.com/reedom/convergen/v8/pkg/internal/events"
	"go.uber.org/zap"
)

// EventOrchestrator manages event-driven coordination between components
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

// ConcreteEventOrchestrator implements EventOrchestrator
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

// NewEventOrchestrator creates a new event orchestrator
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

// StartPipeline begins pipeline execution
func (e *ConcreteEventOrchestrator) StartPipeline(ctx context.Context, input *PipelineInput) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	if e.started {
		return fmt.Errorf("pipeline already started")
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
	
	return e.eventBus.Publish(parseEvent)
}

// HandleEvent processes pipeline events
func (e *ConcreteEventOrchestrator) HandleEvent(ctx context.Context, event events.Event) error {
	if e.cancelled {
		return fmt.Errorf("pipeline cancelled")
	}
	
	select {
	case e.eventQueue <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-e.shutdown:
		return fmt.Errorf("orchestrator shutdown")
	}
}

// GetStatus returns current pipeline status
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

// Cancel stops pipeline execution
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
	
	return e.eventBus.Publish(cancelEvent)
}

// SetComponentManager sets the component manager reference
func (e *ConcreteEventOrchestrator) SetComponentManager(mgr ComponentManager) {
	e.componentMgr = mgr
}

// SetErrorHandler sets the error handler reference
func (e *ConcreteEventOrchestrator) SetErrorHandler(handler ErrorHandler) {
	e.errorHandler = handler
}

// Private methods

func (e *ConcreteEventOrchestrator) registerEventHandlers() {
	// Parser events
	e.eventHandlers["parser.completed"] = e.handleParseComplete
	e.eventHandlers["parser.failed"] = e.handleParseFailed
	
	// Planner events
	e.eventHandlers["planner.completed"] = e.handlePlanComplete
	e.eventHandlers["planner.failed"] = e.handlePlanFailed
	
	// Executor events
	e.eventHandlers["executor.completed"] = e.handleExecuteComplete
	e.eventHandlers["executor.failed"] = e.handleExecuteFailed
	
	// Emitter events
	e.eventHandlers["emitter.completed"] = e.handleEmitComplete
	e.eventHandlers["emitter.failed"] = e.handleEmitFailed
	
	// Component status events
	e.eventHandlers["component.status.changed"] = e.handleComponentStatusChanged
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
	
	return handler(ctx, event)
}

// Event handlers

func (e *ConcreteEventOrchestrator) handleParseComplete(ctx context.Context, event events.Event) error {
	e.updateStage(StagePlanning)
	
	data := event.Data()
	parseResults, ok := data["results"]
	if !ok {
		return fmt.Errorf("parse results not found in event data")
	}
	
	e.results[StageParsing] = parseResults
	
	// Start planning stage
	planEvent := events.NewEvent("pipeline.plan.start", map[string]interface{}{
		"parse_results": parseResults,
		"config":        e.currentInput.Config,
		"pipeline_id":   e.status.PipelineID,
	})
	
	return e.eventBus.Publish(planEvent)
}

func (e *ConcreteEventOrchestrator) handleParseFailed(ctx context.Context, event events.Event) error {
	e.updateStage(StageFailed)
	
	data := event.Data()
	if errData, ok := data["error"]; ok {
		if e.errorHandler != nil {
			e.errorHandler.CollectError("parser", fmt.Errorf("%v", errData))
		}
	}
	
	return fmt.Errorf("parsing stage failed")
}

func (e *ConcreteEventOrchestrator) handlePlanComplete(ctx context.Context, event events.Event) error {
	e.updateStage(StageExecuting)
	
	data := event.Data()
	planResults, ok := data["results"]
	if !ok {
		return fmt.Errorf("plan results not found in event data")
	}
	
	e.results[StagePlanning] = planResults
	
	// Start execution stage
	executeEvent := events.NewEvent("pipeline.execute.start", map[string]interface{}{
		"plan_results": planResults,
		"config":       e.currentInput.Config,
		"pipeline_id":  e.status.PipelineID,
	})
	
	return e.eventBus.Publish(executeEvent)
}

func (e *ConcreteEventOrchestrator) handlePlanFailed(ctx context.Context, event events.Event) error {
	e.updateStage(StageFailed)
	
	data := event.Data()
	if errData, ok := data["error"]; ok {
		if e.errorHandler != nil {
			e.errorHandler.CollectError("planner", fmt.Errorf("%v", errData))
		}
	}
	
	return fmt.Errorf("planning stage failed")
}

func (e *ConcreteEventOrchestrator) handleExecuteComplete(ctx context.Context, event events.Event) error {
	e.updateStage(StageEmitting)
	
	data := event.Data()
	executeResults, ok := data["results"]
	if !ok {
		return fmt.Errorf("execute results not found in event data")
	}
	
	e.results[StageExecuting] = executeResults
	
	// Start emission stage
	emitEvent := events.NewEvent("pipeline.emit.start", map[string]interface{}{
		"execute_results": executeResults,
		"config":          e.currentInput.Config,
		"pipeline_id":     e.status.PipelineID,
	})
	
	return e.eventBus.Publish(emitEvent)
}

func (e *ConcreteEventOrchestrator) handleExecuteFailed(ctx context.Context, event events.Event) error {
	e.updateStage(StageFailed)
	
	data := event.Data()
	if errData, ok := data["error"]; ok {
		if e.errorHandler != nil {
			e.errorHandler.CollectError("executor", fmt.Errorf("%v", errData))
		}
	}
	
	return fmt.Errorf("execution stage failed")
}

func (e *ConcreteEventOrchestrator) handleEmitComplete(ctx context.Context, event events.Event) error {
	e.updateStage(StageCompleted)
	
	data := event.Data()
	emitResults, ok := data["results"]
	if !ok {
		return fmt.Errorf("emit results not found in event data")
	}
	
	e.results[StageEmitting] = emitResults
	
	// Pipeline completed successfully
	completeEvent := events.NewEvent("pipeline.completed", map[string]interface{}{
		"results":     emitResults,
		"pipeline_id": e.status.PipelineID,
		"duration":    time.Since(e.status.StartTime),
	})
	
	return e.eventBus.Publish(completeEvent)
}

func (e *ConcreteEventOrchestrator) handleEmitFailed(ctx context.Context, event events.Event) error {
	e.updateStage(StageFailed)
	
	data := event.Data()
	if errData, ok := data["error"]; ok {
		if e.errorHandler != nil {
			e.errorHandler.CollectError("emitter", fmt.Errorf("%v", errData))
		}
	}
	
	return fmt.Errorf("emission stage failed")
}

func (e *ConcreteEventOrchestrator) handleComponentStatusChanged(ctx context.Context, event events.Event) error {
	data := event.Data()
	component, ok := data["component"].(string)
	if !ok {
		return fmt.Errorf("component name not found in status change event")
	}
	
	status, ok := data["status"].(ComponentStatus)
	if !ok {
		return fmt.Errorf("status not found in status change event")
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

// GetPipelineResults returns the accumulated results from all stages
func (e *ConcreteEventOrchestrator) GetPipelineResults() map[PipelineStage]interface{} {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	results := make(map[PipelineStage]interface{})
	for stage, result := range e.results {
		results[stage] = result
	}
	
	return results
}

// GetFinalResult extracts the final generation result from pipeline results
func (e *ConcreteEventOrchestrator) GetFinalResult() (*domain.ExecutionResults, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	emitResult, exists := e.results[StageEmitting]
	if !exists {
		return nil, fmt.Errorf("emission results not available")
	}
	
	executionResults, ok := emitResult.(*domain.ExecutionResults)
	if !ok {
		return nil, fmt.Errorf("invalid emission result type")
	}
	
	return executionResults, nil
}