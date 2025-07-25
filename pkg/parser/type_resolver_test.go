package parser

import (
	"context"
	"go/types"
	"testing"

	"github.com/reedom/convergen/v8/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestTypeResolver_ResolveBasicTypes(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cache := NewTypeCache(100)
	resolver := NewTypeResolver(cache, logger)

	tests := []struct {
		name         string
		goType       types.Type
		expectedKind domain.TypeKind
		expectedName string
	}{
		{
			name:         "string type",
			goType:       types.Typ[types.String],
			expectedKind: domain.TypeKindString,
			expectedName: "string",
		},
		{
			name:         "int type",
			goType:       types.Typ[types.Int],
			expectedKind: domain.TypeKindInt,
			expectedName: "int",
		},
		{
			name:         "bool type",
			goType:       types.Typ[types.Bool],
			expectedKind: domain.TypeKindBool,
			expectedName: "bool",
		},
		{
			name:         "float64 type",
			goType:       types.Typ[types.Float64],
			expectedKind: domain.TypeKindFloat,
			expectedName: "float64",
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domainType, err := resolver.ResolveType(ctx, tt.goType)
			require.NoError(t, err)
			
			assert.Equal(t, tt.expectedKind, domainType.Kind())
			assert.Equal(t, tt.expectedName, domainType.Name())
			assert.False(t, domainType.Generic())
		})
	}
}

func TestTypeResolver_ResolvePointerType(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cache := NewTypeCache(100)
	resolver := NewTypeResolver(cache, logger)

	// Create a pointer to string
	stringType := types.Typ[types.String]
	pointerType := types.NewPointer(stringType)

	ctx := context.Background()
	domainType, err := resolver.ResolveType(ctx, pointerType)
	require.NoError(t, err)

	assert.Equal(t, domain.TypeKindPointer, domainType.Kind())
	
	// Check that it's a pointer type with correct element
	pointerDomainType, ok := domainType.(*domain.PointerType)
	require.True(t, ok)
	
	assert.Equal(t, domain.TypeKindString, pointerDomainType.ElementType.Kind())
	assert.Equal(t, "string", pointerDomainType.ElementType.Name())
}

func TestTypeResolver_ResolveSliceType(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cache := NewTypeCache(100)
	resolver := NewTypeResolver(cache, logger)

	// Create a slice of strings
	stringType := types.Typ[types.String]
	sliceType := types.NewSlice(stringType)

	ctx := context.Background()
	domainType, err := resolver.ResolveType(ctx, sliceType)
	require.NoError(t, err)

	assert.Equal(t, domain.TypeKindSlice, domainType.Kind())
	
	// Check that it's a slice type with correct element
	sliceDomainType, ok := domainType.(*domain.SliceType)
	require.True(t, ok)
	
	assert.Equal(t, domain.TypeKindString, sliceDomainType.ElementType.Kind())
	assert.Equal(t, "string", sliceDomainType.ElementType.Name())
}

func TestTypeResolver_ResolveArrayType(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cache := NewTypeCache(100)
	resolver := NewTypeResolver(cache, logger)

	// Create an array of 10 strings
	stringType := types.Typ[types.String]
	arrayType := types.NewArray(stringType, 10)

	ctx := context.Background()
	domainType, err := resolver.ResolveType(ctx, arrayType)
	require.NoError(t, err)

	assert.Equal(t, domain.TypeKindArray, domainType.Kind())
	
	// Check that it's an array type with correct element and length
	arrayDomainType, ok := domainType.(*domain.ArrayType)
	require.True(t, ok)
	
	assert.Equal(t, domain.TypeKindString, arrayDomainType.ElementType.Kind())
	assert.Equal(t, "string", arrayDomainType.ElementType.Name())
	assert.Equal(t, 10, arrayDomainType.Length)
}

func TestTypeResolver_ResolveMapType(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cache := NewTypeCache(100)
	resolver := NewTypeResolver(cache, logger)

	// Create a map[string]int
	stringType := types.Typ[types.String]
	intType := types.Typ[types.Int]
	mapType := types.NewMap(stringType, intType)

	ctx := context.Background()
	domainType, err := resolver.ResolveType(ctx, mapType)
	require.NoError(t, err)

	assert.Equal(t, domain.TypeKindMap, domainType.Kind())
	
	// Check that it's a map type with correct key and value
	mapDomainType, ok := domainType.(*domain.MapType)
	require.True(t, ok)
	
	assert.Equal(t, domain.TypeKindString, mapDomainType.KeyType.Kind())
	assert.Equal(t, "string", mapDomainType.KeyType.Name())
	assert.Equal(t, domain.TypeKindInt, mapDomainType.ValueType.Kind())
	assert.Equal(t, "int", mapDomainType.ValueType.Name())
}

func TestTypeResolver_ResolveStructType(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cache := NewTypeCache(100)
	resolver := NewTypeResolver(cache, logger)

	// Create a struct type
	stringType := types.Typ[types.String]
	intType := types.Typ[types.Int]
	
	fields := []*types.Var{
		types.NewField(0, nil, "Name", stringType, false),
		types.NewField(0, nil, "Age", intType, false),
	}
	
	structType := types.NewStruct(fields, nil)

	ctx := context.Background()
	domainType, err := resolver.ResolveType(ctx, structType)
	require.NoError(t, err)

	assert.Equal(t, domain.TypeKindStruct, domainType.Kind())
	
	// Check that it's a struct type with correct fields
	structDomainType, ok := domainType.(*domain.StructType)
	require.True(t, ok)
	
	assert.Len(t, structDomainType.Fields, 2)
	
	nameField := structDomainType.Fields[0]
	assert.Equal(t, "Name", nameField.Name)
	assert.Equal(t, domain.TypeKindString, nameField.Type.Kind())
	
	ageField := structDomainType.Fields[1]
	assert.Equal(t, "Age", ageField.Name)
	assert.Equal(t, domain.TypeKindInt, ageField.Type.Kind())
}

func TestTypeResolver_ResolveChanType(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cache := NewTypeCache(100)
	resolver := NewTypeResolver(cache, logger)

	// Create different channel types
	stringType := types.Typ[types.String]
	
	tests := []struct {
		name              string
		chanType          *types.Chan
		expectedDirection domain.ChannelDirection
	}{
		{
			name:              "bidirectional channel",
			chanType:          types.NewChan(types.SendRecv, stringType),
			expectedDirection: domain.ChannelBidirectional,
		},
		{
			name:              "send-only channel",
			chanType:          types.NewChan(types.SendOnly, stringType),
			expectedDirection: domain.ChannelSendOnly,
		},
		{
			name:              "receive-only channel",
			chanType:          types.NewChan(types.RecvOnly, stringType),
			expectedDirection: domain.ChannelReceiveOnly,
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domainType, err := resolver.ResolveType(ctx, tt.chanType)
			require.NoError(t, err)

			assert.Equal(t, domain.TypeKindChannel, domainType.Kind())
			
			chanDomainType, ok := domainType.(*domain.ChannelType)
			require.True(t, ok)
			
			assert.Equal(t, tt.expectedDirection, chanDomainType.Direction)
			assert.Equal(t, domain.TypeKindString, chanDomainType.ElementType.Kind())
		})
	}
}

func TestTypeResolver_ResolveSignatureType(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cache := NewTypeCache(100)
	resolver := NewTypeResolver(cache, logger)

	// Create a function signature: func(string, int) (bool, error)
	stringType := types.Typ[types.String]
	intType := types.Typ[types.Int]
	boolType := types.Typ[types.Bool]
	
	// Create error type (simplified)
	errorType := types.Universe.Lookup("error").Type()
	
	params := types.NewTuple(
		types.NewVar(0, nil, "s", stringType),
		types.NewVar(0, nil, "i", intType),
	)
	
	results := types.NewTuple(
		types.NewVar(0, nil, "", boolType),
		types.NewVar(0, nil, "", errorType),
	)
	
	signature := types.NewSignature(nil, params, results, false)

	ctx := context.Background()
	domainType, err := resolver.ResolveType(ctx, signature)
	require.NoError(t, err)

	assert.Equal(t, domain.TypeKindFunction, domainType.Kind())
	
	funcDomainType, ok := domainType.(*domain.FunctionType)
	require.True(t, ok)
	
	assert.Len(t, funcDomainType.Parameters, 2)
	assert.Len(t, funcDomainType.Returns, 2)
	assert.False(t, funcDomainType.Variadic)
	
	// Check parameter types
	assert.Equal(t, domain.TypeKindString, funcDomainType.Parameters[0].Kind())
	assert.Equal(t, domain.TypeKindInt, funcDomainType.Parameters[1].Kind())
	
	// Check return types
	assert.Equal(t, domain.TypeKindBool, funcDomainType.Returns[0].Kind())
	// Note: error type checking might need special handling
}

func TestTypeResolverPool(t *testing.T) {
	logger := zaptest.NewLogger(t)
	pool := NewTypeResolverPool(3, logger)
	defer pool.Close()

	// Test getting resolvers in round-robin fashion
	resolver1 := pool.Get()
	assert.NotNil(t, resolver1)
	
	resolver2 := pool.Get()
	assert.NotNil(t, resolver2)
	
	resolver3 := pool.Get()
	assert.NotNil(t, resolver3)
	
	// Fourth request should wrap around to first resolver
	resolver4 := pool.Get()
	assert.Equal(t, resolver1, resolver4)
}

func TestTypeResolverPool_ClosedPool(t *testing.T) {
	logger := zaptest.NewLogger(t)
	pool := NewTypeResolverPool(2, logger)
	
	// Close the pool
	err := pool.Close()
	assert.NoError(t, err)
	
	// Getting from closed pool should return nil
	resolver := pool.Get()
	assert.Nil(t, resolver)
}

func BenchmarkTypeResolver_ResolveBasicType(b *testing.B) {
	logger := zaptest.NewLogger(b)
	cache := NewTypeCache(1000)
	resolver := NewTypeResolver(cache, logger)
	
	stringType := types.Typ[types.String]
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := resolver.ResolveType(ctx, stringType)
		require.NoError(b, err)
	}
}

func BenchmarkTypeResolver_ResolveComplexType(b *testing.B) {
	logger := zaptest.NewLogger(b)
	cache := NewTypeCache(1000)
	resolver := NewTypeResolver(cache, logger)
	
	// Create a complex nested type: map[string]*[]int
	intType := types.Typ[types.Int]
	sliceType := types.NewSlice(intType)
	pointerType := types.NewPointer(sliceType)
	stringType := types.Typ[types.String]
	mapType := types.NewMap(stringType, pointerType)
	
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := resolver.ResolveType(ctx, mapType)
		require.NoError(b, err)
	}
}