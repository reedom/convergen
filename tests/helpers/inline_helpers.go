// Package helpers provides testing utilities for the Convergen behavior-driven testing framework.
package helpers

import (
	"fmt"
	"strings"
)

// NewInlineScenario creates a new inline scenario.
func NewInlineScenario(name, description string) InlineScenario {
	return InlineScenario{
		Name:        name,
		Description: description,
	}
}

// WithTypes adds type definitions to the scenario.
func (is InlineScenario) WithTypes(sourceTypes string) InlineScenario {
	is.SourceTypes = sourceTypes
	return is
}

// WithInterface adds an interface definition to the scenario.
func (is InlineScenario) WithInterface(interfaceDef string) InlineScenario {
	is.Interface = interfaceDef
	return is
}

// WithImports adds imports to the scenario.
func (is InlineScenario) WithImports(imports ...string) InlineScenario {
	is.Imports = imports
	return is
}

// Common type definitions for reuse.

// SimpleUserTypes creates basic User and UserModel types.
func SimpleUserTypes() string {
	return `
type User struct {
	ID   uint64
	Name string
}

type UserModel struct {
	ID   uint64
	Name string
}`
}

// SimpleConverter creates a basic converter interface.
func SimpleConverter(methodSig string) string {
	return fmt.Sprintf(`
type Convergen interface {
	%s
}`, methodSig)
}

// BehaviorTest builders.

// NewBehaviorTest creates a new behavior test.
func NewBehaviorTest(name, description, testFunc string) BehaviorTest {
	return BehaviorTest{
		Name:        name,
		Description: description,
		TestFunc:    testFunc,
	}
}

// WithInput sets the input value for a behavior test.
func (bt BehaviorTest) WithInput(input interface{}) BehaviorTest {
	bt.Input = input
	return bt
}

// WithExpected sets the expected output for a behavior test.
func (bt BehaviorTest) WithExpected(expected interface{}) BehaviorTest {
	bt.Expected = expected
	return bt
}

// ExpectError marks a behavior test as expected to fail.
func (bt BehaviorTest) ExpectError() BehaviorTest {
	bt.ShouldError = true
	return bt
}

// Debugging helpers.

// DebugScenario creates a scenario with verbose debugging enabled.
// Use this when you need to see the full generated code output.
func DebugScenario(name, description string) InlineScenario {
	return InlineScenario{
		Name:        name,
		Description: description,
	}
}

// WithDebug enables verbose debugging for any scenario.
func WithDebug(scenario TestScenario) TestScenario {
	return scenario.WithVerboseDebugging()
}

// Annotation-specific scenario builders.

// StyleAnnotationScenario creates a scenario for testing :style annotation.
func StyleAnnotationScenario(style string) InlineScenario {
	return NewInlineScenario(
		fmt.Sprintf("Style_%s", style),
		fmt.Sprintf("Test :style %s annotation", style),
	).WithTypes(`
type User struct {
	ID   uint64
	Name string
}

type UserModel struct {
	ID   uint64 
	Name string
}`).WithInterface(fmt.Sprintf(`
type Convergen interface {
	// :style %s
	Convert(*User) *UserModel
}`, style))
}

// MatchAnnotationScenario creates a scenario for testing :match annotation.
func MatchAnnotationScenario(algorithm string) InlineScenario {
	return NewInlineScenario(
		fmt.Sprintf("Match_%s", algorithm),
		fmt.Sprintf("Test :match %s annotation", algorithm),
	).WithTypes(`
type Source struct {
	UserName string
	UserAge  int 
}

type Dest struct {
	Name string
	Age  int
}`).WithInterface(fmt.Sprintf(`
type Convergen interface {
	// :match %s
	Convert(*Source) *Dest
}`, algorithm))
}

// ConvertAnnotationScenario creates a scenario for testing :conv annotation.
func ConvertAnnotationScenario(converter, src, dst string) InlineScenario {
	return NewInlineScenario(
		"Conv_Converter",
		"Test :conv annotation with custom converter function",
	).WithTypes(`
type User struct {
	ID       uint64
	Password string
}

type UserModel struct {
	ID             uint64
	HashedPassword string
}

func HashPassword(password string) string {
	return "hashed_" + password
}`).WithInterface(fmt.Sprintf(`
type Convergen interface {
	// :conv %s %s %s
	Convert(*User) *UserModel
}`, converter, src, dst))
}

// LiteralAnnotationScenario creates a scenario for testing :literal annotation.
func LiteralAnnotationScenario(field, literal string) InlineScenario {
	return NewInlineScenario(
		"Literal_Assignment",
		"Test :literal annotation for direct value assignment",
	).WithTypes(`
type User struct {
	Name string
}

type UserModel struct {
	Name   string
	Status string
}`).WithInterface(fmt.Sprintf(`
type Convergen interface {
	// :literal %s %s
	Convert(*User) *UserModel
}`, field, literal))
}

// SkipAnnotationScenario creates a scenario for testing :skip annotation.
func SkipAnnotationScenario(pattern string) InlineScenario {
	return NewInlineScenario(
		"Skip_Fields",
		"Test :skip annotation for field exclusion",
	).WithTypes(`
type User struct {
	Name     string
	Password string
	Email    string
}

type UserModel struct {
	Name  string
	Email string
}`).WithInterface(fmt.Sprintf(`
type Convergen interface {
	// :skip %s
	Convert(*User) *UserModel
}`, pattern))
}

// TypecastAnnotationScenario creates a scenario for testing :typecast annotation.
func TypecastAnnotationScenario() InlineScenario {
	return NewInlineScenario(
		"TypecastAnnotation",
		"Test :typecast annotation for type casting",
	).WithTypes(`
type User struct {
	ID   int64
	Name string
}

type UserModel struct {
	ID   int32  // Different numeric type requiring casting
	Name string
}`).WithInterface(`
type Convergen interface {
	// :typecast
	Convert(*User) *UserModel
}`)
}

// StringerAnnotationScenario creates a scenario for testing :stringer annotation.
func StringerAnnotationScenario() InlineScenario {
	return NewInlineScenario(
		"StringerAnnotation",
		"Test :stringer annotation for String() method usage",
	).WithTypes(`
type Status int

const (
	Active Status = iota
	Inactive
)

func (s Status) String() string {
	switch s {
	case Active:
		return "active"
	case Inactive:
		return "inactive"
	default:
		return "unknown"
	}
}

type User struct {
	Name   string
	Status Status
}

type UserModel struct {
	Name   string
	Status string  // String representation
}`).WithInterface(`
type Convergen interface {
	// :stringer
	Convert(*User) *UserModel
}`)
}

// RecvAnnotationScenario creates a scenario for testing :recv annotation.
func RecvAnnotationScenario(receiverVar string) InlineScenario {
	return NewInlineScenario(
		fmt.Sprintf("RecvAnnotation_%s", receiverVar),
		"Test :recv annotation for receiver method generation",
	).WithTypes(`
type User struct {
	Name string
	Age  int
}

type UserModel struct {
	Name string
	Age  int
}`).WithInterface(fmt.Sprintf(`
type Convergen interface {
	// :recv %s
	Convert(*User) *UserModel
}`, receiverVar))
}

// RecvTypeSpecScenario creates a scenario for testing :recv annotation with type specification.
func RecvTypeSpecScenario(receiverType string) InlineScenario {
	return NewInlineScenario(
		fmt.Sprintf("RecvTypeSpec_%s", strings.ReplaceAll(receiverType, "*", "Ptr")),
		"Test :recv annotation with type specification",
	).WithTypes(`
type User struct {
	Name string
	Age  int
}

type UserModel struct {
	Name string
	Age  int
}

type UserService struct{}
`).WithInterface(fmt.Sprintf(`
type Convergen interface {
	// :recv %s
	Convert(*User) *UserModel
}`, receiverType))
}

// MapAnnotationScenario creates a scenario for testing :map annotation with basic field mapping.
func MapAnnotationScenario(srcField, dstField string) InlineScenario {
	return NewInlineScenario(
		fmt.Sprintf("MapAnnotation_%s_to_%s", srcField, dstField),
		"Test :map annotation for explicit field mapping",
	).WithTypes(`
type User struct {
	FirstName string
	LastName  string
	UserName  string
}

type UserModel struct {
	FullName string
	Name     string
}`).WithInterface(fmt.Sprintf(`
type Convergen interface {
	// :map %s %s
	Convert(*User) *UserModel
}`, srcField, dstField))
}

// MapTemplatedArgumentsScenario creates a scenario for testing :map with templated arguments ($1, $2).
func MapTemplatedArgumentsScenario() InlineScenario {
	return NewInlineScenario(
		"MapTemplatedArguments",
		"Test :map annotation with templated arguments ($1, $2)",
	).WithTypes(`
type User struct {
	Name string
	Age  int
}

type UserModel struct {
	FormattedInfo string
}`).WithInterface(`
type Convergen interface {
	// :map $1 FormattedInfo
	Convert(user *User, additionalInfo string) *UserModel
}`)
}

// MapMethodChainScenario creates a scenario for testing :map with method chains and getters.
func MapMethodChainScenario() InlineScenario {
	return NewInlineScenario(
		"MapMethodChain",
		"Test :map annotation with method chains and getters",
	).WithTypes(`
type Profile struct {
	Name string
}

func (p *Profile) GetDisplayName() string {
	return "Display: " + p.Name
}

type User struct {
	Profile *Profile
}

type UserModel struct {
	DisplayName string
}`).WithInterface(`
type Convergen interface {
	// :map Profile.GetDisplayName() DisplayName
	Convert(*User) *UserModel
}`)
}

// MapNestedFieldScenario creates a scenario for testing :map with nested field paths.
func MapNestedFieldScenario() InlineScenario {
	return NewInlineScenario(
		"MapNestedField",
		"Test :map annotation with nested field access",
	).WithTypes(`
type Address struct {
	Street string
	City   string
}

type User struct {
	Name    string
	Address Address
}

type UserModel struct {
	Name       string
	UserStreet string
	UserCity   string
}`).WithInterface(`
type Convergen interface {
	// :map Address.Street UserStreet
	// :map Address.City UserCity
	Convert(*User) *UserModel
}`)
}

// Error Scenario Builders.

// InvalidSyntaxScenario creates a scenario with invalid Go syntax.
func InvalidSyntaxScenario() InlineScenario {
	return NewInlineScenario(
		"InvalidSyntax",
		"Test error handling for invalid Go syntax",
	).WithTypes(`
type User struct {
	Name string
	Age  int
}

// Invalid interface syntax - missing method body
type UserModel interface {
	GetName() string
	InvalidMethod(
}

type UserData struct {
	Name string
	Age  int
}`).WithInterface(`
type Convergen interface {
	Convert(*User) *UserData
}`)
}

// InvalidAnnotationScenario creates a scenario with invalid annotation.
func InvalidAnnotationScenario() InlineScenario {
	return NewInlineScenario(
		"InvalidAnnotation",
		"Test error handling for invalid annotation syntax",
	).WithTypes(`
type User struct {
	Name string
}

type UserModel struct {
	Name string
}`).WithInterface(`
type Convergen interface {
	// :invalid_annotation_name
	Convert(*User) *UserModel
}`)
}

// TypeMismatchScenario creates a scenario with incompatible types.
func TypeMismatchScenario() InlineScenario {
	return NewInlineScenario(
		"TypeMismatch",
		"Test error handling for incompatible type conversion",
	).WithTypes(`
type User struct {
	ID string // String type
}

type UserModel struct {
	ID int64 // Incompatible int64 type
}`).WithInterface(`
type Convergen interface {
	Convert(*User) *UserModel
}`)
}

// MissingConverterFunctionScenario creates a scenario with missing converter function.
func MissingConverterFunctionScenario() InlineScenario {
	return NewInlineScenario(
		"MissingConverter",
		"Test error handling for missing converter function",
	).WithTypes(`
type User struct {
	Password string
}

type UserModel struct {
	HashedPassword string
}`).WithInterface(`
type Convergen interface {
	// :conv NonExistentFunction Password HashedPassword
	Convert(*User) *UserModel
}`)
}

// InvalidMapAnnotationScenario creates a scenario with invalid :map annotation.
func InvalidMapAnnotationScenario() InlineScenario {
	return NewInlineScenario(
		"InvalidMapAnnotation",
		"Test error handling for invalid :map annotation",
	).WithTypes(`
type User struct {
	Name string
}

type UserModel struct {
	Name string
}`).WithInterface(`
type Convergen interface {
	// :map NonExistentField Name
	Convert(*User) *UserModel
}`)
}

// CircularDependencyScenario creates a scenario with circular type dependency.
func CircularDependencyScenario() InlineScenario {
	return NewInlineScenario(
		"CircularDependency",
		"Test error handling for circular type dependencies",
	).WithTypes(`
type UserA struct {
	Friend *UserB
}

type UserB struct {
	Friend *UserA
}

type ModelA struct {
	Friend *ModelB
}

type ModelB struct {
	Friend *ModelA
}`).WithInterface(`
type Convergen interface {
	ConvertA(*UserA) *ModelA
	ConvertB(*UserB) *ModelB
}`)
}

// EmptyInterfaceScenario creates a scenario with empty interface.
func EmptyInterfaceScenario() InlineScenario {
	return NewInlineScenario(
		"EmptyInterface",
		"Test error handling for empty converter interface",
	).WithTypes(`
type User struct {
	Name string
}

type UserModel struct {
	Name string
}`).WithInterface(`
type Convergen interface {
}`)
}

// InvalidReturnTypeScenario creates a scenario with invalid return type.
func InvalidReturnTypeScenario() InlineScenario {
	return NewInlineScenario(
		"InvalidReturnType",
		"Test error handling for invalid return type in interface",
	).WithTypes(`
type User struct {
	Name string
}`).WithInterface(`
type Convergen interface {
	Convert(*User) NonExistentType
}`)
}

// Generics Scenario Builders (TASK-001-009 implementation)

// BasicGenericInterfaceScenario creates a scenario for testing basic generic interfaces.
func BasicGenericInterfaceScenario() InlineScenario {
	return NewInlineScenario(
		"BasicGenericInterface",
		"Test basic generic interface parsing and recognition",
	).WithTypes(`
type User struct {
	ID   uint64
	Name string
}

type UserModel struct {
	ID   uint64
	Name string
}`).WithInterface(`
type Convergen[T any] interface {
	Convert(*User) *UserModel
}`)
}

// GenericWithConstraintsScenario creates a scenario for testing generic interfaces with constraints.
func GenericWithConstraintsScenario() InlineScenario {
	return NewInlineScenario(
		"GenericWithConstraints",
		"Test generic interface with comparable constraint",
	).WithTypes(`
type User struct {
	ID   string
	Name string
}

type UserModel struct {
	ID   string
	Name string
}`).WithInterface(`
type Convergen[T comparable] interface {
	Convert(*User) *UserModel
}`)
}

// MultipleTypeParametersScenario creates a scenario for testing multiple type parameters.
func MultipleTypeParametersScenario() InlineScenario {
	return NewInlineScenario(
		"MultipleTypeParameters",
		"Test generic interface with multiple type parameters",
	).WithTypes(`
type User struct {
	ID   uint64
	Name string
}

type UserModel struct {
	ID   uint64
	Name string
}`).WithInterface(`
type Convergen[S any, D any] interface {
	Map(*User) *UserModel
}`)
}

// GenericWithAnnotationsScenario creates a scenario for testing generic interfaces with annotations.
func GenericWithAnnotationsScenario() InlineScenario {
	return NewInlineScenario(
		"GenericWithAnnotations",
		"Test generic interface with style annotation",
	).WithTypes(`
type User struct {
	ID   uint64
	Name string
}

type UserModel struct {
	ID   uint64
	Name string
}`).WithInterface(`
type Convergen[T any] interface {
	// :style return
	Convert(*User) *UserModel
}`)
}

// GenericTypeInstantiationScenario creates a scenario for testing type instantiation.
func GenericTypeInstantiationScenario() InlineScenario {
	return NewInlineScenario(
		"GenericTypeInstantiation",
		"Test generic type instantiation with concrete types",
	).WithTypes(`
type User struct {
	ID   uint64
	Name string
}

type UserModel struct {
	ID   uint64
	Name string
}

type StringData struct {
	Value string
}

type IntData struct {
	Value int
}`).WithInterface(`
type Convergen[T any, U any] interface {
	Transform(*User) *UserModel
}`)
}

// GenericMethodProcessingScenario creates a scenario for testing generic method processing.
func GenericMethodProcessingScenario() InlineScenario {
	return NewInlineScenario(
		"GenericMethodProcessing",
		"Test generic method processing with type substitution",
	).WithTypes(`
type ProcessableData struct {
	Value string
	Count int
}

type ProcessedResult struct {
	Value string
	Count int
}`).WithInterface(`
type Convergen[T any] interface {
	Process(*ProcessableData) *ProcessedResult
}`)
}

// GenericFieldMappingScenario creates a scenario for testing field mapping with generics.
func GenericFieldMappingScenario() InlineScenario {
	return NewInlineScenario(
		"GenericFieldMapping",
		"Test field mapping between generic and concrete types",
	).WithTypes(`
type SourceData struct {
	Data string
	Meta string
}

type DestData struct {
	Data string
	Meta string
}`).WithInterface(`
type Convergen[T any] interface {
	MapFields(*SourceData) *DestData
}`)
}

// UnionConstraintParsingScenario creates a scenario for testing union constraint parsing.
func UnionConstraintParsingScenario() InlineScenario {
	return NewInlineScenario(
		"UnionConstraintParsing",
		"Test union constraint parsing (even if generation is limited)",
	).WithTypes(`
type StringValue struct {
	Value string
}

type IntValue struct {
	Value int
}`).WithInterface(`
type Convergen[T string | int] interface {
	// Union constraints are parsed but limited in generation
	ProcessUnion(*StringValue) *IntValue
}`)
}

// NestedGenericTypesScenario creates a scenario for testing nested generic types.
func NestedGenericTypesScenario() InlineScenario {
	return NewInlineScenario(
		"NestedGenericTypes",
		"Test nested generic type structures",
	).WithTypes(`
type Wrapper[T any] struct {
	Inner T
}

type Container[T any] struct {
	Data Wrapper[T]
}

type ConcreteWrapper struct {
	Inner string
}

type ConcreteContainer struct {
	Data ConcreteWrapper
}`).WithInterface(`
type Convergen[T any] interface {
	MapNested(Container[T]) ConcreteContainer
}`)
}

// GenericWithInterfaceConstraintsScenario creates a scenario for testing interface constraints.
func GenericWithInterfaceConstraintsScenario() InlineScenario {
	return NewInlineScenario(
		"GenericWithInterfaceConstraints",
		"Test generic interfaces with interface constraints",
	).WithTypes(`
type Stringer interface {
	String() string
}

type NamedEntity struct {
	Name string
}

func (n NamedEntity) String() string {
	return n.Name
}

type StringEntity struct {
	Value string
}`).WithInterface(`
type Convergen[T Stringer] interface {
	ConvertStringer(*NamedEntity) *StringEntity
}`)
}

// Generics Error Scenario Builders

// InvalidGenericSyntaxScenario creates a scenario with invalid generic syntax.
func InvalidGenericSyntaxScenario() InlineScenario {
	return NewInlineScenario(
		"InvalidGenericSyntax",
		"Test error handling for invalid generic interface syntax",
	).WithTypes(`
type User struct {
	Name string
}

type UserModel struct {
	Name string
}`).WithInterface(`
type Convergen[T any] interface {
	// Should handle gracefully even with complex constraints
	Convert(*User) *UserModel
}`)
}

// UnsupportedConstraintScenario creates a scenario with unsupported constraint combinations.
func UnsupportedConstraintScenario() InlineScenario {
	return NewInlineScenario(
		"UnsupportedConstraint",
		"Test error handling for unsupported constraint combinations",
	).WithTypes(`
type User struct {
	Name string
}

type UserModel struct {
	Name string
}`).WithInterface(`
type Convergen[T interface{ ~string | ~int; comparable; any }] interface {
	// Complex constraints - should handle gracefully
	Convert(*User) *UserModel
}`)
}

// CircularConstraintScenario creates a scenario with circular constraint dependencies.
func CircularConstraintScenario() InlineScenario {
	return NewInlineScenario(
		"CircularConstraint",
		"Test error handling for circular constraint dependencies",
	).WithTypes(`
type User struct {
	Name string
}

type UserModel struct {
	Name string
}

type CircularConstraint[T any] interface {
	Method() T
}`).WithInterface(`
type Convergen[T CircularConstraint[T], U CircularConstraint[U]] interface {
	// Circular constraints - should handle gracefully
	Convert(*User) *UserModel
}`)
}
