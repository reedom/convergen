package domain

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Static errors for err113 compliance.
var (
	ErrFieldNameEmpty             = errors.New("field name cannot be empty")
	ErrFieldTypeNil               = errors.New("field type cannot be nil")
	ErrFieldPathEmpty             = errors.New("field path cannot be empty")
	ErrMappingIDEmpty             = errors.New("mapping ID cannot be empty")
	ErrSourceFieldSpecNil         = errors.New("source field spec cannot be nil")
	ErrDestinationFieldSpecNil    = errors.New("destination field spec cannot be nil")
	ErrConversionStrategyNil      = errors.New("conversion strategy cannot be nil")
	ErrDependencyIDEmpty          = errors.New("dependency ID cannot be empty")
	ErrFieldMappingSelfReference  = errors.New("field mapping cannot depend on itself")
	ErrSourceNotMethodCall        = errors.New("source is not a method call")
	ErrConverterFunctionNotConfig = errors.New("converter function not configured")
	ErrLiteralValueNotConfig      = errors.New("literal value not configured")
)

// Constants for goconst compliance.
const (
	// DirectStrategyType represents the direct assignment strategy type.
	DirectStrategyType = "direct"
	// TypeCastStrategyType represents the type cast strategy type.
	TypeCastStrategyType = "typecast"
	// MethodStrategyType represents the method call strategy type.
	MethodStrategyType = "method"
	// ConverterStrategyType represents the converter function strategy type.
	ConverterStrategyType = "converter"
	// LiteralStrategyType represents the literal value strategy type.
	LiteralStrategyType = "literal"
	// ExpressionStrategyType represents the expression evaluation strategy type.
	ExpressionStrategyType = "expression"
	// CustomStrategyType represents the custom strategy type.
	CustomStrategyType = "custom"
)

// Field represents a struct field with complete metadata.
type Field struct {
	Name      string            `json:"name"`
	Type      Type              `json:"type"`
	Tag       string            `json:"tag"`
	Tags      reflect.StructTag `json:"tags"`
	Position  int               `json:"position"` // For ordering preservation
	Exported  bool              `json:"exported"`
	Embedded  bool              `json:"embedded"`
	Anonymous bool              `json:"anonymous"`
	Doc       string            `json:"doc"`
}

// NewField creates a new field with validation.
func NewField(name string, typ Type, position int, exported bool) (*Field, error) {
	if name == "" {
		return nil, ErrFieldNameEmpty
	}

	if typ == nil {
		return nil, ErrFieldTypeNil
	}

	return &Field{
		Name:     name,
		Type:     typ,
		Position: position,
		Exported: exported,
		Tags:     reflect.StructTag(""),
		Doc:      "",
	}, nil
}

// FieldSpec represents a field path specification.
// Examples: ["User", "Name"], ["User", "GetAddress", "Street"].
type FieldSpec struct {
	Path     []string `json:"path"`
	Type     Type     `json:"type"`
	IsMethod bool     `json:"is_method"` // true for getter methods
	Receiver Type     `json:"receiver"`  // for method calls
}

// NewFieldSpec creates a validated field specification.
func NewFieldSpec(path []string, typ Type) (*FieldSpec, error) {
	if len(path) == 0 {
		return nil, ErrFieldPathEmpty
	}

	if typ == nil {
		return nil, ErrFieldTypeNil
	}

	return &FieldSpec{
		Path:     append([]string(nil), path...), // defensive copy
		Type:     typ,
		IsMethod: false,
		Receiver: nil,
	}, nil
}

// NewMethodSpec creates a method-based field specification.
func NewMethodSpec(path []string, typ Type, receiver Type) (*FieldSpec, error) {
	spec, err := NewFieldSpec(path, typ)
	if err != nil {
		return nil, err
	}

	spec.IsMethod = true
	spec.Receiver = receiver

	return spec, nil
}

// String returns a human-readable representation.
func (fs *FieldSpec) String() string {
	path := strings.Join(fs.Path, ".")
	if fs.IsMethod {
		return path + "()"
	}

	return path
}

// FieldName returns the final field/method name.
func (fs *FieldSpec) FieldName() string {
	if len(fs.Path) == 0 {
		return ""
	}

	return fs.Path[len(fs.Path)-1]
}

// ParentPath returns the path to the parent struct.
func (fs *FieldSpec) ParentPath() []string {
	if len(fs.Path) <= 1 {
		return nil
	}

	return append([]string(nil), fs.Path[:len(fs.Path)-1]...)
}

// ConversionStrategy defines how to convert between field types.
type ConversionStrategy interface {
	Name() string
	CanHandle(source, dest Type) bool
	GenerateCode(mapping *FieldMapping) (*GeneratedCode, error)
	Dependencies() []string
	Priority() int // Higher priority strategies are preferred
}

// FieldMapping represents a conversion between two fields.
type FieldMapping struct {
	ID           string             `json:"id"`
	Source       *FieldSpec         `json:"source"`
	Dest         *FieldSpec         `json:"dest"`
	Strategy     ConversionStrategy `json:"-"` // Not serialized due to interface
	StrategyName string             `json:"strategy_name"`
	Config       *MappingConfig     `json:"config"`
	Dependencies []string           `json:"dependencies"` // Field IDs this mapping depends on
}

// NewFieldMapping creates a validated field mapping.
func NewFieldMapping(id string, source, dest *FieldSpec, strategy ConversionStrategy) (*FieldMapping, error) {
	if id == "" {
		return nil, ErrMappingIDEmpty
	}

	if source == nil {
		return nil, ErrSourceFieldSpecNil
	}

	if dest == nil {
		return nil, ErrDestinationFieldSpecNil
	}

	if strategy == nil {
		return nil, ErrConversionStrategyNil
	}

	return &FieldMapping{
		ID:           id,
		Source:       source,
		Dest:         dest,
		Strategy:     strategy,
		StrategyName: strategy.Name(),
		Config:       &MappingConfig{},
		Dependencies: make([]string, 0),
	}, nil
}

// AddDependency adds a dependency to another field mapping.
func (fm *FieldMapping) AddDependency(dependencyID string) error {
	if dependencyID == "" {
		return ErrDependencyIDEmpty
	}

	if dependencyID == fm.ID {
		return ErrFieldMappingSelfReference
	}

	// Check if dependency already exists
	for _, dep := range fm.Dependencies {
		if dep == dependencyID {
			return nil // Already exists, no error
		}
	}

	fm.Dependencies = append(fm.Dependencies, dependencyID)

	return nil
}

// MappingConfig holds configuration for a specific mapping.
type MappingConfig struct {
	Skip         bool                   `json:"skip"`
	Converter    *ConverterFunc         `json:"converter"`
	Literal      *LiteralValue          `json:"literal"`
	ErrorHandler ErrorHandlingStrategy  `json:"error_handler"`
	Custom       map[string]interface{} `json:"custom"` // For strategy-specific config
}

// ConverterFunc represents a custom converter function.
type ConverterFunc struct {
	Name       string   `json:"name"`
	Package    string   `json:"package"`
	ImportPath string   `json:"import_path"`
	Args       []string `json:"args"` // Additional arguments
	ReturnsErr bool     `json:"returns_err"`
}

// LiteralValue represents a literal value assignment.
type LiteralValue struct {
	Value string `json:"value"`
	Type  Type   `json:"type"`
}

// ErrorHandlingStrategy defines how to handle errors in conversions.
type ErrorHandlingStrategy int

const (
	// ErrorIgnore represents ignoring errors during field mapping.
	ErrorIgnore ErrorHandlingStrategy = iota
	// ErrorPropagate represents propagating errors to the caller.
	ErrorPropagate
	// ErrorPanic represents panicking on errors.
	ErrorPanic
	// ErrorDefault represents using default error handling.
	ErrorDefault
)

func (e ErrorHandlingStrategy) String() string {
	switch e {
	case ErrorIgnore:
		return "ignore"
	case ErrorPropagate:
		return "propagate"
	case ErrorPanic:
		return "panic"
	case ErrorDefault:
		return "default"
	default:
		return UnknownValue
	}
}

// GeneratedCode represents generated code for a field conversion.
type GeneratedCode struct {
	Assignment string           `json:"assignment"`
	Variables  []VarDeclaration `json:"variables"`
	Imports    []Import         `json:"imports"`
	PreCode    []string         `json:"pre_code"`
	PostCode   []string         `json:"post_code"`
	Error      *ErrorHandling   `json:"error"`
}

// VarDeclaration represents a variable declaration.
type VarDeclaration struct {
	Name string `json:"name"`
	Type Type   `json:"type"`
	Init string `json:"init"`
}

// Import represents an import statement.
type Import struct {
	Path  string `json:"path"`
	Alias string `json:"alias"`
}

// ErrorHandling represents error handling code.
type ErrorHandling struct {
	Variable string `json:"variable"`
	Check    string `json:"check"`
	Action   string `json:"action"`
	Message  string `json:"message"`
}

// Built-in conversion strategies

// DirectAssignmentStrategy handles direct field assignments.
type DirectAssignmentStrategy struct{}

// Name returns the strategy name.
func (s *DirectAssignmentStrategy) Name() string { return DirectStrategyType }

// Priority returns the strategy priority.
func (s *DirectAssignmentStrategy) Priority() int { return 100 }

// Dependencies returns strategy dependencies.
func (s *DirectAssignmentStrategy) Dependencies() []string { return nil }

// CanHandle checks if the strategy can handle the conversion.
func (s *DirectAssignmentStrategy) CanHandle(source, dest Type) bool {
	return source.AssignableTo(dest)
}

// GenerateCode generates the conversion code.
func (s *DirectAssignmentStrategy) GenerateCode(mapping *FieldMapping) (*GeneratedCode, error) {
	sourceAccess := strings.Join(mapping.Source.Path, ".")
	destAccess := strings.Join(mapping.Dest.Path, ".")

	assignment := fmt.Sprintf("%s = %s", destAccess, sourceAccess)

	return &GeneratedCode{
		Assignment: assignment,
	}, nil
}

// TypeCastStrategy handles type casting conversions.
type TypeCastStrategy struct{}

// Name returns the strategy name.
func (s *TypeCastStrategy) Name() string { return TypeCastStrategyType }

// Priority returns the strategy priority.
func (s *TypeCastStrategy) Priority() int { return 80 }

// Dependencies returns strategy dependencies.
func (s *TypeCastStrategy) Dependencies() []string { return nil }

// CanHandle checks if the strategy can handle the conversion.
func (s *TypeCastStrategy) CanHandle(source, dest Type) bool {
	// Can cast between compatible basic types
	if source.Kind() == KindBasic && dest.Kind() == KindBasic {
		return true
	}

	// Can cast between named types with same underlying type
	if source.Underlying().AssignableTo(dest.Underlying()) {
		return true
	}

	return false
}

// GenerateCode generates the conversion code for type casting.
func (s *TypeCastStrategy) GenerateCode(mapping *FieldMapping) (*GeneratedCode, error) {
	sourceAccess := strings.Join(mapping.Source.Path, ".")
	destAccess := strings.Join(mapping.Dest.Path, ".")
	destType := mapping.Dest.Type.String()

	assignment := fmt.Sprintf("%s = %s(%s)", destAccess, destType, sourceAccess)

	return &GeneratedCode{
		Assignment: assignment,
	}, nil
}

// MethodCallStrategy handles method-based conversions (getters, stringers).
type MethodCallStrategy struct{}

// Name returns the strategy name.
func (s *MethodCallStrategy) Name() string { return MethodStrategyType }

// Priority returns the strategy priority.
func (s *MethodCallStrategy) Priority() int { return 90 }

// Dependencies returns strategy dependencies.
func (s *MethodCallStrategy) Dependencies() []string { return nil }

// CanHandle checks if the strategy can handle the conversion between source and dest types.
func (s *MethodCallStrategy) CanHandle(source, dest Type) bool {
	// This would be determined by the parser based on available methods
	return false // Implementation depends on method availability analysis
}

// GenerateCode generates code for method call field mapping strategy.
func (s *MethodCallStrategy) GenerateCode(mapping *FieldMapping) (*GeneratedCode, error) {
	if !mapping.Source.IsMethod {
		return nil, ErrSourceNotMethodCall
	}

	// Build method call path
	path := make([]string, 0, len(mapping.Source.Path))

	for i, part := range mapping.Source.Path {
		if i == len(mapping.Source.Path)-1 {
			// Last part is the method call
			path = append(path, part+"()")
		} else {
			path = append(path, part)
		}
	}

	sourceAccess := strings.Join(path, ".")
	destAccess := strings.Join(mapping.Dest.Path, ".")

	assignment := fmt.Sprintf("%s = %s", destAccess, sourceAccess)

	return &GeneratedCode{
		Assignment: assignment,
	}, nil
}

// ConverterFuncStrategy handles custom converter functions.
type ConverterFuncStrategy struct{}

// Name returns the strategy name.
func (s *ConverterFuncStrategy) Name() string { return ConverterStrategyType }

// Priority returns the strategy priority.
func (s *ConverterFuncStrategy) Priority() int { return 70 }

// Dependencies returns the strategy dependencies.
func (s *ConverterFuncStrategy) Dependencies() []string { return nil }

// CanHandle determines if this strategy can handle the given types.
func (s *ConverterFuncStrategy) CanHandle(source, dest Type) bool {
	// Can handle any types if a converter function is configured
	return true
}

// GenerateCode generates code for converter function field mapping strategy.
func (s *ConverterFuncStrategy) GenerateCode(mapping *FieldMapping) (*GeneratedCode, error) {
	if mapping.Config.Converter == nil {
		return nil, ErrConverterFunctionNotConfig
	}

	conv := mapping.Config.Converter
	sourceAccess := strings.Join(mapping.Source.Path, ".")
	destAccess := strings.Join(mapping.Dest.Path, ".")

	// Build function call
	funcCall := fmt.Sprintf("%s(%s", conv.Name, sourceAccess)
	for _, arg := range conv.Args {
		funcCall += ", " + arg
	}

	funcCall += ")"

	code := &GeneratedCode{}

	if conv.ReturnsErr {
		// Handle error return
		errVar := fmt.Sprintf("err_%s", mapping.ID)
		assignment := fmt.Sprintf("%s, %s := %s", destAccess, errVar, funcCall)
		code.Assignment = assignment
		code.Error = &ErrorHandling{
			Variable: errVar,
			Check:    fmt.Sprintf("if %s != nil", errVar),
			Action:   "return",
			Message:  fmt.Sprintf("conversion failed for field %s", mapping.Dest.FieldName()),
		}
	} else {
		assignment := fmt.Sprintf("%s = %s", destAccess, funcCall)
		code.Assignment = assignment
	}

	// Add import if needed
	if conv.ImportPath != "" {
		code.Imports = []Import{{
			Path:  conv.ImportPath,
			Alias: "",
		}}
	}

	return code, nil
}

// LiteralStrategy handles literal value assignments.
type LiteralStrategy struct{}

// Name returns the strategy name.
func (s *LiteralStrategy) Name() string { return LiteralStrategyType }

// Priority returns the strategy priority.
func (s *LiteralStrategy) Priority() int { return 60 }

// Dependencies returns the strategy dependencies.
func (s *LiteralStrategy) Dependencies() []string { return nil }

// CanHandle determines if this strategy can handle the given types.
func (s *LiteralStrategy) CanHandle(source, dest Type) bool {
	// Can assign literals to any compatible type
	return true
}

// GenerateCode generates code for literal field mapping strategy.
func (s *LiteralStrategy) GenerateCode(mapping *FieldMapping) (*GeneratedCode, error) {
	if mapping.Config.Literal == nil {
		return nil, ErrLiteralValueNotConfig
	}

	destAccess := strings.Join(mapping.Dest.Path, ".")
	assignment := fmt.Sprintf("%s = %s", destAccess, mapping.Config.Literal.Value)

	return &GeneratedCode{
		Assignment: assignment,
	}, nil
}

// MappingStrategy is an alias for ConversionStrategy for compatibility with TASK-006.
type MappingStrategy = ConversionStrategy

// GenericDirectAssignmentStrategy handles direct field assignments with generic type awareness.
// This strategy is used for TASK-006 implementation to handle generic type parameter substitution.
type GenericDirectAssignmentStrategy struct {
	InterfaceTypeParams []TypeParam `json:"interface_type_params"`
}

// Name returns the strategy name.
func (s *GenericDirectAssignmentStrategy) Name() string { return "generic_direct" }

// Priority returns the strategy priority (higher than basic direct assignment).
func (s *GenericDirectAssignmentStrategy) Priority() int { return 110 }

// Dependencies returns strategy dependencies.
func (s *GenericDirectAssignmentStrategy) Dependencies() []string { return nil }

// CanHandle checks if the strategy can handle the conversion between generic types.
func (s *GenericDirectAssignmentStrategy) CanHandle(source, dest Type) bool {
	// Can handle direct assignments between types, including generic type parameter substitution
	if source.Generic() || dest.Generic() {
		// For generic types, check constraint compatibility
		return s.areGenericTypesCompatible(source, dest)
	}

	// For concrete types, use standard assignability
	return source.AssignableTo(dest)
}

// GenerateCode generates the conversion code with generic type handling.
func (s *GenericDirectAssignmentStrategy) GenerateCode(mapping *FieldMapping) (*GeneratedCode, error) {
	sourceAccess := strings.Join(mapping.Source.Path, ".")
	destAccess := strings.Join(mapping.Dest.Path, ".")

	// For generic types, we may need type casting or conversion
	if mapping.Source.Type.Generic() || mapping.Dest.Type.Generic() {
		// Generate code that handles type parameter substitution
		// This will be enhanced when type instantiation is integrated
		assignment := fmt.Sprintf("%s = %s // generic type assignment", destAccess, sourceAccess)
		return &GeneratedCode{
			Assignment: assignment,
		}, nil
	}

	// Standard direct assignment for concrete types
	assignment := fmt.Sprintf("%s = %s", destAccess, sourceAccess)
	return &GeneratedCode{
		Assignment: assignment,
	}, nil
}

// areGenericTypesCompatible checks if two types are compatible in a generic context.
func (s *GenericDirectAssignmentStrategy) areGenericTypesCompatible(source, dest Type) bool {
	// If both types are generic, they should be compatible through type parameters
	if source.Generic() && dest.Generic() {
		// Check if they refer to the same type parameter
		return source.String() == dest.String()
	}

	// If one is generic and one is concrete, check constraint satisfaction
	if source.Generic() && !dest.Generic() {
		return s.typeConstraintSatisfied(source, dest)
	}

	if !source.Generic() && dest.Generic() {
		return s.typeConstraintSatisfied(dest, source)
	}

	// Both are concrete types, use regular assignability
	return source.AssignableTo(dest)
}

// typeConstraintSatisfied checks if a concrete type satisfies a generic type's constraints.
func (s *GenericDirectAssignmentStrategy) typeConstraintSatisfied(genericType, concreteType Type) bool {
	// Find the type parameter for the generic type
	for _, param := range s.InterfaceTypeParams {
		if param.Name == genericType.Name() {
			return param.SatisfiesConstraint(concreteType)
		}
	}

	// If type parameter not found, default to allowing the assignment
	return true
}

// DefaultConversionStrategies returns the built-in conversion strategies.
func DefaultConversionStrategies() []ConversionStrategy {
	return []ConversionStrategy{
		&GenericDirectAssignmentStrategy{},
		&DirectAssignmentStrategy{},
		&TypeCastStrategy{},
		&MethodCallStrategy{},
		&ConverterFuncStrategy{},
		&LiteralStrategy{},
	}
}
