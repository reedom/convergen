package domain

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestNewTypeInstantiator(t *testing.T) {
	t.Run("with all dependencies", func(t *testing.T) {
		typeBuilder := NewTypeBuilder()
		logger := zaptest.NewLogger(t)

		instantiator := NewTypeInstantiator(typeBuilder, logger)

		assert.NotNil(t, instantiator)
		assert.NotNil(t, instantiator.cache)
		assert.Equal(t, 10, instantiator.maxRecursionDepth)
		assert.Equal(t, int64(0), instantiator.cacheHits)
		assert.Equal(t, int64(0), instantiator.cacheMisses)
	})

	t.Run("with nil dependencies", func(t *testing.T) {
		instantiator := NewTypeInstantiator(nil, nil)

		assert.NotNil(t, instantiator)
		assert.NotNil(t, instantiator.typeBuilder)
		assert.NotNil(t, instantiator.logger)
		assert.NotNil(t, instantiator.cache)
	})

	t.Run("with custom config", func(t *testing.T) {
		typeBuilder := NewTypeBuilder()
		logger := zaptest.NewLogger(t)
		config := &TypeInstantiatorConfig{
			MaxRecursionDepth:      5,
			EnableCaching:          false,
			EnablePerformanceTrack: false,
			CacheCapacity:          100,
		}

		instantiator := NewTypeInstantiatorWithConfig(typeBuilder, logger, config)

		assert.NotNil(t, instantiator)
		assert.Nil(t, instantiator.cache) // Disabled caching
		assert.Equal(t, 5, instantiator.maxRecursionDepth)
	})
}

func TestNewInstantiatedInterface(t *testing.T) {
	sourceInterface := NewBasicType("TestInterface", reflect.Interface)
	concreteType := NewBasicType("ConcreteType", reflect.Struct)

	t.Run("valid instantiation", func(t *testing.T) {
		typeArgs := map[string]Type{
			"T": StringType,
			"U": IntType,
		}

		result, err := NewInstantiatedInterface(
			sourceInterface,
			typeArgs,
			concreteType,
			"TestInterface[string,int]",
		)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, sourceInterface, result.SourceInterface)
		assert.Equal(t, 2, len(result.TypeArguments))
		assert.Equal(t, StringType, result.TypeArguments["T"])
		assert.Equal(t, IntType, result.TypeArguments["U"])
		assert.Equal(t, concreteType, result.ConcreteType)
		assert.Equal(t, "TestInterface[string,int]", result.TypeSignature)
		assert.False(t, result.CacheHit)
		assert.WithinDuration(t, time.Now(), result.InstantiatedAt, time.Second)
	})

	t.Run("nil source interface", func(t *testing.T) {
		typeArgs := map[string]Type{"T": StringType}

		result, err := NewInstantiatedInterface(
			nil,
			typeArgs,
			concreteType,
			"TestInterface[string]",
		)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrGenericInterfaceNil)
		assert.Nil(t, result)
	})

	t.Run("nil type arguments", func(t *testing.T) {
		result, err := NewInstantiatedInterface(
			sourceInterface,
			nil,
			concreteType,
			"TestInterface",
		)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrTypeArgumentsNil)
		assert.Nil(t, result)
	})

	t.Run("nil concrete type", func(t *testing.T) {
		typeArgs := map[string]Type{"T": StringType}

		result, err := NewInstantiatedInterface(
			sourceInterface,
			typeArgs,
			nil,
			"TestInterface[string]",
		)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidTypeArgument)
		assert.Nil(t, result)
	})

	t.Run("empty type signature", func(t *testing.T) {
		typeArgs := map[string]Type{"T": StringType}

		result, err := NewInstantiatedInterface(
			sourceInterface,
			typeArgs,
			concreteType,
			"",
		)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrCacheKeyGeneration)
		assert.Nil(t, result)
	})

	t.Run("invalid type argument", func(t *testing.T) {
		typeArgs := map[string]Type{
			"T": StringType,
			"":  IntType, // Invalid empty parameter name
		}

		result, err := NewInstantiatedInterface(
			sourceInterface,
			typeArgs,
			concreteType,
			"TestInterface[string,int]",
		)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidTypeArgument)
		assert.Nil(t, result)
	})

	t.Run("nil type argument", func(t *testing.T) {
		typeArgs := map[string]Type{
			"T": StringType,
			"U": nil, // Invalid nil type
		}

		result, err := NewInstantiatedInterface(
			sourceInterface,
			typeArgs,
			concreteType,
			"TestInterface[string,nil]",
		)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidTypeArgument)
		assert.Nil(t, result)
	})
}

func TestNewGenericInterface(t *testing.T) {
	t.Run("valid generic interface", func(t *testing.T) {
		typeParams := []TypeParam{
			*NewTypeParam("T", StringType, 0),
			*NewAnyTypeParam("U", 1),
		}
		methods := []*Method{}

		result, err := NewGenericInterface("TestInterface", typeParams, methods, "testpkg")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "TestInterface", result.Name)
		assert.Equal(t, 2, len(result.TypeParams))
		assert.Equal(t, "testpkg", result.Package)
		assert.Equal(t, "T", result.TypeParams[0].Name)
		assert.Equal(t, "U", result.TypeParams[1].Name)
	})

	t.Run("empty name", func(t *testing.T) {
		typeParams := []TypeParam{*NewAnyTypeParam("T", 0)}

		result, err := NewGenericInterface("", typeParams, nil, "testpkg")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "interface name cannot be empty")
		assert.Nil(t, result)
	})

	t.Run("no type parameters", func(t *testing.T) {
		result, err := NewGenericInterface("TestInterface", []TypeParam{}, nil, "testpkg")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "generic interface must have type parameters")
		assert.Nil(t, result)
	})

	t.Run("invalid type parameter", func(t *testing.T) {
		// Create an invalid type parameter (mutually exclusive constraints)
		invalidParam := &TypeParam{
			Name:         "T",
			Index:        0,
			IsAny:        true,
			IsComparable: true, // Invalid: can't be both any and comparable
		}

		result, err := NewGenericInterface("TestInterface", []TypeParam{*invalidParam}, nil, "testpkg")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid type argument at index 0: T")
		assert.Nil(t, result)
	})
}

func TestInstantiateInterface(t *testing.T) {
	typeBuilder := NewTypeBuilder()
	logger := zaptest.NewLogger(t)
	instantiator := NewTypeInstantiator(typeBuilder, logger)

	t.Run("successful instantiation with any constraint", func(t *testing.T) {
		// Create a generic interface with 'any' constraint
		typeParams := []TypeParam{
			*NewAnyTypeParam("T", 0),
		}
		genericInterface, err := NewGenericInterface("TestInterface", typeParams, nil, "testpkg")
		require.NoError(t, err)

		typeArgs := []Type{StringType}

		result, err := instantiator.InstantiateInterface(genericInterface, typeArgs)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "testpkg.TestInterface[string]", result.TypeSignature)
		assert.True(t, result.ValidationResult.Valid)
		assert.Equal(t, 0, len(result.ValidationResult.ViolatedConstraints))
		assert.False(t, result.CacheHit)
		assert.GreaterOrEqual(t, result.InstantiationDurationMS, int64(0))

		// Verify type arguments mapping
		assert.Equal(t, 1, len(result.TypeArguments))
		assert.Equal(t, StringType, result.TypeArguments["T"])
	})

	t.Run("successful instantiation with comparable constraint", func(t *testing.T) {
		// Create a generic interface with 'comparable' constraint
		typeParams := []TypeParam{
			*NewComparableTypeParam("T", 0),
		}
		genericInterface, err := NewGenericInterface("ComparableInterface", typeParams, nil, "testpkg")
		require.NoError(t, err)

		typeArgs := []Type{StringType} // String is comparable

		result, err := instantiator.InstantiateInterface(genericInterface, typeArgs)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.ValidationResult.Valid)
		assert.Equal(t, 0, len(result.ValidationResult.ViolatedConstraints))
	})

	t.Run("constraint violation with comparable", func(t *testing.T) {
		// Create a generic interface with 'comparable' constraint
		typeParams := []TypeParam{
			*NewComparableTypeParam("T", 0),
		}
		genericInterface, err := NewGenericInterface("ComparableInterface", typeParams, nil, "testpkg")
		require.NoError(t, err)

		// Create a non-comparable type (slice)
		sliceType := NewSliceType(StringType, "")
		typeArgs := []Type{sliceType}

		result, err := instantiator.InstantiateInterface(genericInterface, typeArgs)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrConstraintViolation)
		assert.Nil(t, result)
	})

	t.Run("union constraint validation", func(t *testing.T) {
		// Create a generic interface with union constraint
		unionTypes := []Type{StringType, IntType}
		typeParams := []TypeParam{
			*NewUnionTypeParam("T", unionTypes, 0),
		}
		genericInterface, err := NewGenericInterface("UnionInterface", typeParams, nil, "testpkg")
		require.NoError(t, err)

		// Test with valid type (string)
		typeArgs := []Type{StringType}

		result, err := instantiator.InstantiateInterface(genericInterface, typeArgs)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.ValidationResult.Valid)

		// Test with invalid type (float)
		typeArgs = []Type{Float64Type}

		_, err = instantiator.InstantiateInterface(genericInterface, typeArgs)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrConstraintViolation)
	})

	t.Run("cache functionality", func(t *testing.T) {
		// Create a fresh instantiator for this test to avoid cache contamination
		freshInstantiator := NewTypeInstantiator(NewTypeBuilder(), zaptest.NewLogger(t))

		typeParams := []TypeParam{
			*NewAnyTypeParam("T", 0),
		}
		genericInterface, err := NewGenericInterface("CacheTestInterface", typeParams, nil, "testpkg")
		require.NoError(t, err)

		typeArgs := []Type{StringType}

		// First instantiation - should miss cache
		result1, err := freshInstantiator.InstantiateInterface(genericInterface, typeArgs)
		require.NoError(t, err)
		assert.False(t, result1.CacheHit)

		// Second instantiation - should hit cache
		result2, err := freshInstantiator.InstantiateInterface(genericInterface, typeArgs)
		require.NoError(t, err)
		assert.True(t, result2.CacheHit)
		assert.Equal(t, result1.TypeSignature, result2.TypeSignature)

		// Verify cache stats
		stats := freshInstantiator.GetCacheStats()
		assert.Equal(t, int64(1), stats.CacheHits)
		assert.Equal(t, int64(1), stats.CacheMisses)
		assert.Equal(t, int64(2), stats.TotalInstantiations)
		assert.Equal(t, 50.0, stats.HitRate) // 1 hit out of 2 total
	})

	t.Run("nil generic interface", func(t *testing.T) {
		result, err := instantiator.InstantiateInterface(nil, []Type{StringType})

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrGenericInterfaceNil)
		assert.Nil(t, result)
	})

	t.Run("nil type arguments", func(t *testing.T) {
		typeParams := []TypeParam{*NewAnyTypeParam("T", 0)}
		genericInterface, err := NewGenericInterface("TestInterface", typeParams, nil, "testpkg")
		require.NoError(t, err)

		result, err := instantiator.InstantiateInterface(genericInterface, nil)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrTypeArgumentsNil)
		assert.Nil(t, result)
	})

	t.Run("type argument count mismatch", func(t *testing.T) {
		typeParams := []TypeParam{
			*NewAnyTypeParam("T", 0),
			*NewAnyTypeParam("U", 1),
		}
		genericInterface, err := NewGenericInterface("TestInterface", typeParams, nil, "testpkg")
		require.NoError(t, err)

		// Provide only one type argument for two parameters
		typeArgs := []Type{StringType}

		result, err := instantiator.InstantiateInterface(genericInterface, typeArgs)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrTypeArgumentCountMismatch)
		assert.Nil(t, result)
	})

	t.Run("nil type argument", func(t *testing.T) {
		typeParams := []TypeParam{*NewAnyTypeParam("T", 0)}
		genericInterface, err := NewGenericInterface("TestInterface", typeParams, nil, "testpkg")
		require.NoError(t, err)

		typeArgs := []Type{nil}

		result, err := instantiator.InstantiateInterface(genericInterface, typeArgs)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidTypeArgument)
		assert.Nil(t, result)
	})
}

func TestRecursionDetection(t *testing.T) {
	typeBuilder := NewTypeBuilder()
	logger := zaptest.NewLogger(t)
	config := &TypeInstantiatorConfig{
		MaxRecursionDepth:      2, // Low limit for testing
		EnableCaching:          true,
		EnablePerformanceTrack: true,
		CacheCapacity:          100,
	}
	instantiator := NewTypeInstantiatorWithConfig(typeBuilder, logger, config)

	t.Run("recursion depth limit", func(t *testing.T) {
		// Set up a scenario that would exceed recursion depth
		instantiator.recursionDepth = 2 // At max depth

		err := instantiator.checkRecursion("TestInterface[string]")

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrRecursiveInstantiation)
		assert.Contains(t, err.Error(), "maximum recursion depth")
	})

	t.Run("circular dependency detection", func(t *testing.T) {
		signature := "TestInterface[string]"
		instantiator.instantiationStack = []string{signature}
		instantiator.recursionDepth = 1 // Set to valid depth

		err := instantiator.checkRecursion(signature)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrCircularTypeDetected)
		assert.Contains(t, err.Error(), "circular dependency")
	})

	t.Run("valid recursion", func(t *testing.T) {
		instantiator.recursionDepth = 1
		instantiator.instantiationStack = []string{"OtherInterface[int]"}

		err := instantiator.checkRecursion("TestInterface[string]")

		assert.NoError(t, err)
	})
}

func TestConstraintValidation(t *testing.T) {
	typeBuilder := NewTypeBuilder()
	logger := zaptest.NewLogger(t)
	instantiator := NewTypeInstantiator(typeBuilder, logger)

	t.Run("valid any constraint", func(t *testing.T) {
		typeParams := []TypeParam{*NewAnyTypeParam("T", 0)}
		typeArgs := map[string]Type{"T": StringType}

		result := instantiator.validateConstraints(typeParams, typeArgs)
		assert.True(t, result.Valid)
		assert.Equal(t, 0, len(result.ViolatedConstraints))
		assert.Equal(t, 1, len(result.Details))
		assert.True(t, result.Details["T"].ValidationPassed)
	})

	t.Run("valid comparable constraint", func(t *testing.T) {
		typeParams := []TypeParam{*NewComparableTypeParam("T", 0)}
		typeArgs := map[string]Type{"T": StringType}

		result := instantiator.validateConstraints(typeParams, typeArgs)
		assert.True(t, result.Valid)
		assert.Equal(t, 0, len(result.ViolatedConstraints))
	})

	t.Run("invalid comparable constraint", func(t *testing.T) {
		typeParams := []TypeParam{*NewComparableTypeParam("T", 0)}
		sliceType := NewSliceType(StringType, "")
		typeArgs := map[string]Type{"T": sliceType}

		result := instantiator.validateConstraints(typeParams, typeArgs)
		assert.False(t, result.Valid)
		assert.Equal(t, 1, len(result.ViolatedConstraints))

		violation := result.ViolatedConstraints[0]
		assert.Equal(t, "T", violation.TypeParamName)
		assert.Equal(t, "comparable", violation.ExpectedConstraint)
		assert.Contains(t, violation.ViolationMessage, "not comparable")
	})

	t.Run("missing type argument", func(t *testing.T) {
		typeParams := []TypeParam{*NewAnyTypeParam("T", 0)}
		typeArgs := map[string]Type{} // Missing T

		result := instantiator.validateConstraints(typeParams, typeArgs)
		assert.False(t, result.Valid)
		assert.Equal(t, 1, len(result.ViolatedConstraints))

		violation := result.ViolatedConstraints[0]
		assert.Equal(t, "T", violation.TypeParamName)
		assert.Equal(t, "missing", violation.ActualType)
		assert.Contains(t, violation.ViolationMessage, "missing")
	})

	t.Run("union constraint validation", func(t *testing.T) {
		unionTypes := []Type{StringType, IntType}
		typeParams := []TypeParam{*NewUnionTypeParam("T", unionTypes, 0)}

		// Valid case - string matches union
		typeArgs := map[string]Type{"T": StringType}
		result := instantiator.validateConstraints(typeParams, typeArgs)
		assert.True(t, result.Valid)

		// Invalid case - float doesn't match union
		typeArgs = map[string]Type{"T": Float64Type}
		result = instantiator.validateConstraints(typeParams, typeArgs)
		assert.False(t, result.Valid)
		assert.Equal(t, 1, len(result.ViolatedConstraints))
		assert.Contains(t, result.ViolatedConstraints[0].ViolationMessage, "union types")
	})
}

func TestCacheOperations(t *testing.T) {
	typeBuilder := NewTypeBuilder()
	logger := zaptest.NewLogger(t)
	instantiator := NewTypeInstantiator(typeBuilder, logger)

	t.Run("cache stats with empty cache", func(t *testing.T) {
		stats := instantiator.GetCacheStats()

		assert.Equal(t, int64(0), stats.CacheHits)
		assert.Equal(t, int64(0), stats.CacheMisses)
		assert.Equal(t, int64(0), stats.TotalInstantiations)
		assert.Equal(t, int64(0), stats.CacheSize)
		assert.Equal(t, 0.0, stats.HitRate)
	})

	t.Run("clear cache", func(t *testing.T) {
		// Add some fake data to cache
		sourceInterface := NewBasicType("TestInterface", reflect.Interface)
		typeArgs := map[string]Type{"T": StringType}
		instantiated, err := NewInstantiatedInterface(
			sourceInterface,
			typeArgs,
			StringType,
			"TestInterface[string]",
		)
		require.NoError(t, err)

		instantiator.cache["TestInterface[string]"] = instantiated

		// Verify cache has content
		assert.Equal(t, 1, len(instantiator.cache))

		// Clear cache
		instantiator.ClearCache()

		// Verify cache is empty
		assert.Equal(t, 0, len(instantiator.cache))
	})

	t.Run("get cached instantiation", func(t *testing.T) {
		signature := "TestInterface[string]"
		sourceInterface := NewBasicType("TestInterface", reflect.Interface)
		typeArgs := map[string]Type{"T": StringType}
		expected, err := NewInstantiatedInterface(
			sourceInterface,
			typeArgs,
			StringType,
			signature,
		)
		require.NoError(t, err)

		instantiator.cache[signature] = expected

		// Test retrieval
		result, found := instantiator.GetCachedInstantiation(signature)
		assert.True(t, found)
		assert.Equal(t, expected, result)

		// Test non-existent
		result, found = instantiator.GetCachedInstantiation("NonExistent")
		assert.False(t, found)
		assert.Nil(t, result)
	})

	t.Run("has cached instantiation", func(t *testing.T) {
		signature := "TestInterface[string]"
		sourceInterface := NewBasicType("TestInterface", reflect.Interface)
		typeArgs := map[string]Type{"T": StringType}
		instantiated, err := NewInstantiatedInterface(
			sourceInterface,
			typeArgs,
			StringType,
			signature,
		)
		require.NoError(t, err)

		instantiator.cache[signature] = instantiated

		assert.True(t, instantiator.HasCachedInstantiation(signature))
		assert.False(t, instantiator.HasCachedInstantiation("NonExistent"))
	})
}

func TestTypeSignatureGeneration(t *testing.T) {
	typeBuilder := NewTypeBuilder()
	logger := zaptest.NewLogger(t)
	instantiator := NewTypeInstantiator(typeBuilder, logger)

	t.Run("single type parameter", func(t *testing.T) {
		typeParams := []TypeParam{*NewAnyTypeParam("T", 0)}
		genericInterface, err := NewGenericInterface("TestInterface", typeParams, nil, "testpkg")
		require.NoError(t, err)

		typeArgs := map[string]Type{"T": StringType}

		signature := instantiator.generateTypeSignature(genericInterface, typeArgs)

		assert.Equal(t, "testpkg.TestInterface[string]", signature)
	})

	t.Run("multiple type parameters", func(t *testing.T) {
		typeParams := []TypeParam{
			*NewAnyTypeParam("T", 0),
			*NewAnyTypeParam("U", 1),
		}
		genericInterface, err := NewGenericInterface("TestInterface", typeParams, nil, "testpkg")
		require.NoError(t, err)

		typeArgs := map[string]Type{
			"T": StringType,
			"U": IntType,
		}

		signature := instantiator.generateTypeSignature(genericInterface, typeArgs)

		assert.Equal(t, "testpkg.TestInterface[string,int]", signature)
	})

	t.Run("no package", func(t *testing.T) {
		typeParams := []TypeParam{*NewAnyTypeParam("T", 0)}
		genericInterface, err := NewGenericInterface("TestInterface", typeParams, nil, "")
		require.NoError(t, err)

		typeArgs := map[string]Type{"T": StringType}

		signature := instantiator.generateTypeSignature(genericInterface, typeArgs)

		assert.Equal(t, "TestInterface[string]", signature)
	})
}

func TestConcreteTypeNameGeneration(t *testing.T) {
	typeBuilder := NewTypeBuilder()
	logger := zaptest.NewLogger(t)
	instantiator := NewTypeInstantiator(typeBuilder, logger)

	t.Run("single type parameter", func(t *testing.T) {
		typeParams := []TypeParam{*NewAnyTypeParam("T", 0)}
		genericInterface, err := NewGenericInterface("TestInterface", typeParams, nil, "testpkg")
		require.NoError(t, err)

		typeArgs := map[string]Type{"T": StringType}

		name := instantiator.generateConcreteTypeName(genericInterface, typeArgs)

		assert.Equal(t, "TestInterface[string]", name)
	})

	t.Run("multiple type parameters", func(t *testing.T) {
		typeParams := []TypeParam{
			*NewAnyTypeParam("T", 0),
			*NewAnyTypeParam("U", 1),
		}
		genericInterface, err := NewGenericInterface("TestInterface", typeParams, nil, "testpkg")
		require.NoError(t, err)

		typeArgs := map[string]Type{
			"T": StringType,
			"U": IntType,
		}

		name := instantiator.generateConcreteTypeName(genericInterface, typeArgs)

		assert.Equal(t, "TestInterface[string, int]", name)
	})
}

func TestPerformanceTracking(t *testing.T) {
	t.Run("instantiation duration tracking", func(t *testing.T) {
		typeBuilder := NewTypeBuilder()
		logger := zaptest.NewLogger(t)
		instantiator := NewTypeInstantiator(typeBuilder, logger)

		typeParams := []TypeParam{*NewAnyTypeParam("T", 0)}
		genericInterface, err := NewGenericInterface("TestInterface", typeParams, nil, "testpkg")
		require.NoError(t, err)

		typeArgs := []Type{StringType}

		result, err := instantiator.InstantiateInterface(genericInterface, typeArgs)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, result.InstantiationDurationMS, int64(0))
		assert.GreaterOrEqual(t, result.ValidationResult.ValidationDurationMS, int64(0))
	})

	t.Run("hit rate calculation with zero total", func(t *testing.T) {
		typeBuilder := NewTypeBuilder()
		logger := zaptest.NewLogger(t)
		instantiator := NewTypeInstantiator(typeBuilder, logger)

		hitRate := instantiator.calculateHitRate()
		assert.Equal(t, 0.0, hitRate)
	})

	t.Run("hit rate calculation with data", func(t *testing.T) {
		typeBuilder := NewTypeBuilder()
		logger := zaptest.NewLogger(t)
		instantiator := NewTypeInstantiator(typeBuilder, logger)

		instantiator.cacheHits = 3
		instantiator.cacheMisses = 7

		hitRate := instantiator.calculateHitRate()
		assert.Equal(t, 30.0, hitRate) // 3 hits out of 10 total = 30%
	})
}

func TestConstraintViolationMessages(t *testing.T) {
	typeBuilder := NewTypeBuilder()
	logger := zaptest.NewLogger(t)
	instantiator := NewTypeInstantiator(typeBuilder, logger)

	t.Run("any constraint message", func(t *testing.T) {
		param := NewAnyTypeParam("T", 0)
		message := instantiator.generateConstraintViolationMessage(param, StringType)
		assert.Contains(t, message, "should never happen")
	})

	t.Run("comparable constraint message", func(t *testing.T) {
		param := NewComparableTypeParam("T", 0)
		sliceType := NewSliceType(StringType, "")
		message := instantiator.generateConstraintViolationMessage(param, sliceType)
		assert.Contains(t, message, "not comparable")
		assert.Contains(t, message, sliceType.String())
	})

	t.Run("union constraint message", func(t *testing.T) {
		unionTypes := []Type{StringType, IntType}
		param := NewUnionTypeParam("T", unionTypes, 0)
		message := instantiator.generateConstraintViolationMessage(param, Float64Type)
		assert.Contains(t, message, "union types")
		assert.Contains(t, message, "string | int")
	})

	t.Run("union underlying constraint message", func(t *testing.T) {
		unionTypes := []Type{StringType, IntType}
		param := NewUnionUnderlyingTypeParam("T", unionTypes, 0)
		message := instantiator.generateConstraintViolationMessage(param, Float64Type)
		assert.Contains(t, message, "underlying union types")
		assert.Contains(t, message, "~string | ~int")
	})

	t.Run("underlying constraint message", func(t *testing.T) {
		underlying := NewUnderlyingConstraint(StringType, "")
		param := NewUnderlyingTypeParam("T", underlying, 0)
		message := instantiator.generateConstraintViolationMessage(param, IntType)
		assert.Contains(t, message, "underlying type")
		assert.Contains(t, message, "~string")
	})

	t.Run("interface constraint message", func(t *testing.T) {
		constraint := NewBasicType("Stringer", reflect.Interface)
		param := NewTypeParam("T", constraint, 0)
		message := instantiator.generateConstraintViolationMessage(param, IntType)
		assert.Contains(t, message, "does not implement interface")
		assert.Contains(t, message, "Stringer")
	})

	t.Run("unknown constraint message", func(t *testing.T) {
		// Create a type parameter with no recognizable constraint pattern
		param := &TypeParam{
			Name:       "T",
			Constraint: nil,
			Index:      0,
		}
		message := instantiator.generateConstraintViolationMessage(param, IntType)
		assert.Contains(t, message, "does not satisfy constraint")
	})
}

// Benchmark tests for performance validation.
func BenchmarkInstantiateInterface(b *testing.B) {
	typeBuilder := NewTypeBuilder()
	logger := zaptest.NewLogger(b)
	instantiator := NewTypeInstantiator(typeBuilder, logger)

	typeParams := []TypeParam{*NewAnyTypeParam("T", 0)}
	genericInterface, err := NewGenericInterface("BenchInterface", typeParams, nil, "testpkg")
	require.NoError(b, err)

	typeArgs := []Type{StringType}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := instantiator.InstantiateInterface(genericInterface, typeArgs)
		require.NoError(b, err)
	}
}

func BenchmarkCacheHit(b *testing.B) {
	typeBuilder := NewTypeBuilder()
	logger := zaptest.NewLogger(b)
	instantiator := NewTypeInstantiator(typeBuilder, logger)

	typeParams := []TypeParam{*NewAnyTypeParam("T", 0)}
	genericInterface, err := NewGenericInterface("BenchInterface", typeParams, nil, "testpkg")
	require.NoError(b, err)

	typeArgs := []Type{StringType}

	// Prime the cache
	_, err = instantiator.InstantiateInterface(genericInterface, typeArgs)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := instantiator.InstantiateInterface(genericInterface, typeArgs)
		require.NoError(b, err)
	}
}

func TestCopyTypeArguments(t *testing.T) {
	t.Run("creates defensive copy", func(t *testing.T) {
		original := map[string]Type{
			"T": StringType,
			"U": IntType,
		}

		cp := copyTypeArguments(original)

		// Verify contents are equal
		assert.Equal(t, original, cp)

		// Verify modifying copy doesn't affect original
		cp["V"] = Float64Type
		assert.Equal(t, 2, len(original))
		assert.Equal(t, 3, len(cp))

		// Verify original values are preserved
		assert.Equal(t, StringType, original["T"])
		assert.Equal(t, IntType, original["U"])
		assert.Equal(t, StringType, cp["T"])
		assert.Equal(t, IntType, cp["U"])
		assert.Equal(t, Float64Type, cp["V"])
	})

	t.Run("handles empty map", func(t *testing.T) {
		original := map[string]Type{}
		cp := copyTypeArguments(original)

		assert.Equal(t, 0, len(cp))
		assert.NotNil(t, cp)
	})
}

func TestEdgeCases(t *testing.T) {
	typeBuilder := NewTypeBuilder()
	logger := zaptest.NewLogger(t)

	t.Run("instantiator with disabled caching", func(t *testing.T) {
		config := &TypeInstantiatorConfig{
			MaxRecursionDepth:      10,
			EnableCaching:          false,
			EnablePerformanceTrack: true,
			CacheCapacity:          0,
		}
		instantiator := NewTypeInstantiatorWithConfig(typeBuilder, logger, config)

		assert.Nil(t, instantiator.cache)

		typeParams := []TypeParam{*NewAnyTypeParam("T", 0)}
		genericInterface, err := NewGenericInterface("TestInterface", typeParams, nil, "testpkg")
		require.NoError(t, err)

		typeArgs := []Type{StringType}

		// Should work without caching
		result, err := instantiator.InstantiateInterface(genericInterface, typeArgs)
		require.NoError(t, err)
		assert.False(t, result.CacheHit)

		// Cache operations should be safe
		stats := instantiator.GetCacheStats()
		assert.Equal(t, int64(0), stats.CacheSize)

		instantiator.ClearCache() // Should not panic

		found := instantiator.HasCachedInstantiation("anything")
		assert.False(t, found)

		cached, found := instantiator.GetCachedInstantiation("anything")
		assert.False(t, found)
		assert.Nil(t, cached)
	})
}
