// Package helpers provides testing utilities for the Convergen behavior-driven testing framework.
package helpers

import "fmt"

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
