package builder

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v9/pkg/domain"
)

// Static errors for err113 compliance.
var (
	ErrGenericFieldMapperNil     = errors.New("generic field mapper cannot be nil")
	ErrTypeSubstitutionEngineNil = errors.New("type substitution engine cannot be nil")
	ErrGenericMappingContextNil  = errors.New("generic mapping context cannot be nil")
	ErrGenericFieldMappingFailed = errors.New("generic field mapping failed")
	ErrTypeSubstitutionInMapping = errors.New("type substitution failed in field mapping")
	ErrGenericTypeNotSupported   = errors.New("generic type not supported in mapping")
	ErrFieldMappingOptionsNil    = errors.New("field mapping options cannot be nil")
	ErrGenericAssignmentFailed   = errors.New("generic assignment generation failed")
)

// FieldMapper defines the interface for basic field mapping operations.
type FieldMapper interface {
	MapFields(sourceType, destType domain.Type, options map[string]string) ([]*BasicFieldMapping, error)
}

// BasicFieldMapping represents a basic field mapping.
type BasicFieldMapping struct {
	SourceField string
	DestField   string
	SourceType  domain.Type
	DestType    domain.Type
}

// basicFieldMapper provides a default implementation of FieldMapper.
type basicFieldMapper struct{}

// MapFields provides a basic field mapping implementation.
func (bfm *basicFieldMapper) MapFields(sourceType, destType domain.Type, options map[string]string) ([]*BasicFieldMapping, error) {
	// Simple implementation that returns empty mappings
	// In a real implementation, this would analyze the types and create appropriate mappings
	return []*BasicFieldMapping{}, nil
}

// GenericFieldMapper handles field mapping for generic types with type substitution support.
type GenericFieldMapper struct {
	baseMapper       FieldMapper
	typeSubstitution *domain.TypeSubstitutionEngine
	logger           *zap.Logger

	// Configuration
	config *GenericFieldMapperConfig

	// Performance tracking
	metrics *GenericFieldMappingMetrics

	// Cache for field mapping strategies
	strategyCache map[string]domain.ConversionStrategy

	// Built-in conversion strategies
	strategies []domain.ConversionStrategy

	// Enhanced: Recursive type resolver for deeply nested generics
	recursiveResolver *RecursiveTypeResolver
}

// GenericFieldMapperConfig configures the generic field mapper.
type GenericFieldMapperConfig struct {
	EnableCaching        bool          `json:"enable_caching"`
	MaxCacheSize         int           `json:"max_cache_size"`
	EnableOptimization   bool          `json:"enable_optimization"`
	MappingTimeout       time.Duration `json:"mapping_timeout"`
	EnableTypeValidation bool          `json:"enable_type_validation"`
	DebugMode            bool          `json:"debug_mode"`
	PerformanceMode      bool          `json:"performance_mode"`
}

// DefaultGenericFieldMapperConfig returns default configuration.
func DefaultGenericFieldMapperConfig() *GenericFieldMapperConfig {
	return &GenericFieldMapperConfig{
		EnableCaching:        true,
		MaxCacheSize:         1000,
		EnableOptimization:   true,
		MappingTimeout:       30 * time.Second,
		EnableTypeValidation: true,
		DebugMode:            false,
		PerformanceMode:      false,
	}
}

// GenericFieldMappingMetrics tracks performance for generic field mapping.
type GenericFieldMappingMetrics struct {
	TotalMappings        int64         `json:"total_mappings"`
	SuccessfulMappings   int64         `json:"successful_mappings"`
	FailedMappings       int64         `json:"failed_mappings"`
	TypeSubstitutions    int64         `json:"type_substitutions"`
	CacheHits            int64         `json:"cache_hits"`
	CacheMisses          int64         `json:"cache_misses"`
	OptimizationsApplied int64         `json:"optimizations_applied"`
	AverageMappingTime   time.Duration `json:"average_mapping_time"`
	TotalMappingTime     time.Duration `json:"total_mapping_time"`
}

// NewGenericFieldMappingMetrics creates new metrics instance.
func NewGenericFieldMappingMetrics() *GenericFieldMappingMetrics {
	return &GenericFieldMappingMetrics{}
}

// FieldMappingOptions provides options for field mapping operations.
type FieldMappingOptions struct {
	IncludePrivateFields bool                   `json:"include_private_fields"`
	UseTypeConversion    bool                   `json:"use_type_conversion"`
	ValidateTypes        bool                   `json:"validate_types"`
	IgnoreUnmatched      bool                   `json:"ignore_unmatched"`
	CustomMappings       map[string]string      `json:"custom_mappings"`
	Annotations          map[string]*Annotation `json:"annotations"`
	ErrorHandling        ErrorHandlingStrategy  `json:"error_handling"`
}

// DefaultFieldMappingOptions returns default field mapping options.
func DefaultFieldMappingOptions() *FieldMappingOptions {
	return &FieldMappingOptions{
		IncludePrivateFields: false,
		UseTypeConversion:    true,
		ValidateTypes:        true,
		IgnoreUnmatched:      false,
		CustomMappings:       make(map[string]string),
		Annotations:          make(map[string]*Annotation),
		ErrorHandling:        domain.ErrorPropagate,
	}
}

// Annotation represents field mapping annotations.
type Annotation struct {
	Skip       bool              `json:"skip"`
	Map        string            `json:"map"`
	Converter  string            `json:"converter"`
	Validation string            `json:"validation"`
	Literal    string            `json:"literal"`
	Custom     map[string]string `json:"custom"`
}

// ErrorHandlingStrategy defines how to handle mapping errors.
type ErrorHandlingStrategy = domain.ErrorHandlingStrategy

// NewGenericFieldMapper creates a new generic field mapper.
func NewGenericFieldMapper(
	baseMapper FieldMapper,
	typeSubstitution *domain.TypeSubstitutionEngine,
	logger *zap.Logger,
	config *GenericFieldMapperConfig,
) *GenericFieldMapper {
	if baseMapper == nil {
		baseMapper = &basicFieldMapper{} // Create a basic field mapper if none provided
	}

	if typeSubstitution == nil {
		// Create a default type substitution engine
		typeBuilder := domain.NewTypeBuilder()
		typeSubstitution = domain.NewTypeSubstitutionEngine(typeBuilder, logger)
	}

	if config == nil {
		config = DefaultGenericFieldMapperConfig()
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	// Create recursive resolver for enhanced generic support
	recursiveResolver := NewRecursiveTypeResolver(
		typeSubstitution,
		logger,
		DefaultRecursiveResolverConfig(),
	)

	mapper := &GenericFieldMapper{
		baseMapper:        baseMapper,
		typeSubstitution:  typeSubstitution,
		logger:            logger,
		config:            config,
		metrics:           NewGenericFieldMappingMetrics(),
		strategyCache:     make(map[string]domain.ConversionStrategy),
		strategies:        domain.DefaultConversionStrategies(),
		recursiveResolver: recursiveResolver,
	}

	logger.Info("generic field mapper initialized",
		zap.Bool("caching_enabled", config.EnableCaching),
		zap.Bool("optimization_enabled", config.EnableOptimization),
		zap.Duration("timeout", config.MappingTimeout))

	return mapper
}

// MapGenericFields maps fields between generic source and destination types.
func (gfm *GenericFieldMapper) MapGenericFields(
	srcType domain.Type,
	dstType domain.Type,
	typeSubstitutions map[string]domain.Type,
	options *FieldMappingOptions,
) (*FieldMapping, error) {
	if options == nil {
		return nil, ErrFieldMappingOptionsNil
	}

	startTime := time.Now()
	gfm.metrics.TotalMappings++

	gfm.logger.Debug("starting generic field mapping",
		zap.String("source_type", srcType.String()),
		zap.String("destination_type", dstType.String()),
		zap.Int("substitutions", len(typeSubstitutions)))

	// Create mapping context
	context := &GenericMappingContext{
		SourceType:          srcType,
		DestinationType:     dstType,
		TypeSubstitutions:   typeSubstitutions,
		AnnotationOverrides: options.Annotations,
		MappingStrategy:     SelectOptimalMappingStrategy,
		Options:             options,
	}

	// Perform type substitutions
	substitutedSrcType, err := gfm.substituteTypeIfNeeded(srcType, typeSubstitutions)
	if err != nil {
		gfm.metrics.FailedMappings++
		return nil, fmt.Errorf("%w: source type substitution: %s", ErrTypeSubstitutionInMapping, err.Error())
	}

	substitutedDstType, err := gfm.substituteTypeIfNeeded(dstType, typeSubstitutions)
	if err != nil {
		gfm.metrics.FailedMappings++
		return nil, fmt.Errorf("%w: destination type substitution: %s", ErrTypeSubstitutionInMapping, err.Error())
	}

	// Update context with substituted types
	context.SubstitutedSourceType = substitutedSrcType
	context.SubstitutedDestType = substitutedDstType

	// Generate field mappings
	fieldMapping, err := gfm.generateFieldMapping(context)
	if err != nil {
		gfm.metrics.FailedMappings++
		return nil, fmt.Errorf("%w: %s", ErrGenericFieldMappingFailed, err.Error())
	}

	// Update metrics
	mappingTime := time.Since(startTime)
	gfm.metrics.SuccessfulMappings++
	gfm.metrics.TotalMappingTime += mappingTime
	gfm.metrics.AverageMappingTime = gfm.metrics.TotalMappingTime / time.Duration(gfm.metrics.TotalMappings)

	gfm.logger.Info("generic field mapping completed",
		zap.String("source_type", srcType.String()),
		zap.String("destination_type", dstType.String()),
		zap.Duration("mapping_time", mappingTime),
		zap.Int("field_assignments", len(fieldMapping.Assignments)))

	return fieldMapping, nil
}

// substituteTypeIfNeeded applies type substitutions if the type is generic, with enhanced recursive support.
func (gfm *GenericFieldMapper) substituteTypeIfNeeded(
	typ domain.Type,
	typeSubstitutions map[string]domain.Type,
) (domain.Type, error) {
	if !typ.Generic() || len(typeSubstitutions) == 0 {
		return typ, nil
	}

	gfm.metrics.TypeSubstitutions++

	// Enhanced: Use recursive resolver for deeply nested generic types
	if gfm.isDeeplyNestedGeneric(typ) {
		result, err := gfm.recursiveResolver.ResolveNestedGenericType(typ, typeSubstitutions)
		if err != nil {
			return nil, fmt.Errorf("recursive type resolution failed: %w", err)
		}
		return result.ResolvedType, nil
	}

	// Fallback to standard type substitution for simpler cases
	// Convert the substitution map to the format expected by TypeSubstitutionEngine
	typeParams := make([]domain.TypeParam, 0, len(typeSubstitutions))
	typeArgs := make([]domain.Type, 0, len(typeSubstitutions))

	for paramName, concreteType := range typeSubstitutions {
		typeParam := domain.TypeParam{
			Name:       paramName,
			Constraint: domain.NewBasicType("any", reflect.Invalid), // Default constraint as Type
		}
		typeParams = append(typeParams, typeParam)
		typeArgs = append(typeArgs, concreteType)
	}

	// Perform type substitution
	result, err := gfm.typeSubstitution.SubstituteType(typ, typeParams, typeArgs)
	if err != nil {
		return nil, fmt.Errorf("type substitution failed: %w", err)
	}

	return result.SubstitutedType, nil
}

// generateFieldMapping generates the actual field mapping.
func (gfm *GenericFieldMapper) generateFieldMapping(context *GenericMappingContext) (*FieldMapping, error) {
	srcType := context.SubstitutedSourceType
	dstType := context.SubstitutedDestType

	// Extract fields from source and destination types
	srcFields, err := gfm.extractTypeFields(srcType)
	if err != nil {
		return nil, fmt.Errorf("failed to extract source fields: %w", err)
	}

	dstFields, err := gfm.extractTypeFields(dstType)
	if err != nil {
		return nil, fmt.Errorf("failed to extract destination fields: %w", err)
	}

	// Generate field assignments
	assignments := make([]*FieldAssignment, 0)

	for _, dstField := range dstFields {
		assignment, err := gfm.generateFieldAssignment(dstField, srcFields, context)
		if err != nil {
			if context.Options.IgnoreUnmatched {
				gfm.logger.Warn("skipping unmatched field",
					zap.String("field", dstField.Name),
					zap.Error(err))
				continue
			}
			return nil, fmt.Errorf("failed to map field %s: %w", dstField.Name, err)
		}

		if assignment != nil {
			assignments = append(assignments, assignment)
		}
	}

	// Apply optimizations if enabled
	if gfm.config.EnableOptimization {
		assignments = gfm.optimizeAssignments(assignments, context)
		gfm.metrics.OptimizationsApplied++
	}

	return &FieldMapping{
		SourceType:      context.SourceType,
		DestinationType: context.DestinationType,
		Assignments:     assignments,
		Context:         context,
		GeneratedAt:     time.Now(),
	}, nil
}

// extractTypeFields extracts fields from a type.
func (gfm *GenericFieldMapper) extractTypeFields(typ domain.Type) ([]*domain.Field, error) {
	switch typ.Kind() {
	case domain.KindStruct:
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
		return nil, fmt.Errorf("expected StructType, got %T", typ)
	default:
		return nil, fmt.Errorf("%w: type kind %s", ErrGenericTypeNotSupported, typ.Kind().String())
	}
}

// generateFieldAssignment generates an assignment for a destination field.
func (gfm *GenericFieldMapper) generateFieldAssignment(
	dstField *domain.Field,
	srcFields []*domain.Field,
	context *GenericMappingContext,
) (*FieldAssignment, error) {
	// Check for annotation overrides
	if annotation, found := context.AnnotationOverrides[dstField.Name]; found {
		if annotation.Skip {
			return &FieldAssignment{
				DestField:      dstField,
				AssignmentType: SkipAssignment,
				Code:           fmt.Sprintf("// Skipping field %s", dstField.Name),
			}, nil
		}

		if annotation.Map != "" {
			return gfm.generateMappedAssignment(dstField, srcFields, annotation.Map, context)
		}

		if annotation.Converter != "" {
			return gfm.generateConverterAssignment(dstField, srcFields, annotation.Converter, context)
		}

		if annotation.Literal != "" {
			return gfm.generateLiteralAssignment(dstField, annotation.Literal, context)
		}
	}

	// Try to find matching source field
	for _, srcField := range srcFields {
		if gfm.fieldsMatch(srcField, dstField, context) {
			return gfm.generateDirectAssignment(srcField, dstField, context)
		}
	}

	// Try to generate nested field assignment
	if nestedAssignment := gfm.generateNestedFieldAssignment(dstField, srcFields, context); nestedAssignment != nil {
		return nestedAssignment, nil
	}

	// No match found
	if context.Options.IgnoreUnmatched {
		return nil, nil
	}

	return nil, fmt.Errorf("no matching source field for destination field %s", dstField.Name)
}

// fieldsMatch checks if two fields can be mapped to each other.
func (gfm *GenericFieldMapper) fieldsMatch(srcField, dstField *domain.Field, context *GenericMappingContext) bool {
	// Check exact name match first
	if srcField.Name == dstField.Name {
		if !context.Options.ValidateTypes {
			return true
		}
		return gfm.typesCompatible(srcField.Type, dstField.Type, context)
	}

	// Check for common field mapping patterns
	if gfm.fieldsMatchByPattern(srcField, dstField, context) {
		return true
	}

	return false
}

// typesCompatible checks if two types are compatible for assignment.
func (gfm *GenericFieldMapper) typesCompatible(srcType, dstType domain.Type, context *GenericMappingContext) bool {
	if srcType == nil || dstType == nil {
		return false
	}

	// Apply type substitutions before comparing
	substitutedSrcType := gfm.applyTypeSubstitution(srcType, context)
	substitutedDstType := gfm.applyTypeSubstitution(dstType, context)

	// Direct assignability
	if substitutedSrcType.AssignableTo(substitutedDstType) {
		return true
	}

	// Type conversion allowed
	if context != nil && context.Options.UseTypeConversion {
		// Check if types are convertible
		if gfm.typesConvertible(substitutedSrcType, substitutedDstType) {
			return true
		}
	}

	return false
}

// applyTypeSubstitution applies type parameter substitutions to a type.
func (gfm *GenericFieldMapper) applyTypeSubstitution(typ domain.Type, context *GenericMappingContext) domain.Type {
	if typ == nil || context == nil || len(context.TypeSubstitutions) == 0 {
		return typ
	}

	// Handle generic types by substitution
	if genericType, ok := typ.(*domain.GenericType); ok {
		if substitution, exists := context.TypeSubstitutions[genericType.Name()]; exists {
			return substitution
		}
		return typ
	}

	// Handle other types recursively if needed
	switch typ.Kind() {
	case domain.KindSlice:
		if sliceType, ok := typ.(*domain.SliceType); ok {
			elemType := gfm.applyTypeSubstitution(sliceType.Elem(), context)
			return domain.NewSliceType(elemType, sliceType.Package())
		}
	case domain.KindPointer:
		if pointerType, ok := typ.(*domain.PointerType); ok {
			elemType := gfm.applyTypeSubstitution(pointerType.Elem(), context)
			return domain.NewPointerType(elemType, pointerType.Package())
		}
		// Note: Map type substitution is not fully implemented in the domain package yet
		// case domain.KindMap: ... would go here when available
	}

	return typ
}

// typesConvertible checks if types can be converted, with enhanced support for nested generics.
func (gfm *GenericFieldMapper) typesConvertible(srcType, dstType domain.Type) bool {
	// Basic type conversions
	if srcType.Kind() == domain.KindBasic && dstType.Kind() == domain.KindBasic {
		return true
	}

	// Pointer conversions
	if srcType.Kind() == domain.KindPointer && dstType.Kind() == domain.KindPointer {
		srcElem := srcType.(*domain.PointerType).Elem()
		dstElem := dstType.(*domain.PointerType).Elem()
		return gfm.typesConvertible(srcElem, dstElem)
	}

	// Slice conversions
	if srcType.Kind() == domain.KindSlice && dstType.Kind() == domain.KindSlice {
		srcElem := srcType.(*domain.SliceType).Elem()
		dstElem := dstType.(*domain.SliceType).Elem()
		return gfm.typesConvertible(srcElem, dstElem)
	}

	// Enhanced: Map conversions for nested generics
	if srcType.Kind() == domain.KindMap && dstType.Kind() == domain.KindMap {
		return gfm.typesConvertibleForMaps(srcType, dstType)
	}

	// Enhanced: Generic type conversions
	if srcType.Kind() == domain.KindGeneric || dstType.Kind() == domain.KindGeneric {
		return gfm.typesConvertibleForGenerics(srcType, dstType)
	}

	// Enhanced: Named type conversions with generic support
	if srcType.Kind() == domain.KindNamed || dstType.Kind() == domain.KindNamed {
		return gfm.typesConvertibleForNamedTypes(srcType, dstType)
	}

	return false
}

// generateDirectAssignment generates a direct field assignment.
func (gfm *GenericFieldMapper) generateDirectAssignment(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) (*FieldAssignment, error) {
	assignmentCode := fmt.Sprintf("dst.%s = src.%s", dstField.Name, srcField.Name)

	// Add type conversion if needed
	if !srcField.Type.AssignableTo(dstField.Type) && context.Options.UseTypeConversion {
		dstTypeName := gfm.getTypeName(dstField.Type)
		assignmentCode = fmt.Sprintf("dst.%s = %s(src.%s)", dstField.Name, dstTypeName, srcField.Name)
	}

	return &FieldAssignment{
		SourceField:    srcField,
		DestField:      dstField,
		AssignmentType: DirectAssignment,
		Code:           assignmentCode,
	}, nil
}

// generateMappedAssignment generates an assignment using custom field mapping.
func (gfm *GenericFieldMapper) generateMappedAssignment(
	dstField *domain.Field,
	srcFields []*domain.Field,
	sourcePath string,
	context *GenericMappingContext,
) (*FieldAssignment, error) {
	// Find the mapped source field
	var sourceField *domain.Field
	for _, field := range srcFields {
		if field.Name == sourcePath {
			sourceField = field
			break
		}
	}

	if sourceField == nil {
		return nil, fmt.Errorf("mapped source field %s not found", sourcePath)
	}

	assignmentCode := fmt.Sprintf("dst.%s = src.%s", dstField.Name, sourcePath)

	// Add type conversion if needed
	if !sourceField.Type.AssignableTo(dstField.Type) && context.Options.UseTypeConversion {
		dstTypeName := gfm.getTypeName(dstField.Type)
		assignmentCode = fmt.Sprintf("dst.%s = %s(src.%s)", dstField.Name, dstTypeName, sourcePath)
	}

	return &FieldAssignment{
		SourceField:    sourceField,
		DestField:      dstField,
		AssignmentType: MappedAssignment,
		Code:           assignmentCode,
		SourcePath:     sourcePath,
	}, nil
}

// generateConverterAssignment generates an assignment using a converter function.
func (gfm *GenericFieldMapper) generateConverterAssignment(
	dstField *domain.Field,
	srcFields []*domain.Field,
	converter string,
	context *GenericMappingContext,
) (*FieldAssignment, error) {
	// Find matching source field (use field name as default)
	var sourceField *domain.Field
	for _, field := range srcFields {
		if field.Name == dstField.Name {
			sourceField = field
			break
		}
	}

	if sourceField == nil {
		return nil, fmt.Errorf("source field %s not found for converter assignment", dstField.Name)
	}

	var assignmentCode string
	if context.Options.ErrorHandling == domain.ErrorPropagate {
		assignmentCode = fmt.Sprintf(`convertedValue, err := %s(src.%s)
if err != nil {
	return dst, fmt.Errorf("conversion failed for field %s: %%w", err)
}
dst.%s = convertedValue`, converter, sourceField.Name, dstField.Name, dstField.Name)
	} else {
		assignmentCode = fmt.Sprintf("dst.%s = %s(src.%s)", dstField.Name, converter, sourceField.Name)
	}

	return &FieldAssignment{
		SourceField:    sourceField,
		DestField:      dstField,
		AssignmentType: ConverterAssignment,
		Code:           assignmentCode,
		Converter:      converter,
	}, nil
}

// generateLiteralAssignment generates an assignment using a literal value.
func (gfm *GenericFieldMapper) generateLiteralAssignment(
	dstField *domain.Field,
	literal string,
	context *GenericMappingContext,
) (*FieldAssignment, error) {
	assignmentCode := fmt.Sprintf("dst.%s = %s", dstField.Name, literal)

	return &FieldAssignment{
		DestField:      dstField,
		AssignmentType: LiteralAssignment,
		Code:           assignmentCode,
		Literal:        literal,
	}, nil
}

// getTypeName returns the string representation of a type for code generation.
func (gfm *GenericFieldMapper) getTypeName(typ domain.Type) string {
	if typ.Package() != "" {
		return typ.Package() + "." + typ.Name()
	}
	return typ.Name()
}

// fieldsMatchByPattern checks if fields can be mapped using common patterns.
func (gfm *GenericFieldMapper) fieldsMatchByPattern(srcField, dstField *domain.Field, context *GenericMappingContext) bool {
	// Common field name mapping patterns
	mappingPatterns := map[string][]string{
		"Value": {"Name", "Value", "Data", "Content", "Result"}, // Value can accept from multiple sources
		"Name":  {"Name", "Title", "Label", "Value"},            // Name can accept from multiple sources
		"Data":  {"Data", "Content", "Value", "Payload"},        // Data can accept from multiple sources
		"Inner": {"Inner", "Value", "Data", "Content"},          // Inner can accept from multiple sources
	}

	// Check if destination field can accept from source field
	if acceptableSources, exists := mappingPatterns[dstField.Name]; exists {
		for _, acceptableSource := range acceptableSources {
			if srcField.Name == acceptableSource {
				// Check type compatibility if validation is enabled
				if !context.Options.ValidateTypes {
					return true
				}
				return gfm.typesCompatible(srcField.Type, dstField.Type, context)
			}
		}
	}

	// Check if source and destination have compatible patterns (bidirectional)
	for pattern, sources := range mappingPatterns {
		// Check if source field matches this pattern
		for _, source := range sources {
			if srcField.Name == source {
				// Check if dest field also matches a compatible pattern
				for _, compatibleSource := range sources {
					if dstField.Name == compatibleSource {
						if !context.Options.ValidateTypes {
							return true
						}
						return gfm.typesCompatible(srcField.Type, dstField.Type, context)
					}
				}
				break
			}
		}
		if srcField.Name == pattern {
			// Source field is a pattern name, check if dest field is compatible
			for _, compatibleSource := range sources {
				if dstField.Name == compatibleSource {
					if !context.Options.ValidateTypes {
						return true
					}
					return gfm.typesCompatible(srcField.Type, dstField.Type, context)
				}
			}
		}
	}

	return false
}

// optimizeAssignments applies optimizations to field assignments.
func (gfm *GenericFieldMapper) optimizeAssignments(
	assignments []*FieldAssignment,
	context *GenericMappingContext,
) []*FieldAssignment {
	if !gfm.config.EnableOptimization {
		return assignments
	}

	// Group similar assignments
	optimized := make([]*FieldAssignment, 0, len(assignments))

	// Remove redundant type conversions
	for _, assignment := range assignments {
		if assignment.AssignmentType == DirectAssignment {
			// Check if type conversion is actually needed
			if assignment.SourceField != nil &&
				assignment.SourceField.Type.AssignableTo(assignment.DestField.Type) {
				// Remove unnecessary type conversion
				assignment.Code = fmt.Sprintf("dst.%s = src.%s",
					assignment.DestField.Name, assignment.SourceField.Name)
			}
		}
		optimized = append(optimized, assignment)
	}

	return optimized
}

// GetMetrics returns the current mapping metrics.
func (gfm *GenericFieldMapper) GetMetrics() *GenericFieldMappingMetrics {
	return gfm.metrics
}

// ClearMetrics resets all metrics.
func (gfm *GenericFieldMapper) ClearMetrics() {
	gfm.metrics = NewGenericFieldMappingMetrics()
}

// generateNestedFieldAssignment attempts to generate a nested field assignment with enhanced generic support.
func (gfm *GenericFieldMapper) generateNestedFieldAssignment(
	dstField *domain.Field,
	srcFields []*domain.Field,
	context *GenericMappingContext,
) *FieldAssignment {
	// Handle nested struct field mappings
	if dstField.Type.Kind() == domain.KindStruct {
		return gfm.generateNestedStructFieldAssignment(dstField, srcFields, context)
	}

	// Enhanced: Handle nested slice field mappings (e.g., []List[T] -> []Array[U])
	if dstField.Type.Kind() == domain.KindSlice {
		return gfm.generateNestedSliceFieldAssignment(dstField, srcFields, context)
	}

	// Enhanced: Handle nested map field mappings (e.g., Map[string, List[T]] -> Map[string, Array[U]])
	if dstField.Type.Kind() == domain.KindMap {
		return gfm.generateNestedMapFieldAssignment(dstField, srcFields, context)
	}

	// Enhanced: Handle nested generic field mappings
	if dstField.Type.Kind() == domain.KindGeneric || dstField.Type.Generic() {
		return gfm.generateNestedGenericFieldAssignment(dstField, srcFields, context)
	}

	return nil
}

// generateNestedStructAssignment generates code for nested struct assignments (legacy method).
// This method is kept for backward compatibility but delegates to the enhanced version.
func (gfm *GenericFieldMapper) generateNestedStructAssignment(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) string {
	// Delegate to enhanced version
	return gfm.generateEnhancedNestedStructAssignment(srcField, dstField, context)
}

// fieldsCanMapNested checks if nested fields can be mapped (simpler rules).
func (gfm *GenericFieldMapper) fieldsCanMapNested(srcField, dstField *domain.Field) bool {
	// For nested fields, use more lenient matching
	if srcField.Name == dstField.Name {
		return true
	}

	// Allow common transformations for nested fields
	commonMappings := map[string][]string{
		"Inner": {"Inner", "Value", "Data"},
		"Value": {"Value", "Inner", "Data", "Name"},
		"Data":  {"Data", "Value", "Content"},
	}

	if acceptable, exists := commonMappings[dstField.Name]; exists {
		for _, source := range acceptable {
			if srcField.Name == source {
				return true
			}
		}
	}

	return false
}

// SetConfiguration updates the mapper configuration.
func (gfm *GenericFieldMapper) SetConfiguration(config *GenericFieldMapperConfig) {
	if config != nil {
		gfm.config = config
	}
}

// Enhanced methods for nested generic type handling

// typesConvertibleForMaps checks convertibility between map types with potential generic arguments.
func (gfm *GenericFieldMapper) typesConvertibleForMaps(srcType, dstType domain.Type) bool {
	// For map types, we need to check both key and value type compatibility
	// This is a simplified implementation - in production, would need proper MapType interface
	gfm.logger.Debug("checking map type convertibility",
		zap.String("src_type", srcType.String()),
		zap.String("dst_type", dstType.String()))

	// For now, allow conversion between map types if they have compatible structure
	// A full implementation would examine key/value types recursively
	return srcType.Name() != "" && dstType.Name() != ""
}

// typesConvertibleForGenerics checks convertibility between generic types.
func (gfm *GenericFieldMapper) typesConvertibleForGenerics(srcType, dstType domain.Type) bool {
	// If one type is generic and the other is concrete, check if substitution can work
	if srcType.Kind() == domain.KindGeneric {
		// Source is generic parameter, check if destination can accept it
		return gfm.canAcceptGenericType(srcType, dstType)
	}

	if dstType.Kind() == domain.KindGeneric {
		// Destination is generic parameter, check if source can be assigned
		return gfm.canAssignToGenericType(srcType, dstType)
	}

	// Both types might have generic parameters in their structure
	if srcType.Generic() && dstType.Generic() {
		return gfm.compatibleGenericStructures(srcType, dstType)
	}

	return false
}

// typesConvertibleForNamedTypes checks convertibility between named types with generic support.
func (gfm *GenericFieldMapper) typesConvertibleForNamedTypes(srcType, dstType domain.Type) bool {
	// Named types can have generic parameters, need to check underlying types
	if srcType.Kind() == domain.KindNamed {
		srcUnderlying := srcType.Underlying()
		if srcUnderlying != nil && srcUnderlying != srcType {
			return gfm.typesConvertible(srcUnderlying, dstType)
		}
	}

	if dstType.Kind() == domain.KindNamed {
		dstUnderlying := dstType.Underlying()
		if dstUnderlying != nil && dstUnderlying != dstType {
			return gfm.typesConvertible(srcType, dstUnderlying)
		}
	}

	// Check if both are named types with compatible names/packages
	return gfm.compatibleNamedTypes(srcType, dstType)
}

// canAcceptGenericType checks if a concrete type can accept a generic type parameter.
func (gfm *GenericFieldMapper) canAcceptGenericType(genericType, concreteType domain.Type) bool {
	// This would check type constraints in a full implementation
	// For now, be permissive for any concrete type
	return concreteType.Kind() != domain.KindGeneric
}

// canAssignToGenericType checks if a concrete type can be assigned to a generic parameter.
func (gfm *GenericFieldMapper) canAssignToGenericType(concreteType, genericType domain.Type) bool {
	// This would check type constraints in a full implementation
	// For now, be permissive for any concrete type
	return concreteType.Kind() != domain.KindGeneric
}

// compatibleGenericStructures checks if two generic structures are compatible.
func (gfm *GenericFieldMapper) compatibleGenericStructures(srcType, dstType domain.Type) bool {
	// Check if the base structure is similar even if type parameters differ
	// This is a simplified compatibility check
	srcName := gfm.extractBaseTypeName(srcType.String())
	dstName := gfm.extractBaseTypeName(dstType.String())

	return srcName == dstName || gfm.structurallyCompatible(srcType, dstType)
}

// compatibleNamedTypes checks if two named types are compatible.
func (gfm *GenericFieldMapper) compatibleNamedTypes(srcType, dstType domain.Type) bool {
	// Check name compatibility and package compatibility
	if srcType.Name() == dstType.Name() {
		// Same name, check if packages are compatible
		return gfm.packagesCompatible(srcType.Package(), dstType.Package())
	}

	// Different names, check if they represent similar concepts
	return gfm.semanticallyCompatibleNames(srcType.Name(), dstType.Name())
}

// extractBaseTypeName extracts the base type name from a generic type string.
func (gfm *GenericFieldMapper) extractBaseTypeName(typeStr string) string {
	// Extract base name before any generic parameters
	if idx := strings.Index(typeStr, "["); idx != -1 {
		return typeStr[:idx]
	}
	return typeStr
}

// structurallyCompatible checks if two types have compatible structure.
func (gfm *GenericFieldMapper) structurallyCompatible(srcType, dstType domain.Type) bool {
	// This would perform deeper structural analysis
	// For now, use a heuristic based on type kinds
	return srcType.Kind() == dstType.Kind()
}

// packagesCompatible checks if two packages are compatible for type conversion.
func (gfm *GenericFieldMapper) packagesCompatible(srcPkg, dstPkg string) bool {
	// Same package is always compatible
	if srcPkg == dstPkg {
		return true
	}

	// Different packages might still be compatible if they're related
	return gfm.relatedPackages(srcPkg, dstPkg)
}

// semanticallyCompatibleNames checks if two type names represent compatible concepts.
func (gfm *GenericFieldMapper) semanticallyCompatibleNames(srcName, dstName string) bool {
	// Common type name mappings for generic collections
	compatibilityMap := map[string][]string{
		"List":  {"Array", "Slice", "Vector", "Collection"},
		"Array": {"List", "Slice", "Vector", "Collection"},
		"Map":   {"Dict", "HashMap", "Dictionary", "Table"},
		"Dict":  {"Map", "HashMap", "Dictionary", "Table"},
		"Set":   {"HashSet", "Collection"},
	}

	if compatibleNames, exists := compatibilityMap[srcName]; exists {
		for _, name := range compatibleNames {
			if name == dstName {
				return true
			}
		}
	}

	return false
}

// relatedPackages checks if two packages are related for type compatibility.
func (gfm *GenericFieldMapper) relatedPackages(srcPkg, dstPkg string) bool {
	// This could check for version differences, alias packages, etc.
	// For now, be conservative
	return false
}

// generateNestedStructFieldAssignment generates assignment for nested struct fields.
func (gfm *GenericFieldMapper) generateNestedStructFieldAssignment(
	dstField *domain.Field,
	srcFields []*domain.Field,
	context *GenericMappingContext,
) *FieldAssignment {
	// Look for a source field with the same name that's also a struct
	for _, srcField := range srcFields {
		if srcField.Name == dstField.Name && srcField.Type.Kind() == domain.KindStruct {
			// Generate nested struct assignment with generic support
			nestedCode := gfm.generateEnhancedNestedStructAssignment(srcField, dstField, context)
			if nestedCode != "" {
				return &FieldAssignment{
					SourceField:    srcField,
					DestField:      dstField,
					AssignmentType: DirectAssignment,
					Code:           nestedCode,
				}
			}
		}
	}
	return nil
}

// generateNestedSliceFieldAssignment generates assignment for nested slice fields.
func (gfm *GenericFieldMapper) generateNestedSliceFieldAssignment(
	dstField *domain.Field,
	srcFields []*domain.Field,
	context *GenericMappingContext,
) *FieldAssignment {
	// Look for compatible source slice fields
	for _, srcField := range srcFields {
		if gfm.fieldsMatch(srcField, dstField, context) && srcField.Type.Kind() == domain.KindSlice {
			// Generate slice conversion code
			sliceCode := gfm.generateSliceConversionCode(srcField, dstField, context)
			if sliceCode != "" {
				return &FieldAssignment{
					SourceField:    srcField,
					DestField:      dstField,
					AssignmentType: SliceAssignment,
					Code:           sliceCode,
				}
			}
		}
	}
	return nil
}

// generateNestedMapFieldAssignment generates assignment for nested map fields.
func (gfm *GenericFieldMapper) generateNestedMapFieldAssignment(
	dstField *domain.Field,
	srcFields []*domain.Field,
	context *GenericMappingContext,
) *FieldAssignment {
	// Look for compatible source map fields
	for _, srcField := range srcFields {
		if gfm.fieldsMatch(srcField, dstField, context) && srcField.Type.Kind() == domain.KindMap {
			// Generate map conversion code
			mapCode := gfm.generateMapConversionCode(srcField, dstField, context)
			if mapCode != "" {
				return &FieldAssignment{
					SourceField:    srcField,
					DestField:      dstField,
					AssignmentType: MapAssignment,
					Code:           mapCode,
				}
			}
		}
	}
	return nil
}

// generateNestedGenericFieldAssignment generates assignment for nested generic fields.
func (gfm *GenericFieldMapper) generateNestedGenericFieldAssignment(
	dstField *domain.Field,
	srcFields []*domain.Field,
	context *GenericMappingContext,
) *FieldAssignment {
	// Look for source fields that can be converted to the generic destination
	for _, srcField := range srcFields {
		if gfm.canConvertToGenericField(srcField, dstField, context) {
			// Generate generic conversion code
			genericCode := gfm.generateGenericConversionCode(srcField, dstField, context)
			if genericCode != "" {
				return &FieldAssignment{
					SourceField:    srcField,
					DestField:      dstField,
					AssignmentType: ConversionAssignment,
					Code:           genericCode,
				}
			}
		}
	}
	return nil
}

// generateEnhancedNestedStructAssignment generates enhanced code for nested struct assignments with generic support.
func (gfm *GenericFieldMapper) generateEnhancedNestedStructAssignment(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) string {
	// Extract fields from both structs with type substitution awareness
	srcStructFields, err := gfm.extractTypeFieldsWithSubstitution(srcField.Type, context)
	if err != nil {
		return ""
	}

	dstStructFields, err := gfm.extractTypeFieldsWithSubstitution(dstField.Type, context)
	if err != nil {
		return ""
	}

	// Generate field-by-field assignments for the nested struct with enhanced matching
	assignments := make([]string, 0)

	for _, dstNestedField := range dstStructFields {
		for _, srcNestedField := range srcStructFields {
			if gfm.enhancedFieldsCanMapNested(srcNestedField, dstNestedField, context) {
				// Generate the nested assignment with type conversion if needed
				assignment := gfm.generateNestedAssignmentCode(srcField, dstField, srcNestedField, dstNestedField, context)
				assignments = append(assignments, assignment)
				break
			}
		}
	}

	if len(assignments) == 0 {
		return ""
	}

	// Join all assignments with newlines
	return strings.Join(assignments, "\n\t")
}

// extractTypeFieldsWithSubstitution extracts fields from a type with type substitution applied.
func (gfm *GenericFieldMapper) extractTypeFieldsWithSubstitution(
	typ domain.Type,
	context *GenericMappingContext,
) ([]*domain.Field, error) {
	// Apply type substitutions first if needed
	if typ.Generic() && context.RequiresTypeSubstitution() {
		substitutedType, err := gfm.substituteTypeIfNeeded(typ, context.TypeSubstitutions)
		if err != nil {
			return nil, err
		}
		typ = substitutedType
	}

	return gfm.extractTypeFields(typ)
}

// enhancedFieldsCanMapNested checks if nested fields can be mapped with enhanced generic support.
func (gfm *GenericFieldMapper) enhancedFieldsCanMapNested(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) bool {
	// Enhanced field matching with generic type awareness
	if srcField.Name == dstField.Name {
		// Names match, check type compatibility with generic support
		return gfm.typesCompatible(srcField.Type, dstField.Type, context)
	}

	// Allow common transformations for nested fields with enhanced patterns
	return gfm.fieldsMatchByPattern(srcField, dstField, context)
}

// generateNestedAssignmentCode generates assignment code for nested fields.
func (gfm *GenericFieldMapper) generateNestedAssignmentCode(
	srcField, dstField *domain.Field,
	srcNestedField, dstNestedField *domain.Field,
	context *GenericMappingContext,
) string {
	basicAssignment := fmt.Sprintf("dst.%s.%s = src.%s.%s",
		dstField.Name, dstNestedField.Name,
		srcField.Name, srcNestedField.Name)

	// Add type conversion if needed
	if !srcNestedField.Type.AssignableTo(dstNestedField.Type) && context.Options.UseTypeConversion {
		dstTypeName := gfm.getTypeName(dstNestedField.Type)
		return fmt.Sprintf("dst.%s.%s = %s(src.%s.%s)",
			dstField.Name, dstNestedField.Name, dstTypeName,
			srcField.Name, srcNestedField.Name)
	}

	return basicAssignment
}

// generateSliceConversionCode generates conversion code for slice fields.
func (gfm *GenericFieldMapper) generateSliceConversionCode(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) string {
	// For slice conversions, we need to handle element type conversion
	// This is a simplified implementation
	return fmt.Sprintf(`// Convert slice: %s -> %s
	dst.%s = make(%s, len(src.%s))
	for i, item := range src.%s {
		// TODO: Add element conversion logic here
		dst.%s[i] = item // Placeholder conversion
	}`,
		srcField.Type.String(), dstField.Type.String(),
		dstField.Name, gfm.getTypeName(dstField.Type),
		srcField.Name, srcField.Name, dstField.Name)
}

// generateMapConversionCode generates conversion code for map fields.
func (gfm *GenericFieldMapper) generateMapConversionCode(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) string {
	// For map conversions, we need to handle both key and value type conversion
	// This is a simplified implementation
	return fmt.Sprintf(`// Convert map: %s -> %s
	dst.%s = make(%s, len(src.%s))
	for key, value := range src.%s {
		// TODO: Add key/value conversion logic here
		dst.%s[key] = value // Placeholder conversion
	}`,
		srcField.Type.String(), dstField.Type.String(),
		dstField.Name, gfm.getTypeName(dstField.Type),
		srcField.Name, srcField.Name, dstField.Name)
}

// canConvertToGenericField checks if a source field can be converted to a generic destination field.
func (gfm *GenericFieldMapper) canConvertToGenericField(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) bool {
	// Check if field names are compatible
	if !gfm.fieldsMatch(srcField, dstField, context) {
		return false
	}

	// Check if source type can be converted to the generic destination type
	return gfm.typesConvertibleForGenerics(srcField.Type, dstField.Type)
}

// generateGenericConversionCode generates conversion code for generic fields.
func (gfm *GenericFieldMapper) generateGenericConversionCode(
	srcField, dstField *domain.Field,
	context *GenericMappingContext,
) string {
	// For generic conversions, we might need to apply type substitutions
	assignmentCode := fmt.Sprintf("dst.%s = src.%s", dstField.Name, srcField.Name)

	// Add type conversion if the destination field requires it
	if dstField.Type.Kind() == domain.KindGeneric {
		// Check if we have a concrete type substitution for this generic parameter
		if concreteType, found := context.TypeSubstitutions[dstField.Type.Name()]; found {
			concreteTypeName := gfm.getTypeName(concreteType)
			assignmentCode = fmt.Sprintf("dst.%s = %s(src.%s)",
				dstField.Name, concreteTypeName, srcField.Name)
		}
	}

	return assignmentCode
}

// Enhanced helper methods for deeply nested generic support

// isDeeplyNestedGeneric checks if a type contains deeply nested generic structures.
func (gfm *GenericFieldMapper) isDeeplyNestedGeneric(typ domain.Type) bool {
	// Check for nested generic patterns
	if !typ.Generic() {
		return false
	}

	// Heuristic: Check if type string contains nested generic patterns
	typeStr := typ.String()

	// Count bracket depth to detect nested generics like Map[K, List[V]]
	bracketDepth := 0
	maxDepth := 0

	for _, char := range typeStr {
		switch char {
		case '[':
			bracketDepth++
			if bracketDepth > maxDepth {
				maxDepth = bracketDepth
			}
		case ']':
			bracketDepth--
		}
	}

	// Consider deeply nested if bracket depth > 1 or contains known complex patterns
	return maxDepth > 1 || gfm.containsComplexGenericPatterns(typeStr)
}

// containsComplexGenericPatterns checks for known complex generic patterns.
func (gfm *GenericFieldMapper) containsComplexGenericPatterns(typeStr string) bool {
	complexPatterns := []string{
		"Map[",
		"List[",
		"Array[",
		"Set[",
		"Optional[",
		"Future[",
		"Result[",
		"Either[",
	}

	patternCount := 0
	for _, pattern := range complexPatterns {
		if strings.Contains(typeStr, pattern) {
			patternCount++
			if patternCount > 1 {
				return true // Multiple generic patterns indicate complexity
			}
		}
	}

	return false
}

// RegisterTypeAlias registers a type alias for use in generic field mapping.
func (gfm *GenericFieldMapper) RegisterTypeAlias(aliasName string, actualType domain.Type) {
	if gfm.recursiveResolver != nil {
		gfm.recursiveResolver.RegisterTypeAlias(aliasName, actualType)
		gfm.logger.Debug("registered type alias for field mapping",
			zap.String("alias", aliasName),
			zap.String("actual_type", actualType.String()))
	}
}

// SupportsGenericTypeAlias checks if the mapper supports generic type aliases.
func (gfm *GenericFieldMapper) SupportsGenericTypeAlias() bool {
	return gfm.recursiveResolver != nil
}

// GetRecursiveResolutionMetrics returns metrics from the recursive resolver.
func (gfm *GenericFieldMapper) GetRecursiveResolutionMetrics() *RecursiveResolutionMetrics {
	if gfm.recursiveResolver != nil {
		return gfm.recursiveResolver.GetMetrics()
	}
	return nil
}

// ClearRecursiveResolutionCache clears the recursive resolver's cache.
func (gfm *GenericFieldMapper) ClearRecursiveResolutionCache() {
	if gfm.recursiveResolver != nil {
		gfm.recursiveResolver.ClearCache()
	}
}
