package helpers

import "fmt"

// NewInlineScenario creates a new inline scenario
func NewInlineScenario(name, description string) InlineScenario {
	return InlineScenario{
		Name:        name,
		Description: description,
	}
}

// WithTypes adds type definitions to the scenario
func (is InlineScenario) WithTypes(sourceTypes string) InlineScenario {
	is.SourceTypes = sourceTypes
	return is
}

// WithInterface adds an interface definition to the scenario
func (is InlineScenario) WithInterface(interfaceDef string) InlineScenario {
	is.Interface = interfaceDef
	return is
}

// WithImports adds imports to the scenario
func (is InlineScenario) WithImports(imports ...string) InlineScenario {
	is.Imports = imports
	return is
}

// Common type definitions for reuse

// SimpleUserTypes creates basic User and UserModel types
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

// SimpleConverter creates a basic converter interface
func SimpleConverter(methodSig string) string {
	return fmt.Sprintf(`
type Convergen interface {
	%s
}`, methodSig)
}

// BehaviorTest builders

// NewBehaviorTest creates a new behavior test
func NewBehaviorTest(name, description, testFunc string) BehaviorTest {
	return BehaviorTest{
		Name:        name,
		Description: description,
		TestFunc:    testFunc,
	}
}

// WithInput sets the input value for a behavior test
func (bt BehaviorTest) WithInput(input interface{}) BehaviorTest {
	bt.Input = input
	return bt
}

// WithExpected sets the expected output for a behavior test
func (bt BehaviorTest) WithExpected(expected interface{}) BehaviorTest {
	bt.Expected = expected
	return bt
}

// ExpectError marks a behavior test as expected to fail
func (bt BehaviorTest) ExpectError() BehaviorTest {
	bt.ShouldError = true
	return bt
}

// Annotation-specific scenario builders

// StyleAnnotationScenario creates a scenario for testing :style annotation
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

// MatchAnnotationScenario creates a scenario for testing :match annotation
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

// ConvertAnnotationScenario creates a scenario for testing :conv annotation
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

// LiteralAnnotationScenario creates a scenario for testing :literal annotation
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

// SkipAnnotationScenario creates a scenario for testing :skip annotation
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