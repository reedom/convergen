package coordinator

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v9/pkg/emitter"
	"github.com/reedom/convergen/v9/pkg/executor"
	"github.com/reedom/convergen/v9/pkg/internal/events"
	"github.com/reedom/convergen/v9/pkg/parser"
	"github.com/reedom/convergen/v9/pkg/planner"
)

// Static errors for err113 compliance.
var (
	ErrNoFactoryRegistered     = errors.New("no factory registered for component")
	ErrCannotRegisterAfterInit = errors.New("cannot register component after initialization")
	ErrComponentNotFound       = errors.New("component not found")
	ErrComponentShutdownErrors = errors.New("component shutdown errors")
)

const (
	// ComponentParser is the parser component name.
	ComponentParser = "parser"
	// ComponentPlanner is the planner component name.
	ComponentPlanner = "planner"
	// ComponentEmitter is the emitter component name.
	ComponentEmitter = "emitter"
	// ComponentExecutor is the executor component name.
	ComponentExecutor = "executor"
)

// ComponentManager manages the lifecycle of all pipeline components.
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

// ConcreteComponentManager implements ComponentManager.
type ConcreteComponentManager struct {
	logger   *zap.Logger
	config   *Config
	eventBus events.EventBus

	// Component registry
	mutex      sync.RWMutex
	components map[string]PipelineComponent
	status     map[string]ComponentStatus
	factories  map[string]ComponentFactory

	// Lifecycle management
	initialized bool
	shutdown    chan struct{}
}

// NewComponentManager creates a new component manager.
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

// Initialize all pipeline components.
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
		{ComponentParser, config.ParserConfig},
		{ComponentPlanner, config.PlannerConfig},
		{"executor", config.ExecutorConfig},
		{ComponentEmitter, config.EmitterConfig},
	}

	// Initialize each component
	for _, comp := range components {
		c.status[comp.name] = StatusInitializing

		factory, exists := c.factories[comp.name]
		if !exists {
			return fmt.Errorf("%w: %s", ErrNoFactoryRegistered, comp.name)
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

// RegisterComponent registers a component with event handlers.
func (c *ConcreteComponentManager) RegisterComponent(name string, component PipelineComponent) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.initialized {
		return fmt.Errorf("%w: %s", ErrCannotRegisterAfterInit, name)
	}

	c.components[name] = component
	c.status[name] = StatusReady

	c.logger.Debug("component registered", zap.String("component", name))

	return nil
}

// GetComponent retrieves a component by name.
func (c *ConcreteComponentManager) GetComponent(name string) (PipelineComponent, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	component, exists := c.components[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrComponentNotFound, name)
	}

	return component, nil
}

// GetComponents returns all registered components.
func (c *ConcreteComponentManager) GetComponents() map[string]PipelineComponent {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	result := make(map[string]PipelineComponent)
	for name, component := range c.components {
		result[name] = component
	}

	return result
}

// Shutdown gracefully shuts down all components.
func (c *ConcreteComponentManager) Shutdown(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// If not initialized but we have registered components, still shut them down
	if !c.initialized && len(c.components) == 0 {
		return nil
	}

	c.logger.Info("shutting down pipeline components")

	// Signal shutdown
	close(c.shutdown)

	var shutdownErrors []error

	// Shutdown predefined components in reverse order
	c.shutdownPredefinedComponents(ctx, &shutdownErrors)

	// Shutdown any additional registered components
	c.shutdownAdditionalComponents(ctx, &shutdownErrors)

	c.initialized = false

	return c.handleShutdownErrors(shutdownErrors)
}

// shutdownPredefinedComponents shuts down predefined components in reverse order.
func (c *ConcreteComponentManager) shutdownPredefinedComponents(ctx context.Context, shutdownErrors *[]error) {
	componentOrder := []string{ComponentEmitter, "executor", ComponentPlanner, ComponentParser}

	for _, name := range componentOrder {
		if component, exists := c.components[name]; exists {
			c.shutdownSingleComponent(ctx, name, component, shutdownErrors)
		}
	}
}

// shutdownAdditionalComponents shuts down any additional registered components.
func (c *ConcreteComponentManager) shutdownAdditionalComponents(ctx context.Context, shutdownErrors *[]error) {
	predefinedSet := map[string]bool{
		ComponentEmitter: true, "executor": true, ComponentPlanner: true, ComponentParser: true,
	}

	for name, component := range c.components {
		if !predefinedSet[name] {
			c.shutdownSingleComponent(ctx, name, component, shutdownErrors)
		}
	}
}

// shutdownSingleComponent shuts down a single component with timeout.
func (c *ConcreteComponentManager) shutdownSingleComponent(ctx context.Context, name string, component PipelineComponent, shutdownErrors *[]error) {
	c.status[name] = StatusShutdown

	// Create timeout context for component shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := component.Shutdown(shutdownCtx); err != nil {
		*shutdownErrors = append(*shutdownErrors,
			fmt.Errorf("component %s shutdown failed: %w", name, err))
	}

	c.logger.Debug("component shutdown complete", zap.String("component", name))
}

// handleShutdownErrors processes and returns shutdown errors.
func (c *ConcreteComponentManager) handleShutdownErrors(shutdownErrors []error) error {
	if len(shutdownErrors) > 0 {
		var errorStrings []string
		for _, err := range shutdownErrors {
			errorStrings = append(errorStrings, err.Error())
		}

		return fmt.Errorf("%w: [%s]", ErrComponentShutdownErrors, strings.Join(errorStrings, ", "))
	}

	c.logger.Info("all pipeline components shutdown successfully")

	return nil
}

// GetComponentStatus returns the status of a specific component.
func (c *ConcreteComponentManager) GetComponentStatus(name string) ComponentStatus {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if status, exists := c.status[name]; exists {
		return status
	}

	return StatusFailed // Component not found
}

// UpdateComponentStatus updates the status of a component.
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
	c.factories[ComponentParser] = func(config interface{}) (PipelineComponent, error) {
		parserConfig, ok := config.(*parser.Config)
		if !ok {
			// Create default parser config since DefaultConfig doesn't exist
			parserConfig = &parser.Config{
				BuildTag:              "convergen",
				MaxConcurrentWorkers:  4,
				TypeResolutionTimeout: 30 * time.Second,
			}
		}

		// Note: parser.NewParser has signature (srcPath, dstPath string) (*Parser, error)
		// This is a placeholder adapter that needs proper implementation
		return &ParserAdapter{config: parserConfig}, nil
	}

	// Planner factory
	c.factories[ComponentPlanner] = func(config interface{}) (PipelineComponent, error) {
		plannerConfig, ok := config.(*planner.Config)
		if !ok {
			// Use the default from planner package
			plannerConfig = &planner.Config{
				MaxConcurrentWorkers: 4,
				MaxMemoryMB:          512,
				PlanningTimeout:      30 * time.Second,
				EnableOptimizations:  true,
				OptimizationLevel:    1,
				MinBatchSize:         1,
				MaxBatchSize:         100,
				EnableMetrics:        true,
				DebugMode:            false,
			}
		}

		p := planner.NewExecutionPlanner(c.logger, c.eventBus, plannerConfig)

		return &PlannerAdapter{planner: p}, nil
	}

	// Executor factory
	c.factories["executor"] = func(config interface{}) (PipelineComponent, error) {
		executorConfig, ok := config.(*executor.Config)
		if !ok {
			executorConfig = executor.DefaultConfig()
		}

		e := executor.NewExecutor(c.logger, c.eventBus, executorConfig)

		return &ExecutorAdapter{executor: e}, nil
	}

	// Emitter factory
	c.factories[ComponentEmitter] = func(config interface{}) (PipelineComponent, error) {
		emitterConfig, ok := config.(*emitter.Config)
		if !ok {
			emitterConfig = emitter.DefaultConfig()
		}

		e := emitter.NewEmitter(c.logger, c.eventBus, emitterConfig)

		return &EmitterAdapter{emitter: e}, nil
	}
}

// Component adapters to implement PipelineComponent interface

// ParserAdapter adapts parser.Parser to PipelineComponent.
type ParserAdapter struct {
	config *parser.Config
	status ComponentStatus
}

// Name returns the parser component name.
func (p *ParserAdapter) Name() string { return ComponentParser }

// Initialize initializes the parser adapter.
func (p *ParserAdapter) Initialize(ctx context.Context, eventBus events.EventBus) error {
	p.status = StatusReady
	return nil
}

// Shutdown shuts down the parser adapter.
func (p *ParserAdapter) Shutdown(ctx context.Context) error {
	p.status = StatusShutdown
	return nil
}

// GetMetrics returns parser metrics.
func (p *ParserAdapter) GetMetrics() interface{} {
	// TODO: Implement parser metrics collection
	return map[string]interface{}{
		"status": p.status,
		"config": p.config,
	}
}

// GetStatus returns the parser adapter status.
func (p *ParserAdapter) GetStatus() ComponentStatus {
	return p.status
}

// PlannerAdapter adapts planner.ExecutionPlanner to PipelineComponent.
type PlannerAdapter struct {
	planner *planner.ExecutionPlanner
	status  ComponentStatus
}

// Name returns the planner component name.
func (p *PlannerAdapter) Name() string { return ComponentPlanner }

// Initialize initializes the planner adapter.
func (p *PlannerAdapter) Initialize(ctx context.Context, eventBus events.EventBus) error {
	p.status = StatusReady
	return nil
}

// Shutdown shuts down the planner adapter.
func (p *PlannerAdapter) Shutdown(ctx context.Context) error {
	p.status = StatusShutdown
	return nil
}

// GetMetrics returns planner metrics.
func (p *PlannerAdapter) GetMetrics() interface{} {
	// TODO: Access ExecutionPlanner internal metrics
	return map[string]interface{}{
		"status": p.status,
		"type":   "planner",
	}
}

// GetStatus returns the planner adapter status.
func (p *PlannerAdapter) GetStatus() ComponentStatus {
	return p.status
}

// ExecutorAdapter adapts executor.Executor to PipelineComponent.
type ExecutorAdapter struct {
	executor executor.Executor
	status   ComponentStatus
}

// Name returns the executor component name.
func (e *ExecutorAdapter) Name() string { return ComponentExecutor }

// Initialize initializes the executor adapter.
func (e *ExecutorAdapter) Initialize(ctx context.Context, eventBus events.EventBus) error {
	e.status = StatusReady
	return nil
}

// Shutdown shuts down the executor adapter.
func (e *ExecutorAdapter) Shutdown(ctx context.Context) error {
	e.status = StatusShutdown
	return nil
}

// GetMetrics returns executor metrics.
func (e *ExecutorAdapter) GetMetrics() interface{} {
	return e.executor.GetMetrics()
}

// GetStatus returns the executor adapter status.
func (e *ExecutorAdapter) GetStatus() ComponentStatus {
	return e.status
}

// EmitterAdapter adapts emitter.Emitter to PipelineComponent.
type EmitterAdapter struct {
	emitter emitter.Emitter
	status  ComponentStatus
}

// Name returns the emitter component name.
func (e *EmitterAdapter) Name() string { return ComponentEmitter }

// Initialize initializes the emitter adapter.
func (e *EmitterAdapter) Initialize(ctx context.Context, eventBus events.EventBus) error {
	e.status = StatusReady
	return nil
}

// Shutdown shuts down the emitter adapter.
func (e *EmitterAdapter) Shutdown(ctx context.Context) error {
	e.status = StatusShutdown
	return nil
}

// GetMetrics returns emitter metrics.
func (e *EmitterAdapter) GetMetrics() interface{} {
	return e.emitter.GetMetrics()
}

// GetStatus returns the emitter adapter status.
func (e *EmitterAdapter) GetStatus() ComponentStatus {
	return e.status
}
