# Design: Practical Scenario Testing Framework

## Architecture Overview

A simple, Go-idiomatic testing framework built on enhanced table-driven tests with structured scenario definitions and improved assertions.

## Core Structure

### Test Scenario Definition

```go
type TestScenario struct {
    Name        string
    Description string
    Category    string
    
    // Input
    SourceFile     string
    ExpectedFile   string
    
    // Expectations
    ShouldSucceed  bool
    ExpectedError  string
    CodeChecks     []CodeAssertion
    
    // Test metadata
    Skip           bool
    SkipReason     string
}

type CodeAssertion struct {
    Type     AssertionType  // Contains, NotContains, Regex, Compiles
    Pattern  string
    Message  string
}
```

### Test Organization

```
tests/
├── scenario_test.go           # Main test runner
├── scenarios/
│   ├── annotations/           # Annotation-specific tests
│   │   ├── style_test.go
│   │   ├── match_test.go
│   │   └── ...
│   ├── edge_cases/           # Edge case scenarios
│   │   ├── circular_refs_test.go
│   │   └── complex_types_test.go
│   └── error_cases/          # Error condition tests
│       ├── invalid_syntax_test.go
│       └── type_mismatch_test.go
├── testdata/                 # Test fixtures
│   ├── annotations/
│   ├── edge_cases/
│   └── error_cases/
└── helpers/                  # Test utilities
    ├── assertions.go
    └── fixtures.go
```

### Implementation Approach

1. **Enhanced Table Tests**: Improve current table-driven pattern with structured scenario definitions
2. **Category-Based Organization**: Group tests by annotation type, edge cases, and error conditions  
3. **Helper Functions**: Common assertion and fixture management utilities
4. **Gradual Migration**: Migrate existing tests progressively without breaking current functionality

### Integration Points

- Uses existing Convergen parser/generator pipeline
- Integrates with Go testing framework (`testing.T`)
- Maintains compatibility with current test fixtures
- Supports `go test` command with standard flags

## Key Benefits

- **Simple**: Uses familiar Go testing patterns
- **Maintainable**: Clear organization and structure
- **Extensible**: Easy to add new scenarios and assertions
- **Compatible**: Works with existing tooling and CI/CD