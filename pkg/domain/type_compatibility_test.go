package domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestTypeCompatibilityChecker_CheckAssignability(t *testing.T) {
	logger := zaptest.NewLogger(t)
	checker := NewTypeCompatibilityChecker(logger)

	tests := []struct {
		name          string
		sourceType    Type
		targetType    Type
		expectedLevel CompatibilityLevel
		expectedError bool
	}{
		{
			name:          "identical types",
			sourceType:    NewBasicType("string", 0),
			targetType:    NewBasicType("string", 0),
			expectedLevel: CompatibilityIdentical,
		},
		{
			name:          "nil source type",
			sourceType:    nil,
			targetType:    NewBasicType("string", 0),
			expectedLevel: CompatibilityNone,
			expectedError: true,
		},
		{
			name:          "nil target type",
			sourceType:    NewBasicType("string", 0),
			targetType:    nil,
			expectedLevel: CompatibilityNone,
			expectedError: true,
		},
		{
			name:          "different basic types",
			sourceType:    NewBasicType("int", 0),
			targetType:    NewBasicType("string", 0),
			expectedLevel: CompatibilityNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := checker.CheckAssignability(tt.sourceType, tt.targetType)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedLevel, result.Level)
		})
	}
}

func TestGenericCompatibilityMatrix_AnalyzeCompatibility(t *testing.T) {
	logger := zaptest.NewLogger(t)
	matrix := NewGenericCompatibilityMatrix(logger)

	// Create some test generic types
	anyParam := NewAnyTypeParam("T", 0)
	genericType1 := &mockType{
		name:       "TestGeneric1",
		kind:       KindGeneric,
		generic:    true,
		typeParams: []TypeParam{*anyParam},
	}

	basicType := NewBasicType("string", 0)

	tests := []struct {
		name          string
		sourceType    Type
		targetType    Type
		shouldSucceed bool
	}{
		{
			name:          "generic to basic type",
			sourceType:    genericType1,
			targetType:    basicType,
			shouldSucceed: true,
		},
		{
			name:          "basic to basic type",
			sourceType:    basicType,
			targetType:    basicType,
			shouldSucceed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := matrix.AnalyzeCompatibility(context.Background(), tt.sourceType, tt.targetType)

			if tt.shouldSucceed {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotNil(t, result.MatrixAnalysis)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestFieldMappingValidator_ValidateFieldMappings(t *testing.T) {
	logger := zaptest.NewLogger(t)
	validator := NewFieldMappingValidator(logger)

	// Set up compatibility checker
	compatibilityChecker := NewTypeCompatibilityChecker(logger)
	validator.SetCompatibilityChecker(compatibilityChecker)

	stringType := NewBasicType("string", 0)
	intType := NewBasicType("int", 0)

	mappings := []ValidatorFieldMapping{
		{
			SourceFieldName: "Name",
			TargetFieldName: "Name",
			SourceType:      stringType,
			TargetType:      stringType,
		},
		{
			SourceFieldName: "ID",
			TargetFieldName: "ID",
			SourceType:      intType,
			TargetType:      stringType, // Incompatible conversion
		},
	}

	result, err := validator.ValidateFieldMappings(context.Background(), mappings)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Check that we have both compatible and incompatible fields
	assert.Equal(t, 2, result.ValidationSummary.TotalFields)
	assert.Equal(t, 1, result.ValidationSummary.CompatibleFields)
	assert.Equal(t, 1, result.ValidationSummary.IncompatibleFields)
	assert.False(t, result.Valid) // Overall validation should fail due to incompatible field
}

func TestGenericErrorEnhancer_EnhanceError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	enhancer := NewGenericErrorEnhancer(logger)

	originalError := ErrTypesIncompatible
	sourceType := NewBasicType("int", 0)
	targetType := NewBasicType("string", 0)

	enhanced := enhancer.EnhanceError(originalError, sourceType, targetType, map[string]string{
		"context": "field_mapping",
	})

	assert.NotNil(t, enhanced)
	assert.Equal(t, originalError, enhanced.OriginalError)
	assert.NotEmpty(t, enhanced.ErrorCode)
	assert.NotEmpty(t, enhanced.EnhancedMessage)
	assert.Equal(t, "type_compatibility", enhanced.ErrorCategory)
	assert.NotNil(t, enhanced.Context)
	assert.Equal(t, "field_mapping", enhanced.Context["context"])
}

// mockType is a helper for testing
type mockType struct {
	name         string
	kind         TypeKind
	generic      bool
	typeParams   []TypeParam
	assignableTo bool
	implements   bool
	comparable   bool
	pkg          string
	importPath   string
	underlying   Type
}

func (m *mockType) Name() string            { return m.name }
func (m *mockType) Kind() TypeKind          { return m.kind }
func (m *mockType) String() string          { return m.name }
func (m *mockType) Generic() bool           { return m.generic }
func (m *mockType) TypeParams() []TypeParam { return m.typeParams }
func (m *mockType) Underlying() Type        { return m.underlying }
func (m *mockType) AssignableTo(Type) bool  { return m.assignableTo }
func (m *mockType) Implements(Type) bool    { return m.implements }
func (m *mockType) Comparable() bool        { return m.comparable }
func (m *mockType) Package() string         { return m.pkg }
func (m *mockType) ImportPath() string      { return m.importPath }
