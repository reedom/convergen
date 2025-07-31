package domain

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestField(t *testing.T) {
	t.Run("creation with validation", func(t *testing.T) {
		field, err := NewField("Name", StringType, 0, true)
		require.NoError(t, err)

		assert.Equal(t, "Name", field.Name)
		assert.Equal(t, StringType, field.Type)
		assert.Equal(t, 0, field.Position)
		assert.True(t, field.Exported)
		assert.Equal(t, "", field.Doc)
		assert.Equal(t, reflect.StructTag(""), field.Tags)
	})

	t.Run("validation errors", func(t *testing.T) {
		// Empty name
		_, err := NewField("", StringType, 0, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "field name cannot be empty")

		// Nil type
		_, err = NewField("Name", nil, 0, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "field type cannot be nil")
	})
}

func TestFieldSpec(t *testing.T) {
	t.Run("creation with validation", func(t *testing.T) {
		path := []string{"User", "Name"}
		spec, err := NewFieldSpec(path, StringType)
		require.NoError(t, err)

		assert.Equal(t, path, spec.Path)
		assert.Equal(t, StringType, spec.Type)
		assert.False(t, spec.IsMethod)
		assert.Nil(t, spec.Receiver)
	})

	t.Run("method spec creation", func(t *testing.T) {
		path := []string{"User", "GetName"}
		receiverType := NewPointerType(StringType, "pkg")

		spec, err := NewMethodSpec(path, StringType, receiverType)
		require.NoError(t, err)

		assert.Equal(t, path, spec.Path)
		assert.Equal(t, StringType, spec.Type)
		assert.True(t, spec.IsMethod)
		assert.Equal(t, receiverType, spec.Receiver)
	})

	t.Run("validation errors", func(t *testing.T) {
		// Empty path
		_, err := NewFieldSpec([]string{}, StringType)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "field path cannot be empty")

		// Nil type
		_, err = NewFieldSpec([]string{"User", "Name"}, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "field type cannot be nil")
	})

	t.Run("string representation", func(t *testing.T) {
		// Regular field
		spec, _ := NewFieldSpec([]string{"User", "Name"}, StringType)
		assert.Equal(t, "User.Name", spec.String())

		// Method field
		receiverType := NewPointerType(StringType, "pkg")
		methodSpec, _ := NewMethodSpec([]string{"User", "GetName"}, StringType, receiverType)
		assert.Equal(t, "User.GetName()", methodSpec.String())
	})

	t.Run("field name extraction", func(t *testing.T) {
		spec, _ := NewFieldSpec([]string{"User", "Address", "Street"}, StringType)
		assert.Equal(t, "Street", spec.FieldName())

		// Empty path
		emptySpec := &FieldSpec{Path: []string{}}
		assert.Equal(t, "", emptySpec.FieldName())
	})

	t.Run("parent path extraction", func(t *testing.T) {
		spec, _ := NewFieldSpec([]string{"User", "Address", "Street"}, StringType)
		parentPath := spec.ParentPath()
		expected := []string{"User", "Address"}
		assert.Equal(t, expected, parentPath)

		// Single element path
		singleSpec, _ := NewFieldSpec([]string{"Name"}, StringType)
		assert.Nil(t, singleSpec.ParentPath())

		// Empty path
		emptySpec := &FieldSpec{Path: []string{}}
		assert.Nil(t, emptySpec.ParentPath())
	})

	t.Run("defensive copying", func(t *testing.T) {
		originalPath := []string{"User", "Name"}
		spec, _ := NewFieldSpec(originalPath, StringType)

		// Modify original path
		originalPath[0] = "Modified"

		// Spec should be unaffected
		assert.Equal(t, "User", spec.Path[0])
	})
}

func TestFieldMapping(t *testing.T) {
	t.Run("creation with validation", func(t *testing.T) {
		sourceSpec, _ := NewFieldSpec([]string{"src", "Name"}, StringType)
		destSpec, _ := NewFieldSpec([]string{"dst", "Name"}, StringType)
		strategy := &DirectAssignmentStrategy{}

		mapping, err := NewFieldMapping("mapping1", sourceSpec, destSpec, strategy)
		require.NoError(t, err)

		assert.Equal(t, "mapping1", mapping.ID)
		assert.Equal(t, sourceSpec, mapping.Source)
		assert.Equal(t, destSpec, mapping.Dest)
		assert.Equal(t, strategy, mapping.Strategy)
		assert.Equal(t, "direct", mapping.StrategyName)
		assert.NotNil(t, mapping.Config)
		assert.Empty(t, mapping.Dependencies)
	})

	t.Run("validation errors", func(t *testing.T) {
		sourceSpec, _ := NewFieldSpec([]string{"src", "Name"}, StringType)
		destSpec, _ := NewFieldSpec([]string{"dst", "Name"}, StringType)
		strategy := &DirectAssignmentStrategy{}

		// Empty ID
		_, err := NewFieldMapping("", sourceSpec, destSpec, strategy)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mapping ID cannot be empty")

		// Nil source
		_, err = NewFieldMapping("mapping1", nil, destSpec, strategy)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "source field spec cannot be nil")

		// Nil destination
		_, err = NewFieldMapping("mapping1", sourceSpec, nil, strategy)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "destination field spec cannot be nil")

		// Nil strategy
		_, err = NewFieldMapping("mapping1", sourceSpec, destSpec, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "conversion strategy cannot be nil")
	})

	t.Run("dependency management", func(t *testing.T) {
		sourceSpec, _ := NewFieldSpec([]string{"src", "Name"}, StringType)
		destSpec, _ := NewFieldSpec([]string{"dst", "Name"}, StringType)
		strategy := &DirectAssignmentStrategy{}

		mapping, _ := NewFieldMapping("mapping1", sourceSpec, destSpec, strategy)

		// Add dependency
		err := mapping.AddDependency("mapping2")
		assert.NoError(t, err)
		assert.Contains(t, mapping.Dependencies, "mapping2")

		// Add duplicate dependency (should not error)
		err = mapping.AddDependency("mapping2")
		assert.NoError(t, err)
		assert.Len(t, mapping.Dependencies, 1) // Should still be 1

		// Add self-dependency (should error)
		err = mapping.AddDependency("mapping1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "field mapping cannot depend on itself")

		// Add empty dependency (should error)
		err = mapping.AddDependency("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dependency ID cannot be empty")
	})
}

func TestMappingConfig(t *testing.T) {
	t.Run("converter func", func(t *testing.T) {
		converter := &ConverterFunc{
			Name:       "Convert",
			Package:    "pkg",
			ImportPath: "example.com/pkg",
			Args:       []string{"arg1", "arg2"},
			ReturnsErr: true,
		}

		assert.Equal(t, "Convert", converter.Name)
		assert.Equal(t, "pkg", converter.Package)
		assert.Equal(t, "example.com/pkg", converter.ImportPath)
		assert.Equal(t, []string{"arg1", "arg2"}, converter.Args)
		assert.True(t, converter.ReturnsErr)
	})

	t.Run("literal value", func(t *testing.T) {
		literal := &LiteralValue{
			Value: "\"default\"",
			Type:  StringType,
		}

		assert.Equal(t, "\"default\"", literal.Value)
		assert.Equal(t, StringType, literal.Type)
	})
}

func TestErrorHandlingStrategy(t *testing.T) {
	tests := []struct {
		strategy ErrorHandlingStrategy
		expected string
	}{
		{ErrorIgnore, "ignore"},
		{ErrorPropagate, "propagate"},
		{ErrorPanic, "panic"},
		{ErrorDefault, "default"},
		{ErrorHandlingStrategy(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.strategy.String())
		})
	}
}

func TestDirectAssignmentStrategy(t *testing.T) {
	strategy := &DirectAssignmentStrategy{}

	t.Run("properties", func(t *testing.T) {
		assert.Equal(t, "direct", strategy.Name())
		assert.Equal(t, 100, strategy.Priority())
		assert.Nil(t, strategy.Dependencies())
	})

	t.Run("can handle", func(t *testing.T) {
		// Mock assignable types
		assert.True(t, strategy.CanHandle(StringType, StringType))
		// Note: Real assignability would be tested with proper type system
		assert.NotNil(t, strategy)
	})

	t.Run("code generation", func(t *testing.T) {
		sourceSpec, _ := NewFieldSpec([]string{"src", "Name"}, StringType)
		destSpec, _ := NewFieldSpec([]string{"dst", "Name"}, StringType)
		mapping, _ := NewFieldMapping("test", sourceSpec, destSpec, strategy)

		code, err := strategy.GenerateCode(mapping)
		require.NoError(t, err)

		assert.Equal(t, "dst.Name = src.Name", code.Assignment)
		assert.Empty(t, code.Variables)
		assert.Empty(t, code.Imports)
		assert.Empty(t, code.PreCode)
		assert.Empty(t, code.PostCode)
		assert.Nil(t, code.Error)
	})
}

func TestTypeCastStrategy(t *testing.T) {
	strategy := &TypeCastStrategy{}

	t.Run("properties", func(t *testing.T) {
		assert.Equal(t, "typecast", strategy.Name())
		assert.Equal(t, 80, strategy.Priority())
		assert.Nil(t, strategy.Dependencies())
	})

	t.Run("can handle", func(t *testing.T) {
		// Basic types
		assert.True(t, strategy.CanHandle(IntType, Int64Type))
		assert.True(t, strategy.CanHandle(StringType, StringType))
	})

	t.Run("code generation", func(t *testing.T) {
		sourceSpec, _ := NewFieldSpec([]string{"src", "ID"}, IntType)
		destSpec, _ := NewFieldSpec([]string{"dst", "ID"}, Int64Type)
		mapping, _ := NewFieldMapping("test", sourceSpec, destSpec, strategy)

		code, err := strategy.GenerateCode(mapping)
		require.NoError(t, err)

		assert.Equal(t, "dst.ID = int64(src.ID)", code.Assignment)
	})
}

func TestMethodCallStrategy(t *testing.T) {
	strategy := &MethodCallStrategy{}

	t.Run("properties", func(t *testing.T) {
		assert.Equal(t, "method", strategy.Name())
		assert.Equal(t, 90, strategy.Priority())
		assert.Nil(t, strategy.Dependencies())
	})

	t.Run("code generation with method", func(t *testing.T) {
		receiverType := NewPointerType(StringType, "pkg")
		sourceSpec, _ := NewMethodSpec([]string{"src", "GetName"}, StringType, receiverType)
		destSpec, _ := NewFieldSpec([]string{"dst", "Name"}, StringType)
		mapping, _ := NewFieldMapping("test", sourceSpec, destSpec, strategy)

		code, err := strategy.GenerateCode(mapping)
		require.NoError(t, err)

		assert.Equal(t, "dst.Name = src.GetName()", code.Assignment)
	})

	t.Run("error with non-method source", func(t *testing.T) {
		sourceSpec, _ := NewFieldSpec([]string{"src", "Name"}, StringType)
		destSpec, _ := NewFieldSpec([]string{"dst", "Name"}, StringType)
		mapping, _ := NewFieldMapping("test", sourceSpec, destSpec, strategy)

		_, err := strategy.GenerateCode(mapping)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "source is not a method call")
	})
}

func TestConverterFuncStrategy(t *testing.T) {
	strategy := &ConverterFuncStrategy{}

	t.Run("properties", func(t *testing.T) {
		assert.Equal(t, "converter", strategy.Name())
		assert.Equal(t, 70, strategy.Priority())
		assert.Nil(t, strategy.Dependencies())
		assert.True(t, strategy.CanHandle(StringType, IntType)) // Can handle any types
	})

	t.Run("code generation without error", func(t *testing.T) {
		sourceSpec, _ := NewFieldSpec([]string{"src", "Name"}, StringType)
		destSpec, _ := NewFieldSpec([]string{"dst", "ID"}, IntType)
		mapping, _ := NewFieldMapping("test", sourceSpec, destSpec, strategy)

		mapping.Config.Converter = &ConverterFunc{
			Name:       "ParseInt",
			Package:    "strconv",
			ImportPath: "strconv",
			Args:       []string{"10"},
			ReturnsErr: false,
		}

		code, err := strategy.GenerateCode(mapping)
		require.NoError(t, err)

		assert.Equal(t, "dst.ID = ParseInt(src.Name, 10)", code.Assignment)
		assert.Len(t, code.Imports, 1)
		assert.Equal(t, "strconv", code.Imports[0].Path)
		assert.Nil(t, code.Error)
	})

	t.Run("code generation with error", func(t *testing.T) {
		sourceSpec, _ := NewFieldSpec([]string{"src", "Name"}, StringType)
		destSpec, _ := NewFieldSpec([]string{"dst", "ID"}, IntType)
		mapping, _ := NewFieldMapping("test", sourceSpec, destSpec, strategy)

		mapping.Config.Converter = &ConverterFunc{
			Name:       "ParseInt",
			Package:    "strconv",
			ImportPath: "strconv",
			Args:       []string{"10"},
			ReturnsErr: true,
		}

		code, err := strategy.GenerateCode(mapping)
		require.NoError(t, err)

		assert.Equal(t, "dst.ID, err_test := ParseInt(src.Name, 10)", code.Assignment)
		assert.NotNil(t, code.Error)
		assert.Equal(t, "err_test", code.Error.Variable)
		assert.Equal(t, "if err_test != nil", code.Error.Check)
	})

	t.Run("error without converter config", func(t *testing.T) {
		sourceSpec, _ := NewFieldSpec([]string{"src", "Name"}, StringType)
		destSpec, _ := NewFieldSpec([]string{"dst", "ID"}, IntType)
		mapping, _ := NewFieldMapping("test", sourceSpec, destSpec, strategy)

		_, err := strategy.GenerateCode(mapping)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "converter function not configured")
	})
}

func TestLiteralStrategy(t *testing.T) {
	strategy := &LiteralStrategy{}

	t.Run("properties", func(t *testing.T) {
		assert.Equal(t, "literal", strategy.Name())
		assert.Equal(t, 60, strategy.Priority())
		assert.Nil(t, strategy.Dependencies())
		assert.True(t, strategy.CanHandle(StringType, IntType)) // Can handle any types
	})

	t.Run("code generation", func(t *testing.T) {
		sourceSpec, _ := NewFieldSpec([]string{"src", "Name"}, StringType)
		destSpec, _ := NewFieldSpec([]string{"dst", "Name"}, StringType)
		mapping, _ := NewFieldMapping("test", sourceSpec, destSpec, strategy)

		mapping.Config.Literal = &LiteralValue{
			Value: "\"default\"",
			Type:  StringType,
		}

		code, err := strategy.GenerateCode(mapping)
		require.NoError(t, err)

		assert.Equal(t, "dst.Name = \"default\"", code.Assignment)
	})

	t.Run("error without literal config", func(t *testing.T) {
		sourceSpec, _ := NewFieldSpec([]string{"src", "Name"}, StringType)
		destSpec, _ := NewFieldSpec([]string{"dst", "Name"}, StringType)
		mapping, _ := NewFieldMapping("test", sourceSpec, destSpec, strategy)

		_, err := strategy.GenerateCode(mapping)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "literal value not configured")
	})
}

func TestDefaultConversionStrategies(t *testing.T) {
	strategies := DefaultConversionStrategies()

	assert.Len(t, strategies, 5)

	// Check that all expected strategies are present
	names := make(map[string]bool)
	for _, strategy := range strategies {
		names[strategy.Name()] = true
	}

	assert.True(t, names["direct"])
	assert.True(t, names["typecast"])
	assert.True(t, names["method"])
	assert.True(t, names["converter"])
	assert.True(t, names["literal"])
}
