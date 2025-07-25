package domain

import (
	"fmt"
	"reflect"
	"strings"
)

// Field represents a struct field with complete metadata
type Field struct {
	Name     string            `json:"name"`
	Type     Type              `json:"type"`
	Tags     reflect.StructTag `json:"tags"`
	Position int               `json:"position"` // For ordering preservation
	Exported bool              `json:"exported"`
	Doc      string            `json:"doc"`
}

// NewField creates a new field with validation
func NewField(name string, typ Type, position int, exported bool) (*Field, error) {
	if name == "" {
		return nil, fmt.Errorf("field name cannot be empty")
	}
	if typ == nil {
		return nil, fmt.Errorf("field type cannot be nil")
	}
	
	return &Field{
		Name:     name,
		Type:     typ,
		Position: position,
		Exported: exported,
	}, nil
}

// FieldSpec identifies a specific field access path
// Examples: ["User", "Name"], ["User", "GetAddress", "Street"]
type FieldSpec struct {
	Path     []string `json:"path"`
	Type     Type     `json:"type"`
	IsMethod bool     `json:"is_method"` // true for getter methods
	Receiver Type     `json:"receiver"`  // for method calls
}

// NewFieldSpec creates a validated field specification
func NewFieldSpec(path []string, typ Type) (*FieldSpec, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("field path cannot be empty")
	}
	if typ == nil {
		return nil, fmt.Errorf("field type cannot be nil")
	}
	
	return &FieldSpec{
		Path:     append([]string(nil), path...), // defensive copy
		Type:     typ,
		IsMethod: false,
		Receiver: nil,
	}, nil
}

// NewMethodSpec creates a method-based field specification
func NewMethodSpec(path []string, typ Type, receiver Type) (*FieldSpec, error) {
	spec, err := NewFieldSpec(path, typ)
	if err != nil {
		return nil, err
	}
	
	spec.IsMethod = true
	spec.Receiver = receiver
	return spec, nil
}

// String returns a human-readable representation
func (fs *FieldSpec) String() string {
	path := strings.Join(fs.Path, ".")
	if fs.IsMethod {
		return path + "()"
	}
	return path
}

// FieldName returns the final field/method name
func (fs *FieldSpec) FieldName() string {
	if len(fs.Path) == 0 {
		return ""
	}
	return fs.Path[len(fs.Path)-1]
}

// ParentPath returns the path to the parent struct
func (fs *FieldSpec) ParentPath() []string {
	if len(fs.Path) <= 1 {
		return nil
	}
	return append([]string(nil), fs.Path[:len(fs.Path)-1]...)
}

// ConversionStrategy defines how to convert between field types
type ConversionStrategy interface {
	Name() string
	CanHandle(source, dest Type) bool
	GenerateCode(mapping *FieldMapping) (*GeneratedCode, error)
	Dependencies() []string
	Priority() int // Higher priority strategies are preferred
}

// FieldMapping represents a conversion between two fields
type FieldMapping struct {
	ID           string              `json:"id"`
	Source       *FieldSpec          `json:"source"`
	Dest         *FieldSpec          `json:"dest"`
	Strategy     ConversionStrategy  `json:"-"` // Not serialized due to interface
	StrategyName string              `json:"strategy_name"`
	Config       *MappingConfig      `json:"config"`
	Dependencies []string            `json:"dependencies"` // Field IDs this mapping depends on
}

// NewFieldMapping creates a validated field mapping
func NewFieldMapping(id string, source, dest *FieldSpec, strategy ConversionStrategy) (*FieldMapping, error) {
	if id == "" {
		return nil, fmt.Errorf("mapping ID cannot be empty")
	}
	if source == nil {
		return nil, fmt.Errorf("source field spec cannot be nil")
	}
	if dest == nil {
		return nil, fmt.Errorf("destination field spec cannot be nil")
	}
	if strategy == nil {
		return nil, fmt.Errorf("conversion strategy cannot be nil")
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

// AddDependency adds a dependency to another field mapping
func (fm *FieldMapping) AddDependency(dependencyID string) error {
	if dependencyID == "" {
		return fmt.Errorf("dependency ID cannot be empty")
	}
	if dependencyID == fm.ID {
		return fmt.Errorf("field mapping cannot depend on itself")
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

// MappingConfig holds configuration for a specific mapping
type MappingConfig struct {
	Skip         bool                     `json:"skip"`
	Converter    *ConverterFunc           `json:"converter"`
	Literal      *LiteralValue            `json:"literal"`
	ErrorHandler ErrorHandlingStrategy    `json:"error_handler"`
	Custom       map[string]interface{}   `json:"custom"` // For strategy-specific config
}

// ConverterFunc represents a custom converter function
type ConverterFunc struct {
	Name       string   `json:"name"`
	Package    string   `json:"package"`
	ImportPath string   `json:"import_path"`
	Args       []string `json:"args"`       // Additional arguments
	ReturnsErr bool     `json:"returns_err"`
}

// LiteralValue represents a literal value assignment
type LiteralValue struct {
	Value string `json:"value"`
	Type  Type   `json:"type"`
}

// ErrorHandlingStrategy defines how to handle errors in conversions
type ErrorHandlingStrategy int

const (
	ErrorIgnore ErrorHandlingStrategy = iota
	ErrorPropagate
	ErrorPanic
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
		return "unknown"
	}
}

// GeneratedCode represents generated code for a field conversion
type GeneratedCode struct {
	Assignment string            `json:"assignment"`
	Variables  []VarDeclaration  `json:"variables"`
	Imports    []Import          `json:"imports"`
	PreCode    []string          `json:"pre_code"`
	PostCode   []string          `json:"post_code"`
	Error      *ErrorHandling    `json:"error"`
}

// VarDeclaration represents a variable declaration
type VarDeclaration struct {
	Name string `json:"name"`
	Type Type   `json:"type"`
	Init string `json:"init"`
}

// Import represents an import statement
type Import struct {
	Path  string `json:"path"`
	Alias string `json:"alias"`
}

// ErrorHandling represents error handling code
type ErrorHandling struct {
	Variable   string `json:"variable"`
	Check      string `json:"check"`
	Action     string `json:"action"`
	Message    string `json:"message"`
}

// Built-in conversion strategies

// DirectAssignmentStrategy handles direct field assignments
type DirectAssignmentStrategy struct{}

func (s *DirectAssignmentStrategy) Name() string { return "direct" }
func (s *DirectAssignmentStrategy) Priority() int { return 100 }
func (s *DirectAssignmentStrategy) Dependencies() []string { return nil }

func (s *DirectAssignmentStrategy) CanHandle(source, dest Type) bool {
	return source.AssignableTo(dest)
}

func (s *DirectAssignmentStrategy) GenerateCode(mapping *FieldMapping) (*GeneratedCode, error) {
	sourceAccess := strings.Join(mapping.Source.Path, ".")
	destAccess := strings.Join(mapping.Dest.Path, ".")
	
	assignment := fmt.Sprintf("%s = %s", destAccess, sourceAccess)
	
	return &GeneratedCode{
		Assignment: assignment,
	}, nil
}

// TypeCastStrategy handles type casting conversions
type TypeCastStrategy struct{}

func (s *TypeCastStrategy) Name() string { return "typecast" }
func (s *TypeCastStrategy) Priority() int { return 80 }
func (s *TypeCastStrategy) Dependencies() []string { return nil }

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

func (s *TypeCastStrategy) GenerateCode(mapping *FieldMapping) (*GeneratedCode, error) {
	sourceAccess := strings.Join(mapping.Source.Path, ".")
	destAccess := strings.Join(mapping.Dest.Path, ".")
	destType := mapping.Dest.Type.String()
	
	assignment := fmt.Sprintf("%s = %s(%s)", destAccess, destType, sourceAccess)
	
	return &GeneratedCode{
		Assignment: assignment,
	}, nil
}

// MethodCallStrategy handles method-based conversions (getters, stringers)
type MethodCallStrategy struct{}

func (s *MethodCallStrategy) Name() string { return "method" }
func (s *MethodCallStrategy) Priority() int { return 90 }
func (s *MethodCallStrategy) Dependencies() []string { return nil }

func (s *MethodCallStrategy) CanHandle(source, dest Type) bool {
	// This would be determined by the parser based on available methods
	return false // Implementation depends on method availability analysis
}

func (s *MethodCallStrategy) GenerateCode(mapping *FieldMapping) (*GeneratedCode, error) {
	if !mapping.Source.IsMethod {
		return nil, fmt.Errorf("source is not a method call")
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

// ConverterFuncStrategy handles custom converter functions
type ConverterFuncStrategy struct{}

func (s *ConverterFuncStrategy) Name() string { return "converter" }
func (s *ConverterFuncStrategy) Priority() int { return 70 }
func (s *ConverterFuncStrategy) Dependencies() []string { return nil }

func (s *ConverterFuncStrategy) CanHandle(source, dest Type) bool {
	// Can handle any types if a converter function is configured
	return true
}

func (s *ConverterFuncStrategy) GenerateCode(mapping *FieldMapping) (*GeneratedCode, error) {
	if mapping.Config.Converter == nil {
		return nil, fmt.Errorf("converter function not configured")
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

// LiteralStrategy handles literal value assignments
type LiteralStrategy struct{}

func (s *LiteralStrategy) Name() string { return "literal" }
func (s *LiteralStrategy) Priority() int { return 60 }
func (s *LiteralStrategy) Dependencies() []string { return nil }

func (s *LiteralStrategy) CanHandle(source, dest Type) bool {
	// Can assign literals to any compatible type
	return true
}

func (s *LiteralStrategy) GenerateCode(mapping *FieldMapping) (*GeneratedCode, error) {
	if mapping.Config.Literal == nil {
		return nil, fmt.Errorf("literal value not configured")
	}
	
	destAccess := strings.Join(mapping.Dest.Path, ".")
	assignment := fmt.Sprintf("%s = %s", destAccess, mapping.Config.Literal.Value)
	
	return &GeneratedCode{
		Assignment: assignment,
	}, nil
}

// DefaultConversionStrategies returns the built-in conversion strategies
func DefaultConversionStrategies() []ConversionStrategy {
	return []ConversionStrategy{
		&DirectAssignmentStrategy{},
		&TypeCastStrategy{},
		&MethodCallStrategy{},
		&ConverterFuncStrategy{},
		&LiteralStrategy{},
	}
}