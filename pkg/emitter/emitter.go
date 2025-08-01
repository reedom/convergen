package emitter

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
	ErrExecutionResultsNil    = errors.New("execution results cannot be nil")
	ErrGeneratedCodeNil       = errors.New("generated code cannot be nil")
	ErrMethodGenerationErrors = errors.New("method generation errors")
)

// Config defines configuration parameters for the code generation engine.
type Config struct {
	// Output preferences
	PreferCompositeLiterals bool   `json:"prefer_composite_literals"`
	MaxFieldsForComposite   int    `json:"max_fields_for_composite"`
	IndentStyle             string `json:"indent_style"`
	LineWidth               int    `json:"line_width"`

	// Optimization settings
	OptimizationLevel     OptimizationLevel `json:"optimization_level"`
	EnableDeadCodeElim    bool              `json:"enable_dead_code_elim"`
	EnableVarOptimization bool              `json:"enable_var_optimization"`
	EnableImportOpt       bool              `json:"enable_import_opt"`

	// Template settings
	CustomTemplates map[string]string `json:"custom_templates"`
	DefaultTemplate string            `json:"default_template"`

	// Performance settings
	EnableConcurrentGen  bool          `json:"enable_concurrent_gen"`
	MaxConcurrentMethods int           `json:"max_concurrent_methods"`
	GenerationTimeout    time.Duration `json:"generation_timeout"`

	// Validation settings
	EnableSyntaxValidation   bool `json:"enable_syntax_validation"`
	EnableSemanticValidation bool `json:"enable_semantic_validation"`
	StrictMode               bool `json:"strict_mode"`

	// Metrics and monitoring
	EnableMetrics   bool `json:"enable_metrics"`
	EnableProfiling bool `json:"enable_profiling"`
	DebugMode       bool `json:"debug_mode"`
}

// DefaultConfig returns sensible default configuration.
func DefaultConfig() *Config {
	return &Config{
		PreferCompositeLiterals:  true,
		MaxFieldsForComposite:    5,
		IndentStyle:              "\t",
		LineWidth:                120,
		OptimizationLevel:        OptimizationBasic,
		EnableDeadCodeElim:       true,
		EnableVarOptimization:    true,
		EnableImportOpt:          true,
		CustomTemplates:          make(map[string]string),
		DefaultTemplate:          "standard",
		EnableConcurrentGen:      true,
		MaxConcurrentMethods:     8,
		GenerationTimeout:        30 * time.Second,
		EnableSyntaxValidation:   true,
		EnableSemanticValidation: false,
		StrictMode:               false,
		EnableMetrics:            true,
		EnableProfiling:          false,
		DebugMode:                false,
	}
}

// Emitter coordinates the code generation pipeline with event-driven architecture.
type Emitter interface {
	// GenerateCode generates complete Go code from execution results
	GenerateCode(ctx context.Context, results *domain.ExecutionResults) (*GeneratedCode, error)

	// GenerateMethod generates code for a single method
	GenerateMethod(ctx context.Context, method *domain.MethodResult) (*MethodCode, error)

	// OptimizeOutput applies global optimizations to generated code
	OptimizeOutput(ctx context.Context, code *GeneratedCode) (*GeneratedCode, error)

	// GetMetrics returns current emitter metrics
	GetMetrics() *Metrics

	// Shutdown gracefully shuts down the emitter
	Shutdown(ctx context.Context) error
}

// ConcreteEmitter implements the Emitter interface.
type ConcreteEmitter struct {
	config   *Config
	logger   *zap.Logger
	eventBus events.EventBus

	// Core components
	codeGen     CodeGenerator
	outputStrat OutputStrategy
	formatMgr   FormatManager
	importMgr   ImportManager
	templateSys TemplateSystem
	optimizer   CodeOptimizer

	// State management
	metrics  *Metrics
	shutdown chan struct{}
	wg       sync.WaitGroup
	mutex    sync.RWMutex
}

// NewEmitter creates a new code generation engine.
func NewEmitter(logger *zap.Logger, eventBus events.EventBus, config *Config) Emitter {
	if config == nil {
		config = DefaultConfig()
	}

	metrics := NewMetrics()

	emitter := &ConcreteEmitter{
		config:   config,
		logger:   logger,
		eventBus: eventBus,
		metrics:  metrics,
		shutdown: make(chan struct{}),
	}

	// Initialize components
	emitter.codeGen = NewCodeGenerator(config, logger, metrics)
	emitter.outputStrat = NewOutputStrategy(config, logger)
	emitter.formatMgr = NewFormatManager(config, logger)
	emitter.importMgr = NewImportManager(config, logger)
	emitter.templateSys = NewTemplateSystem()
	emitter.optimizer = NewCodeOptimizer(config, logger, metrics)

	// Register custom templates if provided
	for name, template := range config.CustomTemplates {
		if err := emitter.templateSys.RegisterTemplate(name, template); err != nil {
			logger.Warn("failed to register custom template",
				zap.String("name", name),
				zap.Error(err))
		}
	}

	logger.Info("emitter initialized",
		zap.Bool("concurrent_generation", config.EnableConcurrentGen),
		zap.Int("max_concurrent_methods", config.MaxConcurrentMethods),
		zap.String("optimization_level", config.OptimizationLevel.String()))

	return emitter
}

// GenerateCode generates complete Go code from execution results.
func (e *ConcreteEmitter) GenerateCode(ctx context.Context, results *domain.ExecutionResults) (*GeneratedCode, error) {
	if results == nil {
		return nil, ErrExecutionResultsNil
	}

	e.logger.Info("starting code generation",
		zap.Int("methods", len(results.Methods)),
		zap.String("package", results.PackageName))

	startTime := time.Now()
	generatedCode := &GeneratedCode{
		PackageName: results.PackageName,
		BaseCode:    results.BaseCode,
		Methods:     make([]*MethodCode, 0, len(results.Methods)),
		Metadata: &GenerationMetadata{
			GenerationTime: startTime,
			EmitterVersion: "2.0.0",
			ConfigHash:     e.calculateConfigHash(),
		},
		Metrics: NewGenerationMetrics(),
	}

	// Emit generation started event
	if err := e.emitEvent(ctx, "emit.started", map[string]interface{}{
		"package":    results.PackageName,
		"methods":    len(results.Methods),
		"start_time": startTime,
	}); err != nil {
		e.logger.Warn("failed to emit generation started event", zap.Error(err))
	}

	// Generate methods concurrently or sequentially based on configuration
	var methodCodes []*MethodCode

	var err error

	if e.config.EnableConcurrentGen && len(results.Methods) > 1 {
		methodCodes, err = e.generateMethodsConcurrently(ctx, results.Methods)
	} else {
		methodCodes, err = e.generateMethodsSequentially(ctx, results.Methods)
	}

	if err != nil {
		return nil, fmt.Errorf("method generation failed: %w", err)
	}

	generatedCode.Methods = methodCodes

	// Analyze and generate imports
	importAnalysis, err := e.importMgr.AnalyzeImports(ctx, generatedCode)
	if err != nil {
		return nil, fmt.Errorf("import analysis failed: %w", err)
	}

	importDecl, err := e.importMgr.GenerateImports(ctx, importAnalysis)
	if err != nil {
		return nil, fmt.Errorf("import generation failed: %w", err)
	}

	generatedCode.Imports = importDecl

	// Apply optimizations if enabled
	if e.config.OptimizationLevel > OptimizationNone {
		optimizedCode, err := e.optimizer.OptimizeCode(ctx, generatedCode)
		if err != nil {
			e.logger.Warn("optimization failed", zap.Error(err))
		} else {
			generatedCode = optimizedCode
		}
	}

	// Format the final code
	formattedCode, err := e.formatMgr.FormatCode(ctx, generatedCode)
	if err != nil {
		return nil, fmt.Errorf("code formatting failed: %w", err)
	}

	generatedCode = formattedCode

	// Finalize generation metrics
	generatedCode.Metadata.CompletionTime = time.Now()
	generatedCode.Metadata.GenerationDuration = generatedCode.Metadata.CompletionTime.Sub(startTime)
	generatedCode.Metrics.TotalGenerationTime = generatedCode.Metadata.GenerationDuration
	generatedCode.Metrics.MethodsGenerated = len(methodCodes)

	// Update global metrics
	var firstMethod *MethodCode
	if len(methodCodes) > 0 {
		firstMethod = methodCodes[0]
	}

	e.metrics.RecordGeneration(firstMethod, results.PackageName, methodCodes)

	// Emit generation completed event
	if err := e.emitEvent(ctx, "emit.completed", map[string]interface{}{
		"package":            results.PackageName,
		"methods_generated":  len(methodCodes),
		"generation_time":    generatedCode.Metadata.GenerationDuration.Milliseconds(),
		"lines_generated":    generatedCode.Metrics.LinesGenerated,
		"optimization_level": e.config.OptimizationLevel.String(),
	}); err != nil {
		e.logger.Warn("failed to emit generation completed event", zap.Error(err))
	}

	e.logger.Info("code generation completed",
		zap.Int("methods", len(methodCodes)),
		zap.Duration("duration", generatedCode.Metadata.GenerationDuration),
		zap.Int("lines", generatedCode.Metrics.LinesGenerated),
		zap.Int("imports", len(generatedCode.Imports.Imports)))

	return generatedCode, nil
}

// GenerateMethod generates code for a single method.
func (e *ConcreteEmitter) GenerateMethod(ctx context.Context, method *domain.MethodResult) (*MethodCode, error) {
	if method == nil {
		return nil, ErrMethodResultNil
	}

	e.logger.Debug("generating method code",
		zap.String("method", method.Method.Name))

	// Generate method code using the code generator
	methodCode, err := e.codeGen.GenerateMethodCode(ctx, method)
	if err != nil {
		return nil, fmt.Errorf("method code generation failed: %w", err)
	}

	// Apply method-level optimizations
	if e.config.OptimizationLevel > OptimizationNone {
		if err := e.optimizer.OptimizeMethodCode(methodCode); err != nil {
			e.logger.Warn("method optimization failed",
				zap.String("method", method.Method.Name),
				zap.Error(err))
		}
	}

	e.logger.Debug("method code generated",
		zap.String("method", method.Method.Name),
		zap.Int("lines", methodCode.Complexity.LinesGenerated))

	return methodCode, nil
}

// OptimizeOutput applies global optimizations to generated code.
func (e *ConcreteEmitter) OptimizeOutput(ctx context.Context, code *GeneratedCode) (*GeneratedCode, error) {
	if code == nil {
		return nil, ErrGeneratedCodeNil
	}

	e.logger.Debug("optimizing generated code",
		zap.String("optimization_level", e.config.OptimizationLevel.String()))

	optimizedCode, err := e.optimizer.OptimizeCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("code optimization failed: %w", err)
	}

	return optimizedCode, nil
}

// GetMetrics returns current emitter metrics.
func (e *ConcreteEmitter) GetMetrics() *Metrics {
	return e.metrics.GetSnapshot()
}

// Shutdown gracefully shuts down the emitter.
func (e *ConcreteEmitter) Shutdown(ctx context.Context) error {
	e.logger.Info("shutting down emitter")

	close(e.shutdown)

	// Shutdown components
	if err := e.codeGen.Shutdown(ctx); err != nil {
		e.logger.Warn("code generator shutdown error", zap.Error(err))
	}

	if err := e.optimizer.Shutdown(ctx); err != nil {
		e.logger.Warn("optimizer shutdown error", zap.Error(err))
	}

	// Wait for background tasks
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		e.logger.Info("emitter shutdown completed")
		return nil
	case <-ctx.Done():
		e.logger.Warn("emitter shutdown timed out")
		return fmt.Errorf("emitter shutdown context cancelled: %w", ctx.Err())
	}
}

// Helper methods

func (e *ConcreteEmitter) generateMethodsConcurrently(ctx context.Context, methods []*domain.MethodResult) ([]*MethodCode, error) {
	methodCount := len(methods)
	results := make([]*MethodCode, methodCount)
	errors := make([]error, methodCount)

	// Limit concurrency
	maxConcurrency := e.config.MaxConcurrentMethods
	if maxConcurrency <= 0 || maxConcurrency > methodCount {
		maxConcurrency = methodCount
	}

	semaphore := make(chan struct{}, maxConcurrency)

	var wg sync.WaitGroup

	for i, method := range methods {
		wg.Add(1)

		go func(index int, m *domain.MethodResult) {
			defer wg.Done()

			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			methodCode, err := e.GenerateMethod(ctx, m)
			results[index] = methodCode
			errors[index] = err
		}(i, method)
	}

	wg.Wait()

	// Check for errors
	var generationErrors []error

	var validResults []*MethodCode

	for i, err := range errors {
		if err != nil {
			generationErrors = append(generationErrors, fmt.Errorf("method %s: %w", methods[i].Method.Name, err))
		} else {
			validResults = append(validResults, results[i])
		}
	}

	if len(generationErrors) > 0 {
		return validResults, fmt.Errorf("%w: %v", ErrMethodGenerationErrors, generationErrors)
	}

	return validResults, nil
}

func (e *ConcreteEmitter) generateMethodsSequentially(ctx context.Context, methods []*domain.MethodResult) ([]*MethodCode, error) {
	results := make([]*MethodCode, 0, len(methods))

	for _, method := range methods {
		methodCode, err := e.GenerateMethod(ctx, method)
		if err != nil {
			return results, fmt.Errorf("method %s generation failed: %w", method.Method.Name, err)
		}

		results = append(results, methodCode)
	}

	return results, nil
}

func (e *ConcreteEmitter) calculateConfigHash() string {
	// Simple hash calculation for configuration
	// In practice, this would be more sophisticated
	return fmt.Sprintf("%x", int(e.config.OptimizationLevel)*17+
		e.config.MaxFieldsForComposite*31+
		len(e.config.CustomTemplates)*47)
}

func (e *ConcreteEmitter) emitEvent(ctx context.Context, eventType string, data map[string]interface{}) error {
	if e.eventBus == nil {
		return nil // No event bus configured
	}

	event := events.NewBaseEvent(eventType, ctx)
	for key, value := range data {
		event.WithMetadata(key, value)
	}

	if err := e.eventBus.Publish(event); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// SetEventBus sets the event bus for this emitter.
func (e *ConcreteEmitter) SetEventBus(eventBus events.EventBus) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.eventBus = eventBus
}
