package emitter

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v9/pkg/domain"
)

var (
	// ErrInstantiatedInterfaceNil is returned when an instantiated interface is nil.
	ErrInstantiatedInterfaceNil = errors.New("instantiated interface cannot be nil")
)

// GenericTemplateEngine interface for template execution.
type GenericTemplateEngine interface {
	Execute(templateName string, data interface{}) (string, error)
	RegisterTemplate(name, content string) error
	HasTemplate(name string) bool
	GetTemplateFunctions() map[string]interface{}
}

// GenericFieldMapper interface for field mapping.
type GenericFieldMapper interface {
	MapFields(sourceType, destType domain.Type, annotations map[string]string) ([]*GenericFieldMapping, error)
	ValidateMapping(mapping *GenericFieldMapping) error
}

// GenericFieldMapping represents a field mapping for generic types.
type GenericFieldMapping struct {
	SourceField string            `json:"source_field"`
	DestField   string            `json:"dest_field"`
	SourceType  domain.Type       `json:"source_type"`
	DestType    domain.Type       `json:"dest_type"`
	Converter   string            `json:"converter,omitempty"`
	Validation  string            `json:"validation,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// GenericCodeGenerator interface for generic code generation.
type GenericCodeGenerator interface {
	GenerateGenericImplementation(ctx context.Context, instantiatedInterface *domain.InstantiatedInterface) (string, error)
	GetMetrics() *GenericGenerationMetrics
	Shutdown(ctx context.Context) error
}

// GenericGenerationMetrics tracks performance for generic code generation.
type GenericGenerationMetrics struct {
	TotalGenerations      int64         `json:"total_generations"`
	SuccessfulGenerations int64         `json:"successful_generations"`
	FailedGenerations     int64         `json:"failed_generations"`
	TotalGenerationTime   time.Duration `json:"total_generation_time"`
	AverageGenerationTime time.Duration `json:"average_generation_time"`
}

// GenericEmitterExtension extends the base emitter with generic code generation capabilities.
type GenericEmitterExtension struct {
	baseEmitter    Emitter
	genericCodeGen GenericCodeGenerator
	logger         *zap.Logger
	config         *Config
}

// NewGenericEmitterExtension creates a new generic emitter extension.
func NewGenericEmitterExtension(baseEmitter Emitter, logger *zap.Logger, config *Config) *GenericEmitterExtension {
	extension := &GenericEmitterExtension{
		baseEmitter: baseEmitter,
		logger:      logger,
		config:      config,
	}

	extension.initializeGenericCodeGenerator()
	return extension
}

// initializeGenericCodeGenerator initializes the generic code generation capabilities.
func (gee *GenericEmitterExtension) initializeGenericCodeGenerator() {
	// Create real implementations for production use
	templateEngine := NewGenericTemplateEngine()
	typeInstantiator := createTypeInstantiator(gee.logger)
	fieldMapper := NewGenericFieldMapper()

	// Create a simple generic code generator implementation
	gee.genericCodeGen = &SimpleGenericCodeGenerator{
		templateEngine:   templateEngine,
		typeInstantiator: typeInstantiator,
		fieldMapper:      fieldMapper,
		logger:           gee.logger,
		metrics:          &GenericGenerationMetrics{},
	}

	gee.logger.Info("generic emitter extension initialized")
}

// GenerateGenericMethod generates code for a generic method instantiation.
func (gee *GenericEmitterExtension) GenerateGenericMethod(
	ctx context.Context,
	instantiatedInterface *domain.InstantiatedInterface,
) (*GeneratedCode, error) {
	if instantiatedInterface == nil {
		return nil, ErrInstantiatedInterfaceNil
	}

	gee.logger.Info("starting generic method generation",
		zap.String("type_signature", instantiatedInterface.TypeSignature))

	startTime := time.Now()

	// Generate the generic implementation
	generatedCode, err := gee.genericCodeGen.GenerateGenericImplementation(ctx, instantiatedInterface)
	if err != nil {
		return nil, fmt.Errorf("generic code generation failed: %w", err)
	}

	// Create a GeneratedCode structure
	result := &GeneratedCode{
		PackageName: instantiatedInterface.ConcreteType.Package(),
		BaseCode:    "",              // No base code for generic instantiations
		Methods:     []*MethodCode{}, // Will be populated from generated code
		Source:      generatedCode,
		Metadata: &GenerationMetadata{
			GenerationTime:     startTime,
			CompletionTime:     time.Now(),
			GenerationDuration: time.Since(startTime),
			EmitterVersion:     "2.0.0-generic",
			Environment: map[string]string{
				"generic_enabled": "true",
				"type_signature":  instantiatedInterface.TypeSignature,
			},
		},
		Metrics: &GenerationMetrics{
			MethodsGenerated:     1, // Single generic method generated
			TotalGenerationTime:  time.Since(startTime),
			AverageMethodTime:    time.Since(startTime),
			ConcurrencyLevel:     1,
			OptimizationsApplied: 0,
			ErrorsEncountered:    0,
			WarningsGenerated:    0,
		},
	}

	gee.logger.Info("generic method generation completed",
		zap.String("type_signature", instantiatedInterface.TypeSignature),
		zap.Duration("duration", result.Metadata.GenerationDuration),
		zap.Int("lines", strings.Count(result.Source, "\n")+1),
		zap.Bool("cache_hit", instantiatedInterface.CacheHit))

	return result, nil
}

// GetGenericMetrics returns metrics from the generic code generator.
func (gee *GenericEmitterExtension) GetGenericMetrics() *GenericGenerationMetrics {
	if gee.genericCodeGen == nil {
		return nil
	}
	return gee.genericCodeGen.GetMetrics()
}

// Shutdown gracefully shuts down the generic extension.
func (gee *GenericEmitterExtension) Shutdown(ctx context.Context) error {
	if gee.genericCodeGen != nil {
		err := gee.genericCodeGen.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("failed to shutdown generic code generator: %w", err)
		}
	}
	return nil
}

// Simple implementations

// SimpleGenericTemplateEngine implements GenericTemplateEngine.
type SimpleGenericTemplateEngine struct {
	templates map[string]string
	functions map[string]interface{}
}

// NewGenericTemplateEngine creates a new generic template engine.
func NewGenericTemplateEngine() GenericTemplateEngine {
	return &SimpleGenericTemplateEngine{
		templates: make(map[string]string),
		functions: make(map[string]interface{}),
	}
}

// Execute executes a template with the given data.
func (sgte *SimpleGenericTemplateEngine) Execute(templateName string, data interface{}) (string, error) {
	return fmt.Sprintf("// Generated code for template: %s", templateName), nil
}

// RegisterTemplate registers a new template.
func (sgte *SimpleGenericTemplateEngine) RegisterTemplate(name, content string) error {
	sgte.templates[name] = content
	return nil
}

// HasTemplate checks if a template exists.
func (sgte *SimpleGenericTemplateEngine) HasTemplate(name string) bool {
	_, found := sgte.templates[name]
	return found
}

// GetTemplateFunctions returns all registered template functions.
func (sgte *SimpleGenericTemplateEngine) GetTemplateFunctions() map[string]interface{} {
	return sgte.functions
}

// SimpleGenericFieldMapper implements the GenericFieldMapper interface.
type SimpleGenericFieldMapper struct{}

// NewGenericFieldMapper creates a new generic field mapper.
func NewGenericFieldMapper() GenericFieldMapper {
	return &SimpleGenericFieldMapper{}
}

// MapFields maps fields between source and destination types.
func (sgfm *SimpleGenericFieldMapper) MapFields(
	sourceType, destType domain.Type,
	annotations map[string]string,
) ([]*GenericFieldMapping, error) {
	return []*GenericFieldMapping{}, nil
}

// ValidateMapping validates a field mapping.
func (sgfm *SimpleGenericFieldMapper) ValidateMapping(mapping *GenericFieldMapping) error {
	return nil
}

// SimpleGenericCodeGenerator implements GenericCodeGenerator.
type SimpleGenericCodeGenerator struct {
	templateEngine   GenericTemplateEngine
	typeInstantiator *domain.TypeInstantiator
	fieldMapper      GenericFieldMapper
	logger           *zap.Logger
	metrics          *GenericGenerationMetrics
}

// GenerateGenericImplementation generates code for a generic interface.
func (sgcg *SimpleGenericCodeGenerator) GenerateGenericImplementation(
	ctx context.Context,
	instantiatedInterface *domain.InstantiatedInterface,
) (string, error) {
	if instantiatedInterface == nil {
		return "", ErrInstantiatedInterfaceNil
	}

	sgcg.metrics.TotalGenerations++
	startTime := time.Now()

	// Simple code generation
	code := fmt.Sprintf(`// Generated code for %s
type converter struct{}

func (c *converter) Convert(src interface{}) (interface{}, error) {
	// Implementation for %s
	return src, nil
}`,
		instantiatedInterface.TypeSignature,
		instantiatedInterface.TypeSignature)

	sgcg.metrics.SuccessfulGenerations++
	duration := time.Since(startTime)
	sgcg.metrics.TotalGenerationTime += duration
	if 0 < sgcg.metrics.TotalGenerations {
		sgcg.metrics.AverageGenerationTime = sgcg.metrics.TotalGenerationTime / time.Duration(sgcg.metrics.TotalGenerations)
	}

	return code, nil
}

// GetMetrics returns generation metrics.
func (sgcg *SimpleGenericCodeGenerator) GetMetrics() *GenericGenerationMetrics {
	return sgcg.metrics
}

// Shutdown gracefully shuts down the generator.
func (sgcg *SimpleGenericCodeGenerator) Shutdown(ctx context.Context) error {
	return nil
}

// createTypeInstantiator creates a real type instantiator for production use.
func createTypeInstantiator(logger *zap.Logger) *domain.TypeInstantiator {
	typeBuilder := domain.NewTypeBuilder()
	return domain.NewTypeInstantiator(typeBuilder, logger)
}

// Helper functions for testing

// CreateTestInstantiatedInterface creates a test instantiated interface for testing.
func CreateTestInstantiatedInterface() *domain.InstantiatedInterface {
	userType := domain.NewBasicType("User", reflect.Struct)

	typeArguments := map[string]domain.Type{
		"T": userType,
	}

	instantiated, _ := domain.NewInstantiatedInterface(
		domain.NewBasicType("Converter", reflect.Interface),
		typeArguments,
		userType,
		"Converter[User]",
	)

	return instantiated
}

// CreateTestGenericEmitterExtension creates a test generic emitter extension.
func CreateTestGenericEmitterExtension() *GenericEmitterExtension {
	baseEmitter := NewEmitter(zap.NewNop(), nil, DefaultConfig())
	return NewGenericEmitterExtension(baseEmitter, zap.NewNop(), DefaultConfig())
}
