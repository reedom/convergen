package events

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestBaseEvent(t *testing.T) {
	t.Run("creation and properties", func(t *testing.T) {
		ctx := context.Background()
		event := NewBaseEvent("test.event", ctx)
		
		assert.NotEmpty(t, event.ID())
		assert.Equal(t, "test.event", event.Type())
		assert.Equal(t, ctx, event.Context())
		assert.NotZero(t, event.Timestamp())
		assert.NotNil(t, event.Metadata())
		assert.Empty(t, event.Metadata())
	})
	
	t.Run("metadata management", func(t *testing.T) {
		ctx := context.Background()
		event := NewBaseEvent("test.event", ctx)
		
		event.WithMetadata("key1", "value1")
		event.WithMetadata("key2", 42)
		
		metadata := event.Metadata()
		assert.Equal(t, "value1", metadata["key1"])
		assert.Equal(t, 42, metadata["key2"])
	})
}

func TestInMemoryEventBus(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	t.Run("creation", func(t *testing.T) {
		bus := NewInMemoryEventBus(logger)
		assert.NotNil(t, bus)
		assert.NotNil(t, bus.Stats())
	})
	
	t.Run("subscription and unsubscription", func(t *testing.T) {
		bus := NewInMemoryEventBus(logger)
		defer bus.Close()
		
		handler := NewFuncEventHandler("test.event", func(ctx context.Context, event Event) error {
			return nil
		})
		
		// Subscribe
		err := bus.Subscribe("test.event", handler)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), bus.Stats().GetSubscriptionCount("test.event"))
		
		// Unsubscribe
		err = bus.Unsubscribe("test.event", handler)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), bus.Stats().GetSubscriptionCount("test.event"))
	})
	
	t.Run("publish to no handlers", func(t *testing.T) {
		bus := NewInMemoryEventBus(logger)
		defer bus.Close()
		
		ctx := context.Background()
		event := NewBaseEvent("test.event", ctx)
		
		err := bus.Publish(ctx, event)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), bus.Stats().GetPublishedCount("test.event"))
	})
	
	t.Run("publish to handlers", func(t *testing.T) {
		bus := NewInMemoryEventBus(logger)
		defer bus.Close()
		
		var handledEvents []Event
		var mutex sync.Mutex
		
		handler := NewFuncEventHandler("test.event", func(ctx context.Context, event Event) error {
			mutex.Lock()
			defer mutex.Unlock()
			handledEvents = append(handledEvents, event)
			return nil
		})
		
		err := bus.Subscribe("test.event", handler)
		require.NoError(t, err)
		
		ctx := context.Background()
		event := NewBaseEvent("test.event", ctx)
		
		err = bus.Publish(ctx, event)
		assert.NoError(t, err)
		
		// Wait a bit for async processing
		time.Sleep(10 * time.Millisecond)
		
		mutex.Lock()
		assert.Len(t, handledEvents, 1)
		assert.Equal(t, event.ID(), handledEvents[0].ID())
		mutex.Unlock()
		
		assert.Equal(t, int64(1), bus.Stats().GetPublishedCount("test.event"))
		assert.Equal(t, int64(1), bus.Stats().GetHandlerSuccessCount("test.event"))
		assert.Equal(t, int64(0), bus.Stats().GetHandlerErrorCount("test.event"))
	})
	
	t.Run("handler error", func(t *testing.T) {
		bus := NewInMemoryEventBus(logger)
		defer bus.Close()
		
		handler := NewFuncEventHandler("test.event", func(ctx context.Context, event Event) error {
			return assert.AnError
		})
		
		err := bus.Subscribe("test.event", handler)
		require.NoError(t, err)
		
		ctx := context.Background()
		event := NewBaseEvent("test.event", ctx)
		
		err = bus.Publish(ctx, event)
		assert.Error(t, err)
		
		assert.Equal(t, int64(1), bus.Stats().GetPublishedCount("test.event"))
		assert.Equal(t, int64(0), bus.Stats().GetHandlerSuccessCount("test.event"))
		assert.Equal(t, int64(1), bus.Stats().GetHandlerErrorCount("test.event"))
	})
	
	t.Run("multiple handlers", func(t *testing.T) {
		bus := NewInMemoryEventBus(logger)
		defer bus.Close()
		
		var handledCount int32
		var mutex sync.Mutex
		
		handler1 := NewFuncEventHandler("test.event", func(ctx context.Context, event Event) error {
			mutex.Lock()
			handledCount++
			mutex.Unlock()
			return nil
		})
		
		handler2 := NewFuncEventHandler("test.event", func(ctx context.Context, event Event) error {
			mutex.Lock()
			handledCount++
			mutex.Unlock()
			return nil
		})
		
		err := bus.Subscribe("test.event", handler1)
		require.NoError(t, err)
		err = bus.Subscribe("test.event", handler2)
		require.NoError(t, err)
		
		ctx := context.Background()
		event := NewBaseEvent("test.event", ctx)
		
		err = bus.Publish(ctx, event)
		assert.NoError(t, err)
		
		// Wait for async processing
		time.Sleep(10 * time.Millisecond)
		
		mutex.Lock()
		assert.Equal(t, int32(2), handledCount)
		mutex.Unlock()
		
		assert.Equal(t, int64(2), bus.Stats().GetSubscriptionCount("test.event"))
		assert.Equal(t, int64(2), bus.Stats().GetHandlerSuccessCount("test.event"))
	})
	
	t.Run("closed bus", func(t *testing.T) {
		bus := NewInMemoryEventBus(logger)
		
		handler := NewFuncEventHandler("test.event", func(ctx context.Context, event Event) error {
			return nil
		})
		
		err := bus.Subscribe("test.event", handler)
		require.NoError(t, err)
		
		err = bus.Close()
		assert.NoError(t, err)
		
		// Operations should fail after close
		err = bus.Subscribe("test.event", handler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "event bus is closed")
		
		ctx := context.Background()
		event := NewBaseEvent("test.event", ctx)
		err = bus.Publish(ctx, event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "event bus is closed")
	})
	
	t.Run("subscription validation", func(t *testing.T) {
		bus := NewInMemoryEventBus(logger)
		defer bus.Close()
		
		// Handler that can't handle the event type
		handler := NewFuncEventHandler("other.event", func(ctx context.Context, event Event) error {
			return nil
		})
		
		err := bus.Subscribe("test.event", handler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "handler cannot handle event type")
	})
	
	t.Run("unsubscribe non-existent", func(t *testing.T) {
		bus := NewInMemoryEventBus(logger)
		defer bus.Close()
		
		handler := NewFuncEventHandler("test.event", func(ctx context.Context, event Event) error {
			return nil
		})
		
		// Unsubscribe from non-existent event type
		err := bus.Unsubscribe("test.event", handler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no handlers for event type")
		
		// Subscribe and then unsubscribe a different handler
		err = bus.Subscribe("test.event", handler)
		require.NoError(t, err)
		
		otherHandler := NewFuncEventHandler("test.event", func(ctx context.Context, event Event) error {
			return nil
		})
		
		err = bus.Unsubscribe("test.event", otherHandler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "handler not found")
	})
}

func TestFuncEventHandler(t *testing.T) {
	t.Run("creation and properties", func(t *testing.T) {
		called := false
		handler := NewFuncEventHandler("test.event", func(ctx context.Context, event Event) error {
			called = true
			return nil
		})
		
		assert.True(t, handler.CanHandle("test.event"))
		assert.False(t, handler.CanHandle("other.event"))
		
		ctx := context.Background()
		event := NewBaseEvent("test.event", ctx)
		
		err := handler.Handle(ctx, event)
		assert.NoError(t, err)
		assert.True(t, called)
	})
}

func TestMiddlewareEventBus(t *testing.T) {
	logger := zaptest.NewLogger(t)
	innerBus := NewInMemoryEventBus(logger)
	bus := NewMiddlewareEventBus(innerBus, logger)
	defer bus.Close()
	
	t.Run("delegation without middleware", func(t *testing.T) {
		handler := NewFuncEventHandler("test.event", func(ctx context.Context, event Event) error {
			return nil
		})
		
		err := bus.Subscribe("test.event", handler)
		assert.NoError(t, err)
		
		ctx := context.Background()
		event := NewBaseEvent("test.event", ctx)
		
		err = bus.Publish(ctx, event)
		assert.NoError(t, err)
		
		stats := bus.Stats()
		assert.Equal(t, int64(1), stats.GetPublishedCount("test.event"))
	})
	
	t.Run("middleware processing", func(t *testing.T) {
		var middlewareCalled bool
		var eventProcessed bool
		
		middleware := &testMiddleware{
			process: func(ctx context.Context, event Event, next func(ctx context.Context, event Event) error) error {
				middlewareCalled = true
				return next(ctx, event)
			},
		}
		
		bus.AddMiddleware(middleware)
		
		handler := NewFuncEventHandler("test.event", func(ctx context.Context, event Event) error {
			eventProcessed = true
			return nil
		})
		
		err := bus.Subscribe("test.event", handler)
		require.NoError(t, err)
		
		ctx := context.Background()
		event := NewBaseEvent("test.event", ctx)
		
		err = bus.Publish(ctx, event)
		assert.NoError(t, err)
		
		time.Sleep(10 * time.Millisecond)
		
		assert.True(t, middlewareCalled)
		assert.True(t, eventProcessed)
	})
}

// Test middleware implementation
type testMiddleware struct {
	process func(ctx context.Context, event Event, next func(ctx context.Context, event Event) error) error
}

func (m *testMiddleware) Process(ctx context.Context, event Event, next func(ctx context.Context, event Event) error) error {
	return m.process(ctx, event, next)
}

func TestLoggingMiddleware(t *testing.T) {
	logger := zaptest.NewLogger(t)
	middleware := NewLoggingMiddleware(logger)
	
	t.Run("successful processing", func(t *testing.T) {
		called := false
		next := func(ctx context.Context, event Event) error {
			called = true
			return nil
		}
		
		ctx := context.Background()
		event := NewBaseEvent("test.event", ctx)
		
		err := middleware.Process(ctx, event, next)
		assert.NoError(t, err)
		assert.True(t, called)
	})
	
	t.Run("error processing", func(t *testing.T) {
		next := func(ctx context.Context, event Event) error {
			return assert.AnError
		}
		
		ctx := context.Background()
		event := NewBaseEvent("test.event", ctx)
		
		err := middleware.Process(ctx, event, next)
		assert.Error(t, err)
		assert.Equal(t, assert.AnError, err)
	})
}

func TestTimeoutMiddleware(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	t.Run("successful processing within timeout", func(t *testing.T) {
		middleware := NewTimeoutMiddleware(100*time.Millisecond, logger)
		
		called := false
		next := func(ctx context.Context, event Event) error {
			called = true
			return nil
		}
		
		ctx := context.Background()
		event := NewBaseEvent("test.event", ctx)
		
		err := middleware.Process(ctx, event, next)
		assert.NoError(t, err)
		assert.True(t, called)
	})
	
	t.Run("timeout processing", func(t *testing.T) {
		middleware := NewTimeoutMiddleware(10*time.Millisecond, logger)
		
		next := func(ctx context.Context, event Event) error {
			time.Sleep(50 * time.Millisecond) // Longer than timeout
			return nil
		}
		
		ctx := context.Background()
		event := NewBaseEvent("test.event", ctx)
		
		err := middleware.Process(ctx, event, next)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timed out")
	})
}

func TestBusStats(t *testing.T) {
	t.Run("concurrency safety", func(t *testing.T) {
		stats := &BusStats{}
		
		// Run concurrent operations
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				stats.incrementPublished("test.event")
				stats.incrementSubscriptions("test.event")
				stats.incrementHandlerSuccess("test.event")
				stats.incrementHandlerError("test.event")
			}()
		}
		
		wg.Wait()
		
		assert.Equal(t, int64(100), stats.GetPublishedCount("test.event"))
		assert.Equal(t, int64(100), stats.GetSubscriptionCount("test.event"))
		assert.Equal(t, int64(100), stats.GetHandlerSuccessCount("test.event"))
		assert.Equal(t, int64(100), stats.GetHandlerErrorCount("test.event"))
	})
	
	t.Run("decrement subscriptions", func(t *testing.T) {
		stats := &BusStats{}
		
		stats.incrementSubscriptions("test.event")
		stats.incrementSubscriptions("test.event")
		assert.Equal(t, int64(2), stats.GetSubscriptionCount("test.event"))
		
		stats.decrementSubscriptions("test.event")
		assert.Equal(t, int64(1), stats.GetSubscriptionCount("test.event"))
		
		// Should not go below zero
		stats.decrementSubscriptions("test.event")
		stats.decrementSubscriptions("test.event")
		assert.Equal(t, int64(0), stats.GetSubscriptionCount("test.event"))
	})
}