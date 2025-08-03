package builder

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v8/pkg/domain"
	gmodel "github.com/reedom/convergen/v8/pkg/generator/model"
	"github.com/reedom/convergen/v8/pkg/option"
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

// GenericFieldMapper handles field mapping for generic types with type substitution support.
type GenericFieldMapper struct {
	baseMapper       *FieldMapper
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
	baseMapper *FieldMapper,
	typeSubstitution *domain.TypeSubstitutionEngine,
	logger *zap.Logger,
	config *GenericFieldMapperConfig,
) *GenericFieldMapper {
	if baseMapper == nil {
		baseMapper = &FieldMapper{} // Create a basic field mapper if none provided
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

	mapper := &GenericFieldMapper{
		baseMapper:       baseMapper,
		typeSubstitution: typeSubstitution,
		logger:           logger,
		config:           config,
		metrics:          NewGenericFieldMappingMetrics(),
		strategyCache:    make(map[string]domain.ConversionStrategy),
		strategies:       domain.DefaultConversionStrategies(),
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

// substituteTypeIfNeeded applies type substitutions if the type is generic.
func (gfm *GenericFieldMapper) substituteTypeIfNeeded(
	typ domain.Type,
	typeSubstitutions map[string]domain.Type,
) (domain.Type, error) {
	if !typ.Generic() || len(typeSubstitutions) == 0 {
		return typ, nil
	}

	gfm.metrics.TypeSubstitutions++

	// Convert the substitution map to the format expected by TypeSubstitutionEngine
	typeParams := make([]domain.TypeParam, 0, len(typeSubstitutions))
	typeArgs := make([]domain.Type, 0, len(typeSubstitutions))

	for paramName, concreteType := range typeSubstitutions {
		typeParam := domain.TypeParam{
			Name:       paramName,
			Constraint: "any", // Default constraint
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

	// No match found
	if context.Options.IgnoreUnmatched {
		return nil, nil
	}

	return nil, fmt.Errorf("no matching source field for destination field %s", dstField.Name)
}

// fieldsMatch checks if two fields can be mapped to each other.
func (gfm *GenericFieldMapper) fieldsMatch(srcField, dstField *domain.Field, context *GenericMappingContext) bool {
	// Check name match
	if srcField.Name != dstField.Name {
		return false
	}

	// Check type compatibility
	if context.Options.ValidateTypes {
		return gfm.typesCompatible(srcField.Type, dstField.Type, context)
	}

	return true
}

// typesCompatible checks if two types are compatible for assignment.
func (gfm *GenericFieldMapper) typesCompatible(srcType, dstType domain.Type, context *GenericMappingContext) bool {
	// Direct assignability
	if srcType.AssignableTo(dstType) {
		return true
	}

	// Type conversion allowed
	if context.Options.UseTypeConversion {
		// Check if types are convertible
		if gfm.typesConvertible(srcType, dstType) {
			return true
		}
	}

	return false
}

// typesConvertible checks if types can be converted.
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

// SetConfiguration updates the mapper configuration.
func (gfm *GenericFieldMapper) SetConfiguration(config *GenericFieldMapperConfig) {
	if config != nil {
		gfm.config = config
	}
}
