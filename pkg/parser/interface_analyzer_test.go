package parser

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"golang.org/x/tools/go/packages"

	"github.com/reedom/convergen/v9/pkg/domain"
	"github.com/reedom/convergen/v9/pkg/internal/events"
)

func TestExtractInterfaceTypeParams(t *testing.T) {
	tests := []struct {
		name               string
		sourceCode         string
		interfaceName      string
		expectedTypeParams int
		expectedTypes      []string
		expectedError      bool
	}{
		{
			name: "Basic generic interface with any constraint",
			sourceCode: `package test

//go:convergen
type Converter[T any] interface {
	Convert(src T) T
}
`,
			interfaceName:      "Converter",
			expectedTypeParams: 1,
			expectedTypes:      []string{"T any"},
			expectedError:      false,
		},
		{
			name: "Multiple type parameters with any constraint",
			sourceCode: `package test

//go:convergen
type Mapper[T, U any] interface {
	Map(src T) U
}
`,
			interfaceName:      "Mapper",
			expectedTypeParams: 2,
			expectedTypes:      []string{"T any", "U any"},
			expectedError:      false,
		},
		{
			name: "Generic interface with comparable constraint",
			sourceCode: `package test

//go:convergen
type Processor[T comparable] interface {
	Process(src T) T
}
`,
			interfaceName:      "Processor",
			expectedTypeParams: 1,
			expectedTypes:      []string{"T comparable"},
			expectedError:      false,
		},
		{
			name: "Generic interface with complex constraints",
			sourceCode: `package test

//go:convergen
type Transformer[T ~string | ~int] interface {
	Transform(src T) T
}
`,
			interfaceName:      "Transformer",
			expectedTypeParams: 1,
			expectedTypes:      []string{"T ~string | ~int"},
			expectedError:      false,
		},
		{
			name: "Non-generic interface",
			sourceCode: `package test

//go:convergen
type Converter interface {
	Convert(src string) string
}
`,
			interfaceName:      "Converter",
			expectedTypeParams: 0,
			expectedTypes:      []string{},
			expectedError:      false,
		},
		{
			name: "Mixed constraints interface",
			sourceCode: `package test

//go:convergen
type MixedProcessor[T comparable, U ~string, V any] interface {
	Process(src T, dest U) V
}
`,
			interfaceName:      "MixedProcessor",
			expectedTypeParams: 3,
			expectedTypes:      []string{"T comparable", "U ~string", "V any"},
			expectedError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			logger := zaptest.NewLogger(t)

			// Parse the source code
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.sourceCode, parser.ParseComments)
			require.NoError(t, err)

			// Type check the file
			config := &types.Config{}
			pkg := types.NewPackage("test", "test")
			checker := types.NewChecker(config, fset, pkg, nil)

			// Type check without info - we just need the package

			err = checker.Files([]*ast.File{file})
			require.NoError(t, err)

			// Find the interface object
			var interfaceObj types.Object
			scope := pkg.Scope()
			for _, name := range scope.Names() {
				obj := scope.Lookup(name)
				if obj.Name() == tt.interfaceName {
					interfaceObj = obj
					break
				}
			}
			require.NotNil(t, interfaceObj, "Interface %s not found", tt.interfaceName)

			// Create parser instance
			eventBus := events.NewInMemoryEventBus(logger)
			defer func() { _ = eventBus.Close() }()

			parserConfig := &Config{
				MaxConcurrentWorkers: 1,
				CacheSize:            100,
				EnableProgress:       false,
			}
			astParser := NewASTParser(logger, eventBus, parserConfig)

			// Test the extractInterfaceTypeParams function
			typeParams, err := astParser.extractInterfaceTypeParams(ctx, interfaceObj)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, typeParams, tt.expectedTypeParams)

			// Check each type parameter
			for i, expectedType := range tt.expectedTypes {
				if i < len(typeParams) {
					assert.Equal(t, expectedType, typeParams[i].String(),
						"Type parameter %d mismatch", i)
					assert.True(t, typeParams[i].IsValid(),
						"Type parameter %d should be valid", i)
				}
			}
		})
	}
}

func TestExtractInterfaceTypeParamsEdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		sourceCode    string
		interfaceName string
		expectedError bool
		errorContains string
	}{
		{
			name: "Empty type parameter list",
			sourceCode: `package test

//go:convergen
type Converter interface {
	Convert(src string) string
}
`,
			interfaceName: "Converter",
			expectedError: false,
		},
		{
			name: "Interface with complex underlying constraint",
			sourceCode: `package test

//go:convergen
type StringProcessor[T ~string] interface {
	Process(src T) T
}
`,
			interfaceName: "StringProcessor",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			logger := zaptest.NewLogger(t)

			// Parse the source code
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.sourceCode, parser.ParseComments)
			require.NoError(t, err)

			// Type check the file
			config := &types.Config{}
			pkg := types.NewPackage("test", "test")
			checker := types.NewChecker(config, fset, pkg, nil)

			err = checker.Files([]*ast.File{file})
			require.NoError(t, err)

			// Find the interface object
			var interfaceObj types.Object
			scope := pkg.Scope()
			for _, name := range scope.Names() {
				obj := scope.Lookup(name)
				if obj.Name() == tt.interfaceName {
					interfaceObj = obj
					break
				}
			}
			require.NotNil(t, interfaceObj, "Interface %s not found", tt.interfaceName)

			// Create parser instance
			eventBus := events.NewInMemoryEventBus(logger)
			defer func() { _ = eventBus.Close() }()

			parserConfig := &Config{
				MaxConcurrentWorkers: 1,
				CacheSize:            100,
				EnableProgress:       false,
			}
			astParser := NewASTParser(logger, eventBus, parserConfig)

			// Test the extractInterfaceTypeParams function
			typeParams, err := astParser.extractInterfaceTypeParams(ctx, interfaceObj)

			if tt.expectedError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, typeParams)
		})
	}
}

func TestAnalyzeInterfaceWithTypeParams(t *testing.T) {
	tests := []struct {
		name               string
		sourceCode         string
		interfaceName      string
		expectedTypeParams int
		expectedMethods    int
	}{
		{
			name: "Generic interface with methods",
			sourceCode: `package test

// :convergen
type Converter[T any] interface {
	Convert(src T) T
	Validate(src T) bool
}
`,
			interfaceName:      "Converter",
			expectedTypeParams: 1,
			expectedMethods:    2,
		},
		{
			name: "Non-generic interface with methods",
			sourceCode: `package test

// :convergen
type Converter interface {
	Convert(src string) string
	Validate(src string) bool
}
`,
			interfaceName:      "Converter",
			expectedTypeParams: 0,
			expectedMethods:    2,
		},
		{
			name: "Complex generic interface",
			sourceCode: `package test

// :convergen
type ComplexProcessor[T comparable, U ~string | ~int] interface {
	Process(src T, dest U) T
	Compare(a, b T) bool
	Transform(src U) T
}
`,
			interfaceName:      "ComplexProcessor",
			expectedTypeParams: 2,
			expectedMethods:    3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			logger := zaptest.NewLogger(t)

			// Parse the source code
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.sourceCode, parser.ParseComments)
			require.NoError(t, err)

			// Type check the file
			config := &types.Config{}
			pkg := types.NewPackage("test", "test")
			checker := types.NewChecker(config, fset, pkg, nil)

			err = checker.Files([]*ast.File{file})
			require.NoError(t, err)

			// Find the interface object and type
			var interfaceObj types.Object
			var interfaceType *types.Interface
			scope := pkg.Scope()
			for _, name := range scope.Names() {
				obj := scope.Lookup(name)
				if obj.Name() == tt.interfaceName {
					interfaceObj = obj
					if named, ok := obj.Type().(*types.Named); ok {
						if iface, ok := named.Underlying().(*types.Interface); ok {
							interfaceType = iface
						}
					}
					break
				}
			}
			require.NotNil(t, interfaceObj, "Interface %s not found", tt.interfaceName)
			require.NotNil(t, interfaceType, "Interface type not found")

			// Create parser instance
			eventBus := events.NewInMemoryEventBus(logger)
			defer func() { _ = eventBus.Close() }()

			parserConfig := &Config{
				MaxConcurrentWorkers: 1,
				CacheSize:            100,
				EnableProgress:       false,
			}
			astParser := NewASTParser(logger, eventBus, parserConfig)

			// Create fake package for analyzeInterface
			pkgs := &packages.Package{}

			// Test the analyzeInterface function
			interfaceInfo, err := astParser.analyzeInterface(ctx, pkgs, file, interfaceObj, interfaceType)

			require.NoError(t, err)
			require.NotNil(t, interfaceInfo)

			assert.Equal(t, tt.interfaceName, interfaceInfo.Object.Name())
			assert.Len(t, interfaceInfo.TypeParams, tt.expectedTypeParams)
			assert.Len(t, interfaceInfo.Methods, tt.expectedMethods)

			// Validate type parameters
			for i, typeParam := range interfaceInfo.TypeParams {
				assert.True(t, typeParam.IsValid(),
					"Type parameter %d should be valid", i)
				assert.NotEmpty(t, typeParam.Name,
					"Type parameter %d should have a name", i)
			}
		})
	}
}

func TestNewInterfaceInfoConstructor(t *testing.T) {
	// Test with different type parameter configurations
	tests := []struct {
		name              string
		typeParams        []domain.TypeParam
		expectedIsGeneric bool
	}{
		{
			name:              "Non-generic interface",
			typeParams:        []domain.TypeParam{},
			expectedIsGeneric: false,
		},
		{
			name: "Generic interface with one type parameter",
			typeParams: []domain.TypeParam{
				*domain.NewAnyTypeParam("T", 0),
			},
			expectedIsGeneric: true,
		},
		{
			name: "Generic interface with multiple type parameters",
			typeParams: []domain.TypeParam{
				*domain.NewAnyTypeParam("T", 0),
				*domain.NewComparableTypeParam("U", 1),
			},
			expectedIsGeneric: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test using real parsing to create a proper types.Object
			sourceCode := `package test
//go:convergen
type TestInterface interface {
	TestMethod(src string) string
}`

			// Parse the source code
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", sourceCode, parser.ParseComments)
			require.NoError(t, err)

			// Type check the file
			config := &types.Config{}
			pkg := types.NewPackage("test", "test")
			checker := types.NewChecker(config, fset, pkg, nil)

			err = checker.Files([]*ast.File{file})
			require.NoError(t, err)

			// Find the interface object
			var interfaceObj types.Object
			scope := pkg.Scope()
			for _, name := range scope.Names() {
				obj := scope.Lookup(name)
				if obj.Name() == "TestInterface" {
					interfaceObj = obj
					break
				}
			}
			require.NotNil(t, interfaceObj)

			// Create InterfaceInfo using constructor
			iface := &types.Interface{}
			methods := []types.Object{}
			options := &domain.InterfaceOptions{}
			annotations := []*Annotation{}
			marker := "test-marker"

			info := NewInterfaceInfo(
				interfaceObj,
				iface,
				methods,
				options,
				annotations,
				marker,
				interfaceObj.Pos(),
				tt.typeParams,
			)

			// Verify constructor behavior
			assert.NotNil(t, info)
			assert.Equal(t, interfaceObj, info.Object)
			assert.Equal(t, iface, info.Interface)
			assert.Equal(t, methods, info.Methods)
			assert.Equal(t, options, info.Options)
			assert.Equal(t, annotations, info.Annotations)
			assert.Equal(t, marker, info.Marker)
			assert.Equal(t, interfaceObj.Pos(), info.Position)
			assert.Equal(t, tt.typeParams, info.TypeParams)

			// Verify new generic fields
			assert.Equal(t, tt.expectedIsGeneric, info.IsGeneric)
			assert.NotNil(t, info.Instantiations)
			assert.Empty(t, info.Instantiations)

			// Verify validation passes
			assert.NoError(t, info.ValidateGenericConsistency())
		})
	}
}

func TestInstantiatedInterfaceCreation(t *testing.T) {
	tests := []struct {
		name        string
		typeArgs    map[string]domain.Type
		expectValid bool
	}{
		{
			name: "Valid instantiation with type args",
			typeArgs: map[string]domain.Type{
				"T": domain.StringType,
				"U": domain.IntType,
			},
			expectValid: true,
		},
		{
			name:        "Invalid instantiation with nil type args",
			typeArgs:    nil,
			expectValid: false,
		},
		{
			name:        "Invalid instantiation with empty type args",
			typeArgs:    map[string]domain.Type{},
			expectValid: false,
		},
		{
			name: "Invalid instantiation with nil type",
			typeArgs: map[string]domain.Type{
				"T": nil,
			},
			expectValid: false,
		},
		{
			name: "Invalid instantiation with empty name",
			typeArgs: map[string]domain.Type{
				"": domain.StringType,
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			methods := []types.Object{} // Empty for this test
			inst := NewInstantiatedInterface(tt.typeArgs, methods)

			assert.NotNil(t, inst)
			assert.Equal(t, tt.typeArgs, inst.TypeArgs)
			assert.Equal(t, methods, inst.Methods)
			assert.Equal(t, "", inst.CreatedAt)
			assert.False(t, inst.Validated)

			assert.Equal(t, tt.expectValid, inst.IsValid())
		})
	}
}

func TestInterfaceInfoInstantiationManagement(t *testing.T) {
	// Create a generic interface
	typeParams := []domain.TypeParam{
		*domain.NewAnyTypeParam("T", 0),
		*domain.NewComparableTypeParam("U", 1),
	}

	info := createTestInterfaceInfo(t, typeParams)

	require.True(t, info.IsGeneric)
	require.Equal(t, 0, info.GetInstantiationCount())

	// Test adding valid instantiation
	typeArgs := map[string]domain.Type{
		"T": domain.StringType,
		"U": domain.IntType,
	}
	instantiation := NewInstantiatedInterface(typeArgs, []types.Object{})
	signature := "string,int"

	err := info.AddInstantiation(signature, instantiation)
	assert.NoError(t, err)
	assert.Equal(t, 1, info.GetInstantiationCount())
	assert.True(t, info.HasInstantiation(signature))

	// Test getting instantiation
	retrieved, exists := info.GetInstantiation(signature)
	assert.True(t, exists)
	assert.Equal(t, instantiation, retrieved)

	// Test adding instantiation with wrong type arg count
	wrongTypeArgs := map[string]domain.Type{
		"T": domain.StringType,
		// Missing "U"
	}
	wrongInstantiation := NewInstantiatedInterface(wrongTypeArgs, []types.Object{})
	err = info.AddInstantiation("string", wrongInstantiation)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "type argument count")

	// Test adding nil instantiation
	err = info.AddInstantiation("nil", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "instantiation cannot be nil")

	// Test clearing instantiations
	info.ClearInstantiations()
	assert.Equal(t, 0, info.GetInstantiationCount())
	assert.False(t, info.HasInstantiation(signature))
}

func TestInterfaceInfoTypeParameterAccess(t *testing.T) {
	typeParams := []domain.TypeParam{
		*domain.NewAnyTypeParam("T", 0),
		*domain.NewComparableTypeParam("U", 1),
		*domain.NewUnderlyingTypeParam("V", domain.NewUnderlyingConstraint(domain.StringType, ""), 2),
	}

	info := createTestInterfaceInfo(t, typeParams)

	// Test GetTypeParameterByName
	param, found := info.GetTypeParameterByName("T")
	assert.True(t, found)
	assert.Equal(t, "T", param.Name)
	assert.Equal(t, 0, param.Index)

	param, found = info.GetTypeParameterByName("U")
	assert.True(t, found)
	assert.Equal(t, "U", param.Name)
	assert.Equal(t, 1, param.Index)

	param, found = info.GetTypeParameterByName("NonExistent")
	assert.False(t, found)
	assert.Nil(t, param)

	// Test GetTypeParameterByIndex
	param, found = info.GetTypeParameterByIndex(0)
	assert.True(t, found)
	assert.Equal(t, "T", param.Name)

	param, found = info.GetTypeParameterByIndex(2)
	assert.True(t, found)
	assert.Equal(t, "V", param.Name)

	param, found = info.GetTypeParameterByIndex(-1)
	assert.False(t, found)
	assert.Nil(t, param)

	param, found = info.GetTypeParameterByIndex(10)
	assert.False(t, found)
	assert.Nil(t, param)

	// Test GetTypeParameterNames
	names := info.GetTypeParameterNames()
	assert.Equal(t, []string{"T", "U", "V"}, names)
}

func TestInterfaceInfoValidation(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() *InterfaceInfo
		expectError bool
		errorText   string
	}{
		{
			name: "Valid non-generic interface",
			setup: func() *InterfaceInfo {
				return createTestInterfaceInfo(t, []domain.TypeParam{})
			},
			expectError: false,
		},
		{
			name: "Valid generic interface",
			setup: func() *InterfaceInfo {
				return createTestInterfaceInfo(t, []domain.TypeParam{*domain.NewAnyTypeParam("T", 0)})
			},
			expectError: false,
		},
		{
			name: "Invalid type parameter",
			setup: func() *InterfaceInfo {
				invalidParam := &domain.TypeParam{
					Name:         "", // Invalid: empty name
					Constraint:   nil,
					Index:        0,
					IsAny:        true,
					IsComparable: true, // Invalid: both IsAny and IsComparable
				}
				return createTestInterfaceInfo(t, []domain.TypeParam{*invalidParam})
			},
			expectError: true,
			errorText:   "invalid type parameter",
		},
		{
			name: "Non-generic with instantiations",
			setup: func() *InterfaceInfo {
				info := createTestInterfaceInfo(t, []domain.TypeParam{})
				// Manually add instantiation to non-generic interface (should be invalid)
				info.Instantiations["test"] = NewInstantiatedInterface(
					map[string]domain.Type{"T": domain.StringType},
					[]types.Object{},
				)
				return info
			},
			expectError: true,
			errorText:   "non-generic interface",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := tt.setup()
			err := info.ValidateGenericConsistency()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorText != "" {
					assert.Contains(t, err.Error(), tt.errorText)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNonGenericInterfaceInstantiationRestrictions(t *testing.T) {
	// Create non-generic interface
	info := createTestInterfaceInfo(t, []domain.TypeParam{})

	require.False(t, info.IsGeneric)

	// Try to add instantiation to non-generic interface
	instantiation := NewInstantiatedInterface(
		map[string]domain.Type{"T": domain.StringType},
		[]types.Object{},
	)

	err := info.AddInstantiation("string", instantiation)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot add instantiation to non-generic interface")
}

func TestInstantiatedInterfaceHelperMethods(t *testing.T) {
	typeArgs := map[string]domain.Type{
		"T": domain.StringType,
		"U": domain.IntType,
		"V": domain.BoolType,
	}

	inst := NewInstantiatedInterface(typeArgs, []types.Object{})

	// Test GetTypeArgument
	typ, found := inst.GetTypeArgument("T")
	assert.True(t, found)
	assert.Equal(t, domain.StringType, typ)

	typ, found = inst.GetTypeArgument("NonExistent")
	assert.False(t, found)
	assert.Nil(t, typ)

	// Test GetTypeArgumentNames
	names := inst.GetTypeArgumentNames()
	assert.Len(t, names, 3)
	assert.Contains(t, names, "T")
	assert.Contains(t, names, "U")
	assert.Contains(t, names, "V")

	// Test with nil type args
	emptyInst := NewInstantiatedInterface(nil, []types.Object{})
	names = emptyInst.GetTypeArgumentNames()
	assert.Nil(t, names)

	typ, found = emptyInst.GetTypeArgument("T")
	assert.False(t, found)
	assert.Nil(t, typ)
}

// Helper function to create a test InterfaceInfo with real types.Object.
func createTestInterfaceInfo(t *testing.T, typeParams []domain.TypeParam) *InterfaceInfo {
	// Use real parsing to create proper types.Object
	sourceCode := `package test
//go:convergen
type TestInterface interface {
	TestMethod(src string) string
}`

	// Parse the source code
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", sourceCode, parser.ParseComments)
	require.NoError(t, err)

	// Type check the file
	config := &types.Config{}
	pkg := types.NewPackage("test", "test")
	checker := types.NewChecker(config, fset, pkg, nil)

	err = checker.Files([]*ast.File{file})
	require.NoError(t, err)

	// Find the interface object
	var interfaceObj types.Object
	scope := pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if obj.Name() == "TestInterface" {
			interfaceObj = obj
			break
		}
	}
	require.NotNil(t, interfaceObj)

	return NewInterfaceInfo(
		interfaceObj,
		&types.Interface{},
		[]types.Object{},
		&domain.InterfaceOptions{},
		[]*Annotation{},
		"test-marker",
		interfaceObj.Pos(),
		typeParams,
	)
}

func TestBackwardCompatibilityNonGenericInterfaces(t *testing.T) {
	sourceCode := `package test

// :convergen
type Converter interface {
	Convert(src string) string
	Validate(src string) bool
}

type AnotherConverter interface {
	Transform(input int) string
}
`

	ctx := context.Background()
	logger := zaptest.NewLogger(t)

	// Parse the source code
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", sourceCode, parser.ParseComments)
	require.NoError(t, err)

	// Type check the file
	config := &types.Config{}
	pkg := types.NewPackage("test", "test")
	checker := types.NewChecker(config, fset, pkg, nil)

	err = checker.Files([]*ast.File{file})
	require.NoError(t, err)

	// Find the interface object and type
	var interfaceObj types.Object
	var interfaceType *types.Interface
	scope := pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if obj.Name() == "Converter" {
			interfaceObj = obj
			if named, ok := obj.Type().(*types.Named); ok {
				if iface, ok := named.Underlying().(*types.Interface); ok {
					interfaceType = iface
				}
			} else if iface, ok := obj.Type().(*types.Interface); ok {
				interfaceType = iface
			}
			break
		}
	}
	require.NotNil(t, interfaceObj, "Interface Converter not found")
	require.NotNil(t, interfaceType, "Interface type not found")

	// Create parser instance
	eventBus := events.NewInMemoryEventBus(logger)
	defer func() { _ = eventBus.Close() }()
	parserConfig := &Config{
		MaxConcurrentWorkers: 1,
		CacheSize:            100,
		EnableProgress:       false,
	}
	astParser := NewASTParser(logger, eventBus, parserConfig)

	// Create fake package for analyzeInterface
	pkgs := &packages.Package{}

	// Test that non-generic interfaces work correctly
	interfaceInfo, err := astParser.analyzeInterface(ctx, pkgs, file, interfaceObj, interfaceType)

	require.NoError(t, err)
	require.NotNil(t, interfaceInfo)

	// Should have no type parameters
	assert.Len(t, interfaceInfo.TypeParams, 0)

	// Should still have methods
	assert.Len(t, interfaceInfo.Methods, 2)

	// Should have all other fields populated correctly
	assert.Equal(t, "Converter", interfaceInfo.Object.Name())
	assert.NotNil(t, interfaceInfo.Interface)
	assert.NotNil(t, interfaceInfo.Options)
	assert.NotEmpty(t, interfaceInfo.Marker)
}
