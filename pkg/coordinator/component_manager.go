package coordinator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/reedom/convergen/v8/pkg/emitter"
	"github.com/reedom/convergen/v8/pkg/executor"
	"github.com/reedom/convergen/v8/pkg/internal/events"
	"github.com/reedom/convergen/v8/pkg/parser"
	"github.com/reedom/convergen/v8/pkg/planner"
	"go.uber.org/zap"
)

// ComponentManager manages the lifecycle of all pipeline components
type ComponentManager interface {
	// Initialize all pipeline components
	Initialize(ctx context.Context, config *Config) error
	
	// Register component with event handlers
	RegisterComponent(name string, component PipelineComponent) error
	
	// Get component by name
	GetComponent(name string) (PipelineComponent, error)
	
	// Get all components
	GetComponents() map[string]PipelineComponent
	
	// Shutdown all components
	Shutdown(ctx context.Context) error
	
	// Get component status
	GetComponentStatus(name string) ComponentStatus
	
	// Update component status
	UpdateComponentStatus(name string, status ComponentStatus)
}

// ConcreteComponentManager implements ComponentManager
type ConcreteComponentManager struct {
	logger     *zap.Logger
	config     *Config
	eventBus   events.EventBus
	
	// Component registry
	mutex      sync.RWMutex
	components map[string]PipelineComponent
	status     map[string]ComponentStatus
	factories  map[string]ComponentFactory
	
	// Lifecycle management
	initialized bool
	shutdown    chan struct{}
}

// NewComponentManager creates a new component manager
func NewComponentManager(logger *zap.Logger, config *Config) ComponentManager {
	mgr := &ConcreteComponentManager{
		logger:     logger,
		config:     config,
		components: make(map[string]PipelineComponent),
		status:     make(map[string]ComponentStatus),
		factories:  make(map[string]ComponentFactory),
		shutdown:   make(chan struct{}),
	}
	
	// Register default component factories
	mgr.registerDefaultFactories()
	
	return mgr
}

// Initialize all pipeline components
func (c *ConcreteComponentManager) Initialize(ctx context.Context, config *Config) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if c.initialized {
		return nil
	}
	
	c.logger.Info("initializing pipeline components")
	
	// Create components in dependency order
	components := []struct {
		name   string
		config interface{}
	}{
		{"parser", config.ParserConfig},
		{"planner", config.PlannerConfig}, 
		{"executor", config.ExecutorConfig},
		{"emitter", config.EmitterConfig},
	}
	
	// Initialize each component
	for _, comp := range components {
		c.status[comp.name] = StatusInitializing
		
		factory, exists := c.factories[comp.name]
		if !exists {
			return fmt.Errorf("no factory registered for component: %s", comp.name)
		}
		
		component, err := factory(comp.config)
		if err != nil {
			c.status[comp.name] = StatusFailed
			return fmt.Errorf("failed to create component %s: %w", comp.name, err)
		}
		
		// Initialize component with event bus
		if err := component.Initialize(ctx, c.eventBus); err != nil {
			c.status[comp.name] = StatusFailed
			return fmt.Errorf("failed to initialize component %s: %w", comp.name, err)
		}
		
		c.components[comp.name] = component
		c.status[comp.name] = StatusReady
		
		c.logger.Debug("component initialized",
			zap.String("component", comp.name),
			zap.String("status", string(StatusReady)))
	}
	
	c.initialized = true
	c.logger.Info("all pipeline components initialized successfully")
	
	return nil
}

// RegisterComponent registers a component with event handlers
func (c *ConcreteComponentManager) RegisterComponent(name string, component PipelineComponent) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if c.initialized {
		return fmt.Errorf("cannot register component after initialization: %s", name)
	}
	
	c.components[name] = component
	c.status[name] = StatusReady
	
	c.logger.Debug("component registered", zap.String("component", name))
	
	return nil
}

// GetComponent retrieves a component by name
func (c *ConcreteComponentManager) GetComponent(name string) (PipelineComponent, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	component, exists := c.components[name]
	if !exists {
		return nil, fmt.Errorf("component not found: %s", name)
	}
	
	return component, nil
}

// GetComponents returns all registered components
func (c *ConcreteComponentManager) GetComponents() map[string]PipelineComponent {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	result := make(map[string]PipelineComponent)
	for name, component := range c.components {
		result[name] = component
	}
	
	return result
}

// Shutdown gracefully shuts down all components
func (c *ConcreteComponentManager) Shutdown(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if !c.initialized {
		return nil
	}
	
	c.logger.Info("shutting down pipeline components")
	
	// Signal shutdown
	close(c.shutdown)
	
	// Shutdown components in reverse order
	componentOrder := []string{"emitter", "executor", "planner", "parser"}
	var shutdownErrors []error
	
	for _, name := range componentOrder {
		if component, exists := c.components[name]; exists {
			c.status[name] = StatusShutdown
			
			// Create timeout context for component shutdown
			shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			
			if err := component.Shutdown(shutdownCtx); err != nil {
				shutdownErrors = append(shutdownErrors, 
					fmt.Errorf("component %s shutdown failed: %w", name, err))
			}
			
			cancel()
			
			c.logger.Debug("component shutdown complete", zap.String("component", name))
		}
	}
	
	c.initialized = false
	
	if len(shutdownErrors) > 0 {
		return fmt.Errorf("component shutdown errors: %v", shutdownErrors)
	}
	
	c.logger.Info("all pipeline components shutdown successfully")
	return nil
}

// GetComponentStatus returns the status of a specific component
func (c *ConcreteComponentManager) GetComponentStatus(name string) ComponentStatus {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	if status, exists := c.status[name]; exists {
		return status
	}
	
	return StatusFailed // Component not found
}

// UpdateComponentStatus updates the status of a component
func (c *ConcreteComponentManager) UpdateComponentStatus(name string, status ComponentStatus) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	oldStatus := c.status[name]
	c.status[name] = status
	
	c.logger.Debug("component status updated",
		zap.String("component", name),
		zap.String("old_status", string(oldStatus)),
		zap.String("new_status", string(status)))
}

// Private methods

func (c *ConcreteComponentManager) registerDefaultFactories() {
	// Parser factory
	c.factories["parser"] = func(config interface{}) (PipelineComponent, error) {
		parserConfig, ok := config.(*parser.Config)
		if !ok {
			parserConfig = parser.DefaultConfig()
		}
		
		p := parser.NewParser(c.logger, parserConfig)
		return &ParserAdapter{parser: p}, nil
	}
	
	// Planner factory
	c.factories["planner"] = func(config interface{}) (PipelineComponent, error) {
		plannerConfig, ok := config.(*planner.Config)
		if !ok {
			plannerConfig = planner.DefaultConfig()
		}
		
		p := planner.NewPlanner(c.logger, plannerConfig)
		return &PlannerAdapter{planner: p}, nil
	}
	
	// Executor factory
	c.factories["executor"] = func(config interface{}) (PipelineComponent, error) {
		executorConfig, ok := config.(*executor.Config)
		if !ok {
			executorConfig = executor.DefaultConfig()
		}
		
		e := executor.NewExecutor(c.logger, executorConfig)
		return &ExecutorAdapter{executor: e}, nil
	}
	
	// Emitter factory
	c.factories["emitter"] = func(config interface{}) (PipelineComponent, error) {
		emitterConfig, ok := config.(*emitter.EmitterConfig)
		if !ok {
			emitterConfig = emitter.DefaultEmitterConfig()
		}
		
		e := emitter.NewEmitter(c.logger, emitterConfig)
		return &EmitterAdapter{emitter: e}, nil
	}
}

// Component adapters to implement PipelineComponent interface

// ParserAdapter adapts parser.Parser to PipelineComponent
type ParserAdapter struct {
	parser parser.Parser
	status ComponentStatus
}

func (p *ParserAdapter) Name() string { return "parser" }

func (p *ParserAdapter) Initialize(ctx context.Context, eventBus events.EventBus) error {
	p.status = StatusReady
	return nil
}

func (p *ParserAdapter) Shutdown(ctx context.Context) error {
	p.status = StatusShutdown
	return nil
}

func (p *ParserAdapter) GetMetrics() interface{} {
	return p.parser.GetMetrics()
}

func (p *ParserAdapter) GetStatus() ComponentStatus {
	return p.status
}

// PlannerAdapter adapts planner.Planner to PipelineComponent
type PlannerAdapter struct {
	planner planner.Planner
	status  ComponentStatus
}

func (p *PlannerAdapter) Name() string { return "planner" }

func (p *PlannerAdapter) Initialize(ctx context.Context, eventBus events.EventBus) error {
	p.status = StatusReady
	return nil
}

func (p *PlannerAdapter) Shutdown(ctx context.Context) error {
	p.status = StatusShutdown
	return nil
}

func (p *PlannerAdapter) GetMetrics() interface{} {
	return p.planner.GetMetrics()
}

func (p *PlannerAdapter) GetStatus() ComponentStatus {
	return p.status
}

// ExecutorAdapter adapts executor.Executor to PipelineComponent
type ExecutorAdapter struct {
	executor executor.Executor
	status   ComponentStatus
}

func (e *ExecutorAdapter) Name() string { return "executor" }

func (e *ExecutorAdapter) Initialize(ctx context.Context, eventBus events.EventBus) error {
	e.status = StatusReady
	return nil
}

func (e *ExecutorAdapter) Shutdown(ctx context.Context) error {
	e.status = StatusShutdown
	return nil
}

func (e *ExecutorAdapter) GetMetrics() interface{} {
	return e.executor.GetMetrics()
}

func (e *ExecutorAdapter) GetStatus() ComponentStatus {
	return e.status
}

// EmitterAdapter adapts emitter.Emitter to PipelineComponent
type EmitterAdapter struct {
	emitter emitter.Emitter
	status  ComponentStatus
}

func (e *EmitterAdapter) Name() string { return "emitter" }

func (e *EmitterAdapter) Initialize(ctx context.Context, eventBus events.EventBus) error {
	e.status = StatusReady
	return nil
}

func (e *EmitterAdapter) Shutdown(ctx context.Context) error {
	e.status = StatusShutdown
	return nil
}

func (e *EmitterAdapter) GetMetrics() interface{} {
	return e.emitter.GetMetrics()
}

func (e *EmitterAdapter) GetStatus() ComponentStatus {
	return e.status
}