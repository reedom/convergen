package emitter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v9/pkg/domain"
	"github.com/reedom/convergen/v9/pkg/internal/events"
)

// Static errors for err113 compliance.
var (
	ErrInvalidExecutionResultsMetadata = errors.New("missing or invalid execution results in event metadata")
)

// EmitterEvent types for the emitter pipeline.
const (
	EventEmitterStarted        = "emitter.started"
	EventEmitterCompleted      = "emitter.completed"
	EventEmitterFailed         = "emitter.failed"
	EventCodeGenerationStarted = "emitter.code_generation.started"
	EventCodeGenerated         = "emitter.code.generated"
	EventMethodGenerated       = "emitter.method.generated"
	EventImportsGenerated      = "emitter.imports.generated"
	EventCodeFormatted         = "emitter.code.formatted"
	EventCodeOptimized         = "emitter.code.optimized"
	EventStrategySelected      = "emitter.strategy.selected"
)

// EventHandler handles emitter-specific events.
type EventHandler struct {
	emitter    Emitter
	logger     *zap.Logger
	eventBus   events.EventBus
	middleware []events.EventMiddleware
}

// NewEventHandler creates a new emitter event handler.
func NewEventHandler(emitter Emitter, eventBus events.EventBus, logger *zap.Logger) *EventHandler {
	return &EventHandler{
		emitter:    emitter,
		logger:     logger,
		eventBus:   eventBus,
		middleware: make([]events.EventMiddleware, 0),
	}
}

// RegisterEventHandlers registers all emitter event handlers.
func (h *EventHandler) RegisterEventHandlers() error {
	// Subscribe to execution pipeline events
	if err := h.eventBus.Subscribe("executor.completed", h); err != nil {
		return fmt.Errorf("failed to subscribe to executor.completed: %w", err)
	}

	// Subscribe to planner events for optimization
	if err := h.eventBus.Subscribe("planner.method_planned", h); err != nil {
		return fmt.Errorf("failed to subscribe to planner.method_planned: %w", err)
	}

	h.logger.Info("emitter event handlers registered successfully")

	return nil
}

// Handle processes events for the emitter.
func (h *EventHandler) Handle(ctx context.Context, event events.Event) error {
	h.logger.Debug("handling emitter event",
		zap.String("event_type", event.Type()),
		zap.String("event_id", event.ID()))

	switch event.Type() {
	case "executor.completed":
		return h.handleExecutorCompleted(ctx, event)
	case "planner.method_planned":
		return h.handleMethodPlanned(ctx, event)
	default:
		h.logger.Debug("unhandled event type",
			zap.String("event_type", event.Type()))
		return nil
	}
}

// CanHandle returns true if this handler can process the event type.
func (h *EventHandler) CanHandle(eventType string) bool {
	switch eventType {
	case "executor.completed", "planner.method_planned":
		return true
	default:
		return false
	}
}

// Event publishing methods

// PublishEmitterStarted publishes an emitter started event.
func (h *EventHandler) PublishEmitterStarted(ctx context.Context, packageName string, methodCount int) error {
	event := NewStartedEvent(ctx, packageName, methodCount)
	if err := h.eventBus.Publish(event); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// PublishEmitterCompleted publishes an emitter completed event.
func (h *EventHandler) PublishEmitterCompleted(ctx context.Context, code *GeneratedCode, metrics *Metrics) error {
	event := NewCompletedEvent(ctx, code, metrics)
	if err := h.eventBus.Publish(event); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// PublishEmitterFailed publishes an emitter failed event.
func (h *EventHandler) PublishEmitterFailed(ctx context.Context, err error, partialCode *GeneratedCode) error {
	event := NewFailedEvent(ctx, err, partialCode)
	if err := h.eventBus.Publish(event); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// PublishCodeGenerationStarted publishes a code generation started event.
func (h *EventHandler) PublishCodeGenerationStarted(ctx context.Context, methodName string, strategy ConstructionStrategy) error {
	event := NewCodeGenerationStartedEvent(ctx, methodName, strategy)
	if err := h.eventBus.Publish(event); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// PublishMethodGenerated publishes a method generated event.
func (h *EventHandler) PublishMethodGenerated(ctx context.Context, method *MethodCode, duration time.Duration) error {
	event := NewMethodGeneratedEvent(ctx, method, duration)
	if err := h.eventBus.Publish(event); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// PublishStrategySelected publishes a strategy selected event.
func (h *EventHandler) PublishStrategySelected(ctx context.Context, methodName string, strategy ConstructionStrategy, reason string) error {
	event := NewStrategySelectedEvent(ctx, methodName, strategy, reason)
	if err := h.eventBus.Publish(event); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// Event handlers

func (h *EventHandler) handleExecutorCompleted(ctx context.Context, event events.Event) error {
	h.logger.Debug("handling executor completed event")

	// Extract execution results from event metadata
	metadata := event.Metadata()

	results, ok := metadata["execution_results"].(*domain.ExecutionResults)
	if !ok {
		return ErrInvalidExecutionResultsMetadata
	}

	// Trigger code generation
	go func() {
		codeCtx := ctx
		if err := h.PublishEmitterStarted(codeCtx, results.PackageName, len(results.Methods)); err != nil {
			h.logger.Error("failed to publish emitter started event", zap.Error(err))
		}

		code, err := h.emitter.GenerateCode(codeCtx, results)
		if err != nil {
			h.logger.Error("code generation failed", zap.Error(err))

			if publishErr := h.PublishEmitterFailed(codeCtx, err, nil); publishErr != nil {
				h.logger.Error("failed to publish emitter failed event", zap.Error(publishErr))
			}

			return
		}

		metrics := h.emitter.GetMetrics()
		if err := h.PublishEmitterCompleted(codeCtx, code, metrics); err != nil {
			h.logger.Error("failed to publish emitter completed event", zap.Error(err))
		}
	}()

	return nil
}

func (h *EventHandler) handleMethodPlanned(_ context.Context, event events.Event) error {
	h.logger.Debug("handling method planned event")

	// This could be used for pre-planning optimization or strategy hints
	metadata := event.Metadata()
	if methodName, ok := metadata["method_name"].(string); ok {
		h.logger.Debug("method planned for generation",
			zap.String("method", methodName))
	}

	return nil
}

// Event types

// StartedEvent represents the start of code generation.
type StartedEvent struct {
	events.BaseEvent
	PackageName string `json:"package_name"`
	MethodCount int    `json:"method_count"`
}

// Type returns the event type.
func (e *StartedEvent) Type() string {
	return e.BaseEvent.Type()
}

// NewStartedEvent creates a new emitter started event.
func NewStartedEvent(ctx context.Context, packageName string, methodCount int) *StartedEvent {
	base := events.NewBaseEvent(ctx, EventEmitterStarted)

	return &StartedEvent{
		BaseEvent:   *base,
		PackageName: packageName,
		MethodCount: methodCount,
	}
}

// CompletedEvent represents successful code generation completion.
type CompletedEvent struct {
	events.BaseEvent
	Code    *GeneratedCode `json:"code"`
	Metrics *Metrics       `json:"metrics"`
}

// Type returns the event type.
func (e *CompletedEvent) Type() string {
	return e.BaseEvent.Type()
}

// NewCompletedEvent creates a new emitter completed event.
func NewCompletedEvent(ctx context.Context, code *GeneratedCode, metrics *Metrics) *CompletedEvent {
	base := events.NewBaseEvent(ctx, EventEmitterCompleted)
	base.WithMetadata("package_name", code.PackageName)
	base.WithMetadata("methods_generated", len(code.Methods))
	base.WithMetadata("lines_generated", metrics.TotalLines)

	return &CompletedEvent{
		BaseEvent: *base,
		Code:      code,
		Metrics:   metrics,
	}
}

// FailedEvent represents code generation failure.
type FailedEvent struct {
	events.BaseEvent
	Error       error          `json:"error"`
	PartialCode *GeneratedCode `json:"partial_code,omitempty"`
}

// Type returns the event type.
func (e *FailedEvent) Type() string {
	return e.BaseEvent.Type()
}

// NewFailedEvent creates a new emitter failed event.
func NewFailedEvent(ctx context.Context, err error, partialCode *GeneratedCode) *FailedEvent {
	base := events.NewBaseEvent(ctx, EventEmitterFailed)
	base.WithMetadata("error_message", err.Error())

	if partialCode != nil {
		base.WithMetadata("partial_methods", len(partialCode.Methods))
	}

	return &FailedEvent{
		BaseEvent:   *base,
		Error:       err,
		PartialCode: partialCode,
	}
}

// CodeGenerationStartedEvent represents the start of method code generation.
type CodeGenerationStartedEvent struct {
	events.BaseEvent
	MethodName string               `json:"method_name"`
	Strategy   ConstructionStrategy `json:"strategy"`
}

// Type returns the event type.
func (e *CodeGenerationStartedEvent) Type() string {
	return e.BaseEvent.Type()
}

// NewCodeGenerationStartedEvent creates a new code generation started event.
func NewCodeGenerationStartedEvent(ctx context.Context, methodName string, strategy ConstructionStrategy) *CodeGenerationStartedEvent {
	base := events.NewBaseEvent(ctx, EventCodeGenerationStarted)
	base.WithMetadata("method_name", methodName)
	base.WithMetadata("strategy", strategy.String())

	return &CodeGenerationStartedEvent{
		BaseEvent:  *base,
		MethodName: methodName,
		Strategy:   strategy,
	}
}

// MethodGeneratedEvent represents successful method generation.
type MethodGeneratedEvent struct {
	events.BaseEvent
	Method   *MethodCode   `json:"method"`
	Duration time.Duration `json:"duration"`
}

// Type returns the event type.
func (e *MethodGeneratedEvent) Type() string {
	return e.BaseEvent.Type()
}

// NewMethodGeneratedEvent creates a new method generated event.
func NewMethodGeneratedEvent(ctx context.Context, method *MethodCode, duration time.Duration) *MethodGeneratedEvent {
	base := events.NewBaseEvent(ctx, EventMethodGenerated)
	base.WithMetadata("method_name", method.Name)
	base.WithMetadata("lines_generated", len(method.Body))
	base.WithMetadata("generation_duration_ms", duration.Milliseconds())

	return &MethodGeneratedEvent{
		BaseEvent: *base,
		Method:    method,
		Duration:  duration,
	}
}

// StrategySelectedEvent represents strategy selection for a method.
type StrategySelectedEvent struct {
	events.BaseEvent
	MethodName string               `json:"method_name"`
	Strategy   ConstructionStrategy `json:"strategy"`
	Reason     string               `json:"reason"`
}

// Type returns the event type.
func (e *StrategySelectedEvent) Type() string {
	return e.BaseEvent.Type()
}

// NewStrategySelectedEvent creates a new strategy selected event.
func NewStrategySelectedEvent(ctx context.Context, methodName string, strategy ConstructionStrategy, reason string) *StrategySelectedEvent {
	base := events.NewBaseEvent(ctx, EventStrategySelected)
	base.WithMetadata("method_name", methodName)
	base.WithMetadata("strategy", strategy.String())
	base.WithMetadata("selection_reason", reason)

	return &StrategySelectedEvent{
		BaseEvent:  *base,
		MethodName: methodName,
		Strategy:   strategy,
		Reason:     reason,
	}
}

// EventAwareEmitter wraps an emitter with event publishing capabilities.
type EventAwareEmitter struct {
	inner        Emitter
	eventHandler *EventHandler
	logger       *zap.Logger
}

// NewEventAwareEmitter creates a new event-aware emitter.
func NewEventAwareEmitter(inner Emitter, eventBus events.EventBus, logger *zap.Logger) *EventAwareEmitter {
	eventHandler := NewEventHandler(inner, eventBus, logger)

	// Register event handlers
	if err := eventHandler.RegisterEventHandlers(); err != nil {
		logger.Error("failed to register event handlers", zap.Error(err))
	}

	return &EventAwareEmitter{
		inner:        inner,
		eventHandler: eventHandler,
		logger:       logger,
	}
}

// GenerateCode generates code with event publishing.
func (e *EventAwareEmitter) GenerateCode(ctx context.Context, results *domain.ExecutionResults) (*GeneratedCode, error) {
	// Publish start event
	if err := e.eventHandler.PublishEmitterStarted(ctx, results.PackageName, len(results.Methods)); err != nil {
		e.logger.Warn("failed to publish emitter started event", zap.Error(err))
	}

	start := time.Now()

	code, err := e.inner.GenerateCode(ctx, results)
	if err != nil {
		err = fmt.Errorf("code generation failed: %w", err)
	}

	duration := time.Since(start)

	if err != nil {
		// Publish failure event
		if publishErr := e.eventHandler.PublishEmitterFailed(ctx, err, code); publishErr != nil {
			e.logger.Warn("failed to publish emitter failed event", zap.Error(publishErr))
		}

		return code, err
	}

	// Publish completion event
	metrics := e.inner.GetMetrics()
	if metrics != nil {
		metrics.TotalGenerationTime = duration
	}

	if err := e.eventHandler.PublishEmitterCompleted(ctx, code, metrics); err != nil {
		e.logger.Warn("failed to publish emitter completed event", zap.Error(err))
	}

	return code, nil
}

// GenerateMethod generates a single method with event publishing.
func (e *EventAwareEmitter) GenerateMethod(ctx context.Context, method *domain.MethodResult) (*MethodCode, error) {
	start := time.Now()

	methodCode, err := e.inner.GenerateMethod(ctx, method)
	duration := time.Since(start)

	if err == nil && methodCode != nil {
		// Publish method generated event
		if publishErr := e.eventHandler.PublishMethodGenerated(ctx, methodCode, duration); publishErr != nil {
			e.logger.Warn("failed to publish method generated event", zap.Error(publishErr))
		}
	}

	if err != nil {
		return methodCode, fmt.Errorf("method generation failed: %w", err)
	}

	return methodCode, nil
}

// OptimizeOutput delegates the output optimization to the inner emitter.
func (e *EventAwareEmitter) OptimizeOutput(ctx context.Context, code *GeneratedCode) (*GeneratedCode, error) {
	optimizedCode, err := e.inner.OptimizeOutput(ctx, code)
	if err != nil {
		return optimizedCode, fmt.Errorf("output optimization failed: %w", err)
	}

	return optimizedCode, nil
}

// GetMetrics returns the metrics from the inner emitter.
func (e *EventAwareEmitter) GetMetrics() *Metrics {
	return e.inner.GetMetrics()
}

// Shutdown gracefully shuts down the inner emitter.
func (e *EventAwareEmitter) Shutdown(ctx context.Context) error {
	if err := e.inner.Shutdown(ctx); err != nil {
		return fmt.Errorf("emitter shutdown failed: %w", err)
	}

	return nil
}
