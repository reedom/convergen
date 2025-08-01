package events

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	gonanoid "github.com/matoous/go-nanoid"
	"go.uber.org/zap"
)

// Static errors for err113 compliance.
var (
	ErrEventBusClosed           = errors.New("event bus is closed")
	ErrHandlerCannotHandleEvent = errors.New("handler cannot handle event type")
	ErrNoHandlersForEventType   = errors.New("no handlers for event type")
	ErrHandlerNotFound          = errors.New("handler not found for event type")
	ErrEventProcessingTimeout   = errors.New("event processing timed out")
	ErrHandlerPanic             = errors.New("handler panic")
)

// Event represents a pipeline event with context.
type Event interface {
	ID() string
	Type() string
	Data() map[string]interface{}
	Timestamp() time.Time
	Context() context.Context
	Metadata() map[string]interface{}
}

// EventHandler processes events of a specific type.
type EventHandler interface {
	Handle(ctx context.Context, event Event) error
	CanHandle(eventType string) bool
}

// EventBus manages event publishing and subscription.
type EventBus interface {
	Publish(event Event) error
	Subscribe(eventType string, handler EventHandler) error
	Unsubscribe(eventType string, handler EventHandler) error
	Close() error
	Stats() *BusStats
	Emit(ctx context.Context, event Event) error
}

// BaseEvent provides common event functionality.
type BaseEvent struct {
	id        string
	eventType string
	data      map[string]interface{}
	timestamp time.Time
	ctx       context.Context
	metadata  map[string]interface{}
}

// NewBaseEvent creates a new base event.
func NewBaseEvent(eventType string, ctx context.Context) *BaseEvent {
	id, _ := gonanoid.Nanoid()

	return &BaseEvent{
		id:        id,
		eventType: eventType,
		data:      make(map[string]interface{}),
		timestamp: time.Now(),
		ctx:       ctx,
		metadata:  make(map[string]interface{}),
	}
}

// NewEvent creates a new event with data.
func NewEvent(eventType string, data map[string]interface{}) Event {
	id, _ := gonanoid.Nanoid()

	return &BaseEvent{
		id:        id,
		eventType: eventType,
		data:      data,
		timestamp: time.Now(),
		ctx:       context.Background(),
		metadata:  make(map[string]interface{}),
	}
}

func (e *BaseEvent) ID() string                       { return e.id }
func (e *BaseEvent) Type() string                     { return e.eventType }
func (e *BaseEvent) Data() map[string]interface{}     { return e.data }
func (e *BaseEvent) Timestamp() time.Time             { return e.timestamp }
func (e *BaseEvent) Context() context.Context         { return e.ctx }
func (e *BaseEvent) Metadata() map[string]interface{} { return e.metadata }

// WithMetadata adds metadata to the event.
func (e *BaseEvent) WithMetadata(key string, value interface{}) *BaseEvent {
	e.metadata[key] = value
	return e
}

// InMemoryEventBus implements EventBus with in-memory storage.
type InMemoryEventBus struct {
	handlers map[string][]EventHandler
	mutex    sync.RWMutex
	logger   *zap.Logger
	stats    *BusStats
	closed   bool
}

// NewInMemoryEventBus creates a new in-memory event bus.
func NewInMemoryEventBus(logger *zap.Logger) *InMemoryEventBus {
	return &InMemoryEventBus{
		handlers: make(map[string][]EventHandler),
		logger:   logger,
		stats:    &BusStats{},
		closed:   false,
	}
}

// Publish publishes an event to all registered handlers.
func (bus *InMemoryEventBus) Publish(event Event) error {
	ctx := event.Context()

	bus.mutex.RLock()
	defer bus.mutex.RUnlock()

	if bus.closed {
		return ErrEventBusClosed
	}

	eventType := event.Type()
	handlers, exists := bus.handlers[eventType]

	bus.stats.incrementPublished(eventType)

	if !exists || len(handlers) == 0 {
		bus.logger.Debug("no handlers for event type",
			zap.String("event_type", eventType),
			zap.String("event_id", event.ID()))

		return nil
	}

	bus.logger.Debug("publishing event",
		zap.String("event_type", eventType),
		zap.String("event_id", event.ID()),
		zap.Int("handler_count", len(handlers)))

	// Process handlers concurrently
	errChan := make(chan error, len(handlers))

	for _, handler := range handlers {
		go func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					errChan <- fmt.Errorf("%w: %v", ErrHandlerPanic, r)
				}
			}()

			if err := h.Handle(ctx, event); err != nil {
				bus.stats.incrementHandlerError(eventType)
				errChan <- fmt.Errorf("handler error: %w", err)
			} else {
				bus.stats.incrementHandlerSuccess(eventType)
				errChan <- nil
			}
		}(handler)
	}

	// Collect results
	var errors []error

	for i := 0; i < len(handlers); i++ {
		if err := <-errChan; err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		bus.logger.Error("handler errors during event processing",
			zap.String("event_type", eventType),
			zap.String("event_id", event.ID()),
			zap.Int("error_count", len(errors)))

		// Return first error (could be enhanced to return all)
		return errors[0]
	}

	return nil
}

// Subscribe registers a handler for a specific event type.
func (bus *InMemoryEventBus) Subscribe(eventType string, handler EventHandler) error {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	if bus.closed {
		return ErrEventBusClosed
	}

	if !handler.CanHandle(eventType) {
		return fmt.Errorf("%w: %s", ErrHandlerCannotHandleEvent, eventType)
	}

	bus.handlers[eventType] = append(bus.handlers[eventType], handler)
	bus.stats.incrementSubscriptions(eventType)

	bus.logger.Debug("handler subscribed",
		zap.String("event_type", eventType),
		zap.Int("total_handlers", len(bus.handlers[eventType])))

	return nil
}

// Unsubscribe removes a handler for a specific event type.
func (bus *InMemoryEventBus) Unsubscribe(eventType string, handler EventHandler) error {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	handlers, exists := bus.handlers[eventType]
	if !exists {
		return fmt.Errorf("%w: %s", ErrNoHandlersForEventType, eventType)
	}

	// Find and remove the handler
	for i, h := range handlers {
		if h == handler {
			bus.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			bus.stats.decrementSubscriptions(eventType)

			bus.logger.Debug("handler unsubscribed",
				zap.String("event_type", eventType),
				zap.Int("remaining_handlers", len(bus.handlers[eventType])))

			return nil
		}
	}

	return fmt.Errorf("%w: %s", ErrHandlerNotFound, eventType)
}

// Close closes the event bus.
func (bus *InMemoryEventBus) Close() error {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	bus.closed = true
	bus.handlers = make(map[string][]EventHandler)

	bus.logger.Info("event bus closed")

	return nil
}

// Stats returns bus statistics.
func (bus *InMemoryEventBus) Stats() *BusStats {
	bus.mutex.RLock()
	defer bus.mutex.RUnlock()

	// Return a copy to avoid race conditions (exclude mutex to fix copylocks issue)
	statsCopy := &BusStats{
		publishedEvents:  copyStringInt64Map(bus.stats.publishedEvents),
		subscriptions:    copyStringInt64Map(bus.stats.subscriptions),
		handlerSuccesses: copyStringInt64Map(bus.stats.handlerSuccesses),
		handlerErrors:    copyStringInt64Map(bus.stats.handlerErrors),
		// mutex field intentionally omitted to avoid copying lock
	}

	return statsCopy
}

// copyStringInt64Map creates a deep copy of a map[string]int64.
func copyStringInt64Map(original map[string]int64) map[string]int64 {
	if original == nil {
		return nil
	}

	copy := make(map[string]int64, len(original))
	for k, v := range original {
		copy[k] = v
	}

	return copy
}

// Emit emits an event with context (alias for Publish for compatibility).
func (bus *InMemoryEventBus) Emit(ctx context.Context, event Event) error {
	return bus.Publish(event)
}

// BusStats tracks event bus statistics.
type BusStats struct {
	publishedEvents  map[string]int64
	subscriptions    map[string]int64
	handlerSuccesses map[string]int64
	handlerErrors    map[string]int64
	mutex            sync.RWMutex
}

func (s *BusStats) incrementPublished(eventType string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.publishedEvents == nil {
		s.publishedEvents = make(map[string]int64)
	}

	s.publishedEvents[eventType]++
}

func (s *BusStats) incrementSubscriptions(eventType string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.subscriptions == nil {
		s.subscriptions = make(map[string]int64)
	}

	s.subscriptions[eventType]++
}

func (s *BusStats) decrementSubscriptions(eventType string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.subscriptions == nil {
		s.subscriptions = make(map[string]int64)
	}

	if s.subscriptions[eventType] > 0 {
		s.subscriptions[eventType]--
	}
}

func (s *BusStats) incrementHandlerSuccess(eventType string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.handlerSuccesses == nil {
		s.handlerSuccesses = make(map[string]int64)
	}

	s.handlerSuccesses[eventType]++
}

func (s *BusStats) incrementHandlerError(eventType string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.handlerErrors == nil {
		s.handlerErrors = make(map[string]int64)
	}

	s.handlerErrors[eventType]++
}

// GetPublishedCount returns the number of published events for a type.
func (s *BusStats) GetPublishedCount(eventType string) int64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.publishedEvents == nil {
		return 0
	}

	return s.publishedEvents[eventType]
}

// GetSubscriptionCount returns the number of subscriptions for a type.
func (s *BusStats) GetSubscriptionCount(eventType string) int64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.subscriptions == nil {
		return 0
	}

	return s.subscriptions[eventType]
}

// GetHandlerSuccessCount returns the number of successful handler executions.
func (s *BusStats) GetHandlerSuccessCount(eventType string) int64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.handlerSuccesses == nil {
		return 0
	}

	return s.handlerSuccesses[eventType]
}

// GetHandlerErrorCount returns the number of handler errors.
func (s *BusStats) GetHandlerErrorCount(eventType string) int64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.handlerErrors == nil {
		return 0
	}

	return s.handlerErrors[eventType]
}

// FuncEventHandler adapts a function to the EventHandler interface.
type FuncEventHandler struct {
	eventType string
	handler   func(ctx context.Context, event Event) error
}

// NewFuncEventHandler creates a new function-based event handler.
func NewFuncEventHandler(eventType string, handler func(ctx context.Context, event Event) error) *FuncEventHandler {
	return &FuncEventHandler{
		eventType: eventType,
		handler:   handler,
	}
}

func (h *FuncEventHandler) Handle(ctx context.Context, event Event) error {
	return h.handler(ctx, event)
}

func (h *FuncEventHandler) CanHandle(eventType string) bool {
	return h.eventType == eventType
}

// EventMiddleware allows intercepting and modifying events.
type EventMiddleware interface {
	Process(ctx context.Context, event Event, next func(ctx context.Context, event Event) error) error
}

// MiddlewareEventBus wraps an EventBus with middleware support.
type MiddlewareEventBus struct {
	inner       EventBus
	middlewares []EventMiddleware
	logger      *zap.Logger
}

// NewMiddlewareEventBus creates a new middleware-enabled event bus.
func NewMiddlewareEventBus(inner EventBus, logger *zap.Logger) *MiddlewareEventBus {
	return &MiddlewareEventBus{
		inner:       inner,
		middlewares: make([]EventMiddleware, 0),
		logger:      logger,
	}
}

// AddMiddleware adds middleware to the bus.
func (bus *MiddlewareEventBus) AddMiddleware(middleware EventMiddleware) {
	bus.middlewares = append(bus.middlewares, middleware)
}

// Publish publishes an event through the middleware chain.
func (bus *MiddlewareEventBus) Publish(event Event) error {
	ctx := event.Context()

	if len(bus.middlewares) == 0 {
		if err := bus.inner.Publish(event); err != nil {
			return fmt.Errorf("failed to publish event: %w", err)
		}

		return nil
	}

	// Build middleware chain
	next := func(ctx context.Context, event Event) error {
		return bus.inner.Publish(event)
	}

	// Apply middlewares in reverse order
	for i := len(bus.middlewares) - 1; i >= 0; i-- {
		middleware := bus.middlewares[i]
		currentNext := next
		next = func(ctx context.Context, event Event) error {
			return middleware.Process(ctx, event, currentNext)
		}
	}

	return next(ctx, event)
}

// Delegate other methods to inner bus.
func (bus *MiddlewareEventBus) Subscribe(eventType string, handler EventHandler) error {
	if err := bus.inner.Subscribe(eventType, handler); err != nil {
		return fmt.Errorf("failed to subscribe to event %s: %w", eventType, err)
	}

	return nil
}

func (bus *MiddlewareEventBus) Unsubscribe(eventType string, handler EventHandler) error {
	if err := bus.inner.Unsubscribe(eventType, handler); err != nil {
		return fmt.Errorf("failed to unsubscribe from event %s: %w", eventType, err)
	}

	return nil
}

func (bus *MiddlewareEventBus) Close() error {
	if err := bus.inner.Close(); err != nil {
		return fmt.Errorf("failed to close event bus: %w", err)
	}

	return nil
}

func (bus *MiddlewareEventBus) Stats() *BusStats {
	return bus.inner.Stats()
}

// LoggingMiddleware logs all events.
type LoggingMiddleware struct {
	logger *zap.Logger
}

// NewLoggingMiddleware creates a new logging middleware.
func NewLoggingMiddleware(logger *zap.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{logger: logger}
}

func (m *LoggingMiddleware) Process(ctx context.Context, event Event, next func(ctx context.Context, event Event) error) error {
	start := time.Now()

	m.logger.Debug("processing event",
		zap.String("event_type", event.Type()),
		zap.String("event_id", event.ID()),
		zap.Time("timestamp", event.Timestamp()))

	err := next(ctx, event)
	duration := time.Since(start)

	if err != nil {
		m.logger.Error("event processing failed",
			zap.String("event_type", event.Type()),
			zap.String("event_id", event.ID()),
			zap.Duration("duration", duration),
			zap.Error(err))
	} else {
		m.logger.Debug("event processing completed",
			zap.String("event_type", event.Type()),
			zap.String("event_id", event.ID()),
			zap.Duration("duration", duration))
	}

	return err
}

// TimeoutMiddleware adds timeout protection to event processing.
type TimeoutMiddleware struct {
	timeout time.Duration
	logger  *zap.Logger
}

// NewTimeoutMiddleware creates a new timeout middleware.
func NewTimeoutMiddleware(timeout time.Duration, logger *zap.Logger) *TimeoutMiddleware {
	return &TimeoutMiddleware{
		timeout: timeout,
		logger:  logger,
	}
}

func (m *TimeoutMiddleware) Process(ctx context.Context, event Event, next func(ctx context.Context, event Event) error) error {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	done := make(chan error, 1)

	go func() {
		done <- next(ctx, event)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		m.logger.Warn("event processing timed out",
			zap.String("event_type", event.Type()),
			zap.String("event_id", event.ID()),
			zap.Duration("timeout", m.timeout))

		return fmt.Errorf("%w after %v", ErrEventProcessingTimeout, m.timeout)
	}
}
