package generator

import (
	"errors"
	"fmt"
	"time"

	"github.com/reedom/convergen/v9/pkg/domain"
)

var (
	// ErrTemplateNameCannotBeEmpty is returned when a template name is empty.
	ErrTemplateNameCannotBeEmpty = errors.New("template name cannot be empty")
	// ErrTemplateContentCannotBeEmpty is returned when template content is empty.
	ErrTemplateContentCannotBeEmpty = errors.New("template content cannot be empty")
	// ErrFunctionNameCannotBeEmpty is returned when a function name is empty.
	ErrFunctionNameCannotBeEmpty = errors.New("function name cannot be empty")
	// ErrFunctionCannotBeNil is returned when a function is nil.
	ErrFunctionCannotBeNil = errors.New("function cannot be nil")
)

// BaseTemplateData provides basic template data without emitter dependency.
type BaseTemplateData struct {
	Package         string                 `json:"package"`
	Imports         []*ImportInfo          `json:"imports"`
	Metadata        map[string]interface{} `json:"metadata"`
	HelperFunctions map[string]interface{} `json:"helper_functions"`
}

// GenericTemplateData extends the base TemplateData with generic-specific information.
type GenericTemplateData struct {
	BaseTemplateData // Embed basic template data

	// Generic-specific data
	TypeParameters    []TypeParam                 `json:"type_parameters"`    // Type parameters from the generic interface
	TypeArguments     []TypeArg                   `json:"type_arguments"`     // Concrete type arguments
	TypeSubstitutions map[string]TypeSubstitution `json:"type_substitutions"` // Mapping from type params to concrete types
	IsGenericFlag     bool                        `json:"is_generic"`         // Whether this is a generic instantiation
	Methods           []*MethodData               `json:"methods"`            // Methods to generate
	SourceInterface   domain.Type                 `json:"source_interface"`   // Original generic interface
	ConcreteType      domain.Type                 `json:"concrete_type"`      // Instantiated concrete type
	TypeSignature     string                      `json:"type_signature"`     // Unique type signature
}

// MethodTemplateData contains template data for a specific method.
type MethodTemplateData struct {
	*GenericTemplateData                 // Parent template data
	Method               *MethodData     `json:"method"`         // Current method being generated
	FieldMappings        []*FieldMapping `json:"field_mappings"` // Field mappings for this method
}

// TypeParam represents a type parameter in the generic interface.
type TypeParam struct {
	Name       string `json:"name"`       // Type parameter name (e.g., "T", "K", "V")
	Constraint string `json:"constraint"` // Constraint (e.g., "any", "comparable")
	Position   int    `json:"position"`   // Position in type parameter list
	Used       bool   `json:"used"`       // Whether this parameter is used in the method
}

// TypeArg represents a concrete type argument.
type TypeArg struct {
	Name        string      `json:"name"`         // Type name (e.g., "User", "string")
	Type        domain.Type `json:"type"`         // Actual type object
	PackagePath string      `json:"package_path"` // Package path if external type
	IsPointer   bool        `json:"is_pointer"`   // Whether this is a pointer type
	IsSlice     bool        `json:"is_slice"`     // Whether this is a slice type
	IsGeneric   bool        `json:"is_generic"`   // Whether this type itself is generic
}

// TypeSubstitution represents a substitution from type parameter to concrete type.
type TypeSubstitution struct {
	ParameterName string      `json:"parameter_name"` // Original type parameter name
	ConcreteType  domain.Type `json:"concrete_type"`  // Concrete type to substitute
	PackagePath   string      `json:"package_path"`   // Package path for imports
	Alias         string      `json:"alias"`          // Type alias if needed
}

// MethodData represents method information for template generation.
type MethodData struct {
	Name         string            `json:"name"`          // Method name
	Parameters   []*ParameterData  `json:"parameters"`    // Method parameters
	ReturnType   domain.Type       `json:"return_type"`   // Return type
	ReturnsError bool              `json:"returns_error"` // Whether method returns error
	Annotations  map[string]string `json:"annotations"`   // Method annotations
	Comments     []string          `json:"comments"`      // Method comments
	Receiver     *ReceiverData     `json:"receiver"`      // Receiver information
}

// ParameterData represents method parameter information.
type ParameterData struct {
	Name     string      `json:"name"`     // Parameter name
	Type     domain.Type `json:"type"`     // Parameter type
	Optional bool        `json:"optional"` // Whether parameter is optional
	Variadic bool        `json:"variadic"` // Whether parameter is variadic
}

// ReceiverData represents method receiver information.
type ReceiverData struct {
	Name      string      `json:"name"`       // Receiver variable name
	Type      domain.Type `json:"type"`       // Receiver type
	IsPointer bool        `json:"is_pointer"` // Whether receiver is pointer
}

// TemplateDefinitions contains all template definitions for generic code generation.
type TemplateDefinitions struct {
	// Basic templates
	BasicMethod       string `json:"basic_method"`
	MethodWithError   string `json:"method_with_error"`
	SimpleConversion  string `json:"simple_conversion"`
	ComplexConversion string `json:"complex_conversion"`

	// Generic-specific templates
	GenericBasicMethod       string `json:"generic_basic_method"`
	GenericMethodWithError   string `json:"generic_method_with_error"`
	GenericSimpleConversion  string `json:"generic_simple_conversion"`
	GenericComplexConversion string `json:"generic_complex_conversion"`

	// Field mapping templates
	FieldAssignment string `json:"field_assignment"`
	FieldConversion string `json:"field_conversion"`
	FieldValidation string `json:"field_validation"`

	// Error handling templates
	ErrorCheck   string `json:"error_check"`
	ErrorWrapper string `json:"error_wrapper"`
	ErrorReturn  string `json:"error_return"`

	// Utility templates
	TypeDeclaration   string `json:"type_declaration"`
	ImportDeclaration string `json:"import_declaration"`
	MethodSignature   string `json:"method_signature"`
}

// GetDefaultTemplateDefinitions returns the default template definitions.
func GetDefaultTemplateDefinitions() *TemplateDefinitions {
	return &TemplateDefinitions{
		GenericBasicMethod: `
// {{.Method.Name}} converts {{.Method.Parameters.0.Type.Name}} to {{.Method.ReturnType.Name}}.
func ({{.Method.Receiver.Name}} *{{.Method.Receiver.Type.Name}}) {{.Method.Name}}({{range $i, $p := .Method.Parameters}}{{if $i}}, {{end}}{{$p.Name}} {{substituteType $p.Type $.TypeSubstitutions}}{{end}}) {{if .Method.ReturnType}}{{substituteType .Method.ReturnType $.TypeSubstitutions}}{{end}} {
	{{if .Method.ReturnType}}var result {{substituteType .Method.ReturnType $.TypeSubstitutions}}

	{{range .FieldMappings}}{{generateFieldAccess .}}
	{{end}}

	return result{{end}}
}`,

		GenericMethodWithError: `
// {{.Method.Name}} converts {{.Method.Parameters.0.Type.Name}} to {{.Method.ReturnType.Name}}.
func ({{.Method.Receiver.Name}} *{{.Method.Receiver.Type.Name}}) {{.Method.Name}}({{range $i, $p := .Method.Parameters}}{{if $i}}, {{end}}{{$p.Name}} {{substituteType $p.Type $.TypeSubstitutions}}{{end}}) ({{if .Method.ReturnType}}{{substituteType .Method.ReturnType $.TypeSubstitutions}}, {{end}}error) {
	{{if .Method.ReturnType}}var result {{substituteType .Method.ReturnType $.TypeSubstitutions}}

	{{range .FieldMappings}}{{generateFieldAccess .}}
	{{end}}

	return result, nil{{else}}return nil{{end}}
}`,

		GenericSimpleConversion: `
// {{.Method.Name}} converts {{.Method.Parameters.0.Type.Name}} to {{.Method.ReturnType.Name}}.
func ({{.Method.Receiver.Name}} *{{.Method.Receiver.Type.Name}}) {{.Method.Name}}(src {{substituteType .Method.Parameters.0.Type $.TypeSubstitutions}}) {{if .Method.ReturnsError}}({{end}}{{substituteType .Method.ReturnType $.TypeSubstitutions}}{{if .Method.ReturnsError}}, error){{end}} {
	{{if .Method.ReturnType}}var dst {{substituteType .Method.ReturnType $.TypeSubstitutions}}

	{{range .FieldMappings}}dst.{{.DestField}} = src.{{.SourceField}}{{if .Converter}}
	// Apply converter: {{.Converter}}{{end}}
	{{end}}

	return dst{{if .Method.ReturnsError}}, nil{{end}}{{else}}{{if .Method.ReturnsError}}return nil{{end}}{{end}}
}`,

		GenericComplexConversion: `
// {{.Method.Name}} converts {{.Method.Parameters.0.Type.Name}} to {{.Method.ReturnType.Name}} with advanced mapping.
func ({{.Method.Receiver.Name}} *{{.Method.Receiver.Type.Name}}) {{.Method.Name}}(src {{substituteType .Method.Parameters.0.Type $.TypeSubstitutions}}) ({{substituteType .Method.ReturnType $.TypeSubstitutions}}, error) {
	var dst {{substituteType .Method.ReturnType $.TypeSubstitutions}}

	{{range .FieldMappings}}// Map {{.SourceField}} -> {{.DestField}}
	{{if .Validation}}if {{.Validation}} {
		return dst, fmt.Errorf("validation failed for field {{.SourceField}}")
	}{{end}}
	{{if .Converter}}converted, err := {{.Converter}}(src.{{.SourceField}})
	if err != nil {
		return dst, fmt.Errorf("conversion failed for field {{.SourceField}}: %w", err)
	}
	dst.{{.DestField}} = converted{{else}}dst.{{.DestField}} = src.{{.SourceField}}{{end}}

	{{end}}
	return dst, nil
}`,

		FieldAssignment: `dst.{{.DestField}} = src.{{.SourceField}}`,

		FieldConversion: `{{if .Converter}}converted, err := {{.Converter}}(src.{{.SourceField}})
if err != nil {
	return {{if .Method.ReturnType}}dst, {{end}}fmt.Errorf("conversion failed for field {{.SourceField}}: %w", err)
}
dst.{{.DestField}} = converted{{else}}dst.{{.DestField}} = src.{{.SourceField}}{{end}}`,

		FieldValidation: `{{if .Validation}}if {{.Validation}} {
	return {{if .Method.ReturnType}}dst, {{end}}fmt.Errorf("validation failed for field {{.SourceField}}")
}{{end}}`,

		ErrorCheck: `if err != nil {
	return {{if .Method.ReturnType}}dst, {{end}}fmt.Errorf("{{.Method.Name}} failed: %w", err)
}`,

		ErrorWrapper: `if err != nil {
	return {{if .Method.ReturnType}}dst, {{end}}fmt.Errorf("{{.Phase}} failed in {{.Method.Name}}: %w", err)
}`,

		ErrorReturn: `return {{if .Method.ReturnType}}dst, {{end}}err`,

		TypeDeclaration: `type {{.Name}} {{.Type}}`,

		ImportDeclaration: `import {{if .Alias}}"{{.Alias}}" {{end}}"{{.Path}}"`,

		MethodSignature: `func ({{.Receiver.Name}} {{if .Receiver.IsPointer}}*{{end}}{{.Receiver.Type.Name}}) {{.Name}}({{range $i, $p := .Parameters}}{{if $i}}, {{end}}{{$p.Name}} {{$p.Type.Name}}{{end}}) {{if .ReturnType}}{{.ReturnType.Name}}{{end}}{{if .ReturnsError}}{{if .ReturnType}}, {{end}}error{{end}}`,
	}
}

// TemplateContext provides context for template execution.
type TemplateContext struct {
	Data         interface{}            `json:"data"`          // Template data
	Functions    map[string]interface{} `json:"functions"`     // Template functions
	Metadata     map[string]interface{} `json:"metadata"`      // Additional metadata
	GeneratedAt  time.Time              `json:"generated_at"`  // Generation timestamp
	TemplateName string                 `json:"template_name"` // Template being executed
	Depth        int                    `json:"depth"`         // Template execution depth
}

// TemplateError represents a template execution error.
type TemplateError struct {
	Template  string                 `json:"template"`  // Template name
	Message   string                 `json:"message"`   // Error message
	Line      int                    `json:"line"`      // Line number if available
	Column    int                    `json:"column"`    // Column number if available
	Context   *TemplateContext       `json:"context"`   // Template context
	Cause     error                  `json:"cause"`     // Underlying error
	Data      map[string]interface{} `json:"data"`      // Additional error data
	Timestamp time.Time              `json:"timestamp"` // Error timestamp
}

// Error implements the error interface.
func (te *TemplateError) Error() string {
	return fmt.Sprintf("template error in %s: %s", te.Template, te.Message)
}

// Unwrap returns the underlying error.
func (te *TemplateError) Unwrap() error {
	return te.Cause
}

// NewTemplateError creates a new template error.
func NewTemplateError(template, message string, cause error) *TemplateError {
	return &TemplateError{
		Template:  template,
		Message:   message,
		Cause:     cause,
		Data:      make(map[string]interface{}),
		Timestamp: time.Now(),
	}
}

// TemplateRegistry manages template registration and retrieval.
type TemplateRegistry struct {
	templates        map[string]string
	functions        map[string]interface{}
	defaultTemplates *TemplateDefinitions
}

// NewTemplateRegistry creates a new template registry.
func NewTemplateRegistry() *TemplateRegistry {
	registry := &TemplateRegistry{
		templates:        make(map[string]string),
		functions:        make(map[string]interface{}),
		defaultTemplates: GetDefaultTemplateDefinitions(),
	}

	// Register default templates
	registry.registerDefaultTemplates()

	return registry
}

// registerDefaultTemplates registers all default templates.
func (tr *TemplateRegistry) registerDefaultTemplates() {
	templates := map[string]string{
		"generic_method_basic":       tr.defaultTemplates.GenericBasicMethod,
		"generic_method_with_error":  tr.defaultTemplates.GenericMethodWithError,
		"generic_simple_conversion":  tr.defaultTemplates.GenericSimpleConversion,
		"generic_complex_conversion": tr.defaultTemplates.GenericComplexConversion,
		"field_assignment":           tr.defaultTemplates.FieldAssignment,
		"field_conversion":           tr.defaultTemplates.FieldConversion,
		"field_validation":           tr.defaultTemplates.FieldValidation,
		"error_check":                tr.defaultTemplates.ErrorCheck,
		"error_wrapper":              tr.defaultTemplates.ErrorWrapper,
		"error_return":               tr.defaultTemplates.ErrorReturn,
		"type_declaration":           tr.defaultTemplates.TypeDeclaration,
		"import_declaration":         tr.defaultTemplates.ImportDeclaration,
		"method_signature":           tr.defaultTemplates.MethodSignature,
	}

	for name, template := range templates {
		tr.templates[name] = template
	}
}

// RegisterTemplate registers a new template.
func (tr *TemplateRegistry) RegisterTemplate(name, content string) error {
	if name == "" {
		return ErrTemplateNameCannotBeEmpty
	}

	if content == "" {
		return ErrTemplateContentCannotBeEmpty
	}

	tr.templates[name] = content
	return nil
}

// GetTemplate retrieves a template by name.
func (tr *TemplateRegistry) GetTemplate(name string) (string, bool) {
	template, found := tr.templates[name]
	return template, found
}

// HasTemplate checks if a template exists.
func (tr *TemplateRegistry) HasTemplate(name string) bool {
	_, found := tr.templates[name]
	return found
}

// ListTemplates returns all registered template names.
func (tr *TemplateRegistry) ListTemplates() []string {
	names := make([]string, 0, len(tr.templates))
	for name := range tr.templates {
		names = append(names, name)
	}
	return names
}

// RegisterFunction registers a template function.
func (tr *TemplateRegistry) RegisterFunction(name string, fn interface{}) error {
	if name == "" {
		return ErrFunctionNameCannotBeEmpty
	}

	if fn == nil {
		return ErrFunctionCannotBeNil
	}

	tr.functions[name] = fn
	return nil
}

// GetFunctions returns all registered template functions.
func (tr *TemplateRegistry) GetFunctions() map[string]interface{} {
	functions := make(map[string]interface{})
	for name, fn := range tr.functions {
		functions[name] = fn
	}
	return functions
}

// IsGeneric returns whether the template data represents a generic instantiation.
func (gtd *GenericTemplateData) IsGeneric() bool {
	return gtd.IsGenericFlag
}

// GetTypeSubstitution gets a type substitution by parameter name.
func (gtd *GenericTemplateData) GetTypeSubstitution(paramName string) (TypeSubstitution, bool) {
	substitution, found := gtd.TypeSubstitutions[paramName]
	return substitution, found
}

// GetTypeArgumentByName gets a type argument by name.
func (gtd *GenericTemplateData) GetTypeArgumentByName(name string) (*TypeArg, bool) {
	for _, arg := range gtd.TypeArguments {
		if arg.Name == name {
			return &arg, true
		}
	}
	return nil, false
}

// GetTypeParameterByName gets a type parameter by name.
func (gtd *GenericTemplateData) GetTypeParameterByName(name string) (*TypeParam, bool) {
	for _, param := range gtd.TypeParameters {
		if param.Name == name {
			return &param, true
		}
	}
	return nil, false
}

// HasAnnotation checks if a method has a specific annotation.
func (md *MethodData) HasAnnotation(key string) bool {
	_, found := md.Annotations[key]
	return found
}

// GetAnnotation gets an annotation value.
func (md *MethodData) GetAnnotation(key string) (string, bool) {
	value, found := md.Annotations[key]
	return value, found
}

// IsPointer checks if the parameter type is a pointer.
func (pd *ParameterData) IsPointer() bool {
	return pd.Type.Kind() == domain.KindPointer
}

// IsSlice checks if the parameter type is a slice.
func (pd *ParameterData) IsSlice() bool {
	return pd.Type.Kind() == domain.KindSlice
}

// RequiresImport checks if the type argument requires an import.
func (ta *TypeArg) RequiresImport() bool {
	return ta.PackagePath != ""
}

// GetImportPath returns the import path for the type argument.
func (ta *TypeArg) GetImportPath() string {
	return ta.PackagePath
}

// GetQualifiedName returns the fully qualified name for the type argument.
func (ta *TypeArg) GetQualifiedName() string {
	if ta.PackagePath != "" {
		return ta.PackagePath + "." + ta.Name
	}
	return ta.Name
}

// NeedsValidation checks if the field mapping requires validation.
func (fm *FieldMapping) NeedsValidation() bool {
	return fm.Validation != ""
}

// NeedsConversion checks if the field mapping requires conversion.
func (fm *FieldMapping) NeedsConversion() bool {
	return fm.Converter != ""
}

// IsSimpleAssignment checks if this is a simple field assignment.
func (fm *FieldMapping) IsSimpleAssignment() bool {
	return fm.Converter == "" && fm.Validation == ""
}

// GetConversionFunction returns the conversion function name.
func (fm *FieldMapping) GetConversionFunction() string {
	return fm.Converter
}

// GetValidationExpression returns the validation expression.
func (fm *FieldMapping) GetValidationExpression() string {
	return fm.Validation
}
