package events

import (
	"context"
	"sync"
	"time"

	"github.com/reedom/convergen/v8/pkg/domain"
)

// Event type constants for the generation pipeline.
const (
	EventTypeParseStarted   = "parse.started"
	EventTypeParsed         = "parse.completed"
	EventTypePlanStarted    = "plan.started"
	EventTypePlanned        = "plan.completed"
	EventTypeExecuteStarted = "execute.started"
	EventTypeExecuted       = "execute.completed"
	EventTypeEmitStarted    = "emit.started"
	EventTypeEmitted        = "emit.completed"
	EventTypeProgress       = "progress.update"
	EventTypeError          = "error.occurred"
)

// ParseStartedEvent signals the beginning of parsing.
type ParseStartedEvent struct {
	*BaseEvent
	SourcePath string `json:"source_path"`
}

// NewParseStartedEvent creates a new parse started event.
func NewParseStartedEvent(ctx context.Context, sourcePath string) *ParseStartedEvent {
	return &ParseStartedEvent{
		BaseEvent:  NewBaseEvent(ctx, EventTypeParseStarted),
		SourcePath: sourcePath,
	}
}

// ParsedEvent represents successful parsing completion.
type ParsedEvent struct {
	*BaseEvent
	Methods  []*domain.Method `json:"methods"`
	BaseCode string           `json:"base_code"`
	Metrics  *ParseMetrics    `json:"metrics"`
}

// NewParsedEvent creates a new parsed event.
func NewParsedEvent(ctx context.Context, methods []*domain.Method, baseCode string) *ParsedEvent {
	return &ParsedEvent{
		BaseEvent: NewBaseEvent(ctx, EventTypeParsed),
		Methods:   methods,
		BaseCode:  baseCode,
		Metrics:   &ParseMetrics{},
	}
}

// ParseMetrics tracks parsing performance.
type ParseMetrics struct {
	ParseDurationMS      int64   `json:"parse_duration_ms"`
	InterfacesFound      int     `json:"interfaces_found"`
	MethodsProcessed     int     `json:"methods_processed"`
	AnnotationsProcessed int     `json:"annotations_processed"`
	TypesResolved        int     `json:"types_resolved"`
	CacheHitRate         float64 `json:"cache_hit_rate"`
}

// PlanStartedEvent signals the beginning of planning.
type PlanStartedEvent struct {
	*BaseEvent
	Methods []*domain.Method `json:"methods"`
}

// NewPlanStartedEvent creates a new plan started event.
func NewPlanStartedEvent(ctx context.Context, methods []*domain.Method) *PlanStartedEvent {
	return &PlanStartedEvent{
		BaseEvent: NewBaseEvent(ctx, EventTypePlanStarted),
		Methods:   methods,
	}
}

// PlannedEvent represents successful planning completion.
type PlannedEvent struct {
	*BaseEvent
	Plan    *domain.ExecutionPlan `json:"plan"`
	Metrics *PlanMetrics          `json:"metrics"`
}

// NewPlannedEvent creates a new planned event.
func NewPlannedEvent(ctx context.Context, plan *domain.ExecutionPlan) *PlannedEvent {
	return &PlannedEvent{
		BaseEvent: NewBaseEvent(ctx, EventTypePlanned),
		Plan:      plan,
		Metrics:   &PlanMetrics{}, // Create new metrics for event
	}
}

// PlanMetrics tracks planning performance.
type PlanMetrics struct {
	PlanningDurationMS    int64   `json:"planning_duration_ms"`
	MethodsPlanned        int     `json:"methods_planned"`
	TotalFields           int     `json:"total_fields"`
	ConcurrentBatches     int     `json:"concurrent_batches"`
	ParallelizationRatio  float64 `json:"parallelization_ratio"`
	EstimatedSpeedupRatio float64 `json:"estimated_speedup_ratio"`
}

// ExecuteStartedEvent signals the beginning of execution.
type ExecuteStartedEvent struct {
	*BaseEvent
	Plan *domain.ExecutionPlan `json:"plan"`
}

// NewExecuteStartedEvent creates a new execute started event.
func NewExecuteStartedEvent(ctx context.Context, plan *domain.ExecutionPlan) *ExecuteStartedEvent {
	return &ExecuteStartedEvent{
		BaseEvent: NewBaseEvent(ctx, EventTypeExecuteStarted),
		Plan:      plan,
	}
}

// ExecutedEvent represents successful execution completion.
type ExecutedEvent struct {
	*BaseEvent
	Results []*domain.FieldResult     `json:"results"`
	Errors  []*domain.GenerationError `json:"errors"`
	Metrics *ExecutionMetrics         `json:"metrics"`
}

// NewExecutedEvent creates a new executed event.
func NewExecutedEvent(ctx context.Context, results []*domain.FieldResult, errors []*domain.GenerationError) *ExecutedEvent {
	return &ExecutedEvent{
		BaseEvent: NewBaseEvent(ctx, EventTypeExecuted),
		Results:   results,
		Errors:    errors,
		Metrics:   &ExecutionMetrics{},
	}
}

// ExecutionMetrics tracks execution performance.
type ExecutionMetrics struct {
	ExecutionDurationMS int64   `json:"execution_duration_ms"`
	FieldsProcessed     int     `json:"fields_processed"`
	BatchesExecuted     int     `json:"batches_executed"`
	ConcurrentWorkers   int     `json:"concurrent_workers"`
	MemoryUsageMB       int     `json:"memory_usage_mb"`
	ErrorRate           float64 `json:"error_rate"`
}

// EmitStartedEvent signals the beginning of code emission.
type EmitStartedEvent struct {
	*BaseEvent
	Results []*domain.FieldResult `json:"results"`
}

// NewEmitStartedEvent creates a new emit started event.
func NewEmitStartedEvent(ctx context.Context, results []*domain.FieldResult) *EmitStartedEvent {
	return &EmitStartedEvent{
		BaseEvent: NewBaseEvent(ctx, EventTypeEmitStarted),
		Results:   results,
	}
}

// EmittedEvent represents successful code emission completion.
type EmittedEvent struct {
	*BaseEvent
	Generated *domain.GeneratedFunction `json:"generated"`
	Metrics   *EmissionMetrics          `json:"metrics"`
}

// NewEmittedEvent creates a new emitted event.
func NewEmittedEvent(ctx context.Context, generated *domain.GeneratedFunction) *EmittedEvent {
	return &EmittedEvent{
		BaseEvent: NewBaseEvent(ctx, EventTypeEmitted),
		Generated: generated,
		Metrics:   &EmissionMetrics{},
	}
}

// EmissionMetrics tracks code emission performance.
type EmissionMetrics struct {
	EmissionDurationMS   int64 `json:"emission_duration_ms"`
	LinesGenerated       int   `json:"lines_generated"`
	ImportsGenerated     int   `json:"imports_generated"`
	OptimizationsApplied int   `json:"optimizations_applied"`
	CodeSizeBytes        int   `json:"code_size_bytes"`
}

// ProgressEvent represents progress updates during processing.
type ProgressEvent struct {
	*BaseEvent
	Phase       domain.ProcessingPhase `json:"phase"`
	Progress    float64                `json:"progress"` // 0.0 to 1.0
	Message     string                 `json:"message"`
	Current     int                    `json:"current"`
	Total       int                    `json:"total"`
	ElapsedMS   int64                  `json:"elapsed_ms"`
	EstimatedMS int64                  `json:"estimated_ms"`
}

// NewProgressEvent creates a new progress event.
func NewProgressEvent(ctx context.Context, phase domain.ProcessingPhase, current, total int, message string) *ProgressEvent {
	progress := float64(current) / float64(total)
	if total == 0 {
		progress = 0.0
	}

	return &ProgressEvent{
		BaseEvent:   NewBaseEvent(ctx, EventTypeProgress),
		Phase:       phase,
		Progress:    progress,
		Message:     message,
		Current:     current,
		Total:       total,
		ElapsedMS:   0,
		EstimatedMS: 0,
	}
}

// WithTiming adds timing information to the progress event.
func (e *ProgressEvent) WithTiming(elapsedMS, estimatedMS int64) *ProgressEvent {
	e.ElapsedMS = elapsedMS
	e.EstimatedMS = estimatedMS

	return e
}

// ErrorEvent represents error occurrences during processing.
type ErrorEvent struct {
	*BaseEvent
	Error       *domain.GenerationError `json:"error"`
	Phase       domain.ProcessingPhase  `json:"phase"`
	Recoverable bool                    `json:"recoverable"`
	Context     map[string]interface{}  `json:"context"`
}

// NewErrorEvent creates a new error event.
func NewErrorEvent(ctx context.Context, err *domain.GenerationError, recoverable bool) *ErrorEvent {
	return &ErrorEvent{
		BaseEvent:   NewBaseEvent(ctx, EventTypeError),
		Error:       err,
		Phase:       err.Phase,
		Recoverable: recoverable,
		Context:     make(map[string]interface{}),
	}
}

// WithContext adds context information to the error event.
func (e *ErrorEvent) WithContext(key string, value interface{}) *ErrorEvent {
	e.Context[key] = value
	return e
}

// EventCollector collects events for analysis and debugging.
type EventCollector struct {
	events    []Event
	mutex     sync.RWMutex
	maxEvents int
}

// NewEventCollector creates a new event collector.
func NewEventCollector(maxEvents int) *EventCollector {
	if maxEvents <= 0 {
		maxEvents = 1000 // Default limit
	}

	return &EventCollector{
		events:    make([]Event, 0),
		maxEvents: maxEvents,
	}
}

// Collect adds an event to the collection.
func (ec *EventCollector) Collect(event Event) {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()

	// Add event, removing oldest if at capacity
	if len(ec.events) >= ec.maxEvents {
		ec.events = ec.events[1:]
	}

	ec.events = append(ec.events, event)
}

// Events returns all collected events.
func (ec *EventCollector) Events() []Event {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()

	// Return defensive copy
	events := make([]Event, len(ec.events))
	copy(events, ec.events)

	return events
}

// EventsByType returns events filtered by type.
func (ec *EventCollector) EventsByType(eventType string) []Event {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()

	var filtered []Event

	for _, event := range ec.events {
		if event.Type() == eventType {
			filtered = append(filtered, event)
		}
	}

	return filtered
}

// EventsInTimeRange returns events within a time range.
func (ec *EventCollector) EventsInTimeRange(start, end time.Time) []Event {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()

	var filtered []Event

	for _, event := range ec.events {
		timestamp := event.Timestamp()
		if (timestamp.Equal(start) || timestamp.After(start)) &&
			(timestamp.Equal(end) || timestamp.Before(end)) {
			filtered = append(filtered, event)
		}
	}

	return filtered
}

// Clear removes all collected events.
func (ec *EventCollector) Clear() {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()

	ec.events = ec.events[:0]
}

// Count returns the number of collected events.
func (ec *EventCollector) Count() int {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()

	return len(ec.events)
}

// PipelineEventHandler provides a base for pipeline event handlers.
type PipelineEventHandler struct {
	eventTypes []string
	handler    func(ctx context.Context, event Event) error
}

// NewPipelineEventHandler creates a new pipeline event handler.
func NewPipelineEventHandler(eventTypes []string, handler func(ctx context.Context, event Event) error) *PipelineEventHandler {
	return &PipelineEventHandler{
		eventTypes: eventTypes,
		handler:    handler,
	}
}

// Handle processes an event.
func (h *PipelineEventHandler) Handle(ctx context.Context, event Event) error {
	return h.handler(ctx, event)
}

// CanHandle returns true if the handler can process the event type.
func (h *PipelineEventHandler) CanHandle(eventType string) bool {
	for _, supported := range h.eventTypes {
		if supported == eventType {
			return true
		}
	}

	return false
}

// ProgressTracker tracks progress across the entire pipeline.
type ProgressTracker struct {
	totalPhases   int
	currentPhase  int
	phaseProgress map[domain.ProcessingPhase]float64
	mutex         sync.RWMutex
	startTime     time.Time
}

// NewProgressTracker creates a new progress tracker.
func NewProgressTracker() *ProgressTracker {
	return &ProgressTracker{
		totalPhases:   4, // parsing, planning, execution, emission
		currentPhase:  0,
		phaseProgress: make(map[domain.ProcessingPhase]float64),
		startTime:     time.Now(),
	}
}

// UpdatePhaseProgress updates progress for a specific phase.
func (pt *ProgressTracker) UpdatePhaseProgress(phase domain.ProcessingPhase, progress float64) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	pt.phaseProgress[phase] = progress
}

// OverallProgress calculates overall pipeline progress.
func (pt *ProgressTracker) OverallProgress() float64 {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	totalProgress := 0.0

	for phase := domain.PhaseParsing; phase <= domain.PhaseEmission; phase++ {
		if progress, exists := pt.phaseProgress[phase]; exists {
			totalProgress += progress
		}
	}

	return totalProgress / float64(pt.totalPhases)
}

// ElapsedTime returns time elapsed since tracking started.
func (pt *ProgressTracker) ElapsedTime() time.Duration {
	return time.Since(pt.startTime)
}

// EstimatedTimeRemaining estimates remaining time based on current progress.
func (pt *ProgressTracker) EstimatedTimeRemaining() time.Duration {
	progress := pt.OverallProgress()
	if progress <= 0 {
		return 0
	}

	elapsed := pt.ElapsedTime()
	totalEstimated := time.Duration(float64(elapsed) / progress)

	return totalEstimated - elapsed
}
