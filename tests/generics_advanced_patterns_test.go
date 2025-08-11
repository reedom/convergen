package testing

import (
	"testing"

	"github.com/reedom/convergen/v8/tests/helpers"
)

// TestGenericsMultipleTypeParameters tests scenarios with multiple type parameters
func TestGenericsMultipleTypeParameters(t *testing.T) {
	t.Parallel()
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	multiTypeTests := []struct {
		name        string
		description string
		types       string
		interface_  string
		checks      []helpers.CodeAssertion
	}{
		{
			name:        "ThreeTypeParameters",
			description: "Should handle converter with three type parameters",
			types: `
type SourceA struct {
	ID   int
	Name string
}

type SourceB struct {
	Value  string
	Active bool
}

type SourceC struct {
	Data   []string
	Count  int
}

type TargetResult struct {
	AID     int
	AName   string
	BValue  string
	BActive bool
	CData   []string
	CCount  int
}`,
			interface_: `
// :convergen
type TripleConverter[T any, U any, V any] interface {
	ConvertTriple(T, U, V) TargetResult
}`,
			checks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func ConvertTriple"),
				helpers.CompilesSuccessfully(),
			},
		},
		{
			name:        "FiveTypeParameters",
			description: "Should handle converter with five type parameters",
			types: `
type Input1 struct { Field1 string }
type Input2 struct { Field2 int }
type Input3 struct { Field3 bool }
type Input4 struct { Field4 float64 }
type Input5 struct { Field5 []string }

type CombinedOutput struct {
	Field1 string
	Field2 int
	Field3 bool
	Field4 float64
	Field5 []string
}`,
			interface_: `
// :convergen
type PentaConverter[T1 any, T2 any, T3 any, T4 any, T5 any] interface {
	CombineAll(T1, T2, T3, T4, T5) CombinedOutput
}`,
			checks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func CombineAll"),
				helpers.CompilesSuccessfully(),
			},
		},
		{
			name:        "MixedConstraintParameters",
			description: "Should handle mixed constraint types across multiple parameters",
			types: `
type ComparableData struct {
	ID int
}

type StringerData struct {
	Value string
}

func (s StringerData) String() string {
	return s.Value
}

type AnyData struct {
	Content interface{}
}

type MixedResult struct {
	ComparableField int
	StringerField   string
	AnyField        interface{}
}`,
			interface_: `
// :convergen
type MixedConstraintConverter[T comparable, U fmt.Stringer, V any] interface {
	ConvertMixed(T, U, V) MixedResult
}`,
			checks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func ConvertMixed"),
				helpers.CompilesSuccessfully(),
			},
		},
	}

	for _, tt := range multiTypeTests {
		t.Run(tt.name, func(t *testing.T) {
			scenario := helpers.NewInlineScenario(tt.name, tt.description).
				WithTypes(tt.types).
				WithInterface(tt.interface_).
				WithBehaviorTests().
				WithCodeChecks(tt.checks...)

			runner.RunScenario(scenario)
		})
	}
}

// TestGenericsComplexConstraints tests advanced constraint scenarios
func TestGenericsComplexConstraints(t *testing.T) {
	t.Parallel()
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	constraintTests := []struct {
		name        string
		description string
		types       string
		interface_  string
		checks      []helpers.CodeAssertion
	}{
		{
			name:        "UnionConstraints",
			description: "Should handle union constraint types",
			types: `
type IntOrString interface {
	~int | ~string
}

type IntData struct {
	Value int
}

type StringData struct {
	Value string
}

type UnifiedTarget struct {
	IntValue    int
	StringValue string
}`,
			interface_: `
// :convergen
type UnionConverter[T IntOrString, U any] interface {
	ConvertUnion(T) U
}`,
			checks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func ConvertUnion"),
				helpers.CompilesSuccessfully(),
			},
		},
		{
			name:        "InterfaceConstraints",
			description: "Should handle interface constraints with methods",
			types: `
type Processor interface {
	Process() string
	Validate() bool
}

type DataProcessor struct {
	Data string
}

func (d DataProcessor) Process() string {
	return d.Data
}

func (d DataProcessor) Validate() bool {
	return len(d.Data) > 0
}

type ProcessorTarget struct {
	ProcessedData string
	IsValid       bool
}`,
			interface_: `
// :convergen
type InterfaceConstraintConverter[T Processor, U any] interface {
	ConvertProcessor(T) U
}`,
			checks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func ConvertProcessor"),
				helpers.CompilesSuccessfully(),
			},
		},
		{
			name:        "CombinedInterfaceConstraints",
			description: "Should handle combined interface constraints",
			types: `
type Serializer interface {
	Serialize() []byte
}

type Validator interface {
	IsValid() bool
}

type ComplexData struct {
	Content string
}

func (c ComplexData) String() string {
	return c.Content
}

func (c ComplexData) Serialize() []byte {
	return []byte(c.Content)
}

func (c ComplexData) IsValid() bool {
	return len(c.Content) > 0
}

type ComplexTarget struct {
	StringValue      string
	SerializedValue  []byte
	ValidationResult bool
}`,
			interface_: `
// :convergen
type CombinedConstraintConverter[T interface {
	fmt.Stringer
	Serializer
	Validator
}, U any] interface {
	ConvertComplex(T) U
}`,
			checks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func ConvertComplex"),
				helpers.CompilesSuccessfully(),
			},
		},
	}

	for _, tt := range constraintTests {
		t.Run(tt.name, func(t *testing.T) {
			scenario := helpers.NewInlineScenario(tt.name, tt.description).
				WithTypes(tt.types).
				WithInterface(tt.interface_).
				WithBehaviorTests().
				WithCodeChecks(tt.checks...)

			runner.RunScenario(scenario)
		})
	}
}

// TestGenericsNestedGenericTypes tests nested generic type scenarios
func TestGenericsNestedGenericTypes(t *testing.T) {
	t.Parallel()
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	nestedTests := []struct {
		name        string
		description string
		types       string
		interface_  string
		checks      []helpers.CodeAssertion
	}{
		{
			name:        "GenericSliceConversion",
			description: "Should convert generic slice types",
			types: `
type Container[T any] struct {
	Items []T
	Count int
}

type StringContainer struct {
	Items []string
	Count int
}`,
			interface_: `
// :convergen
type GenericSliceConverter[T any, U any] interface {
	ConvertContainer(Container[T]) U
}`,
			checks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func ConvertContainer"),
				helpers.Contains("Items"),
				helpers.Contains("Count"),
				helpers.CompilesSuccessfully(),
			},
		},
		{
			name:        "GenericMapConversion",
			description: "Should convert generic map types",
			types: `
type KeyValue[K comparable, V any] struct {
	Data map[K]V
	Size int
}

type StringIntMap struct {
	Data map[string]int
	Size int
}`,
			interface_: `
// :convergen
type GenericMapConverter[K comparable, V any, T any] interface {
	ConvertMap(KeyValue[K, V]) T
}`,
			checks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func ConvertMap"),
				helpers.Contains("Data"),
				helpers.Contains("Size"),
				helpers.CompilesSuccessfully(),
			},
		},
		{
			name:        "DeeplyNestedGenerics",
			description: "Should handle deeply nested generic structures",
			types: `
type Level1[T any] struct {
	Data T
}

type Level2[T any] struct {
	Container Level1[[]T]
	Meta      string
}

type Level3[T any] struct {
	Nested Level2[map[string]T]
	ID     int
}

type FlatResult struct {
	NestedData interface{}
	Meta       string
	ID         int
}`,
			interface_: `
// :convergen
type DeepNestedConverter[T any, U any] interface {
	ConvertDeep(Level3[T]) U
}`,
			checks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func ConvertDeep"),
				helpers.CompilesSuccessfully(),
			},
		},
		{
			name:        "GenericChannelConversion",
			description: "Should handle generic channel types",
			types: `
type ChannelWrapper[T any] struct {
	Input  chan T
	Output chan T
	Buffer []T
}

type StringChannelResult struct {
	Buffer []string
	Size   int
}`,
			interface_: `
// :convergen
type ChannelConverter[T any, U any] interface {
	ConvertChannel(ChannelWrapper[T]) U
}`,
			checks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func ConvertChannel"),
				helpers.Contains("Buffer"),
				helpers.CompilesSuccessfully(),
			},
		},
	}

	for _, tt := range nestedTests {
		t.Run(tt.name, func(t *testing.T) {
			scenario := helpers.NewInlineScenario(tt.name, tt.description).
				WithTypes(tt.types).
				WithInterface(tt.interface_).
				WithBehaviorTests().
				WithCodeChecks(tt.checks...)

			runner.RunScenario(scenario)
		})
	}
}

// TestGenericsGenericMethodsAndInterfaces tests generic methods and interfaces
func TestGenericsGenericMethodsAndInterfaces(t *testing.T) {
	t.Parallel()
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	methodTests := []struct {
		name        string
		description string
		types       string
		interface_  string
		checks      []helpers.CodeAssertion
	}{
		{
			name:        "MultipleGenericMethods",
			description: "Should handle multiple generic methods in interface",
			types: `
type SourceData struct {
	StringValue string
	IntValue    int
	BoolValue   bool
}

type StringTarget struct {
	Value string
}

type IntTarget struct {
	Value int
}

type BoolTarget struct {
	Value bool
}`,
			interface_: `
// :convergen
type MultiMethodConverter[S any, T1 any, T2 any, T3 any] interface {
	ConvertToString(S) T1
	ConvertToInt(S) T2
	ConvertToBool(S) T3
}`,
			checks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func ConvertToString"),
				helpers.Contains("func ConvertToInt"),
				helpers.Contains("func ConvertToBool"),
				helpers.CompilesSuccessfully(),
			},
		},
		{
			name:        "GenericMethodWithConstraints",
			description: "Should handle generic methods with different constraints",
			types: `
type ComparableSource struct {
	ID   int
	Name string
}

type StringerSource struct {
	Value string
}

func (s StringerSource) String() string {
	return s.Value
}

type AnySource struct {
	Data interface{}
}

type UniversalTarget struct {
	ComparableData interface{}
	StringerData   string
	AnyData        interface{}
}`,
			interface_: `
// :convergen
type ConstrainedMethodConverter[C comparable, S fmt.Stringer, A any, T any] interface {
	ConvertComparable(C) T
	ConvertStringer(S) T
	ConvertAny(A) T
}`,
			checks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func ConvertComparable"),
				helpers.Contains("func ConvertStringer"),
				helpers.Contains("func ConvertAny"),
				helpers.CompilesSuccessfully(),
			},
		},
		{
			name:        "GenericVariadicMethods",
			description: "Should handle generic methods with variadic parameters",
			types: `
type Item struct {
	Name string
	ID   int
}

type BatchResult struct {
	Items []Item
	Count int
}`,
			interface_: `
// :convergen
type VariadicConverter[T any, U any] interface {
	ConvertBatch(...T) U
}`,
			checks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func ConvertBatch"),
				helpers.CompilesSuccessfully(),
			},
		},
	}

	for _, tt := range methodTests {
		t.Run(tt.name, func(t *testing.T) {
			scenario := helpers.NewInlineScenario(tt.name, tt.description).
				WithTypes(tt.types).
				WithInterface(tt.interface_).
				WithBehaviorTests().
				WithCodeChecks(tt.checks...)

			runner.RunScenario(scenario)
		})
	}
}

// TestGenericsPerformanceScenarios tests performance-related scenarios
func TestGenericsPerformanceScenarios(t *testing.T) {
	t.Parallel()
	runner := helpers.NewInlineScenarioRunner(t)
	defer runner.Cleanup()

	performanceTests := []struct {
		name        string
		description string
		types       string
		interface_  string
		checks      []helpers.CodeAssertion
	}{
		{
			name:        "LargeStructConversion",
			description: "Should efficiently handle large struct conversions",
			types: `
type LargeSource struct {
	Field01, Field02, Field03, Field04, Field05 string
	Field06, Field07, Field08, Field09, Field10 string
	Field11, Field12, Field13, Field14, Field15 int
	Field16, Field17, Field18, Field19, Field20 int
	Field21, Field22, Field23, Field24, Field25 bool
	Field26, Field27, Field28, Field29, Field30 bool
}

type LargeTarget struct {
	Field01, Field02, Field03, Field04, Field05 string
	Field06, Field07, Field08, Field09, Field10 string
	Field11, Field12, Field13, Field14, Field15 int
	Field16, Field17, Field18, Field19, Field20 int
	Field21, Field22, Field23, Field24, Field25 bool
	Field26, Field27, Field28, Field29, Field30 bool
}`,
			interface_: `
// :convergen
type LargeStructConverter[T any, U any] interface {
	ConvertLarge(T) U
}`,
			checks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func ConvertLarge"),
				helpers.Contains("Field01"),
				helpers.Contains("Field15"),
				helpers.Contains("Field30"),
				helpers.CompilesSuccessfully(),
			},
		},
		{
			name:        "ManyTypeParametersPerformance",
			description: "Should handle many type parameters efficiently",
			types: `
type Source1 struct { Data1 string }
type Source2 struct { Data2 int }
type Source3 struct { Data3 bool }
type Source4 struct { Data4 float64 }
type Source5 struct { Data5 []string }
type Source6 struct { Data6 map[string]int }
type Source7 struct { Data7 interface{} }
type Source8 struct { Data8 chan string }

type ComplexTarget struct {
	Data1 string
	Data2 int
	Data3 bool
	Data4 float64
	Data5 []string
	Data6 map[string]int
	Data7 interface{}
	Data8 chan string
}`,
			interface_: `
// :convergen
type ManyTypeConverter[T1, T2, T3, T4, T5, T6, T7, T8 any] interface {
	ConvertMany(T1, T2, T3, T4, T5, T6, T7, T8) ComplexTarget
}`,
			checks: []helpers.CodeAssertion{
				helpers.AssertHasGeneratedFunction(),
				helpers.Contains("func ConvertMany"),
				helpers.Contains("Data1"),
				helpers.Contains("Data8"),
				helpers.CompilesSuccessfully(),
			},
		},
	}

	for _, tt := range performanceTests {
		t.Run(tt.name, func(t *testing.T) {
			scenario := helpers.NewInlineScenario(tt.name, tt.description).
				WithTypes(tt.types).
				WithInterface(tt.interface_).
				WithBehaviorTests().
				WithCodeChecks(tt.checks...)

			runner.RunScenario(scenario)
		})
	}
}
