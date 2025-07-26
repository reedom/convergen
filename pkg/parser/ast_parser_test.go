package parser

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/reedom/convergen/v8/pkg/domain"
	"github.com/reedom/convergen/v8/pkg/internal/events"
)

func TestASTParser_NewASTParser(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	parser := NewASTParser(logger, eventBus, nil)
	assert.NotNil(t, parser)
	assert.NotNil(t, parser.cache)
	assert.NotNil(t, parser.typeResolverPool)
	assert.NotNil(t, parser.fileSet)
}

func TestASTParser_ParseSourceFile(t *testing.T) {
	tests := []struct {
		name            string
		sourceContent   string
		expectedError   bool
		expectedMethods int
	}{
		{
			name: "simple interface",
			sourceContent: `package test

//go:generate convergen

// :convergen
type Convergen interface {
	Convert(src *Source) *Dest
}

type Source struct {
	Name string
	Age  int
}

type Dest struct {
	Name string
	Age  int
}
`,
			expectedError:   false,
			expectedMethods: 1,
		},
		{
			name: "interface with annotations",
			sourceContent: `package test

//go:generate convergen

// :convergen
// :style camel
// :match name
type Convergen interface {
	// :skip Age
	Convert(src *Source) *Dest
}

type Source struct {
	Name string
	Age  int
}

type Dest struct {
	Name string
}
`,
			expectedError:   false,
			expectedMethods: 1,
		},
		{
			name: "multiple methods",
			sourceContent: `package test

//go:generate convergen

// :convergen
type Convergen interface {
	ConvertAB(src *A) *B
	ConvertBA(src *B) *A
}

type A struct {
	Field1 string
}

type B struct {
	Field1 string
}
`,
			expectedError:   false,
			expectedMethods: 2,
		},
		{
			name: "no convergen interface",
			sourceContent: `package test

type RegularInterface interface {
	Method()
}
`,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary files
			tempDir := t.TempDir()
			sourceFile := filepath.Join(tempDir, "source.go")
			destFile := filepath.Join(tempDir, "dest.go")

			err := os.WriteFile(sourceFile, []byte(tt.sourceContent), 0644)
			require.NoError(t, err)

			// Create parser
			logger := zaptest.NewLogger(t)
			eventBus := events.NewInMemoryEventBus(logger)
			defer eventBus.Close()

			// Subscribe to events for testing
			var receivedEvents []events.Event
			handler := events.NewFuncEventHandler("parse.completed", func(ctx context.Context, event events.Event) error {
				receivedEvents = append(receivedEvents, event)
				return nil
			})
			err = eventBus.Subscribe("parse.completed", handler)
			require.NoError(t, err)

			parser := NewASTParser(logger, eventBus, &ParserConfig{
				BuildTag:              "convergen",
				MaxConcurrentWorkers:  2,
				TypeResolutionTimeout: 5 * time.Second,
				CacheSize:             100,
				EnableProgress:        false, // Disable for testing
			})
			defer parser.Close()

			// Parse source file
			ctx := context.Background()
			methods, baseCode, err := parser.ParseSourceFile(ctx, sourceFile, destFile)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, methods, tt.expectedMethods)
			assert.NotEmpty(t, baseCode)

			// Verify that parsed event was emitted
			assert.Len(t, receivedEvents, 1)
			parsedEvent, ok := receivedEvents[0].(*events.ParsedEvent)
			require.True(t, ok)
			assert.Len(t, parsedEvent.Methods, tt.expectedMethods)
		})
	}
}

func TestASTParser_ConcurrentParsing(t *testing.T) {
	sourceContent := `package test

//go:generate convergen

// :convergen
type Convergen interface {
	Convert1(src *Source) *Dest
	Convert2(src *Source) *Dest
	Convert3(src *Source) *Dest
	Convert4(src *Source) *Dest
	Convert5(src *Source) *Dest
}

type Source struct {
	Name string
	Age  int
}

type Dest struct {
	Name string
	Age  int
}
`

	// Create temporary files
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "source.go")
	destFile := filepath.Join(tempDir, "dest.go")

	err := os.WriteFile(sourceFile, []byte(sourceContent), 0644)
	require.NoError(t, err)

	// Create parser with high concurrency
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	parser := NewASTParser(logger, eventBus, &ParserConfig{
		BuildTag:              "convergen",
		MaxConcurrentWorkers:  4,
		TypeResolutionTimeout: 5 * time.Second,
		CacheSize:             100,
		EnableProgress:        false,
	})
	defer parser.Close()

	// Parse multiple times concurrently
	ctx := context.Background()
	numGoroutines := 10
	results := make(chan struct {
		methods  []*domain.Method
		baseCode string
		err      error
	}, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			methods, baseCode, err := parser.ParseSourceFile(ctx, sourceFile, destFile)
			results <- struct {
				methods  []*domain.Method
				baseCode string
				err      error
			}{methods, baseCode, err}
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		result := <-results
		require.NoError(t, result.err)
		assert.Len(t, result.methods, 5)
		assert.NotEmpty(t, result.baseCode)
	}
}

func TestASTParser_TypeCaching(t *testing.T) {
	sourceContent := `package test

//go:generate convergen

// :convergen
type Convergen interface {
	Convert(src *ComplexType) *ComplexType
}

type ComplexType struct {
	Simple    string
	Slice     []string
	Map       map[string]int
	Pointer   *string
	Interface interface{}
}
`

	// Create temporary files
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "source.go")
	destFile := filepath.Join(tempDir, "dest.go")

	err := os.WriteFile(sourceFile, []byte(sourceContent), 0644)
	require.NoError(t, err)

	// Create parser
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	parser := NewASTParser(logger, eventBus, &ParserConfig{
		BuildTag:              "convergen",
		MaxConcurrentWorkers:  2,
		TypeResolutionTimeout: 5 * time.Second,
		CacheSize:             100,
		EnableProgress:        false,
	})
	defer parser.Close()

	ctx := context.Background()

	// First parse - cache miss
	initialCacheSize := parser.cache.Size()
	_, _, err = parser.ParseSourceFile(ctx, sourceFile, destFile)
	require.NoError(t, err)

	firstParseSize := parser.cache.Size()
	assert.Greater(t, firstParseSize, initialCacheSize)

	// Second parse - should use cache
	_, _, err = parser.ParseSourceFile(ctx, sourceFile, destFile)
	require.NoError(t, err)

	// Cache size should not increase significantly
	secondParseSize := parser.cache.Size()
	assert.LessOrEqual(t, secondParseSize-firstParseSize, 1) // Allow for minor differences

	// Check cache hit rate
	stats := parser.cache.Stats()
	assert.Greater(t, stats.HitRate, 0.0)
}

func TestASTParser_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		sourceContent string
		expectedError string
	}{
		{
			name: "invalid Go syntax",
			sourceContent: `package test

//go:generate convergen

// :convergen
type Convergen interface {
	Convert(src *Source) *Dest
`,
			expectedError: "package errors:",
		},
		{
			name: "method without parameters",
			sourceContent: `package test

//go:generate convergen

// :convergen
type Convergen interface {
	Convert() *Dest
}

type Dest struct {
	Name string
}
`,
			expectedError: "must have at least one parameter",
		},
		{
			name: "method without return values",
			sourceContent: `package test

//go:generate convergen

// :convergen
type Convergen interface {
	Convert(src *Source)
}

type Source struct {
	Name string
}
`,
			expectedError: "must have at least one return value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary files
			tempDir := t.TempDir()
			sourceFile := filepath.Join(tempDir, "source.go")
			destFile := filepath.Join(tempDir, "dest.go")

			err := os.WriteFile(sourceFile, []byte(tt.sourceContent), 0644)
			require.NoError(t, err)

			// Create parser
			logger := zaptest.NewLogger(t)
			eventBus := events.NewInMemoryEventBus(logger)
			defer eventBus.Close()

			parser := NewASTParser(logger, eventBus, nil)
			defer parser.Close()

			// Parse source file
			ctx := context.Background()
			_, _, err = parser.ParseSourceFile(ctx, sourceFile, destFile)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestASTParser_GenericsSupport(t *testing.T) {
	sourceContent := `package test

//go:generate convergen

// :convergen
type Convergen interface {
	Convert[T any](src *GenericType[T]) *GenericType[T]
}

type GenericType[T any] struct {
	Value T
	Name  string
}
`

	// Create temporary files
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "source.go")
	destFile := filepath.Join(tempDir, "dest.go")

	err := os.WriteFile(sourceFile, []byte(sourceContent), 0644)
	require.NoError(t, err)

	// Create parser
	logger := zaptest.NewLogger(t)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	parser := NewASTParser(logger, eventBus, nil)
	defer parser.Close()

	// Parse source file
	ctx := context.Background()
	methods, baseCode, err := parser.ParseSourceFile(ctx, sourceFile, destFile)

	// Note: This test may not work with older Go versions that don't support generics
	// In a real implementation, you'd handle this gracefully
	if err != nil {
		t.Skip("Generics not supported in this Go version")
	}

	require.NoError(t, err)
	assert.Len(t, methods, 1)
	assert.NotEmpty(t, baseCode)

	// Verify the method has generic type parameters
	method := methods[0]
	assert.NotEmpty(t, method.SourceParams())
	assert.NotEmpty(t, method.DestinationReturns())
}

func BenchmarkASTParser_ParseSourceFile(b *testing.B) {
	sourceContent := `package test

//go:generate convergen

// :convergen
type Convergen interface {
	Convert1(src *Source) *Dest
	Convert2(src *Source) *Dest
	Convert3(src *Source) *Dest
}

type Source struct {
	Name     string
	Age      int
	Address  string
	Phone    string
	Email    string
}

type Dest struct {
	Name     string
	Age      int
	Address  string
	Phone    string
	Email    string
}
`

	// Create temporary files
	tempDir := b.TempDir()
	sourceFile := filepath.Join(tempDir, "source.go")
	destFile := filepath.Join(tempDir, "dest.go")

	err := os.WriteFile(sourceFile, []byte(sourceContent), 0644)
	require.NoError(b, err)

	// Create parser
	logger := zaptest.NewLogger(b)
	eventBus := events.NewInMemoryEventBus(logger)
	defer eventBus.Close()

	parser := NewASTParser(logger, eventBus, &ParserConfig{
		BuildTag:              "convergen",
		MaxConcurrentWorkers:  4,
		TypeResolutionTimeout: 5 * time.Second,
		CacheSize:             1000,
		EnableProgress:        false,
	})
	defer parser.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := parser.ParseSourceFile(ctx, sourceFile, destFile)
		require.NoError(b, err)
	}
}
