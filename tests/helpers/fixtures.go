package helpers

import (
	"fmt"
	"path/filepath"
)

// Common fixture paths for use in test scenarios
const (
	FixtureBasePath = "fixtures/usecase"
)

// FixturePath constructs a path to a test fixture
func FixturePath(usecaseName, filename string) string {
	return filepath.Join(FixtureBasePath, usecaseName, filename)
}

// SourceFixture returns the path to a setup.go fixture
func SourceFixture(usecaseName string) string {
	return FixturePath(usecaseName, "setup.go")
}

// ExpectedFixture returns the path to a setup.gen.go fixture
func ExpectedFixture(usecaseName string) string {
	return FixturePath(usecaseName, "setup.gen.go")
}

// QuickScenario creates a scenario using standard fixture naming
func QuickScenario(name, usecaseName string) TestScenario {
	return NewScenario(
		name,
		SourceFixture(usecaseName),
		ExpectedFixture(usecaseName),
	)
}

// Common assertion patterns for generated code

// AssertFunction checks for the presence of a function with the given name
func AssertFunction(funcName string) CodeAssertion {
	return Contains(fmt.Sprintf("func %s", funcName))
}

// AssertNoComments checks that generated code doesn't contain comment markers
func AssertNoComments(pattern string) CodeAssertion {
	return NotContains(fmt.Sprintf("// %s", pattern))
}

// AssertImport checks for specific import statements
func AssertImport(importPath string) CodeAssertion {
	return Contains(fmt.Sprintf("\"%s\"", importPath))
}

// AssertStructInit checks for struct initialization patterns
func AssertStructInit(structName string) CodeAssertion {
	return Contains(fmt.Sprintf("= &%s{}", structName))
}