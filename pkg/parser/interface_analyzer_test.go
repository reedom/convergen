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

	"github.com/reedom/convergen/v8/pkg/internal/events"
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
			defer eventBus.Close()

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
			defer eventBus.Close()

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
			defer eventBus.Close()

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
	defer eventBus.Close()
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
