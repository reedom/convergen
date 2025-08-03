package generator

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v8/pkg/domain"
)

// ExampleUsage demonstrates how to use the generic template system.
// This file shows real-world usage patterns and examples.

// ExampleGenericConverter demonstrates a complete generic converter workflow.
func ExampleGenericConverter() {
	// Step 1: Create a logger
	logger := zap.NewNop()

	// Step 2: Create the type builder and instantiator
	typeBuilder := domain.NewTypeBuilder()
	typeInstantiator := domain.NewTypeInstantiator(typeBuilder, logger)

	// Step 3: Create template engine and field mapper
	templateEngine := &ExampleTemplateEngine{}
	fieldMapper := &ExampleFieldMapper{}

	// Step 4: Create the generic code generator
	config := DefaultGenericGeneratorConfig()
	generator := NewGenericCodeGenerator(
		templateEngine,
		typeInstantiator,
		fieldMapper,
		logger,
		config,
	)

	// Step 5: Create a generic interface definition
	userType := domain.NewBasicType("User", reflect.Struct)
	userDTOType := domain.NewBasicType("UserDTO", reflect.Struct)

	// Create type arguments map
	typeArguments := map[string]domain.Type{
		"T": userType,
		"U": userDTOType,
	}

	// Step 6: Create an instantiated interface
	instantiatedInterface, err := domain.NewInstantiatedInterface(
		domain.NewBasicType("Converter", reflect.Interface),
		typeArguments,
		userType,
		"Converter[User, UserDTO]",
	)
	if err != nil {
		fmt.Printf("Error creating instantiated interface: %v\n", err)
		return
	}

	// Step 7: Generate the generic implementation
	ctx := context.Background()
	generatedCode, err := generator.GenerateGenericImplementation(ctx, instantiatedInterface)
	if err != nil {
		fmt.Printf("Error generating code: %v\n", err)
		return
	}

	fmt.Printf("Generated Code:\n%s\n", generatedCode)

	// Step 8: Get metrics
	metrics := generator.GetMetrics()
	fmt.Printf("Generation Metrics:\n")
	fmt.Printf("  Total Generations: %d\n", metrics.TotalGenerations)
	fmt.Printf("  Successful Generations: %d\n", metrics.SuccessfulGenerations)
	fmt.Printf("  Average Generation Time: %v\n", metrics.AverageGenerationTime)
}

// ExampleAdvancedGenericUsage demonstrates advanced features.
func ExampleAdvancedGenericUsage() {
	logger := zap.NewNop()

	// Create complex type mappings
	userType := domain.NewBasicType("User", reflect.Struct)
	addressType := domain.NewBasicType("Address", reflect.Struct)
	orderType := domain.NewBasicType("Order", reflect.Struct)

	// Create slice and pointer types
	userSliceType := domain.NewSliceType(userType, "")
	userPointerType := domain.NewPointerType(userType, "")

	typeArguments := map[string]domain.Type{
		"TUser":    userType,
		"TAddress": addressType,
		"TOrder":   orderType,
		"TUsers":   userSliceType,
		"TUserPtr": userPointerType,
	}

	instantiatedInterface, err := domain.NewInstantiatedInterface(
		domain.NewBasicType("ComplexConverter", reflect.Interface),
		typeArguments,
		userType,
		"ComplexConverter[User, Address, Order, []User, *User]",
	)
	if err != nil {
		fmt.Printf("Error creating complex instantiated interface: %v\n", err)
		return
	}

	// Configure generator for advanced features
	config := &GenericGeneratorConfig{
		EnableOptimization:   true,
		MaxTemplateDepth:     15,
		EnableCaching:        true,
		GenerationTimeout:    60 * time.Second,
		EnableTypeValidation: true,
		PreferCompactOutput:  true,
		EnableErrorWrapping:  true,
		DebugMode:            true,
	}

	templateEngine := &ExampleTemplateEngine{}
	fieldMapper := &ExampleFieldMapper{}
	typeInstantiator := domain.NewTypeInstantiator(domain.NewTypeBuilder(), logger)

	generator := NewGenericCodeGenerator(
		templateEngine,
		typeInstantiator,
		fieldMapper,
		logger,
		config,
	)

	ctx := context.Background()
	generatedCode, err := generator.GenerateGenericImplementation(ctx, instantiatedInterface)
	if err != nil {
		fmt.Printf("Error generating advanced code: %v\n", err)
		return
	}

	fmt.Printf("Advanced Generated Code:\n%s\n", generatedCode)
}

// ExampleCustomTemplates demonstrates custom template usage.
func ExampleCustomTemplates() {
	logger := zap.NewNop()

	// Create a custom template engine with custom templates
	templateEngine := &ExampleTemplateEngine{
		customTemplates: map[string]string{
			"custom_converter": `
// Custom converter for {{.Method.Name}}
func (c *converter) {{.Method.Name}}(src {{substituteType .Method.Parameters.0.Type $.TypeSubstitutions}}) ({{substituteType .Method.ReturnType $.TypeSubstitutions}}, error) {
	// Custom conversion logic
	var result {{substituteType .Method.ReturnType $.TypeSubstitutions}}
	
	// Apply custom mapping
	{{range .FieldMappings}}
	result.{{.DestField}} = convertField(src.{{.SourceField}})
	{{end}}
	
	return result, nil
}`,
		},
	}

	fieldMapper := &ExampleFieldMapper{}
	typeInstantiator := domain.NewTypeInstantiator(domain.NewTypeBuilder(), logger)
	config := DefaultGenericGeneratorConfig()

	generator := NewGenericCodeGenerator(
		templateEngine,
		typeInstantiator,
		fieldMapper,
		logger,
		config,
	)

	// Create an instantiated interface with custom template annotation
	userType := domain.NewBasicType("User", reflect.Struct)
	typeArguments := map[string]domain.Type{"T": userType}

	instantiatedInterface, _ := domain.NewInstantiatedInterface(
		domain.NewBasicType("CustomConverter", reflect.Interface),
		typeArguments,
		userType,
		"CustomConverter[User]",
	)

	ctx := context.Background()
	generatedCode, err := generator.GenerateGenericImplementation(ctx, instantiatedInterface)
	if err != nil {
		fmt.Printf("Error generating custom code: %v\n", err)
		return
	}

	fmt.Printf("Custom Template Generated Code:\n%s\n", generatedCode)
}

// Example implementations for testing

// ExampleTemplateEngine provides example template functionality.
type ExampleTemplateEngine struct {
	customTemplates map[string]string
}

// Execute executes a template with the given data.
func (ete *ExampleTemplateEngine) Execute(templateName string, data interface{}) (string, error) {
	// Check for custom templates first
	if ete.customTemplates != nil {
		if template, found := ete.customTemplates[templateName]; found {
			return ete.renderTemplate(template, data)
		}
	}

	// Default templates
	switch templateName {
	case "generic_simple_conversion":
		return ete.renderSimpleConversion(data)
	case "generic_complex_conversion":
		return ete.renderComplexConversion(data)
	case "generic_method_basic":
		return ete.renderBasicMethod(data)
	case "generic_method_with_error":
		return ete.renderMethodWithError(data)
	default:
		return fmt.Sprintf("// Generated method using template: %s", templateName), nil
	}
}

// RegisterTemplate registers a new template.
func (ete *ExampleTemplateEngine) RegisterTemplate(name, content string) error {
	if ete.customTemplates == nil {
		ete.customTemplates = make(map[string]string)
	}
	ete.customTemplates[name] = content
	return nil
}

// HasTemplate checks if a template exists.
func (ete *ExampleTemplateEngine) HasTemplate(name string) bool {
	if ete.customTemplates != nil {
		if _, found := ete.customTemplates[name]; found {
			return true
		}
	}

	// Check default templates
	defaults := []string{
		"generic_simple_conversion",
		"generic_complex_conversion",
		"generic_method_basic",
		"generic_method_with_error",
	}

	for _, defaultTemplate := range defaults {
		if name == defaultTemplate {
			return true
		}
	}

	return false
}

// GetTemplateFunctions returns template functions.
func (ete *ExampleTemplateEngine) GetTemplateFunctions() map[string]interface{} {
	return map[string]interface{}{
		"substituteType": func(typ domain.Type, substitutions map[string]TypeSubstitution) string {
			if substitution, found := substitutions[typ.Name()]; found {
				return substitution.ConcreteType.Name()
			}
			return typ.Name()
		},
	}
}

// renderTemplate renders a template with simple substitution.
func (ete *ExampleTemplateEngine) renderTemplate(template string, data interface{}) (string, error) {
	// Simple template rendering - in production this would use text/template
	if methodData, ok := data.(*MethodTemplateData); ok {
		// Basic substitutions
		result := template
		if methodData.Method != nil {
			result = fmt.Sprintf(result, methodData.Method.Name)
		}
		return result, nil
	}
	return template, nil
}

// renderSimpleConversion renders a simple conversion method.
func (ete *ExampleTemplateEngine) renderSimpleConversion(data interface{}) (string, error) {
	if methodData, ok := data.(*MethodTemplateData); ok && methodData.Method != nil {
		method := methodData.Method
		result := fmt.Sprintf(`
// %s converts %s to %s
func (c *converter) %s(src %s) (%s, error) {
	var result %s
	
	// Field mappings
	%s
	
	return result, nil
}`,
			method.Name,
			method.Parameters[0].Type.Name(),
			method.ReturnType.Name(),
			method.Name,
			method.Parameters[0].Type.Name(),
			method.ReturnType.Name(),
			method.ReturnType.Name(),
			ete.renderFieldMappings(methodData.FieldMappings),
		)
		return result, nil
	}
	return "// Simple conversion method", nil
}

// renderComplexConversion renders a complex conversion method.
func (ete *ExampleTemplateEngine) renderComplexConversion(data interface{}) (string, error) {
	if methodData, ok := data.(*MethodTemplateData); ok && methodData.Method != nil {
		method := methodData.Method
		result := fmt.Sprintf(`
// %s converts %s to %s with validation and error handling
func (c *converter) %s(src %s) (%s, error) {
	var result %s
	
	// Validation and complex field mappings
	%s
	
	return result, nil
}`,
			method.Name,
			method.Parameters[0].Type.Name(),
			method.ReturnType.Name(),
			method.Name,
			method.Parameters[0].Type.Name(),
			method.ReturnType.Name(),
			method.ReturnType.Name(),
			ete.renderComplexFieldMappings(methodData.FieldMappings),
		)
		return result, nil
	}
	return "// Complex conversion method", nil
}

// renderBasicMethod renders a basic method.
func (ete *ExampleTemplateEngine) renderBasicMethod(data interface{}) (string, error) {
	if methodData, ok := data.(*MethodTemplateData); ok && methodData.Method != nil {
		method := methodData.Method
		return fmt.Sprintf(`
// %s is a basic method
func (c *converter) %s() {
	// Basic method implementation
}`, method.Name, method.Name), nil
	}
	return "// Basic method", nil
}

// renderMethodWithError renders a method with error handling.
func (ete *ExampleTemplateEngine) renderMethodWithError(data interface{}) (string, error) {
	if methodData, ok := data.(*MethodTemplateData); ok && methodData.Method != nil {
		method := methodData.Method
		return fmt.Sprintf(`
// %s is a method with error handling
func (c *converter) %s() error {
	// Method implementation with error handling
	return nil
}`, method.Name, method.Name), nil
	}
	return "// Method with error", nil
}

// renderFieldMappings renders simple field mappings.
func (ete *ExampleTemplateEngine) renderFieldMappings(mappings []*FieldMapping) string {
	if len(mappings) == 0 {
		return "\t// No field mappings"
	}

	var result string
	for _, mapping := range mappings {
		result += fmt.Sprintf("\tresult.%s = src.%s\n", mapping.DestField, mapping.SourceField)
	}
	return result
}

// renderComplexFieldMappings renders complex field mappings with validation.
func (ete *ExampleTemplateEngine) renderComplexFieldMappings(mappings []*FieldMapping) string {
	if len(mappings) == 0 {
		return "\t// No field mappings"
	}

	var result string
	for _, mapping := range mappings {
		if mapping.Validation != "" {
			result += fmt.Sprintf("\tif %s {\n\t\treturn result, fmt.Errorf(\"validation failed for %s\")\n\t}\n",
				mapping.Validation, mapping.SourceField)
		}

		if mapping.Converter != "" {
			result += fmt.Sprintf("\tconverted, err := %s(src.%s)\n", mapping.Converter, mapping.SourceField)
			result += "\tif err != nil {\n\t\treturn result, err\n\t}\n"
			result += fmt.Sprintf("\tresult.%s = converted\n", mapping.DestField)
		} else {
			result += fmt.Sprintf("\tresult.%s = src.%s\n", mapping.DestField, mapping.SourceField)
		}
	}
	return result
}

// ExampleFieldMapper provides example field mapping functionality.
type ExampleFieldMapper struct{}

// MapFields maps fields between source and destination types.
func (efm *ExampleFieldMapper) MapFields(
	sourceType, destType domain.Type,
	annotations map[string]string,
) ([]*FieldMapping, error) {
	// Example field mappings - in production this would analyze struct fields
	mappings := []*FieldMapping{
		{
			SourceField: "ID",
			DestField:   "ID",
			SourceType:  sourceType,
			DestType:    destType,
			Annotations: annotations,
		},
		{
			SourceField: "Name",
			DestField:   "Name",
			SourceType:  sourceType,
			DestType:    destType,
			Annotations: annotations,
		},
		{
			SourceField: "Email",
			DestField:   "Email",
			SourceType:  sourceType,
			DestType:    destType,
			Annotations: annotations,
		},
	}

	// Add converter if specified in annotations
	if converter, found := annotations["conv"]; found {
		for _, mapping := range mappings {
			if mapping.SourceField == "Email" {
				mapping.Converter = converter
			}
		}
	}

	// Add validation if specified in annotations
	if validation, found := annotations["validate"]; found {
		for _, mapping := range mappings {
			mapping.Validation = validation
		}
	}

	return mappings, nil
}

// ValidateMapping validates a field mapping.
func (efm *ExampleFieldMapper) ValidateMapping(mapping *FieldMapping) error {
	if mapping == nil {
		return fmt.Errorf("mapping cannot be nil")
	}

	if mapping.SourceField == "" {
		return fmt.Errorf("source field cannot be empty")
	}

	if mapping.DestField == "" {
		return fmt.Errorf("destination field cannot be empty")
	}

	return nil
}

// RunAllExamples runs all example functions.
func RunAllExamples() {
	fmt.Println("=== Basic Generic Converter Example ===")
	ExampleGenericConverter()

	fmt.Println("\n=== Advanced Generic Usage Example ===")
	ExampleAdvancedGenericUsage()

	fmt.Println("\n=== Custom Templates Example ===")
	ExampleCustomTemplates()
}
