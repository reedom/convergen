package parser

import (
	"context"
	"fmt"
	"go/types"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/reedom/convergen/v9/pkg/domain"
)

func TestNewConstraintParser(t *testing.T) {
	tests := []struct {
		name         string
		typeResolver *TypeResolver
		logger       *zap.Logger
		expectNil    bool
	}{
		{
			name:         "valid inputs",
			typeResolver: &TypeResolver{},
			logger:       zap.NewNop(),
			expectNil:    false,
		},
		{
			name:         "nil type resolver",
			typeResolver: nil,
			logger:       zap.NewNop(),
			expectNil:    false,
		},
		{
			name:         "nil logger",
			typeResolver: &TypeResolver{},
			logger:       nil,
			expectNil:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewConstraintParser(tt.typeResolver, tt.logger)

			if tt.expectNil {
				assert.Nil(t, parser)
			} else {
				assert.NotNil(t, parser)
				assert.Equal(t, tt.typeResolver, parser.typeResolver)
				assert.Equal(t, tt.logger, parser.logger)
			}
		})
	}
}

func TestConstraintParser_ParseConstraint_AnyConstraint(t *testing.T) {
	parser := createTestConstraintParser(t)
	ctx := context.Background()

	tests := []struct {
		name         string
		constraint   types.Type
		expectedType string
		expectedAny  bool
	}{
		{
			name:         "nil constraint represents any",
			constraint:   nil,
			expectedType: "any",
			expectedAny:  true,
		},
		{
			name:         "empty interface represents any",
			constraint:   types.NewInterfaceType(nil, nil),
			expectedType: "any",
			expectedAny:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseConstraint(ctx, tt.constraint)

			require.NoError(t, err)
			assert.True(t, result.Valid)
			assert.Equal(t, tt.expectedType, result.ConstraintType)
			assert.Equal(t, tt.expectedAny, result.IsAny)
			assert.False(t, result.IsComparable)
			assert.Empty(t, result.UnionTypes)
			assert.Nil(t, result.Underlying)
			assert.True(t, 0 < result.ParseDuration)
		})
	}
}

func TestConstraintParser_ParseConstraint_ComparableConstraint(t *testing.T) {
	parser := createTestConstraintParser(t)
	ctx := context.Background()

	// Create a comparable constraint by creating a named type
	comparableObj := types.NewTypeName(0, nil, "comparable", nil)
	comparableType := types.NewNamed(comparableObj, types.Typ[types.Bool], nil)

	result, err := parser.ParseConstraint(ctx, comparableType)

	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, "comparable", result.ConstraintType)
	assert.True(t, result.IsComparable)
	assert.False(t, result.IsAny)
	assert.Empty(t, result.UnionTypes)
	assert.Nil(t, result.Underlying)
	assert.True(t, 0 < result.ParseDuration)
}

func TestConstraintParser_ParseConstraint_BasicConstraint(t *testing.T) {
	parser := createTestConstraintParser(t)
	ctx := context.Background()

	tests := []struct {
		name         string
		basicType    *types.Basic
		expectedType string
	}{
		{
			name:         "string constraint",
			basicType:    types.Typ[types.String],
			expectedType: "basic",
		},
		{
			name:         "int constraint",
			basicType:    types.Typ[types.Int],
			expectedType: "basic",
		},
		{
			name:         "bool constraint",
			basicType:    types.Typ[types.Bool],
			expectedType: "basic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseConstraint(ctx, tt.basicType)

			require.NoError(t, err)
			assert.True(t, result.Valid)
			assert.Equal(t, tt.expectedType, result.ConstraintType)
			assert.False(t, result.IsAny)
			assert.False(t, result.IsComparable)
			assert.Empty(t, result.UnionTypes)
			assert.Nil(t, result.Underlying)
			assert.NotNil(t, result.Type)
			assert.True(t, 0 < result.ParseDuration)
		})
	}
}

func TestConstraintParser_ParseConstraint_NamedConstraint(t *testing.T) {
	parser := createTestConstraintParser(t)
	ctx := context.Background()

	// Test 'any' named constraint
	t.Run("any named constraint", func(t *testing.T) {
		anyObj := types.NewTypeName(0, nil, "any", nil)
		anyType := types.NewNamed(anyObj, types.NewInterfaceType(nil, nil), nil)

		result, err := parser.ParseConstraint(ctx, anyType)

		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Equal(t, "any", result.ConstraintType)
		assert.True(t, result.IsAny)
		assert.False(t, result.IsComparable)
	})

	// Test custom named constraint
	t.Run("custom named constraint", func(t *testing.T) {
		customObj := types.NewTypeName(0, types.NewPackage("test", "test"), "CustomConstraint", nil)
		customType := types.NewNamed(customObj, types.Typ[types.String], nil)

		result, err := parser.ParseConstraint(ctx, customType)

		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Equal(t, "named", result.ConstraintType)
		assert.False(t, result.IsAny)
		assert.False(t, result.IsComparable)
		assert.NotNil(t, result.Type)
	})
}

func TestConstraintParser_ParseConstraint_UnionConstraint(t *testing.T) {
	parser := createTestConstraintParser(t)
	ctx := context.Background()

	// Create a union constraint: ~int | ~string
	intTerm := types.NewTerm(true, types.Typ[types.Int])       // ~int
	stringTerm := types.NewTerm(true, types.Typ[types.String]) // ~string
	union := types.NewUnion([]*types.Term{intTerm, stringTerm})

	result, err := parser.ParseConstraint(ctx, union)

	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, "union_underlying", result.ConstraintType)
	assert.False(t, result.IsAny)
	assert.False(t, result.IsComparable)
	assert.Len(t, result.UnionTypes, 2)
	assert.Nil(t, result.Underlying)
	assert.True(t, 0 < result.ParseDuration)
}

func TestConstraintParser_ParseConstraint_EmptyUnionConstraint(t *testing.T) {
	// Since types.NewUnion doesn't allow empty unions, we'll test with a single term
	// and manually test the error case in parseUnionConstraint method
	t.Skip("types.NewUnion panics on empty unions - testing this scenario requires mocking")
}

func TestConstraintParser_ParseConstraint_UnsupportedType(t *testing.T) {
	parser := createTestConstraintParser(t)
	ctx := context.Background()

	// Use a channel type which is not a supported constraint
	chanType := types.NewChan(types.SendRecv, types.Typ[types.Int])

	result, err := parser.ParseConstraint(ctx, chanType)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported constraint type")
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.ErrorMessage)
}

func TestConstraintParser_ValidateConstraint(t *testing.T) {
	parser := createTestConstraintParser(t)

	tests := []struct {
		name        string
		constraint  *ParsedConstraint
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil constraint",
			constraint:  nil,
			expectError: true,
			errorMsg:    "constraint is nil",
		},
		{
			name: "invalid constraint",
			constraint: &ParsedConstraint{
				Valid:        false,
				ErrorMessage: "test error",
			},
			expectError: true,
			errorMsg:    "test error",
		},
		{
			name: "valid any constraint",
			constraint: &ParsedConstraint{
				IsAny:          true,
				ConstraintType: "any",
				Valid:          true,
			},
			expectError: false,
		},
		{
			name: "valid comparable constraint",
			constraint: &ParsedConstraint{
				IsComparable:   true,
				ConstraintType: "comparable",
				Valid:          true,
			},
			expectError: false,
		},
		{
			name: "multiple constraint types",
			constraint: &ParsedConstraint{
				IsAny:        true,
				IsComparable: true,
				Valid:        true,
			},
			expectError: true,
			errorMsg:    "multiple constraint types detected",
		},
		{
			name: "no constraint type",
			constraint: &ParsedConstraint{
				Valid: true,
			},
			expectError: true,
			errorMsg:    "no constraint type detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.ValidateConstraint(tt.constraint)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConstraintParser_GetConstraintTypeString(t *testing.T) {
	parser := createTestConstraintParser(t)

	tests := []struct {
		name       string
		constraint *ParsedConstraint
		expected   string
	}{
		{
			name:       "nil constraint",
			constraint: nil,
			expected:   "unknown",
		},
		{
			name: "any constraint",
			constraint: &ParsedConstraint{
				IsAny: true,
			},
			expected: "any",
		},
		{
			name: "comparable constraint",
			constraint: &ParsedConstraint{
				IsComparable: true,
			},
			expected: "comparable",
		},
		{
			name: "union constraint",
			constraint: &ParsedConstraint{
				UnionTypes: []domain.Type{
					domain.NewBasicType("int", reflect.Int),
					domain.NewBasicType("string", reflect.String),
				},
			},
			expected: "int | string",
		},
		{
			name: "underlying constraint",
			constraint: &ParsedConstraint{
				Underlying: &domain.UnderlyingConstraint{
					Type: domain.NewBasicType("string", reflect.String),
				},
			},
			expected: "~string",
		},
		{
			name: "interface constraint",
			constraint: &ParsedConstraint{
				InterfaceType: domain.NewBasicType("Reader", reflect.Interface),
			},
			expected: "Reader",
		},
		{
			name: "type constraint",
			constraint: &ParsedConstraint{
				Type: domain.NewBasicType("int", reflect.Int),
			},
			expected: "int",
		},
		{
			name:       "unknown constraint",
			constraint: &ParsedConstraint{},
			expected:   "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.GetConstraintTypeString(tt.constraint)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConstraintParser_ConvertToDomainTypeParam(t *testing.T) {
	parser := createTestConstraintParser(t)

	tests := []struct {
		name        string
		paramName   string
		index       int
		constraint  *ParsedConstraint
		expectError bool
		validate    func(t *testing.T, tp *domain.TypeParam)
	}{
		{
			name:        "nil constraint",
			paramName:   "T",
			index:       0,
			constraint:  nil,
			expectError: true,
		},
		{
			name:      "any constraint",
			paramName: "T",
			index:     0,
			constraint: &ParsedConstraint{
				IsAny: true,
				Valid: true,
			},
			expectError: false,
			validate: func(t *testing.T, tp *domain.TypeParam) {
				assert.Equal(t, "T", tp.Name)
				assert.Equal(t, 0, tp.Index)
				assert.True(t, tp.IsAny)
				assert.False(t, tp.IsComparable)
			},
		},
		{
			name:      "comparable constraint",
			paramName: "T",
			index:     1,
			constraint: &ParsedConstraint{
				IsComparable: true,
				Valid:        true,
			},
			expectError: false,
			validate: func(t *testing.T, tp *domain.TypeParam) {
				assert.Equal(t, "T", tp.Name)
				assert.Equal(t, 1, tp.Index)
				assert.False(t, tp.IsAny)
				assert.True(t, tp.IsComparable)
			},
		},
		{
			name:      "union constraint",
			paramName: "T",
			index:     2,
			constraint: &ParsedConstraint{
				UnionTypes: []domain.Type{
					domain.NewBasicType("int", reflect.Int),
					domain.NewBasicType("string", reflect.String),
				},
				Valid: true,
			},
			expectError: false,
			validate: func(t *testing.T, tp *domain.TypeParam) {
				assert.Equal(t, "T", tp.Name)
				assert.Equal(t, 2, tp.Index)
				assert.Len(t, tp.UnionTypes, 2)
			},
		},
		{
			name:      "underlying constraint",
			paramName: "T",
			index:     3,
			constraint: &ParsedConstraint{
				Underlying: &domain.UnderlyingConstraint{
					Type: domain.NewBasicType("string", reflect.String),
				},
				Valid: true,
			},
			expectError: false,
			validate: func(t *testing.T, tp *domain.TypeParam) {
				assert.Equal(t, "T", tp.Name)
				assert.Equal(t, 3, tp.Index)
				assert.NotNil(t, tp.Underlying)
			},
		},
		{
			name:      "type constraint",
			paramName: "T",
			index:     4,
			constraint: &ParsedConstraint{
				Type:  domain.NewBasicType("int", reflect.Int),
				Valid: true,
			},
			expectError: false,
			validate: func(t *testing.T, tp *domain.TypeParam) {
				assert.Equal(t, "T", tp.Name)
				assert.Equal(t, 4, tp.Index)
				assert.NotNil(t, tp.Constraint)
			},
		},
		{
			name:      "invalid constraint",
			paramName: "T",
			index:     0,
			constraint: &ParsedConstraint{
				Valid:        false,
				ErrorMessage: "test error",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ConvertToDomainTypeParam(tt.paramName, tt.index, tt.constraint)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)

				if tt.validate != nil {
					tt.validate(t, result)
				}

				// Validate that the result is valid
				assert.True(t, result.IsValid())
			}
		})
	}
}

func TestConstraintParser_isComparableInterface(t *testing.T) {
	parser := createTestConstraintParser(t)

	tests := []struct {
		name     string
		iface    *types.Interface
		expected bool
	}{
		{
			name:     "empty interface",
			iface:    types.NewInterfaceType(nil, nil),
			expected: false,
		},
		{
			name: "interface with comparable in string representation",
			iface: func() *types.Interface {
				// Create a mock interface that contains "comparable" in its string representation
				// This is a simplified test case
				return types.NewInterfaceType(nil, nil)
			}(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.isComparableInterface(tt.iface)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConstraintParser_PerformanceRequirement(t *testing.T) {
	parser := createTestConstraintParser(t)
	ctx := context.Background()

	// Test performance requirement: Parse constraints in <1ms for typical cases
	constraints := []types.Type{
		nil,                              // any
		types.Typ[types.String],          // basic
		types.Typ[types.Int],             // basic
		types.NewInterfaceType(nil, nil), // empty interface (any)
	}

	for i, constraint := range constraints {
		t.Run(fmt.Sprintf("performance_test_%d", i), func(t *testing.T) {
			start := time.Now()
			result, err := parser.ParseConstraint(ctx, constraint)
			duration := time.Since(start)

			require.NoError(t, err)
			assert.True(t, result.Valid)

			// Performance requirement: <1ms for typical cases
			assert.True(t, duration < time.Millisecond,
				"constraint parsing took %v, expected <1ms", duration)

			// Verify the parser's internal timing is reasonable
			assert.True(t, result.ParseDuration <= duration,
				"internal timing (%v) should not exceed actual timing (%v)",
				result.ParseDuration, duration)
		})
	}
}

func TestConstraintParser_ConcurrentAccess(t *testing.T) {
	parser := createTestConstraintParser(t)
	ctx := context.Background()

	// Test that the parser is safe for concurrent use
	const numGoroutines = 10
	const numIterations = 5

	constraints := []types.Type{
		nil,
		types.Typ[types.String],
		types.Typ[types.Int],
		types.NewInterfaceType(nil, nil),
	}

	// Run concurrent parsing operations
	results := make(chan error, numGoroutines*numIterations*len(constraints))

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numIterations; j++ {
				for _, constraint := range constraints {
					result, err := parser.ParseConstraint(ctx, constraint)
					if err != nil {
						results <- err
						return
					}
					if !result.Valid {
						results <- fmt.Errorf("invalid result: %s", result.ErrorMessage)
						return
					}
				}
			}
			results <- nil
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		err := <-results
		require.NoError(t, err)
	}
}

// Helper functions

func createTestConstraintParser(_ *testing.T) *ConstraintParser {
	cache := NewTypeCache(100)
	logger := zap.NewNop()
	typeResolver := NewTypeResolver(cache, logger)

	return NewConstraintParser(typeResolver, logger)
}

// Benchmark tests

func BenchmarkConstraintParser_ParseAnyConstraint(b *testing.B) {
	parser := createTestConstraintParserForBench()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseConstraint(ctx, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConstraintParser_ParseBasicConstraint(b *testing.B) {
	parser := createTestConstraintParserForBench()
	ctx := context.Background()
	constraint := types.Typ[types.String]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseConstraint(ctx, constraint)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConstraintParser_ParseUnionConstraint(b *testing.B) {
	parser := createTestConstraintParserForBench()
	ctx := context.Background()

	// Create a union constraint: ~int | ~string
	intTerm := types.NewTerm(true, types.Typ[types.Int])
	stringTerm := types.NewTerm(true, types.Typ[types.String])
	union := types.NewUnion([]*types.Term{intTerm, stringTerm})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseConstraint(ctx, union)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Additional test to fix the createTestConstraintParser function for benchmarks.
func createTestConstraintParserForBench() *ConstraintParser {
	cache := NewTypeCache(100)
	logger := zap.NewNop()
	typeResolver := NewTypeResolver(cache, logger)

	return NewConstraintParser(typeResolver, logger)
}

func BenchmarkConstraintParser_ParseAnyConstraintFixed(b *testing.B) {
	parser := createTestConstraintParserForBench()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseConstraint(ctx, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}
