package builder

import (
	"fmt"
	"strings"
	"time"

	"github.com/reedom/convergen/v8/pkg/domain"
)

// GenericMappingStrategy defines strategies for generic field mapping.
type GenericMappingStrategy int

const (
	// SelectOptimalMappingStrategy automatically selects the best strategy.
	SelectOptimalMappingStrategy GenericMappingStrategy = iota
	// DirectMappingStrategy uses direct field assignment where possible.
	DirectMappingStrategy
	// ConversionMappingStrategy uses type conversion for all fields.
	ConversionMappingStrategy
	// AnnotationDrivenMappingStrategy relies on annotations for mapping decisions.
	AnnotationDrivenMappingStrategy
	// PerformanceMappingStrategy optimizes for performance over flexibility.
	PerformanceMappingStrategy
)

// String returns the string representation of the mapping strategy.
func (gms GenericMappingStrategy) String() string {
	switch gms {
	case SelectOptimalMappingStrategy:
		return "optimal"
	case DirectMappingStrategy:
		return "direct"
	case ConversionMappingStrategy:
		return "conversion"
	case AnnotationDrivenMappingStrategy:
		return "annotation_driven"
	case PerformanceMappingStrategy:
		return "performance"
	default:
		return "unknown"
	}
}

// GenericMappingContext provides context for generic field mapping operations.
type GenericMappingContext struct {
	// Source and destination types
	SourceType      domain.Type `json:"source_type"`
	DestinationType domain.Type `json:"destination_type"`

	// Type substitution information
	TypeSubstitutions     map[string]domain.Type `json:"type_substitutions"`
	SubstitutedSourceType domain.Type            `json:"substituted_source_type"`
	SubstitutedDestType   domain.Type            `json:"substituted_dest_type"`

	// Mapping configuration
	AnnotationOverrides map[string]*Annotation `json:"annotation_overrides"`
	MappingStrategy     GenericMappingStrategy `json:"mapping_strategy"`
	Options             *FieldMappingOptions   `json:"options"`

	// Context metadata
	CreatedAt time.Time `json:"created_at"`
	ContextID string    `json:"context_id"`
}

// NewGenericMappingContext creates a new generic mapping context.
func NewGenericMappingContext(
	sourceType, destType domain.Type,
	typeSubstitutions map[string]domain.Type,
	options *FieldMappingOptions,
) *GenericMappingContext {
	if options == nil {
		options = DefaultFieldMappingOptions()
	}

	return &GenericMappingContext{
		SourceType:          sourceType,
		DestinationType:     destType,
		TypeSubstitutions:   copyTypeSubstitutions(typeSubstitutions),
		AnnotationOverrides: make(map[string]*Annotation),
		MappingStrategy:     SelectOptimalMappingStrategy,
		Options:             options,
		CreatedAt:           time.Now(),
		ContextID:           generateContextID(),
	}
}

// copyTypeSubstitutions creates a defensive copy of type substitutions.
func copyTypeSubstitutions(substitutions map[string]domain.Type) map[string]domain.Type {
	if substitutions == nil {
		return make(map[string]domain.Type)
	}

	result := make(map[string]domain.Type, len(substitutions))
	for k, v := range substitutions {
		result[k] = v
	}
	return result
}

// generateContextID generates a unique context ID.
func generateContextID() string {
	// Simple ID generation - in production, use a proper UUID library
	return fmt.Sprintf("ctx_%d", time.Now().UnixNano())
}

// HasTypeSubstitution checks if a type parameter has a substitution.
func (gmc *GenericMappingContext) HasTypeSubstitution(paramName string) bool {
	_, found := gmc.TypeSubstitutions[paramName]
	return found
}

// GetTypeSubstitution gets a type substitution for a parameter.
func (gmc *GenericMappingContext) GetTypeSubstitution(paramName string) (domain.Type, bool) {
	substitution, found := gmc.TypeSubstitutions[paramName]
	return substitution, found
}

// AddAnnotationOverride adds an annotation override for a field.
func (gmc *GenericMappingContext) AddAnnotationOverride(fieldName string, annotation *Annotation) {
	if gmc.AnnotationOverrides == nil {
		gmc.AnnotationOverrides = make(map[string]*Annotation)
	}
	gmc.AnnotationOverrides[fieldName] = annotation
}

// HasAnnotationOverride checks if a field has an annotation override.
func (gmc *GenericMappingContext) HasAnnotationOverride(fieldName string) bool {
	_, found := gmc.AnnotationOverrides[fieldName]
	return found
}

// GetAnnotationOverride gets an annotation override for a field.
func (gmc *GenericMappingContext) GetAnnotationOverride(fieldName string) (*Annotation, bool) {
	annotation, found := gmc.AnnotationOverrides[fieldName]
	return annotation, found
}

// IsGenericContext returns true if this context involves generic types.
func (gmc *GenericMappingContext) IsGenericContext() bool {
	return len(gmc.TypeSubstitutions) > 0 ||
		gmc.SourceType.Generic() ||
		gmc.DestinationType.Generic()
}

// RequiresTypeSubstitution returns true if type substitution is needed.
func (gmc *GenericMappingContext) RequiresTypeSubstitution() bool {
	return len(gmc.TypeSubstitutions) > 0
}

// GetSubstitutedTypes returns the substituted source and destination types.
func (gmc *GenericMappingContext) GetSubstitutedTypes() (domain.Type, domain.Type) {
	srcType := gmc.SubstitutedSourceType
	if srcType == nil {
		srcType = gmc.SourceType
	}

	dstType := gmc.SubstitutedDestType
	if dstType == nil {
		dstType = gmc.DestinationType
	}

	return srcType, dstType
}

// FieldMapping represents the result of generic field mapping.
type FieldMapping struct {
	SourceType      domain.Type            `json:"source_type"`
	DestinationType domain.Type            `json:"destination_type"`
	Assignments     []*FieldAssignment     `json:"assignments"`
	Context         *GenericMappingContext `json:"context"`
	GeneratedAt     time.Time              `json:"generated_at"`
	MappingID       string                 `json:"mapping_id"`
}

// AssignmentType defines the type of field assignment.
type AssignmentType int

const (
	// DirectAssignment represents direct field assignment.
	DirectAssignment AssignmentType = iota
	// MappedAssignment represents assignment using custom field mapping.
	MappedAssignment
	// ConverterAssignment represents assignment using a converter function.
	ConverterAssignment
	// LiteralAssignment represents assignment using a literal value.
	LiteralAssignment
	// SkipAssignment represents skipping the field assignment.
	SkipAssignment
	// ConversionAssignment represents assignment with type conversion.
	ConversionAssignment
	// SliceAssignment represents assignment for slice types.
	SliceAssignment
	// MapAssignment represents assignment for map types.
	MapAssignment
)

// String returns the string representation of assignment type.
func (at AssignmentType) String() string {
	switch at {
	case DirectAssignment:
		return "direct"
	case MappedAssignment:
		return "mapped"
	case ConverterAssignment:
		return "converter"
	case LiteralAssignment:
		return "literal"
	case SkipAssignment:
		return "skip"
	case ConversionAssignment:
		return "conversion"
	case SliceAssignment:
		return "slice"
	case MapAssignment:
		return "map"
	default:
		return "unknown"
	}
}

// FieldAssignment represents a single field assignment.
type FieldAssignment struct {
	SourceField    *domain.Field  `json:"source_field,omitempty"`
	DestField      *domain.Field  `json:"dest_field"`
	AssignmentType AssignmentType `json:"assignment_type"`
	Code           string         `json:"code"`

	// Optional assignment details
	SourcePath    string `json:"source_path,omitempty"`
	Converter     string `json:"converter,omitempty"`
	Literal       string `json:"literal,omitempty"`
	Validation    string `json:"validation,omitempty"`
	ErrorHandling string `json:"error_handling,omitempty"`

	// Metadata
	GeneratedAt  time.Time `json:"generated_at"`
	AssignmentID string    `json:"assignment_id"`
}

// NewFieldAssignment creates a new field assignment.
func NewFieldAssignment(
	sourceField, destField *domain.Field,
	assignmentType AssignmentType,
	code string,
) *FieldAssignment {
	return &FieldAssignment{
		SourceField:    sourceField,
		DestField:      destField,
		AssignmentType: assignmentType,
		Code:           code,
		GeneratedAt:    time.Now(),
		AssignmentID:   generateAssignmentID(),
	}
}

// generateAssignmentID generates a unique assignment ID.
func generateAssignmentID() string {
	return fmt.Sprintf("assign_%d", time.Now().UnixNano())
}

// IsSkipped returns true if this assignment should be skipped.
func (fa *FieldAssignment) IsSkipped() bool {
	return fa.AssignmentType == SkipAssignment
}

// RequiresErrorHandling returns true if this assignment requires error handling.
func (fa *FieldAssignment) RequiresErrorHandling() bool {
	return fa.AssignmentType == ConverterAssignment && fa.ErrorHandling != ""
}

// IsSimple returns true if this is a simple direct assignment.
func (fa *FieldAssignment) IsSimple() bool {
	return fa.AssignmentType == DirectAssignment &&
		fa.Converter == "" &&
		fa.Validation == ""
}

// GetSourceFieldName returns the source field name, handling nil case.
func (fa *FieldAssignment) GetSourceFieldName() string {
	if fa.SourceField != nil {
		return fa.SourceField.Name
	}
	return ""
}

// GetDestFieldName returns the destination field name.
func (fa *FieldAssignment) GetDestFieldName() string {
	if fa.DestField != nil {
		return fa.DestField.Name
	}
	return ""
}

// GetAssignmentSummary returns a summary of the assignment for logging.
func (fa *FieldAssignment) GetAssignmentSummary() string {
	switch fa.AssignmentType {
	case DirectAssignment:
		return fmt.Sprintf("direct: %s -> %s", fa.GetSourceFieldName(), fa.GetDestFieldName())
	case MappedAssignment:
		return fmt.Sprintf("mapped: %s -> %s", fa.SourcePath, fa.GetDestFieldName())
	case ConverterAssignment:
		return fmt.Sprintf("converter: %s(%s) -> %s", fa.Converter, fa.GetSourceFieldName(), fa.GetDestFieldName())
	case LiteralAssignment:
		return fmt.Sprintf("literal: %s -> %s", fa.Literal, fa.GetDestFieldName())
	case SkipAssignment:
		return fmt.Sprintf("skip: %s", fa.GetDestFieldName())
	default:
		return fmt.Sprintf("%s: %s -> %s", fa.AssignmentType.String(), fa.GetSourceFieldName(), fa.GetDestFieldName())
	}
}

// Validate validates the field assignment.
func (fa *FieldAssignment) Validate() error {
	if fa.DestField == nil {
		return fmt.Errorf("destination field cannot be nil")
	}

	if fa.Code == "" && fa.AssignmentType != SkipAssignment {
		return fmt.Errorf("assignment code cannot be empty for non-skip assignments")
	}

	switch fa.AssignmentType {
	case DirectAssignment, ConversionAssignment:
		if fa.SourceField == nil {
			return fmt.Errorf("source field cannot be nil for %s assignments", fa.AssignmentType.String())
		}
	case MappedAssignment:
		if fa.SourcePath == "" {
			return fmt.Errorf("source path cannot be empty for mapped assignments")
		}
	case ConverterAssignment:
		if fa.Converter == "" {
			return fmt.Errorf("converter cannot be empty for converter assignments")
		}
	case LiteralAssignment:
		if fa.Literal == "" {
			return fmt.Errorf("literal value cannot be empty for literal assignments")
		}
	}

	return nil
}

// GetRequiredImports returns any imports required for this assignment.
func (fa *FieldAssignment) GetRequiredImports() []string {
	var imports []string

	// Add imports for source field type
	if fa.SourceField != nil && fa.SourceField.Type.Package() != "" {
		imports = append(imports, fa.SourceField.Type.Package())
	}

	// Add imports for destination field type
	if fa.DestField != nil && fa.DestField.Type.Package() != "" {
		imports = append(imports, fa.DestField.Type.Package())
	}

	// Add imports for converter function
	if fa.Converter != "" && strings.Contains(fa.Converter, ".") {
		// Extract package from converter function name
		parts := strings.Split(fa.Converter, ".")
		if len(parts) > 1 {
			pkg := strings.Join(parts[:len(parts)-1], ".")
			imports = append(imports, pkg)
		}
	}

	return imports
}

// GetFieldMappingStatistics returns statistics about the field mapping.
func (fm *FieldMapping) GetFieldMappingStatistics() *FieldMappingStatistics {
	stats := &FieldMappingStatistics{
		TotalAssignments: len(fm.Assignments),
	}

	for _, assignment := range fm.Assignments {
		switch assignment.AssignmentType {
		case DirectAssignment:
			stats.DirectAssignments++
		case MappedAssignment:
			stats.MappedAssignments++
		case ConverterAssignment:
			stats.ConverterAssignments++
		case LiteralAssignment:
			stats.LiteralAssignments++
		case SkipAssignment:
			stats.SkippedAssignments++
		case ConversionAssignment:
			stats.ConversionAssignments++
		case SliceAssignment:
			stats.SliceAssignments++
		case MapAssignment:
			stats.MapAssignments++
		}

		if assignment.RequiresErrorHandling() {
			stats.ErrorHandlingAssignments++
		}

		if assignment.Validation != "" {
			stats.ValidationAssignments++
		}
	}

	return stats
}

// FieldMappingStatistics provides statistics about field mapping results.
type FieldMappingStatistics struct {
	TotalAssignments         int `json:"total_assignments"`
	DirectAssignments        int `json:"direct_assignments"`
	MappedAssignments        int `json:"mapped_assignments"`
	ConverterAssignments     int `json:"converter_assignments"`
	LiteralAssignments       int `json:"literal_assignments"`
	SkippedAssignments       int `json:"skipped_assignments"`
	ConversionAssignments    int `json:"conversion_assignments"`
	SliceAssignments         int `json:"slice_assignments"`
	MapAssignments           int `json:"map_assignments"`
	ErrorHandlingAssignments int `json:"error_handling_assignments"`
	ValidationAssignments    int `json:"validation_assignments"`
}

// GetAssignmentEfficiency returns the efficiency of the mapping (percentage of direct assignments).
func (fms *FieldMappingStatistics) GetAssignmentEfficiency() float64 {
	if fms.TotalAssignments == 0 {
		return 0.0
	}
	return float64(fms.DirectAssignments) / float64(fms.TotalAssignments) * 100.0
}

// GetComplexityScore returns a complexity score for the mapping.
func (fms *FieldMappingStatistics) GetComplexityScore() float64 {
	if fms.TotalAssignments == 0 {
		return 0.0
	}

	// Simple scoring: direct assignments = 1, others = 2-4 points
	score := float64(fms.DirectAssignments) +
		float64(fms.MappedAssignments)*2 +
		float64(fms.ConverterAssignments)*3 +
		float64(fms.ConversionAssignments)*2 +
		float64(fms.SliceAssignments)*3 +
		float64(fms.MapAssignments)*4 +
		float64(fms.LiteralAssignments)*1 +
		float64(fms.ErrorHandlingAssignments)*2 +
		float64(fms.ValidationAssignments)*2

	return score / float64(fms.TotalAssignments)
}
