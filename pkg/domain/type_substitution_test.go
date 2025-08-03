package domain

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewTypeSubstitutionEngine(t *testing.T) {
	tests := []struct {
		name        string
		typeBuilder *TypeBuilder
		logger      *zap.Logger
		expectNil   bool
	}{
		{
			name:        "with valid dependencies",
			typeBuilder: NewTypeBuilder(),
			logger:      zap.NewNop(),
			expectNil:   false,
		},
		{
			name:        "with nil type builder",
			typeBuilder: nil,
			logger:      zap.NewNop(),
			expectNil:   false, // should create default
		},
		{
			name:        "with nil logger",
			typeBuilder: NewTypeBuilder(),
			logger:      nil,
			expectNil:   false, // should create no-op
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewTypeSubstitutionEngine(tt.typeBuilder, tt.logger)

			if tt.expectNil && engine != nil {
				t.Errorf("expected nil engine, got %v", engine)
			}

			if !tt.expectNil && engine == nil {
				t.Errorf("expected non-nil engine, got nil")
			}

			if engine != nil {
				// Verify default configuration
				if engine.maxRecursionDepth != DefaultMaxRecursionDepth {
					t.Errorf("expected max recursion depth %d, got %d", DefaultMaxRecursionDepth, engine.maxRecursionDepth)
				}

				if !engine.enableCaching {
					t.Errorf("expected caching to be enabled by default")
				}
			}
		})
	}
}

func TestNewTypeSubstitutionEngineWithConfig(t *testing.T) {
	config := &SubstitutionEngineConfig{
		MaxRecursionDepth:    50,
		EnableCaching:        false,
		CacheCapacity:        500,
		CycleLimitDepth:      25,
		OptimizePerformance:  false,
		EnableDetailedStats:  false,
		EnableMemoryTracking: true,
	}

	engine := NewTypeSubstitutionEngineWithConfig(NewTypeBuilder(), config, zap.NewNop())

	if engine.maxRecursionDepth != 50 {
		t.Errorf("expected max recursion depth 50, got %d", engine.maxRecursionDepth)
	}

	if engine.enableCaching {
		t.Errorf("expected caching to be disabled")
	}

	if engine.cacheCapacity != 500 {
		t.Errorf("expected cache capacity 500, got %d", engine.cacheCapacity)
	}

	if engine.cycleLimitDepth != 25 {
		t.Errorf("expected cycle limit depth 25, got %d", engine.cycleLimitDepth)
	}
}

func TestSubstituteType_BasicTypes(t *testing.T) {
	engine := NewTypeSubstitutionEngine(NewTypeBuilder(), zap.NewNop())

	// Test basic type substitution (should return unchanged)
	basicType := StringType
	typeParams := []TypeParam{}
	typeArgs := []Type{}

	result, err := engine.SubstituteType(basicType, typeParams, typeArgs)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.SubstitutedType != basicType {
		t.Errorf("expected substituted type to be same as original for basic types")
	}

	if result.OriginalType != basicType {
		t.Errorf("expected original type to be preserved")
	}
}

func TestSubstituteType_GenericTypes(t *testing.T) {
	engine := NewTypeSubstitutionEngine(NewTypeBuilder(), zap.NewNop())

	// Create a generic type parameter T
	genericType := NewGenericType("T", nil, 0, "")

	// Create type parameter and argument
	typeParam := NewTypeParam("T", nil, 0)
	typeArg := StringType

	result, err := engine.SubstituteType(genericType, []TypeParam{*typeParam}, []Type{typeArg})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.SubstitutedType != typeArg {
		t.Errorf("expected substituted type to be %v, got %v", typeArg, result.SubstitutedType)
	}

	// Verify type mapping
	if len(result.TypeMapping) != 1 {
		t.Errorf("expected 1 type mapping, got %d", len(result.TypeMapping))
	}

	if mappedType, found := result.TypeMapping["T"]; !found || mappedType != typeArg {
		t.Errorf("expected type mapping T -> %v, got %v", typeArg, mappedType)
	}
}

func TestSubstituteType_SliceTypes(t *testing.T) {
	engine := NewTypeSubstitutionEngine(NewTypeBuilder(), zap.NewNop())

	// Create []T where T is a generic type parameter
	genericElemType := NewGenericType("T", nil, 0, "")
	sliceType := NewSliceType(genericElemType, "")

	// Create substitution T -> string
	typeParam := NewTypeParam("T", nil, 0)
	typeArg := StringType

	result, err := engine.SubstituteType(sliceType, []TypeParam{*typeParam}, []Type{typeArg})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify the result is []string
	substitutedSlice, ok := result.SubstitutedType.(*SliceType)
	if !ok {
		t.Errorf("expected result to be SliceType, got %T", result.SubstitutedType)
	}

	if substitutedSlice.Elem() != typeArg {
		t.Errorf("expected slice element type to be %v, got %v", typeArg, substitutedSlice.Elem())
	}
}

func TestSubstituteType_PointerTypes(t *testing.T) {
	engine := NewTypeSubstitutionEngine(NewTypeBuilder(), zap.NewNop())

	// Create *T where T is a generic type parameter
	genericElemType := NewGenericType("T", nil, 0, "")
	pointerType := NewPointerType(genericElemType, "")

	// Create substitution T -> int
	typeParam := NewTypeParam("T", nil, 0)
	typeArg := IntType

	result, err := engine.SubstituteType(pointerType, []TypeParam{*typeParam}, []Type{typeArg})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify the result is *int
	substitutedPointer, ok := result.SubstitutedType.(*PointerType)
	if !ok {
		t.Errorf("expected result to be PointerType, got %T", result.SubstitutedType)
	}

	if substitutedPointer.Elem() != typeArg {
		t.Errorf("expected pointer element type to be %v, got %v", typeArg, substitutedPointer.Elem())
	}
}

func TestSubstituteType_StructTypes(t *testing.T) {
	engine := NewTypeSubstitutionEngine(NewTypeBuilder(), zap.NewNop())

	// Create struct with generic field: struct { Field T }
	genericFieldType := NewGenericType("T", nil, 0, "")
	field, err := NewField("Field", genericFieldType, 0, true)
	if err != nil {
		t.Fatalf("failed to create field: %v", err)
	}

	structType := NewStructType("TestStruct", []Field{*field}, "")

	// Create substitution T -> bool
	typeParam := NewTypeParam("T", nil, 0)
	typeArg := BoolType

	result, err := engine.SubstituteType(structType, []TypeParam{*typeParam}, []Type{typeArg})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify the field type was substituted
	substitutedStruct, ok := result.SubstitutedType.(*StructType)
	if !ok {
		t.Errorf("expected result to be StructType, got %T", result.SubstitutedType)
	}

	fields := substitutedStruct.Fields()
	if len(fields) != 1 {
		t.Errorf("expected 1 field, got %d", len(fields))
	}

	if fields[0].Type != typeArg {
		t.Errorf("expected field type to be %v, got %v", typeArg, fields[0].Type)
	}
}

func TestSubstituteType_NestedGenericTypes(t *testing.T) {
	engine := NewTypeSubstitutionEngine(NewTypeBuilder(), zap.NewNop())

	// Create []*T where T is a generic type parameter
	genericElemType := NewGenericType("T", nil, 0, "")
	pointerType := NewPointerType(genericElemType, "")
	sliceType := NewSliceType(pointerType, "")

	// Create substitution T -> float64
	typeParam := NewTypeParam("T", nil, 0)
	typeArg := Float64Type

	result, err := engine.SubstituteType(sliceType, []TypeParam{*typeParam}, []Type{typeArg})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify the result is []*float64
	substitutedSlice, ok := result.SubstitutedType.(*SliceType)
	if !ok {
		t.Errorf("expected result to be SliceType, got %T", result.SubstitutedType)
	}

	substitutedPointer, ok := substitutedSlice.Elem().(*PointerType)
	if !ok {
		t.Errorf("expected slice element to be PointerType, got %T", substitutedSlice.Elem())
	}

	if substitutedPointer.Elem() != typeArg {
		t.Errorf("expected pointer element type to be %v, got %v", typeArg, substitutedPointer.Elem())
	}
}

func TestSubstituteType_MultipleTypeParameters(t *testing.T) {
	engine := NewTypeSubstitutionEngine(NewTypeBuilder(), zap.NewNop())

	// Create struct with multiple generic fields: struct { A T; B U }
	genericTypeT := NewGenericType("T", nil, 0, "")
	genericTypeU := NewGenericType("U", nil, 1, "")

	fieldA, err := NewField("A", genericTypeT, 0, true)
	if err != nil {
		t.Fatalf("failed to create field A: %v", err)
	}

	fieldB, err := NewField("B", genericTypeU, 1, true)
	if err != nil {
		t.Fatalf("failed to create field B: %v", err)
	}

	structType := NewStructType("TestStruct", []Field{*fieldA, *fieldB}, "")

	// Create substitutions T -> string, U -> int
	typeParamT := NewTypeParam("T", nil, 0)
	typeParamU := NewTypeParam("U", nil, 1)
	typeArgT := StringType
	typeArgU := IntType

	result, err := engine.SubstituteType(structType,
		[]TypeParam{*typeParamT, *typeParamU},
		[]Type{typeArgT, typeArgU})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify both field types were substituted
	substitutedStruct, ok := result.SubstitutedType.(*StructType)
	if !ok {
		t.Errorf("expected result to be StructType, got %T", result.SubstitutedType)
	}

	fields := substitutedStruct.Fields()
	if len(fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(fields))
	}

	if fields[0].Type != typeArgT {
		t.Errorf("expected field A type to be %v, got %v", typeArgT, fields[0].Type)
	}

	if fields[1].Type != typeArgU {
		t.Errorf("expected field B type to be %v, got %v", typeArgU, fields[1].Type)
	}

	// Verify type mappings
	if len(result.TypeMapping) != 2 {
		t.Errorf("expected 2 type mappings, got %d", len(result.TypeMapping))
	}
}

func TestSubstituteType_InputValidation(t *testing.T) {
	engine := NewTypeSubstitutionEngine(NewTypeBuilder(), zap.NewNop())

	tests := []struct {
		name        string
		genericType Type
		typeParams  []TypeParam
		typeArgs    []Type
		expectError bool
		errorType   error
	}{
		{
			name:        "nil generic type",
			genericType: nil,
			typeParams:  []TypeParam{},
			typeArgs:    []Type{},
			expectError: true,
			errorType:   ErrSubstitutionTypeNil,
		},
		{
			name:        "nil type params",
			genericType: StringType,
			typeParams:  nil,
			typeArgs:    []Type{},
			expectError: true,
			errorType:   ErrSubstitutionTypeParamsNil,
		},
		{
			name:        "nil type args",
			genericType: StringType,
			typeParams:  []TypeParam{},
			typeArgs:    nil,
			expectError: true,
			errorType:   ErrSubstitutionTypeArgsNil,
		},
		{
			name:        "mismatched parameter count",
			genericType: StringType,
			typeParams:  []TypeParam{*NewTypeParam("T", nil, 0)},
			typeArgs:    []Type{},
			expectError: true,
			errorType:   ErrSubstitutionParameterMismatch,
		},
		{
			name:        "nil type argument",
			genericType: StringType,
			typeParams:  []TypeParam{*NewTypeParam("T", nil, 0)},
			typeArgs:    []Type{nil},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := engine.SubstituteType(tt.genericType, tt.typeParams, tt.typeArgs)

			if tt.expectError && err == nil {
				t.Errorf("expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.expectError && tt.errorType != nil {
				if !strings.Contains(err.Error(), tt.errorType.Error()) {
					t.Errorf("expected error containing %v, got %v", tt.errorType, err)
				}
			}
		})
	}
}

func TestSubstituteType_Caching(t *testing.T) {
	engine := NewTypeSubstitutionEngine(NewTypeBuilder(), zap.NewNop())

	// Create a substitution scenario
	genericType := NewGenericType("T", nil, 0, "")
	typeParam := NewTypeParam("T", nil, 0)
	typeArg := StringType

	// First substitution - should be cache miss
	result1, err := engine.SubstituteType(genericType, []TypeParam{*typeParam}, []Type{typeArg})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result1.CacheHit {
		t.Errorf("expected cache miss for first substitution")
	}

	// Second substitution - should be cache hit
	result2, err := engine.SubstituteType(genericType, []TypeParam{*typeParam}, []Type{typeArg})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !result2.CacheHit {
		t.Errorf("expected cache hit for second substitution")
	}

	// Verify cache size
	if engine.GetCacheSize() != 1 {
		t.Errorf("expected cache size 1, got %d", engine.GetCacheSize())
	}
}

func TestSubstituteType_CacheDisabled(t *testing.T) {
	config := NewSubstitutionEngineConfig()
	config.EnableCaching = false

	engine := NewTypeSubstitutionEngineWithConfig(NewTypeBuilder(), config, zap.NewNop())

	// Create a substitution scenario
	genericType := NewGenericType("T", nil, 0, "")
	typeParam := NewTypeParam("T", nil, 0)
	typeArg := StringType

	// First substitution
	result1, err := engine.SubstituteType(genericType, []TypeParam{*typeParam}, []Type{typeArg})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result1.CacheHit {
		t.Errorf("expected no cache hit when caching disabled")
	}

	// Second substitution
	result2, err := engine.SubstituteType(genericType, []TypeParam{*typeParam}, []Type{typeArg})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result2.CacheHit {
		t.Errorf("expected no cache hit when caching disabled")
	}

	// Verify cache size is 0
	if engine.GetCacheSize() != 0 {
		t.Errorf("expected cache size 0 when disabled, got %d", engine.GetCacheSize())
	}
}

func TestSubstituteType_RecursionLimit(t *testing.T) {
	config := NewSubstitutionEngineConfig()
	config.MaxRecursionDepth = 5 // Set low limit for testing

	engine := NewTypeSubstitutionEngineWithConfig(NewTypeBuilder(), config, zap.NewNop())

	// Create deeply nested type: [][][][][][]T (6 levels deep)
	genericType := NewGenericType("T", nil, 0, "")
	nestedType := Type(genericType)

	for i := 0; i < 6; i++ {
		nestedType = NewSliceType(nestedType, "")
	}

	typeParam := NewTypeParam("T", nil, 0)
	typeArg := StringType

	_, err := engine.SubstituteType(nestedType, []TypeParam{*typeParam}, []Type{typeArg})
	if err == nil {
		t.Errorf("expected recursion limit error")
	}

	if !strings.Contains(err.Error(), "recursion limit") {
		t.Errorf("expected recursion limit error, got: %v", err)
	}
}

func TestSubstituteType_PerformanceStats(t *testing.T) {
	config := NewSubstitutionEngineConfig()
	config.EnableDetailedStats = true

	engine := NewTypeSubstitutionEngineWithConfig(NewTypeBuilder(), config, zap.NewNop())

	// Create a substitution scenario
	genericType := NewGenericType("T", nil, 0, "")
	typeParam := NewTypeParam("T", nil, 0)
	typeArg := StringType

	result, err := engine.SubstituteType(genericType, []TypeParam{*typeParam}, []Type{typeArg})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify performance stats are populated
	if result.PerformanceStats == nil {
		t.Errorf("expected performance stats to be populated")
	}

	if result.PerformanceStats.GenericTypeSubstitutions != 1 {
		t.Errorf("expected 1 generic type substitution, got %d", result.PerformanceStats.GenericTypeSubstitutions)
	}

	if result.SubstitutionTime < 0 {
		t.Errorf("expected non-negative substitution time, got %d", result.SubstitutionTime)
	}

	if result.TypesProcessed <= 0 {
		t.Errorf("expected positive types processed count")
	}
}

func TestSubstituteType_Context(t *testing.T) {
	engine := NewTypeSubstitutionEngine(NewTypeBuilder(), zap.NewNop())

	// Test with context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	genericType := NewGenericType("T", nil, 0, "")
	typeParam := NewTypeParam("T", nil, 0)
	typeArg := StringType

	result, err := engine.SubstituteTypeWithContext(ctx, genericType, []TypeParam{*typeParam}, []Type{typeArg})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Errorf("expected non-nil result")
	}
}

func TestSubstituteType_NoSubstitutionNeeded(t *testing.T) {
	engine := NewTypeSubstitutionEngine(NewTypeBuilder(), zap.NewNop())

	// Test substitution where no generic types are present
	sliceType := NewSliceType(StringType, "")

	result, err := engine.SubstituteType(sliceType, []TypeParam{}, []Type{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should return the original type unchanged
	if result.SubstitutedType != sliceType {
		t.Errorf("expected substituted type to be same as original")
	}

	if len(result.TypeMapping) != 0 {
		t.Errorf("expected empty type mapping")
	}
}

func TestClearCache(t *testing.T) {
	engine := NewTypeSubstitutionEngine(NewTypeBuilder(), zap.NewNop())

	// Add something to cache
	genericType := NewGenericType("T", nil, 0, "")
	typeParam := NewTypeParam("T", nil, 0)
	typeArg := StringType

	_, err := engine.SubstituteType(genericType, []TypeParam{*typeParam}, []Type{typeArg})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify cache has content
	if engine.GetCacheSize() == 0 {
		t.Errorf("expected cache to have content")
	}

	// Clear cache
	engine.ClearCache()

	// Verify cache is empty
	if engine.GetCacheSize() != 0 {
		t.Errorf("expected cache to be empty after clear")
	}
}

func TestGenerateCacheKey(t *testing.T) {
	engine := NewTypeSubstitutionEngine(NewTypeBuilder(), zap.NewNop())

	genericType := NewGenericType("T", nil, 0, "")
	typeMapping := map[string]Type{
		"T": StringType,
		"U": IntType,
	}

	key := engine.generateCacheKey(genericType, typeMapping)

	// Verify key is not empty
	if key == "" {
		t.Errorf("expected non-empty cache key")
	}

	// Verify key is deterministic
	key2 := engine.generateCacheKey(genericType, typeMapping)
	if key != key2 {
		t.Errorf("expected deterministic cache key generation")
	}

	// Verify key changes with different input
	differentMapping := map[string]Type{
		"T": IntType,
		"U": StringType,
	}
	key3 := engine.generateCacheKey(genericType, differentMapping)
	if key == key3 {
		t.Errorf("expected different cache key for different mapping")
	}
}

func TestSubstitutionStats(t *testing.T) {
	engine := NewTypeSubstitutionEngine(NewTypeBuilder(), zap.NewNop())

	// Perform various substitutions to test stats
	tests := []struct {
		name         string
		createType   func() Type
		expectedStat string
	}{
		{
			name: "basic type",
			createType: func() Type {
				return StringType
			},
			expectedStat: "basic",
		},
		{
			name: "generic type",
			createType: func() Type {
				return NewGenericType("T", nil, 0, "")
			},
			expectedStat: "generic",
		},
		{
			name: "slice type",
			createType: func() Type {
				return NewSliceType(NewGenericType("T", nil, 0, ""), "")
			},
			expectedStat: "slice",
		},
		{
			name: "pointer type",
			createType: func() Type {
				return NewPointerType(NewGenericType("T", nil, 0, ""), "")
			},
			expectedStat: "pointer",
		},
	}

	typeParam := NewTypeParam("T", nil, 0)
	typeArg := StringType

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear stats before test
			engine.substitutionStats = &SubstitutionStats{}

			typ := tt.createType()
			_, err := engine.SubstituteType(typ, []TypeParam{*typeParam}, []Type{typeArg})
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			stats := engine.GetSubstitutionStats()

			// Verify that at least one substitution was recorded
			totalSubstitutions := stats.BasicTypeSubstitutions +
				stats.GenericTypeSubstitutions +
				stats.SliceTypeSubstitutions +
				stats.PointerTypeSubstitutions +
				stats.StructTypeSubstitutions

			if totalSubstitutions == 0 {
				t.Errorf("expected at least one substitution to be recorded")
			}
		})
	}
}

// Benchmark tests for performance validation

func BenchmarkSubstituteType_Simple(b *testing.B) {
	engine := NewTypeSubstitutionEngine(NewTypeBuilder(), zap.NewNop())
	genericType := NewGenericType("T", nil, 0, "")
	typeParam := NewTypeParam("T", nil, 0)
	typeArg := StringType

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.SubstituteType(genericType, []TypeParam{*typeParam}, []Type{typeArg})
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkSubstituteType_Complex(b *testing.B) {
	engine := NewTypeSubstitutionEngine(NewTypeBuilder(), zap.NewNop())

	// Create complex nested type: []*map[T][]U
	genericTypeT := NewGenericType("T", nil, 0, "")
	genericTypeU := NewGenericType("U", nil, 1, "")
	sliceU := NewSliceType(genericTypeU, "")
	mapType := NewMapType(genericTypeT, sliceU)
	pointerType := NewPointerType(mapType, "")
	sliceType := NewSliceType(pointerType, "")

	typeParamT := NewTypeParam("T", nil, 0)
	typeParamU := NewTypeParam("U", nil, 1)
	typeArgT := StringType
	typeArgU := IntType

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.SubstituteType(sliceType,
			[]TypeParam{*typeParamT, *typeParamU},
			[]Type{typeArgT, typeArgU})
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkSubstituteType_WithCaching(b *testing.B) {
	engine := NewTypeSubstitutionEngine(NewTypeBuilder(), zap.NewNop())
	genericType := NewGenericType("T", nil, 0, "")
	typeParam := NewTypeParam("T", nil, 0)
	typeArg := StringType

	// Warm up cache
	_, _ = engine.SubstituteType(genericType, []TypeParam{*typeParam}, []Type{typeArg})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.SubstituteType(genericType, []TypeParam{*typeParam}, []Type{typeArg})
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}
