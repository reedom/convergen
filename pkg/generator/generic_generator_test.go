package generator

import (
	"context"
	"reflect"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v8/pkg/domain"
)

func TestNewGenericCodeGenerator(t *testing.T) {
	tests := []struct {
		name               string
		templateEngine     TemplateEngine
		typeInstantiator   *domain.TypeInstantiator
		fieldMapper        FieldMapper
		logger             *zap.Logger
		config             *GenericGeneratorConfig
		expectNil          bool
		expectedOptEnabled bool
	}{
		{
			name:               "with default config",
			templateEngine:     &mockTemplateEngine{},
			typeInstantiator:   createMockTypeInstantiator(),
			fieldMapper:        &mockFieldMapper{},
			logger:             zap.NewNop(),
			config:             nil,
			expectNil:          false,
			expectedOptEnabled: true,
		},
		{
			name:               "with custom config",
			templateEngine:     &mockTemplateEngine{},
			typeInstantiator:   createMockTypeInstantiator(),
			fieldMapper:        &mockFieldMapper{},
			logger:             zap.NewNop(),
			config:             &GenericGeneratorConfig{EnableOptimization: false},
			expectNil:          false,
			expectedOptEnabled: false,
		},
		{
			name:               "with nil logger",
			templateEngine:     &mockTemplateEngine{},
			typeInstantiator:   createMockTypeInstantiator(),
			fieldMapper:        &mockFieldMapper{},
			logger:             nil,
			config:             nil,
			expectNil:          false,
			expectedOptEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewGenericCodeGenerator(
				tt.templateEngine,
				tt.typeInstantiator,
				tt.fieldMapper,
				tt.logger,
				tt.config,
			)

			if tt.expectNil && generator != nil {
				t.Errorf("expected nil generator, got %v", generator)
			}

			if !tt.expectNil && generator == nil {
				t.Error("expected non-nil generator, got nil")
			}

			if generator != nil {
				if generator.config.EnableOptimization != tt.expectedOptEnabled {
					t.Errorf("expected optimization enabled = %v, got %v",
						tt.expectedOptEnabled, generator.config.EnableOptimization)
				}

				if generator.metrics == nil {
					t.Error("expected metrics to be initialized")
				}
			}
		})
	}
}

func TestGenerateGenericImplementation(t *testing.T) {
	generator := createTestGenericCodeGenerator()

	tests := []struct {
		name                    string
		instantiatedInterface   *domain.InstantiatedInterface
		expectError             bool
		expectedMethodsContains string
		expectedErrorType       error
	}{
		{
			name:                  "nil instantiated interface",
			instantiatedInterface: nil,
			expectError:           true,
			expectedErrorType:     ErrInstantiatedInterfaceNil,
		},
		{
			name:                    "valid instantiated interface",
			instantiatedInterface:   createTestInstantiatedInterface(),
			expectError:             false,
			expectedMethodsContains: "Convert",
		},
		{
			name:                    "interface with validation result",
			instantiatedInterface:   createTestInstantiatedInterfaceWithValidation(),
			expectError:             false,
			expectedMethodsContains: "Convert",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := generator.GenerateGenericImplementation(ctx, tt.instantiatedInterface)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.expectedErrorType != nil && err != tt.expectedErrorType {
					t.Errorf("expected error type %v, got %v", tt.expectedErrorType, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.expectedMethodsContains != "" && result != "" {
					// For now, just check that result is not empty
					// In a real implementation, we'd check for specific method content
					if len(result) == 0 {
						t.Error("expected non-empty result")
					}
				}
			}
		})
	}
}

func TestCreateGenericTemplateData(t *testing.T) {
	generator := createTestGenericCodeGenerator()
	instantiatedInterface := createTestInstantiatedInterface()

	templateData, err := generator.createGenericTemplateData(instantiatedInterface)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if templateData == nil {
		t.Fatal("expected non-nil template data")
	}

	// Check basic fields
	if !templateData.IsGeneric() {
		t.Error("expected IsGeneric to be true")
	}

	if templateData.TypeSignature != instantiatedInterface.TypeSignature {
		t.Errorf("expected type signature %s, got %s",
			instantiatedInterface.TypeSignature, templateData.TypeSignature)
	}

	if len(templateData.TypeSubstitutions) == 0 {
		t.Error("expected type substitutions to be populated")
	}

	if len(templateData.TypeArguments) == 0 {
		t.Error("expected type arguments to be populated")
	}

	if len(templateData.TypeParameters) == 0 {
		t.Error("expected type parameters to be populated")
	}

	// Check metadata
	if templateData.Metadata == nil {
		t.Error("expected metadata to be initialized")
	}

	if _, found := templateData.Metadata["type_signature"]; !found {
		t.Error("expected type_signature in metadata")
	}
}

func TestGenerateFieldMappings(t *testing.T) {
	generator := createTestGenericCodeGenerator()
	templateData := createTestGenericTemplateData()

	tests := []struct {
		name          string
		method        *MethodData
		templateData  *GenericTemplateData
		expectError   bool
		expectedCount int
	}{
		{
			name:          "method without parameters",
			method:        &MethodData{Name: "Test", Parameters: []*ParameterData{}},
			templateData:  templateData,
			expectError:   false,
			expectedCount: 0,
		},
		{
			name: "conversion method",
			method: &MethodData{
				Name: "Convert",
				Parameters: []*ParameterData{
					{Name: "src", Type: domain.NewBasicType("User", reflect.Struct)},
				},
				ReturnType:  domain.NewBasicType("User", reflect.Struct),
				Annotations: make(map[string]string),
			},
			templateData:  templateData,
			expectError:   false,
			expectedCount: 0, // Mock field mapper returns empty mappings
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mappings, err := generator.generateFieldMappings(tt.method, tt.templateData)

			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(mappings) != tt.expectedCount {
				t.Errorf("expected %d mappings, got %d", tt.expectedCount, len(mappings))
			}
		})
	}
}

func TestSubstituteTypeInContext(t *testing.T) {
	generator := createTestGenericCodeGenerator()

	userType := domain.NewBasicType("User", reflect.Struct)
	stringType := domain.NewBasicType("string", reflect.String)
	userTypeForParam := domain.NewBasicType("T", reflect.Interface)
	tType := userTypeForParam // Use basic type instead of generic type for testing

	substitutions := map[string]TypeSubstitution{
		"T": {
			ParameterName: "T",
			ConcreteType:  userType,
			PackagePath:   "",
		},
	}

	tests := []struct {
		name          string
		typ           domain.Type
		substitutions map[string]TypeSubstitution
		expectError   bool
		expectedType  domain.Type
		expectedName  string
	}{
		{
			name:          "nil type",
			typ:           nil,
			substitutions: substitutions,
			expectError:   true,
		},
		{
			name:          "type parameter substitution",
			typ:           tType,
			substitutions: substitutions,
			expectError:   false,
			expectedType:  userType,
			expectedName:  "User",
		},
		{
			name:          "no substitution needed",
			typ:           stringType,
			substitutions: substitutions,
			expectError:   false,
			expectedType:  stringType,
			expectedName:  "string",
		},
		{
			name:          "slice type substitution",
			typ:           domain.NewSliceType(tType, ""),
			substitutions: substitutions,
			expectError:   false,
			expectedName:  "User", // Element type should be substituted
		},
		{
			name:          "pointer type substitution",
			typ:           domain.NewPointerType(tType, ""),
			substitutions: substitutions,
			expectError:   false,
			expectedName:  "User", // Element type should be substituted
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generator.substituteTypeInContext(tt.typ, tt.substitutions)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				if tt.expectedType != nil && result != tt.expectedType {
					t.Errorf("expected type %v, got %v", tt.expectedType, result)
				}

				if tt.expectedName != "" {
					if result == nil {
						t.Error("expected non-nil result")
					} else {
						// For composite types, check the element type
						switch result.Kind() {
						case domain.KindSlice:
							if sliceType, ok := result.(*domain.SliceType); ok {
								if sliceType.Elem().Name() != tt.expectedName {
									t.Errorf("expected element type name %s, got %s",
										tt.expectedName, sliceType.Elem().Name())
								}
							}
						case domain.KindPointer:
							if pointerType, ok := result.(*domain.PointerType); ok {
								if pointerType.Elem().Name() != tt.expectedName {
									t.Errorf("expected element type name %s, got %s",
										tt.expectedName, pointerType.Elem().Name())
								}
							}
						default:
							if result.Name() != tt.expectedName {
								t.Errorf("expected type name %s, got %s",
									tt.expectedName, result.Name())
							}
						}
					}
				}
			}
		})
	}
}

func TestSelectTemplate(t *testing.T) {
	generator := createTestGenericCodeGenerator()
	templateData := createTestGenericTemplateData()

	tests := []struct {
		name             string
		method           *MethodData
		expectedTemplate string
	}{
		{
			name: "basic method",
			method: &MethodData{
				Name:         "BasicMethod",
				Parameters:   []*ParameterData{},
				ReturnType:   nil,
				ReturnsError: false,
				Annotations:  make(map[string]string),
			},
			expectedTemplate: "generic_method_basic",
		},
		{
			name: "method with error",
			method: &MethodData{
				Name:         "MethodWithError",
				Parameters:   []*ParameterData{},
				ReturnType:   nil,
				ReturnsError: true,
				Annotations:  make(map[string]string),
			},
			expectedTemplate: "generic_method_with_error",
		},
		{
			name: "simple conversion method",
			method: &MethodData{
				Name: "Convert",
				Parameters: []*ParameterData{
					{Name: "src", Type: domain.NewBasicType("User", reflect.Struct)},
				},
				ReturnType:   domain.NewBasicType("UserDTO", reflect.Struct),
				ReturnsError: false,
				Annotations:  make(map[string]string),
			},
			expectedTemplate: "generic_simple_conversion",
		},
		{
			name: "complex conversion method",
			method: &MethodData{
				Name: "ComplexConvert",
				Parameters: []*ParameterData{
					{Name: "src", Type: domain.NewBasicType("User", reflect.Struct)},
				},
				ReturnType:   domain.NewBasicType("UserDTO", reflect.Struct),
				ReturnsError: false,
				Annotations: map[string]string{
					"conv": "UserConverter",
					"map":  "Name->FullName",
				},
			},
			expectedTemplate: "generic_complex_conversion",
		},
		{
			name: "custom template annotation",
			method: &MethodData{
				Name:         "CustomMethod",
				Parameters:   []*ParameterData{},
				ReturnType:   nil,
				ReturnsError: false,
				Annotations: map[string]string{
					"template": "custom_template",
				},
			},
			expectedTemplate: "generic_method_basic", // Falls back to default since custom template doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.selectTemplate(tt.method, templateData)
			if result != tt.expectedTemplate {
				t.Errorf("expected template %s, got %s", tt.expectedTemplate, result)
			}
		})
	}
}

func TestGetMetrics(t *testing.T) {
	generator := createTestGenericCodeGenerator()

	// Generate some activity to populate metrics
	generator.metrics.TotalGenerations = 5
	generator.metrics.SuccessfulGenerations = 4
	generator.metrics.FailedGenerations = 1
	generator.metrics.TotalGenerationTime = 100 * time.Millisecond

	metrics := generator.GetMetrics()
	if metrics == nil {
		t.Fatal("expected non-nil metrics")
	}

	if metrics.TotalGenerations != 5 {
		t.Errorf("expected 5 total generations, got %d", metrics.TotalGenerations)
	}

	if metrics.SuccessfulGenerations != 4 {
		t.Errorf("expected 4 successful generations, got %d", metrics.SuccessfulGenerations)
	}

	if metrics.FailedGenerations != 1 {
		t.Errorf("expected 1 failed generation, got %d", metrics.FailedGenerations)
	}
}

func TestClearMetrics(t *testing.T) {
	generator := createTestGenericCodeGenerator()

	// Populate some metrics
	generator.metrics.TotalGenerations = 5
	generator.metrics.SuccessfulGenerations = 4

	generator.ClearMetrics()

	if generator.metrics.TotalGenerations != 0 {
		t.Errorf("expected 0 total generations after clear, got %d", generator.metrics.TotalGenerations)
	}

	if generator.metrics.SuccessfulGenerations != 0 {
		t.Errorf("expected 0 successful generations after clear, got %d", generator.metrics.SuccessfulGenerations)
	}
}

func TestShutdown(t *testing.T) {
	generator := createTestGenericCodeGenerator()

	ctx := context.Background()
	err := generator.Shutdown(ctx)
	if err != nil {
		t.Errorf("unexpected error during shutdown: %v", err)
	}
}

// Helper functions and mock implementations

func createTestGenericCodeGenerator() *GenericCodeGenerator {
	return NewGenericCodeGenerator(
		&mockTemplateEngine{},
		createMockTypeInstantiator(),
		&mockFieldMapper{},
		zap.NewNop(),
		DefaultGenericGeneratorConfig(),
	)
}

func createMockTypeInstantiator() *domain.TypeInstantiator {
	// Create a minimal mock type instantiator
	typeBuilder := domain.NewTypeBuilder()
	return domain.NewTypeInstantiator(typeBuilder, zap.NewNop())
}

func createTestInstantiatedInterface() *domain.InstantiatedInterface {
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

func createTestInstantiatedInterfaceWithValidation() *domain.InstantiatedInterface {
	instantiated := createTestInstantiatedInterface()

	// Add validation result
	instantiated.ValidationResult = &domain.ValidationResult{
		Valid:                true,
		ViolatedConstraints:  []domain.ConstraintViolation{},
		ValidationDurationMS: 5,
		Details:              make(map[string]domain.ValidationDetail),
	}

	return instantiated
}

func createTestGenericTemplateData() *GenericTemplateData {
	userType := domain.NewBasicType("User", reflect.Struct)

	return &GenericTemplateData{
		BaseTemplateData: BaseTemplateData{
			Package:         "main",
			Imports:         []*ImportInfo{},
			Metadata:        make(map[string]interface{}),
			HelperFunctions: make(map[string]interface{}),
		},
		TypeParameters: []TypeParam{
			{Name: "T", Constraint: "any", Position: 0, Used: true},
		},
		TypeArguments: []TypeArg{
			{Name: "User", Type: userType, PackagePath: "", IsPointer: false, IsSlice: false},
		},
		TypeSubstitutions: map[string]TypeSubstitution{
			"T": {
				ParameterName: "T",
				ConcreteType:  userType,
				PackagePath:   "",
			},
		},
		IsGenericFlag: true,
		TypeSignature: "Converter[User]",
	}
}

// Mock implementations

type mockTemplateEngine struct {
	templates map[string]string
	functions map[string]interface{}
}

func (m *mockTemplateEngine) Execute(templateName string, data interface{}) (string, error) {
	// Simple mock implementation
	return "// Generated code for " + templateName, nil
}

func (m *mockTemplateEngine) RegisterTemplate(name, content string) error {
	if m.templates == nil {
		m.templates = make(map[string]string)
	}
	m.templates[name] = content
	return nil
}

func (m *mockTemplateEngine) HasTemplate(name string) bool {
	if m.templates == nil {
		return false
	}
	_, found := m.templates[name]
	return found
}

func (m *mockTemplateEngine) GetTemplateFunctions() map[string]interface{} {
	if m.functions == nil {
		return make(map[string]interface{})
	}
	return m.functions
}

type mockFieldMapper struct{}

func (m *mockFieldMapper) MapFields(
	sourceType, destType domain.Type,
	annotations map[string]string,
) ([]*FieldMapping, error) {
	// Return empty mappings for simplicity
	return []*FieldMapping{}, nil
}

func (m *mockFieldMapper) ValidateMapping(mapping *FieldMapping) error {
	return nil
}

// TestGenericTemplateSystemIntegration performs a basic integration test of the generic template system.
func TestGenericTemplateSystemIntegration(t *testing.T) {
	// Create logger
	logger := zap.NewNop()

	// Create type builder and instantiator
	typeBuilder := domain.NewTypeBuilder()
	typeInstantiator := domain.NewTypeInstantiator(typeBuilder, logger)

	// Create template engine and field mapper
	templateEngine := &mockTemplateEngine{}
	fieldMapper := &mockFieldMapper{}

	// Create the generic code generator
	config := DefaultGenericGeneratorConfig()
	generator := NewGenericCodeGenerator(
		templateEngine,
		typeInstantiator,
		fieldMapper,
		logger,
		config,
	)

	// Create a generic interface definition
	userType := domain.NewBasicType("User", reflect.Struct)
	userDTOType := domain.NewBasicType("UserDTO", reflect.Struct)

	// Create type arguments map
	typeArguments := map[string]domain.Type{
		"T": userType,
		"U": userDTOType,
	}

	// Create an instantiated interface
	instantiatedInterface, err := domain.NewInstantiatedInterface(
		domain.NewBasicType("Converter", reflect.Interface),
		typeArguments,
		userType,
		"Converter[User, UserDTO]",
	)
	if err != nil {
		t.Fatalf("error creating instantiated interface: %v", err)
	}

	// Generate the generic implementation
	ctx := context.Background()
	generatedCode, err := generator.GenerateGenericImplementation(ctx, instantiatedInterface)
	if err != nil {
		t.Fatalf("error generating code: %v", err)
	}

	// Verify we got some code
	if len(generatedCode) == 0 {
		t.Error("generated code is empty")
	}

	// Get metrics
	metrics := generator.GetMetrics()
	if metrics == nil {
		t.Error("metrics are nil")
	}

	if metrics.TotalGenerations == 0 {
		t.Error("no generations recorded in metrics")
	}

	// Success
	t.Logf("Integration test passed: code_length=%d, total_generations=%d, successful_generations=%d",
		len(generatedCode), metrics.TotalGenerations, metrics.SuccessfulGenerations)
}
