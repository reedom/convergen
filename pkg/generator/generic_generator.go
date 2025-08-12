package generator

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v9/pkg/builder"
	"github.com/reedom/convergen/v9/pkg/domain"
)

// Package-level error definitions for static error handling.
var (
	ErrNotStructType     = errors.New("cannot extract fields from non-struct type")
	ErrInvalidStructType = errors.New("type is not a proper struct type")
)

// ImportInfo represents import information without emitter dependency.
type ImportInfo struct {
	Path     string `json:"path"`
	Alias    string `json:"alias"`
	Used     bool   `json:"used"`
	Standard bool   `json:"standard"`
	Local    bool   `json:"local"`
	Required bool   `json:"required"`
}

// Static errors for err113 compliance.
var (
	ErrInstantiatedInterfaceNil    = errors.New("instantiated interface cannot be nil")
	ErrTemplateDataNil             = errors.New("template data cannot be nil")
	ErrGenericCodeGenerationFailed = errors.New("generic code generation failed")
	ErrTypeSubstitutionFailed      = errors.New("type substitution failed in template")
	ErrTemplateExecutionFailed     = errors.New("template execution failed")
	ErrTypeCannotBeNil             = errors.New("type cannot be nil")
)

// GenericCodeGenerator handles code generation for generic interfaces.
// It integrates with the existing template system and type substitution engine.
type GenericCodeGenerator struct {
	templateEngine   TemplateEngine
	typeInstantiator *domain.TypeInstantiator
	fieldMapper      FieldMapper
	logger           *zap.Logger

	// Configuration
	config *GenericGeneratorConfig

	// Performance tracking
	metrics *GenericGenerationMetrics
}

// GenericGeneratorConfig configures the generic code generator.
type GenericGeneratorConfig struct {
	EnableOptimization   bool          `json:"enable_optimization"`
	MaxTemplateDepth     int           `json:"max_template_depth"`
	EnableCaching        bool          `json:"enable_caching"`
	GenerationTimeout    time.Duration `json:"generation_timeout"`
	EnableTypeValidation bool          `json:"enable_type_validation"`
	PreferCompactOutput  bool          `json:"prefer_compact_output"`
	EnableErrorWrapping  bool          `json:"enable_error_wrapping"`
	DebugMode            bool          `json:"debug_mode"`
}

// DefaultGenericGeneratorConfig returns default configuration.
func DefaultGenericGeneratorConfig() *GenericGeneratorConfig {
	return &GenericGeneratorConfig{
		EnableOptimization:   true,
		MaxTemplateDepth:     10,
		EnableCaching:        true,
		GenerationTimeout:    30 * time.Second,
		EnableTypeValidation: true,
		PreferCompactOutput:  false,
		EnableErrorWrapping:  true,
		DebugMode:            false,
	}
}

// GenericGenerationMetrics tracks performance for generic code generation.
type GenericGenerationMetrics struct {
	TotalGenerations         int64         `json:"total_generations"`
	SuccessfulGenerations    int64         `json:"successful_generations"`
	FailedGenerations        int64         `json:"failed_generations"`
	TotalGenerationTime      time.Duration `json:"total_generation_time"`
	AverageGenerationTime    time.Duration `json:"average_generation_time"`
	TypeSubstitutions        int64         `json:"type_substitutions"`
	TemplateExecutions       int64         `json:"template_executions"`
	CacheHits                int64         `json:"cache_hits"`
	CacheMisses              int64         `json:"cache_misses"`
	ErrorsEncountered        int64         `json:"errors_encountered"`
	PerformanceOptimizations int64         `json:"performance_optimizations"`
}

// NewGenericGenerationMetrics creates a new metrics instance.
func NewGenericGenerationMetrics() *GenericGenerationMetrics {
	return &GenericGenerationMetrics{}
}

// TemplateEngine defines the interface for template execution.
type TemplateEngine interface {
	Execute(templateName string, data interface{}) (string, error)
	RegisterTemplate(name, content string) error
	HasTemplate(name string) bool
	GetTemplateFunctions() map[string]interface{}
}

// FieldMapper defines the interface for field mapping logic.
type FieldMapper interface {
	MapFields(sourceType, destType domain.Type, annotations map[string]string) ([]*FieldMapping, error)
	ValidateMapping(mapping *FieldMapping) error
}

// FieldMapping represents a field mapping between source and destination.
type FieldMapping struct {
	SourceField string            `json:"source_field"`
	DestField   string            `json:"dest_field"`
	SourceType  domain.Type       `json:"source_type"`
	DestType    domain.Type       `json:"dest_type"`
	Converter   string            `json:"converter,omitempty"`
	Validation  string            `json:"validation,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// NewGenericCodeGenerator creates a new generic code generator.
func NewGenericCodeGenerator(
	templateEngine TemplateEngine,
	typeInstantiator *domain.TypeInstantiator,
	fieldMapper FieldMapper,
	logger *zap.Logger,
	config *GenericGeneratorConfig,
) *GenericCodeGenerator {
	if config == nil {
		config = DefaultGenericGeneratorConfig()
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	generator := &GenericCodeGenerator{
		templateEngine:   templateEngine,
		typeInstantiator: typeInstantiator,
		fieldMapper:      fieldMapper,
		logger:           logger,
		config:           config,
		metrics:          NewGenericGenerationMetrics(),
	}

	// Register generic template functions
	generator.registerGenericTemplateFunctions()

	logger.Info("generic code generator initialized",
		zap.Bool("optimization_enabled", config.EnableOptimization),
		zap.Bool("caching_enabled", config.EnableCaching),
		zap.Duration("timeout", config.GenerationTimeout))

	return generator
}

// GenerateGenericImplementation generates concrete method implementations from generic templates.
func (gcg *GenericCodeGenerator) GenerateGenericImplementation(
	ctx context.Context,
	instantiatedInterface *domain.InstantiatedInterface,
) (string, error) {
	if instantiatedInterface == nil {
		return "", ErrInstantiatedInterfaceNil
	}

	startTime := time.Now()
	gcg.metrics.TotalGenerations++

	gcg.logger.Info("starting generic code generation",
		zap.String("type_signature", instantiatedInterface.TypeSignature),
		zap.Int("type_arguments", len(instantiatedInterface.TypeArguments)))

	// Create generation context with timeout
	genCtx, cancel := context.WithTimeout(ctx, gcg.config.GenerationTimeout)
	defer cancel()

	// Create template data with generic-specific information
	templateData := gcg.createGenericTemplateData(instantiatedInterface)

	// Generate method implementations for each method in the interface
	generatedMethods := make([]string, 0, len(templateData.Methods))
	for _, method := range templateData.Methods {
		methodCode, err := gcg.generateGenericMethod(genCtx, method, templateData)
		if err != nil {
			gcg.logger.Error("failed to generate method",
				zap.String("method", method.Name),
				zap.Error(err))
			gcg.metrics.ErrorsEncountered++
			continue
		}

		generatedMethods = append(generatedMethods, methodCode)
	}

	if len(generatedMethods) == 0 {
		gcg.metrics.FailedGenerations++
		return "", fmt.Errorf("%w: no methods generated successfully", ErrGenericCodeGenerationFailed)
	}

	// Combine all method implementations
	finalCode := strings.Join(generatedMethods, "\n\n")

	// Apply optimizations if enabled
	if gcg.config.EnableOptimization {
		finalCode = gcg.optimizeGeneratedCode(finalCode)
		gcg.metrics.PerformanceOptimizations++
	}

	// Update metrics
	generationTime := time.Since(startTime)
	gcg.metrics.SuccessfulGenerations++
	gcg.metrics.TotalGenerationTime += generationTime
	gcg.metrics.AverageGenerationTime = gcg.metrics.TotalGenerationTime / time.Duration(gcg.metrics.TotalGenerations)

	gcg.logger.Info("generic code generation completed",
		zap.String("type_signature", instantiatedInterface.TypeSignature),
		zap.Int("methods_generated", len(generatedMethods)),
		zap.Duration("generation_time", generationTime),
		zap.Int("lines_generated", strings.Count(finalCode, "\n")+1))

	return finalCode, nil
}

// createGenericTemplateData creates template data with generic-specific information.
func (gcg *GenericCodeGenerator) createGenericTemplateData(
	instantiatedInterface *domain.InstantiatedInterface,
) *GenericTemplateData {
	gcg.logger.Debug("creating generic template data",
		zap.String("type_signature", instantiatedInterface.TypeSignature))

	// Extract methods from the concrete type
	methods := gcg.extractMethodsFromType(instantiatedInterface.ConcreteType)

	// Build type parameter mapping
	typeParams := make([]TypeParam, 0, len(instantiatedInterface.TypeArguments))
	typeArgs := make([]TypeArg, 0, len(instantiatedInterface.TypeArguments))
	typeSubstitutions := make(map[string]TypeSubstitution)

	for paramName, concreteType := range instantiatedInterface.TypeArguments {
		typeParam := TypeParam{
			Name:       paramName,
			Constraint: "any", // Default constraint, could be extracted from source
		}

		typeArg := TypeArg{
			Name:        concreteType.Name(),
			Type:        concreteType,
			PackagePath: concreteType.Package(),
		}

		typeSubstitution := TypeSubstitution{
			ParameterName: paramName,
			ConcreteType:  concreteType,
			PackagePath:   concreteType.Package(),
		}

		typeParams = append(typeParams, typeParam)
		typeArgs = append(typeArgs, typeArg)
		typeSubstitutions[paramName] = typeSubstitution
	}

	// Create the generic template data
	templateData := &GenericTemplateData{
		BaseTemplateData: BaseTemplateData{
			Package:         instantiatedInterface.ConcreteType.Package(),
			Imports:         gcg.extractRequiredImports(instantiatedInterface),
			Metadata:        make(map[string]interface{}),
			HelperFunctions: gcg.getGenericHelperFunctions(),
		},

		// Generic-specific data
		TypeParameters:    typeParams,
		TypeArguments:     typeArgs,
		TypeSubstitutions: typeSubstitutions,
		IsGenericFlag:     true,
		Methods:           methods,
		SourceInterface:   instantiatedInterface.SourceInterface,
		ConcreteType:      instantiatedInterface.ConcreteType,
		TypeSignature:     instantiatedInterface.TypeSignature,
	}

	// Add generic-specific metadata
	templateData.Metadata["generation_time"] = time.Now()
	templateData.Metadata["type_signature"] = instantiatedInterface.TypeSignature
	templateData.Metadata["validation_result"] = instantiatedInterface.ValidationResult
	templateData.Metadata["cache_hit"] = instantiatedInterface.CacheHit

	return templateData
}

// generateGenericMethod generates code for a single generic method.
func (gcg *GenericCodeGenerator) generateGenericMethod(
	_ context.Context,
	method *MethodData,
	templateData *GenericTemplateData,
) (string, error) {
	gcg.logger.Debug("generating generic method",
		zap.String("method", method.Name))

	// Create method-specific template data
	methodTemplateData := &MethodTemplateData{
		GenericTemplateData: templateData,
		Method:              method,
		FieldMappings:       make([]*FieldMapping, 0),
	}

	// Generate field mappings if this is a conversion method
	if gcg.isConversionMethod(method) {
		fieldMappings, err := gcg.generateFieldMappings(method, templateData)
		if err != nil {
			return "", fmt.Errorf("failed to generate field mappings: %w", err)
		}
		methodTemplateData.FieldMappings = fieldMappings
	}

	// Select appropriate template based on method characteristics
	templateName := gcg.selectTemplate(method, templateData)

	gcg.logger.Debug("selected template for method",
		zap.String("method", method.Name),
		zap.String("template", templateName))

	// Execute the template
	gcg.metrics.TemplateExecutions++

	generatedCode, err := gcg.templateEngine.Execute(templateName, methodTemplateData)
	if err != nil {
		return "", fmt.Errorf("%w: template %s: %s", ErrTemplateExecutionFailed, templateName, err.Error())
	}

	// Apply type substitutions in the generated code
	substitutedCode := gcg.applyTypeSubstitutions(generatedCode, templateData.TypeSubstitutions)

	return substitutedCode, nil
}

// generateFieldMappings generates field mappings for conversion methods.
func (gcg *GenericCodeGenerator) generateFieldMappings(
	method *MethodData,
	templateData *GenericTemplateData,
) ([]*FieldMapping, error) {
	if len(method.Parameters) == 0 || method.ReturnType == nil {
		return []*FieldMapping{}, nil
	}

	sourceType := method.Parameters[0].Type
	destType := method.ReturnType

	// Apply type substitutions to source and destination types
	substitutedSourceType, err := gcg.substituteTypeInContext(sourceType, templateData.TypeSubstitutions)
	if err != nil {
		return nil, fmt.Errorf("failed to substitute source type: %w", err)
	}

	substitutedDestType, err := gcg.substituteTypeInContext(destType, templateData.TypeSubstitutions)
	if err != nil {
		return nil, fmt.Errorf("failed to substitute destination type: %w", err)
	}

	gcg.logger.Debug("generating field mappings for generic method",
		zap.String("method", method.Name),
		zap.String("source_type", substitutedSourceType.String()),
		zap.String("dest_type", substitutedDestType.String()))

	// Use the enhanced field mapping logic
	mappings, err := gcg.generateEnhancedFieldMappings(substitutedSourceType, substitutedDestType, method.Annotations)
	if err != nil {
		return nil, fmt.Errorf("field mapping failed: %w", err)
	}

	return mappings, nil
}

// substituteTypeInContext applies type substitutions to a type.
func (gcg *GenericCodeGenerator) substituteTypeInContext(
	typ domain.Type,
	substitutions map[string]TypeSubstitution,
) (domain.Type, error) {
	if typ == nil {
		return nil, ErrTypeCannotBeNil
	}

	// Check if this type needs substitution
	if substitution, found := substitutions[typ.Name()]; found {
		gcg.metrics.TypeSubstitutions++
		return substitution.ConcreteType, nil
	}

	// For composite types, recursively substitute components
	switch typ.Kind() {
	case domain.KindSlice:
		if sliceType, ok := typ.(*domain.SliceType); ok {
			substitutedElem, err := gcg.substituteTypeInContext(sliceType.Elem(), substitutions)
			if err != nil {
				return nil, err
			}
			return domain.NewSliceType(substitutedElem, sliceType.Package()), nil
		}
	case domain.KindPointer:
		if pointerType, ok := typ.(*domain.PointerType); ok {
			substitutedElem, err := gcg.substituteTypeInContext(pointerType.Elem(), substitutions)
			if err != nil {
				return nil, err
			}
			return domain.NewPointerType(substitutedElem, pointerType.Package()), nil
		}
	}

	// No substitution needed
	return typ, nil
}

// selectTemplate selects the appropriate template for a method.
func (gcg *GenericCodeGenerator) selectTemplate(method *MethodData, templateData *GenericTemplateData) string {
	// Check for custom template annotation
	if customTemplate, found := method.Annotations["template"]; found {
		if gcg.templateEngine.HasTemplate(customTemplate) {
			return customTemplate
		}
		gcg.logger.Warn("custom template not found, using default",
			zap.String("requested_template", customTemplate),
			zap.String("method", method.Name))
	}

	// Select template based on method characteristics
	if gcg.isConversionMethod(method) {
		if gcg.isComplexConversion(method, templateData) {
			return "generic_complex_conversion"
		}
		return "generic_simple_conversion"
	}

	if method.ReturnsError {
		return "generic_method_with_error"
	}

	return "generic_method_basic"
}

// isConversionMethod determines if a method is a conversion method.
func (gcg *GenericCodeGenerator) isConversionMethod(method *MethodData) bool {
	return len(method.Parameters) == 1 && method.ReturnType != nil
}

// isComplexConversion determines if a conversion is complex.
func (gcg *GenericCodeGenerator) isComplexConversion(method *MethodData, templateData *GenericTemplateData) bool {
	// Consider complex if there are custom converters or complex type mappings
	return len(method.Annotations) > 1 ||
		strings.Contains(method.Name, "Complex") ||
		len(templateData.TypeSubstitutions) > 2
}

// applyTypeSubstitutions applies type substitutions to generated code.
func (gcg *GenericCodeGenerator) applyTypeSubstitutions(
	code string,
	substitutions map[string]TypeSubstitution,
) string {
	result := code

	for paramName, substitution := range substitutions {
		// Replace type parameter references with concrete types
		placeholder := fmt.Sprintf("{{.%s}}", paramName)
		concreteTypeName := substitution.ConcreteType.Name()

		// Add package prefix if needed
		if substitution.PackagePath != "" && substitution.ConcreteType.Package() != "" {
			concreteTypeName = substitution.ConcreteType.Package() + "." + concreteTypeName
		}

		result = strings.ReplaceAll(result, placeholder, concreteTypeName)

		// Also replace direct type parameter references
		result = strings.ReplaceAll(result, paramName, concreteTypeName)
	}

	return result
}

// extractMethodsFromType extracts method information from a type.
func (gcg *GenericCodeGenerator) extractMethodsFromType(typ domain.Type) []*MethodData {
	// For interface types, extract method signatures
	// This is a simplified implementation - in practice would analyze the AST
	methods := []*MethodData{
		{
			Name: "Convert",
			Parameters: []*ParameterData{
				{
					Name: "src",
					Type: typ,
				},
			},
			ReturnType:   typ,
			ReturnsError: true,
			Annotations:  make(map[string]string),
		},
	}

	return methods
}

// extractRequiredImports extracts required imports from the instantiated interface.
func (gcg *GenericCodeGenerator) extractRequiredImports(
	instantiatedInterface *domain.InstantiatedInterface,
) []*ImportInfo {
	imports := make([]*ImportInfo, 0)

	// Add imports for type arguments
	for _, typeArg := range instantiatedInterface.TypeArguments {
		if typeArg.Package() != "" {
			imports = append(imports, &ImportInfo{
				Path:     typeArg.Package(),
				Standard: false,
				Local:    false,
				Required: true,
			})
		}
	}

	return imports
}

// optimizeGeneratedCode applies optimizations to generated code.
func (gcg *GenericCodeGenerator) optimizeGeneratedCode(
	code string,
) string {
	if !gcg.config.EnableOptimization {
		return code
	}

	optimized := code

	// Remove redundant type conversions
	optimized = gcg.removeRedundantConversions(optimized)

	// Optimize error handling patterns
	optimized = gcg.optimizeErrorHandling(optimized)

	// Compact whitespace if preferred
	if gcg.config.PreferCompactOutput {
		optimized = gcg.compactWhitespace(optimized)
	}

	gcg.logger.Debug("applied code optimizations",
		zap.Int("original_length", len(code)),
		zap.Int("optimized_length", len(optimized)))

	return optimized
}

// removeRedundantConversions removes redundant type conversions.
func (gcg *GenericCodeGenerator) removeRedundantConversions(code string) string {
	// Simple optimization: remove same-type conversions like Type(value) where value is already Type
	// This is a placeholder for more sophisticated analysis
	return code
}

// optimizeErrorHandling optimizes error handling patterns.
func (gcg *GenericCodeGenerator) optimizeErrorHandling(code string) string {
	// Optimize common error handling patterns
	// This is a placeholder for more sophisticated optimizations
	return code
}

// compactWhitespace compacts whitespace in generated code.
func (gcg *GenericCodeGenerator) compactWhitespace(code string) string {
	// Remove excessive blank lines
	lines := strings.Split(code, "\n")
	compacted := make([]string, 0, len(lines))

	previousBlank := false
	for _, line := range lines {
		isBlank := strings.TrimSpace(line) == ""
		if isBlank && previousBlank {
			continue // Skip consecutive blank lines
		}
		compacted = append(compacted, line)
		previousBlank = isBlank
	}

	return strings.Join(compacted, "\n")
}

// getGenericHelperFunctions returns template helper functions for generic generation.
func (gcg *GenericCodeGenerator) getGenericHelperFunctions() map[string]interface{} {
	return map[string]interface{}{
		"substituteType":      gcg.substituteTypeInTemplate,
		"isGenericType":       gcg.isGenericTypeInTemplate,
		"formatTypeParam":     gcg.formatTypeParamInTemplate,
		"generateTypeSwitch":  gcg.generateTypeSwitchInTemplate,
		"hasAnnotation":       gcg.hasAnnotationInTemplate,
		"getAnnotation":       gcg.getAnnotationInTemplate,
		"generateFieldAccess": gcg.generateFieldAccessInTemplate,
	}
}

// GetMetrics returns current generation metrics.
func (gcg *GenericCodeGenerator) GetMetrics() *GenericGenerationMetrics {
	return gcg.metrics
}

// ClearMetrics resets all metrics.
func (gcg *GenericCodeGenerator) ClearMetrics() {
	gcg.metrics = NewGenericGenerationMetrics()
}

// generateEnhancedFieldMappings uses the builder package's GenericFieldMapper for proper field mapping.
func (gcg *GenericCodeGenerator) generateEnhancedFieldMappings(
	sourceType, destType domain.Type,
	annotations map[string]string,
) ([]*FieldMapping, error) {
	// Create a GenericFieldMapper from the builder package
	genericMapper := builder.NewGenericFieldMapper(
		nil, // Use default base mapper
		nil, // Use default type substitution engine
		gcg.logger,
		nil, // Use default config
	)

	// Convert annotations to the format expected by the builder
	options := builder.DefaultFieldMappingOptions()
	if len(annotations) > 0 {
		options.CustomMappings = annotations
	}

	// Create empty type substitutions for now - this could be enhanced later
	typeSubstitutions := make(map[string]domain.Type)

	// Use the builder's field mapping logic
	builderFieldMapping, err := genericMapper.MapGenericFields(
		sourceType,
		destType,
		typeSubstitutions,
		options,
	)
	if err != nil {
		gcg.logger.Debug("builder field mapping failed, using fallback", zap.Error(err))
		// Fall back to simple field mapping
		return gcg.generateSimpleFieldMappings(sourceType, destType, annotations)
	}

	// Convert builder.FieldMapping to generator.FieldMapping
	mappings := make([]*FieldMapping, len(builderFieldMapping.Assignments))
	for i, assignment := range builderFieldMapping.Assignments {
		mapping := &FieldMapping{
			SourceField: assignment.GetSourceFieldName(),
			DestField:   assignment.GetDestFieldName(),
			Annotations: annotations,
		}

		// Set types if available
		if assignment.SourceField != nil {
			mapping.SourceType = assignment.SourceField.Type
		}
		if assignment.DestField != nil {
			mapping.DestType = assignment.DestField.Type
		}

		// Extract converter from assignment code if present
		if assignment.Converter != "" {
			mapping.Converter = assignment.Converter
		}

		mappings[i] = mapping
	}

	gcg.logger.Debug("generated field mappings using builder",
		zap.Int("mappings_count", len(mappings)))

	return mappings, nil
}

// extractStructFields extracts fields from a struct type.
func (gcg *GenericCodeGenerator) extractStructFields(typ domain.Type) ([]*domain.Field, error) {
	// Handle pointer types by dereferencing
	if typ.Kind() == domain.KindPointer {
		if ptrType, ok := typ.(*domain.PointerType); ok {
			return gcg.extractStructFields(ptrType.Elem())
		}
	}

	// Only handle struct types
	if typ.Kind() != domain.KindStruct {
		return nil, fmt.Errorf("%w: %s", ErrNotStructType, typ.Kind())
	}

	// Cast to struct type and get fields
	if structType, ok := typ.(*domain.StructType); ok {
		fields := structType.Fields()
		result := make([]*domain.Field, len(fields))
		for i, field := range fields {
			result[i] = &domain.Field{
				Name:     field.Name,
				Type:     field.Type,
				Position: field.Position,
				Exported: field.Exported,
			}
		}
		return result, nil
	}

	return nil, fmt.Errorf("%w: %T", ErrInvalidStructType, typ)
}

// generateSimpleFieldMappings provides a fallback field mapping implementation.
func (gcg *GenericCodeGenerator) generateSimpleFieldMappings(
	sourceType, destType domain.Type,
	annotations map[string]string,
) ([]*FieldMapping, error) {
	// Extract fields from source and destination types
	srcFields, err := gcg.extractStructFields(sourceType)
	if err != nil {
		gcg.logger.Debug("failed to extract source fields", zap.Error(err))
		return []*FieldMapping{}, nil // Return empty mappings rather than error for non-struct types
	}

	dstFields, err := gcg.extractStructFields(destType)
	if err != nil {
		gcg.logger.Debug("failed to extract destination fields", zap.Error(err))
		return []*FieldMapping{}, nil
	}

	gcg.logger.Debug("extracted fields for simple mapping",
		zap.Int("source_fields", len(srcFields)),
		zap.Int("dest_fields", len(dstFields)))

	// Generate field mappings using simple name-based matching
	mappings := make([]*FieldMapping, 0, len(dstFields))

	for _, dstField := range dstFields {
		// Try to find matching source field by name
		for _, srcField := range srcFields {
			if gcg.fieldsCanMap(srcField, dstField) {
				mapping := &FieldMapping{
					SourceField: srcField.Name,
					DestField:   dstField.Name,
					SourceType:  srcField.Type,
					DestType:    dstField.Type,
					Annotations: annotations,
				}

				// Add type conversion if needed
				if !srcField.Type.AssignableTo(dstField.Type) {
					mapping.Converter = gcg.generateTypeConversion(srcField.Type, dstField.Type)
				}

				mappings = append(mappings, mapping)
				break
			}
		}
	}

	gcg.logger.Debug("generated simple field mappings",
		zap.Int("mappings_count", len(mappings)))

	return mappings, nil
}

// fieldsCanMap determines if two fields can be mapped to each other.
func (gcg *GenericCodeGenerator) fieldsCanMap(srcField, dstField *domain.Field) bool {
	// Check name match (primary criteria)
	if srcField.Name == dstField.Name {
		return true
	}

	// Check for common field name patterns
	commonMappings := map[string]string{
		"Name":  "Value", // Name -> Value mapping
		"Value": "Value", // Value -> Value mapping
		"Inner": "Inner", // Inner -> Inner mapping
	}

	for srcName, dstName := range commonMappings {
		if srcField.Name == srcName && dstField.Name == dstName {
			return true
		}
	}

	// Special case for nested fields - could be enhanced
	return false
}

// generateTypeConversion generates type conversion code if needed.
func (gcg *GenericCodeGenerator) generateTypeConversion(srcType, dstType domain.Type) string {
	// Handle basic type conversions
	if srcType.Kind() == domain.KindBasic && dstType.Kind() == domain.KindBasic {
		return dstType.Name() // Simple type conversion
	}

	// For more complex types, return empty string (direct assignment)
	return ""
}

// Shutdown gracefully shuts down the generator.
func (gcg *GenericCodeGenerator) Shutdown(ctx context.Context) error {
	gcg.logger.Info("shutting down generic code generator",
		zap.Int64("total_generations", gcg.metrics.TotalGenerations),
		zap.Int64("successful_generations", gcg.metrics.SuccessfulGenerations))
	return nil
}
