package domain

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicType(t *testing.T) {
	t.Run("creation and properties", func(t *testing.T) {
		basicType := NewBasicType("string", reflect.String)

		assert.Equal(t, "string", basicType.Name())
		assert.Equal(t, KindBasic, basicType.Kind())
		assert.Equal(t, "string", basicType.String())
		assert.False(t, basicType.Generic())
		assert.Nil(t, basicType.TypeParams())
		assert.Equal(t, basicType, basicType.Underlying())
		assert.Equal(t, "", basicType.Package())
		assert.Equal(t, "", basicType.ImportPath())
		assert.True(t, basicType.Comparable())
	})

	t.Run("assignability", func(t *testing.T) {
		stringType1 := NewBasicType("string", reflect.String)
		stringType2 := NewBasicType("string", reflect.String)
		intType := NewBasicType("int", reflect.Int)

		assert.True(t, stringType1.AssignableTo(stringType2))
		assert.False(t, stringType1.AssignableTo(intType))
		assert.False(t, stringType1.AssignableTo(nil))
	})

	t.Run("comparability", func(t *testing.T) {
		stringType := NewBasicType("string", reflect.String)
		funcType := NewBasicType("func", reflect.Func)
		mapType := NewBasicType("map", reflect.Map)
		sliceType := NewBasicType("slice", reflect.Slice)

		assert.True(t, stringType.Comparable())
		assert.False(t, funcType.Comparable())
		assert.False(t, mapType.Comparable())
		assert.False(t, sliceType.Comparable())
	})
}

func TestStructType(t *testing.T) {
	t.Run("creation and properties", func(t *testing.T) {
		field1, err := NewField("Name", StringType, 0, true)
		require.NoError(t, err)

		field2, err := NewField("Age", IntType, 1, true)
		require.NoError(t, err)

		fields := []Field{*field1, *field2}
		structType := NewStructType("User", fields, "example.com/pkg")

		assert.Equal(t, "User", structType.Name())
		assert.Equal(t, KindStruct, structType.Kind())
		assert.Equal(t, "User", structType.String())
		assert.False(t, structType.Generic())
		assert.Empty(t, structType.TypeParams())
		assert.Equal(t, structType, structType.Underlying())
		assert.Equal(t, "example.com/pkg", structType.Package())
		assert.Equal(t, "example.com/pkg", structType.ImportPath())
		assert.True(t, structType.Comparable())
	})

	t.Run("field access", func(t *testing.T) {
		field1, _ := NewField("Name", StringType, 0, true)
		field2, _ := NewField("Age", IntType, 1, true)

		fields := []Field{*field1, *field2}
		structType := NewStructType("User", fields, "example.com/pkg")

		// Test Fields() returns defensive copy
		returnedFields := structType.Fields()
		assert.Equal(t, fields, returnedFields)

		// Modify returned slice shouldn't affect original
		returnedFields[0].Name = "Modified"
		originalFields := structType.Fields()
		assert.Equal(t, "Name", originalFields[0].Name)

		// Test FieldByName
		foundField, exists := structType.FieldByName("Name")
		assert.True(t, exists)
		assert.Equal(t, "Name", foundField.Name)

		_, exists = structType.FieldByName("NonExistent")
		assert.False(t, exists)
	})

	t.Run("assignability", func(t *testing.T) {
		field1, _ := NewField("Name", StringType, 0, true)
		fields := []Field{*field1}

		structType1 := NewStructType("User", fields, "example.com/pkg")
		structType2 := NewStructType("User", fields, "example.com/pkg")
		structType3 := NewStructType("User", fields, "other.com/pkg")
		structType4 := NewStructType("Customer", fields, "example.com/pkg")

		assert.True(t, structType1.AssignableTo(structType2))
		assert.False(t, structType1.AssignableTo(structType3)) // Different package
		assert.False(t, structType1.AssignableTo(structType4)) // Different name
		assert.False(t, structType1.AssignableTo(nil))
	})
}

func TestSliceType(t *testing.T) {
	t.Run("creation and properties", func(t *testing.T) {
		sliceType := NewSliceType(StringType, "example.com/pkg")

		assert.Equal(t, "[]string", sliceType.Name())
		assert.Equal(t, KindSlice, sliceType.Kind())
		assert.Equal(t, "[]string", sliceType.String())
		assert.False(t, sliceType.Generic())
		assert.Empty(t, sliceType.TypeParams())
		assert.Equal(t, sliceType, sliceType.Underlying())
		assert.Equal(t, "example.com/pkg", sliceType.Package())
		assert.Equal(t, "example.com/pkg", sliceType.ImportPath())
		assert.False(t, sliceType.Comparable())
		assert.Equal(t, StringType, sliceType.Elem())
	})

	t.Run("assignability", func(t *testing.T) {
		stringSlice1 := NewSliceType(StringType, "pkg")
		stringSlice2 := NewSliceType(StringType, "pkg")
		intSlice := NewSliceType(IntType, "pkg")

		assert.True(t, stringSlice1.AssignableTo(stringSlice2))
		assert.False(t, stringSlice1.AssignableTo(intSlice))
		assert.False(t, stringSlice1.AssignableTo(nil))
	})
}

func TestPointerType(t *testing.T) {
	t.Run("creation and properties", func(t *testing.T) {
		ptrType := NewPointerType(StringType, "example.com/pkg")

		assert.Equal(t, "*string", ptrType.Name())
		assert.Equal(t, KindPointer, ptrType.Kind())
		assert.Equal(t, "*string", ptrType.String())
		assert.False(t, ptrType.Generic())
		assert.Empty(t, ptrType.TypeParams())
		assert.Equal(t, ptrType, ptrType.Underlying())
		assert.Equal(t, "example.com/pkg", ptrType.Package())
		assert.Equal(t, "example.com/pkg", ptrType.ImportPath())
		assert.True(t, ptrType.Comparable())
		assert.Equal(t, StringType, ptrType.Elem())
	})

	t.Run("assignability", func(t *testing.T) {
		stringPtr1 := NewPointerType(StringType, "pkg")
		stringPtr2 := NewPointerType(StringType, "pkg")
		intPtr := NewPointerType(IntType, "pkg")

		assert.True(t, stringPtr1.AssignableTo(stringPtr2))
		assert.False(t, stringPtr1.AssignableTo(intPtr))
		assert.False(t, stringPtr1.AssignableTo(nil))
	})
}

func TestGenericType(t *testing.T) {
	t.Run("creation and properties", func(t *testing.T) {
		constraint := StringType
		genericType := NewGenericType("T", constraint, 0, "example.com/pkg")

		assert.Equal(t, "T", genericType.Name())
		assert.Equal(t, KindGeneric, genericType.Kind())
		assert.Equal(t, "T", genericType.String())
		assert.True(t, genericType.Generic())

		typeParams := genericType.TypeParams()
		require.Len(t, typeParams, 1)
		assert.Equal(t, "T", typeParams[0].Name)
		assert.Equal(t, constraint, typeParams[0].Constraint)
		assert.Equal(t, 0, typeParams[0].Index)

		assert.Equal(t, constraint, genericType.Underlying())
		assert.Equal(t, "example.com/pkg", genericType.Package())
		assert.Equal(t, "example.com/pkg", genericType.ImportPath())
		assert.Equal(t, constraint, genericType.Constraint())
		assert.Equal(t, 0, genericType.Index())
	})

	t.Run("assignability based on constraint", func(t *testing.T) {
		genericType1 := NewGenericType("T", StringType, 0, "pkg")

		assert.True(t, genericType1.AssignableTo(StringType))
		// Note: Generic type assignability is complex and depends on constraint satisfaction
		// For now, we'll test that it delegates to constraint
		assert.False(t, genericType1.AssignableTo(IntType))
		assert.False(t, genericType1.AssignableTo(nil))
	})
}

func TestTypeBuilder(t *testing.T) {
	t.Run("build struct with validation", func(t *testing.T) {
		builder := NewTypeBuilder()

		field1, _ := NewField("Name", StringType, 0, true)
		field2, _ := NewField("Age", IntType, 1, true)
		fields := []Field{*field1, *field2}

		structType, err := builder.BuildStruct("User", "example.com/pkg", fields)
		require.NoError(t, err)

		assert.Equal(t, "User", structType.Name())
		assert.Equal(t, "example.com/pkg", structType.Package())
		assert.Len(t, structType.Fields(), 2)
	})

	t.Run("validation errors", func(t *testing.T) {
		builder := NewTypeBuilder()

		// Empty name
		_, err := builder.BuildStruct("", "pkg", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "struct name cannot be empty")

		// Duplicate field names
		field1, _ := NewField("Name", StringType, 0, true)
		field2, _ := NewField("Name", IntType, 1, true)
		fields := []Field{*field1, *field2}

		_, err = builder.BuildStruct("User", "pkg", fields)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate field name")

		// Nil field type
		fieldWithNilType := Field{Name: "Test", Type: nil, Position: 0, Exported: true}
		fields = []Field{fieldWithNilType}

		_, err = builder.BuildStruct("User", "pkg", fields)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "has nil type")
	})

	t.Run("caching", func(t *testing.T) {
		builder := NewTypeBuilder()

		field1, _ := NewField("Name", StringType, 0, true)
		fields := []Field{*field1}

		structType, err := builder.BuildStruct("User", "example.com/pkg", fields)
		require.NoError(t, err)

		// Should be cached
		cachedType, exists := builder.GetCachedType("example.com/pkg", "User")
		assert.True(t, exists)
		assert.Equal(t, structType, cachedType)

		// Non-existent type
		_, exists = builder.GetCachedType("other.com/pkg", "User")
		assert.False(t, exists)
	})
}

func TestEnhancedTypeParam(t *testing.T) {
	t.Run("basic type parameter creation", func(t *testing.T) {
		constraint := StringType
		tp := NewTypeParam("T", constraint, 0)

		assert.Equal(t, "T", tp.Name)
		assert.Equal(t, constraint, tp.Constraint)
		assert.Equal(t, 0, tp.Index)
		assert.False(t, tp.IsAny)
		assert.False(t, tp.IsComparable)
		assert.Nil(t, tp.UnionTypes)
		assert.Nil(t, tp.Underlying)
		assert.True(t, tp.IsValid())
		assert.Equal(t, "interface", tp.GetConstraintType())
	})

	t.Run("any type parameter", func(t *testing.T) {
		tp := NewAnyTypeParam("T", 0)

		assert.Equal(t, "T", tp.Name)
		assert.Nil(t, tp.Constraint)
		assert.Equal(t, 0, tp.Index)
		assert.True(t, tp.IsAny)
		assert.False(t, tp.IsComparable)
		assert.Nil(t, tp.UnionTypes)
		assert.Nil(t, tp.Underlying)
		assert.True(t, tp.IsValid())
		assert.Equal(t, "any", tp.GetConstraintType())
		assert.Equal(t, "T any", tp.String())
	})

	t.Run("comparable type parameter", func(t *testing.T) {
		tp := NewComparableTypeParam("T", 1)

		assert.Equal(t, "T", tp.Name)
		assert.Nil(t, tp.Constraint)
		assert.Equal(t, 1, tp.Index)
		assert.False(t, tp.IsAny)
		assert.True(t, tp.IsComparable)
		assert.Nil(t, tp.UnionTypes)
		assert.Nil(t, tp.Underlying)
		assert.True(t, tp.IsValid())
		assert.Equal(t, "comparable", tp.GetConstraintType())
		assert.Equal(t, "T comparable", tp.String())
	})

	t.Run("union type parameter", func(t *testing.T) {
		unionTypes := []Type{IntType, StringType, Float64Type}
		tp := NewUnionTypeParam("T", unionTypes, 2)

		assert.Equal(t, "T", tp.Name)
		assert.Nil(t, tp.Constraint)
		assert.Equal(t, 2, tp.Index)
		assert.False(t, tp.IsAny)
		assert.False(t, tp.IsComparable)
		assert.Len(t, tp.UnionTypes, 3)
		assert.Equal(t, IntType, tp.UnionTypes[0])
		assert.Equal(t, StringType, tp.UnionTypes[1])
		assert.Equal(t, Float64Type, tp.UnionTypes[2])
		assert.Nil(t, tp.Underlying)
		assert.True(t, tp.IsValid())
		assert.Equal(t, "union", tp.GetConstraintType())
		assert.Equal(t, "T int | string | float64", tp.String())
	})

	t.Run("underlying type parameter", func(t *testing.T) {
		underlyingConstraint := NewUnderlyingConstraint(StringType, "")
		tp := NewUnderlyingTypeParam("T", underlyingConstraint, 3)

		assert.Equal(t, "T", tp.Name)
		assert.Equal(t, StringType, tp.Constraint)
		assert.Equal(t, 3, tp.Index)
		assert.False(t, tp.IsAny)
		assert.False(t, tp.IsComparable)
		assert.Nil(t, tp.UnionTypes)
		assert.NotNil(t, tp.Underlying)
		assert.Equal(t, StringType, tp.Underlying.Type)
		assert.True(t, tp.IsValid())
		assert.Equal(t, "underlying", tp.GetConstraintType())
		assert.Equal(t, "T ~string", tp.String())
	})

	t.Run("constraint satisfaction - any constraint", func(t *testing.T) {
		tp := NewAnyTypeParam("T", 0)

		assert.True(t, tp.SatisfiesConstraint(StringType))
		assert.True(t, tp.SatisfiesConstraint(IntType))
		assert.True(t, tp.SatisfiesConstraint(BoolType))
		assert.False(t, tp.SatisfiesConstraint(nil))
	})

	t.Run("constraint satisfaction - comparable constraint", func(t *testing.T) {
		tp := NewComparableTypeParam("T", 0)

		assert.True(t, tp.SatisfiesConstraint(StringType))
		assert.True(t, tp.SatisfiesConstraint(IntType))
		assert.False(t, tp.SatisfiesConstraint(NewSliceType(IntType, "")))
		assert.False(t, tp.SatisfiesConstraint(nil))
	})

	t.Run("constraint satisfaction - union constraint", func(t *testing.T) {
		unionTypes := []Type{IntType, StringType}
		tp := NewUnionTypeParam("T", unionTypes, 0)

		assert.True(t, tp.SatisfiesConstraint(IntType))
		assert.True(t, tp.SatisfiesConstraint(StringType))
		assert.False(t, tp.SatisfiesConstraint(BoolType))
		assert.False(t, tp.SatisfiesConstraint(nil))
	})

	t.Run("constraint satisfaction - underlying constraint", func(t *testing.T) {
		underlyingConstraint := NewUnderlyingConstraint(StringType, "")
		tp := NewUnderlyingTypeParam("T", underlyingConstraint, 0)

		assert.True(t, tp.SatisfiesConstraint(StringType))
		assert.False(t, tp.SatisfiesConstraint(IntType))
		assert.False(t, tp.SatisfiesConstraint(nil))
	})

	t.Run("validation - empty name", func(t *testing.T) {
		tp := &TypeParam{Name: "", Constraint: StringType, Index: 0}
		assert.False(t, tp.IsValid())
	})

	t.Run("validation - multiple constraints invalid", func(t *testing.T) {
		tp := &TypeParam{
			Name:         "T",
			Constraint:   StringType,
			Index:        0,
			IsAny:        true, // This creates a conflict
			IsComparable: false,
		}
		assert.False(t, tp.IsValid())
	})

	t.Run("validation - single constraint valid", func(t *testing.T) {
		tp := &TypeParam{
			Name:       "T",
			Constraint: StringType,
			Index:      0,
		}
		assert.True(t, tp.IsValid())
	})

	t.Run("no constraint type parameter", func(t *testing.T) {
		tp := &TypeParam{Name: "T", Index: 0}
		assert.Equal(t, "none", tp.GetConstraintType())
		assert.Equal(t, "T", tp.String())
		assert.True(t, tp.SatisfiesConstraint(StringType)) // no constraint accepts any type
	})
}

func TestUnderlyingConstraint(t *testing.T) {
	t.Run("creation and properties", func(t *testing.T) {
		uc := NewUnderlyingConstraint(IntType, "math")

		assert.Equal(t, IntType, uc.Type)
		assert.Equal(t, "math", uc.Package)
	})

	t.Run("empty package", func(t *testing.T) {
		uc := NewUnderlyingConstraint(StringType, "")

		assert.Equal(t, StringType, uc.Type)
		assert.Equal(t, "", uc.Package)
	})
}

func TestTypeParamJSONSerialization(t *testing.T) {
	t.Run("any type parameter JSON", func(t *testing.T) {
		tp := NewAnyTypeParam("T", 0)

		data, err := json.Marshal(tp)
		require.NoError(t, err)

		// Verify serialized fields that don't contain Type interface
		assert.Contains(t, string(data), `"name":"T"`)
		assert.Contains(t, string(data), `"any":true`)
		assert.Contains(t, string(data), `"index":0`)

		// Test roundtrip for fields that don't contain Type interface
		var restored TypeParam
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		assert.Equal(t, "T", restored.Name)
		assert.True(t, restored.IsAny)
		assert.Equal(t, 0, restored.Index)
		assert.Equal(t, "any", restored.GetConstraintType())
	})

	t.Run("comparable type parameter JSON", func(t *testing.T) {
		tp := NewComparableTypeParam("T", 1)

		data, err := json.Marshal(tp)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"name":"T"`)
		assert.Contains(t, string(data), `"comparable":true`)
		assert.Contains(t, string(data), `"index":1`)

		var restored TypeParam
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		assert.Equal(t, "T", restored.Name)
		assert.True(t, restored.IsComparable)
		assert.Equal(t, 1, restored.Index)
		assert.Equal(t, "comparable", restored.GetConstraintType())
	})

	t.Run("serialization contains correct JSON tags", func(t *testing.T) {
		// Test that new fields have correct JSON tags in the struct
		tp := &TypeParam{
			Name:         "T",
			Index:        0,
			IsAny:        true,
			IsComparable: false,
		}

		data, err := json.Marshal(tp)
		require.NoError(t, err)

		// Verify JSON tag naming follows convention
		assert.Contains(t, string(data), `"name":"T"`)
		assert.Contains(t, string(data), `"index":0`)
		assert.Contains(t, string(data), `"any":true`)
		// comparable field should be omitted when false with omitempty tag
		assert.NotContains(t, string(data), `"comparable"`)
	})
}

func TestBackwardCompatibility(t *testing.T) {
	t.Run("legacy TypeParam struct literal", func(t *testing.T) {
		// Test that existing code using struct literals still works
		tp := &TypeParam{
			Name:       "T",
			Constraint: StringType,
			Index:      0,
		}

		assert.Equal(t, "T", tp.Name)
		assert.Equal(t, StringType, tp.Constraint)
		assert.Equal(t, 0, tp.Index)
		assert.True(t, tp.IsValid())
		assert.Equal(t, "interface", tp.GetConstraintType())
		assert.Equal(t, "T string", tp.String())

		// New fields should have zero values
		assert.False(t, tp.IsAny)
		assert.False(t, tp.IsComparable)
		assert.Nil(t, tp.UnionTypes)
		assert.Nil(t, tp.Underlying)
	})

	t.Run("legacy GenericType still works", func(t *testing.T) {
		// Test that existing GenericType usage still works
		constraint := StringType
		genericType := NewGenericType("T", constraint, 0, "example.com/pkg")

		assert.Equal(t, "T", genericType.Name())
		assert.Equal(t, KindGeneric, genericType.Kind())
		assert.True(t, genericType.Generic())

		typeParams := genericType.TypeParams()
		require.Len(t, typeParams, 1)
		assert.Equal(t, "T", typeParams[0].Name)
		assert.Equal(t, constraint, typeParams[0].Constraint)
		assert.Equal(t, 0, typeParams[0].Index)

		// Should work with the enhanced validation too
		assert.True(t, typeParams[0].IsValid())
	})
}

func TestCommonBasicTypes(t *testing.T) {
	tests := []struct {
		name     string
		typ      *BasicType
		expected string
		kind     reflect.Kind
	}{
		{"string", StringType, "string", reflect.String},
		{"int", IntType, "int", reflect.Int},
		{"int8", Int8Type, "int8", reflect.Int8},
		{"int16", Int16Type, "int16", reflect.Int16},
		{"int32", Int32Type, "int32", reflect.Int32},
		{"int64", Int64Type, "int64", reflect.Int64},
		{"uint", UintType, "uint", reflect.Uint},
		{"uint8", Uint8Type, "uint8", reflect.Uint8},
		{"uint16", Uint16Type, "uint16", reflect.Uint16},
		{"uint32", Uint32Type, "uint32", reflect.Uint32},
		{"uint64", Uint64Type, "uint64", reflect.Uint64},
		{"float32", Float32Type, "float32", reflect.Float32},
		{"float64", Float64Type, "float64", reflect.Float64},
		{"bool", BoolType, "bool", reflect.Bool},
		{"byte", ByteType, "byte", reflect.Uint8},
		{"rune", RuneType, "rune", reflect.Int32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.typ.Name())
			assert.Equal(t, tt.expected, tt.typ.String())
			assert.Equal(t, KindBasic, tt.typ.Kind())
			assert.Equal(t, tt.kind, tt.typ.kind)
		})
	}
}

func TestTypeKindString(t *testing.T) {
	tests := []struct {
		kind     TypeKind
		expected string
	}{
		{KindBasic, "basic"},
		{KindStruct, "struct"},
		{KindSlice, "slice"},
		{KindMap, "map"},
		{KindInterface, "interface"},
		{KindPointer, "pointer"},
		{KindGeneric, "generic"},
		{KindNamed, "named"},
		{KindFunction, "function"},
		{TypeKind(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.kind.String())
		})
	}
}
