package coordinator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ContextManager manages context propagation and cancellation throughout the pipeline
type ContextManager interface {
	// Create pipeline context with timeout
	CreatePipelineContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc)
	
	// Create component context with tracing information
	CreateComponentContext(parent context.Context, component string) context.Context
	
	// Check if context should cancel pipeline
	ShouldCancel(ctx context.Context) bool
	
	// Propagate cancellation to all active contexts
	CancelAll()
	
	// Add context metadata
	AddMetadata(ctx context.Context, key string, value interface{}) context.Context
	
	// Get context metadata
	GetMetadata(ctx context.Context, key string) (interface{}, bool)
	
	// Track active contexts
	TrackContext(ctx context.Context, description string)
	
	// Get active context count
	GetActiveContextCount() int
}

// ConcreteContextManager implements ContextManager
type ConcreteContextManager struct {
	logger *zap.Logger
	config *Config
	
	// Context tracking
	mutex           sync.RWMutex
	activeContexts  map[context.Context]string
	cancelFuncs     []context.CancelFunc
	globalCancel    context.CancelFunc
	
	// Context metadata
	contextMetadata map[context.Context]map[string]interface{}
	
	// Lifecycle management
	shutdown chan struct{}
}

// NewContextManager creates a new context manager
func NewContextManager(logger *zap.Logger, config *Config) ContextManager {
	mgr := &ConcreteContextManager{
		logger:          logger,
		config:          config,
		activeContexts:  make(map[context.Context]string),
		contextMetadata: make(map[context.Context]map[string]interface{}),
		shutdown:        make(chan struct{}),
	}
	
	logger.Debug("context manager initialized")
	
	return mgr
}

// CreatePipelineContext creates a new pipeline context with timeout
func (c *ConcreteContextManager) CreatePipelineContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(parent, timeout)
	
	// Add pipeline metadata
	ctx = c.addPipelineMetadata(ctx)
	
	// Track the context
	c.activeContexts[ctx] = "pipeline"
	c.cancelFuncs = append(c.cancelFuncs, cancel)
	
	// Store global cancel function for the first pipeline context
	if c.globalCancel == nil {
		c.globalCancel = cancel
	}
	
	c.logger.Debug("pipeline context created",
		zap.Duration("timeout", timeout),
		zap.Int("active_contexts", len(c.activeContexts)))
	
	// Return wrapped cancel function for cleanup
	wrappedCancel := c.wrapCancelFunc(ctx, cancel, "pipeline")
	
	return ctx, wrappedCancel
}

// CreateComponentContext creates a context for a specific component
func (c *ConcreteContextManager) CreateComponentContext(parent context.Context, component string) context.Context {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	// Create derived context
	ctx := context.WithValue(parent, contextKeyComponent, component)
	ctx = context.WithValue(ctx, contextKeyStartTime, time.Now())
	
	// Add component-specific metadata
	ctx = c.addComponentMetadata(ctx, component)
	
	// Track the context
	c.activeContexts[ctx] = component
	
	c.logger.Debug("component context created",
		zap.String("component", component),
		zap.Int("active_contexts", len(c.activeContexts)))
	
	return ctx
}

// ShouldCancel checks if the context indicates pipeline should be cancelled
func (c *ConcreteContextManager) ShouldCancel(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		err := ctx.Err()
		c.logger.Debug("context cancellation detected",
			zap.Error(err),
			zap.String("reason", c.getCancellationReason(err)))
		return true
	case <-c.shutdown:
		c.logger.Debug("context manager shutdown detected")
		return true
	default:
		return false
	}
}

// CancelAll cancels all active contexts
func (c *ConcreteContextManager) CancelAll() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.logger.Info("cancelling all active contexts",
		zap.Int("context_count", len(c.activeContexts)))
	
	// Cancel all tracked contexts
	for _, cancel := range c.cancelFuncs {
		cancel()
	}
	
	// Clear tracking
	c.activeContexts = make(map[context.Context]string)
	c.cancelFuncs = c.cancelFuncs[:0]
	c.globalCancel = nil
	
	// Signal shutdown
	close(c.shutdown)
}

// AddMetadata adds metadata to a context
func (c *ConcreteContextManager) AddMetadata(ctx context.Context, key string, value interface{}) context.Context {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	// Initialize metadata map if needed
	if c.contextMetadata[ctx] == nil {
		c.contextMetadata[ctx] = make(map[string]interface{})
	}
	
	c.contextMetadata[ctx][key] = value
	
	c.logger.Debug("context metadata added",
		zap.String("key", key),
		zap.Any("value", value))
	
	// Return context with metadata
	return context.WithValue(ctx, contextKey(key), value)
}

// GetMetadata retrieves metadata from a context
func (c *ConcreteContextManager) GetMetadata(ctx context.Context, key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	// First check our metadata store
	if metadata, exists := c.contextMetadata[ctx]; exists {
		if value, found := metadata[key]; found {
			return value, true
		}
	}
	
	// Then check context values
	if value := ctx.Value(contextKey(key)); value != nil {
		return value, true
	}
	
	return nil, false
}

// TrackContext adds a context to the tracking system
func (c *ConcreteContextManager) TrackContext(ctx context.Context, description string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.activeContexts[ctx] = description
	
	c.logger.Debug("context tracked",
		zap.String("description", description),
		zap.Int("active_contexts", len(c.activeContexts)))
}

// GetActiveContextCount returns the number of active contexts
func (c *ConcreteContextManager) GetActiveContextCount() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	return len(c.activeContexts)
}

// Private methods

func (c *ConcreteContextManager) addPipelineMetadata(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, contextKeyPipelineID, c.generatePipelineID())
	ctx = context.WithValue(ctx, contextKeyStartTime, time.Now())
	ctx = context.WithValue(ctx, contextKeyManagerID, "coordinator")
	
	return ctx
}

func (c *ConcreteContextManager) addComponentMetadata(ctx context.Context, component string) context.Context {
	ctx = context.WithValue(ctx, contextKeyComponent, component)
	ctx = context.WithValue(ctx, contextKeyComponentStartTime, time.Now())
	
	// Add component-specific configuration
	switch component {
	case "parser":
		ctx = context.WithValue(ctx, contextKeyConfig, c.config.ParserConfig)
	case "planner":
		ctx = context.WithValue(ctx, contextKeyConfig, c.config.PlannerConfig)
	case "executor":
		ctx = context.WithValue(ctx, contextKeyConfig, c.config.ExecutorConfig)
	case "emitter":
		ctx = context.WithValue(ctx, contextKeyConfig, c.config.EmitterConfig)
	}
	
	return ctx
}

func (c *ConcreteContextManager) wrapCancelFunc(ctx context.Context, cancel context.CancelFunc, description string) context.CancelFunc {
	return func() {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		
		// Remove from tracking
		delete(c.activeContexts, ctx)
		delete(c.contextMetadata, ctx)
		
		// Call original cancel
		cancel()
		
		c.logger.Debug("context cancelled and cleaned up",
			zap.String("description", description),
			zap.Int("remaining_contexts", len(c.activeContexts)))
	}
}

func (c *ConcreteContextManager) getCancellationReason(err error) string {
	switch err {
	case context.Canceled:
		return "cancelled"
	case context.DeadlineExceeded:
		return "timeout"
	default:
		return err.Error()
	}
}

func (c *ConcreteContextManager) generatePipelineID() string {
	return fmt.Sprintf("ctx_%d_%d", time.Now().Unix(), time.Now().Nanosecond())
}

// Context cleanup methods

// CleanupContext removes a context from tracking
func (c *ConcreteContextManager) CleanupContext(ctx context.Context) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	description := c.activeContexts[ctx]
	delete(c.activeContexts, ctx)
	delete(c.contextMetadata, ctx)
	
	c.logger.Debug("context cleaned up",
		zap.String("description", description),
		zap.Int("remaining_contexts", len(c.activeContexts)))
}

// GetContextInfo returns information about a specific context
func (c *ConcreteContextManager) GetContextInfo(ctx context.Context) map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	info := make(map[string]interface{})
	
	// Add description
	if desc, exists := c.activeContexts[ctx]; exists {
		info["description"] = desc
	}
	
	// Add metadata
	if metadata, exists := c.contextMetadata[ctx]; exists {
		info["metadata"] = metadata
	}
	
	// Add context values
	if pipelineID := ctx.Value(contextKeyPipelineID); pipelineID != nil {
		info["pipeline_id"] = pipelineID
	}
	
	if component := ctx.Value(contextKeyComponent); component != nil {
		info["component"] = component
	}
	
	if startTime := ctx.Value(contextKeyStartTime); startTime != nil {
		if start, ok := startTime.(time.Time); ok {
			info["start_time"] = start
			info["elapsed_time"] = time.Since(start)
		}
	}
	
	// Add context status
	select {
	case <-ctx.Done():
		info["status"] = "cancelled"
		info["error"] = ctx.Err().Error()
	default:
		info["status"] = "active"
	}
	
	return info
}

// GetAllContextInfo returns information about all active contexts
func (c *ConcreteContextManager) GetAllContextInfo() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	info := make(map[string]interface{})
	info["active_count"] = len(c.activeContexts)
	info["total_cancel_funcs"] = len(c.cancelFuncs)
	
	// Add details for each context
	contexts := make([]map[string]interface{}, 0, len(c.activeContexts))
	for ctx, description := range c.activeContexts {
		contextInfo := map[string]interface{}{
			"description": description,
		}
		
		// Add timing information
		if startTime := ctx.Value(contextKeyStartTime); startTime != nil {
			if start, ok := startTime.(time.Time); ok {
				contextInfo["elapsed_time"] = time.Since(start)
			}
		}
		
		// Add status
		select {
		case <-ctx.Done():
			contextInfo["status"] = "cancelled"
		default:
			contextInfo["status"] = "active"
		}
		
		contexts = append(contexts, contextInfo)
	}
	
	info["contexts"] = contexts
	
	return info
}

// Context key definitions
type contextKey string

const (
	contextKeyPipelineID        contextKey = "pipeline_id"
	contextKeyComponent         contextKey = "component"
	contextKeyStartTime         contextKey = "start_time"
	contextKeyComponentStartTime contextKey = "component_start_time"
	contextKeyManagerID         contextKey = "manager_id"
	contextKeyConfig            contextKey = "config"
)

// Utility functions for extracting context values

// GetPipelineID extracts pipeline ID from context
func GetPipelineID(ctx context.Context) (string, bool) {
	if value := ctx.Value(contextKeyPipelineID); value != nil {
		if pipelineID, ok := value.(string); ok {
			return pipelineID, true
		}
	}
	return "", false
}

// GetComponentName extracts component name from context
func GetComponentName(ctx context.Context) (string, bool) {
	if value := ctx.Value(contextKeyComponent); value != nil {
		if component, ok := value.(string); ok {
			return component, true
		}
	}
	return "", false
}

// GetStartTime extracts start time from context
func GetStartTime(ctx context.Context) (time.Time, bool) {
	if value := ctx.Value(contextKeyStartTime); value != nil {
		if startTime, ok := value.(time.Time); ok {
			return startTime, true
		}
	}
	return time.Time{}, false
}

// GetElapsedTime calculates elapsed time since context creation
func GetElapsedTime(ctx context.Context) (time.Duration, bool) {
	if startTime, ok := GetStartTime(ctx); ok {
		return time.Since(startTime), true
	}
	return 0, false
}

// IsContextActive checks if context is still active (not cancelled)
func IsContextActive(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	default:
		return true
	}
}