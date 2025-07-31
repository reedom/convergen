package parser

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestParserFactory_CreateParser(t *testing.T) {
	factory := NewParserFactory(nil)

	tests := []struct {
		name     string
		strategy ParseStrategy
		wantType string
	}{
		{
			name:     "create_legacy_parser",
			strategy: StrategyLegacy,
			wantType: "*parser.LegacyParser",
		},
		{
			name:     "create_modern_parser",
			strategy: StrategyModern,
			wantType: "*parser.ModernParser",
		},
		{
			name:     "create_adaptive_parser",
			strategy: StrategyAuto,
			wantType: "*parser.AdaptiveParser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := factory.CreateParser(tt.strategy)
			if err != nil {
				t.Errorf("CreateParser() error = %v", err)
				return
			}

			if parser == nil {
				t.Error("CreateParser() returned nil parser")
				return
			}

			// Test that parser implements the interface
			if parser.GetStrategy() != tt.strategy && tt.strategy != StrategyAuto {
				t.Errorf("GetStrategy() = %v, want %v", parser.GetStrategy(), tt.strategy)
			}

			// Test configuration
			config := parser.GetConfig()
			if config == nil {
				t.Error("GetConfig() returned nil")
			}

			// Test metrics
			metrics := parser.GetMetrics()
			if metrics == nil {
				t.Error("GetMetrics() returned nil")
			}

			// Test cleanup
			err = parser.Close()
			if err != nil {
				t.Errorf("Close() error = %v", err)
			}
		})
	}
}

func TestGetRecommendedStrategy(t *testing.T) {
	tests := []struct {
		name                       string
		fileCount                  int
		averageMethodsPerInterface int
		totalMethods               int
		want                       ParseStrategy
	}{
		{
			name:                       "simple_case_legacy",
			fileCount:                  1,
			averageMethodsPerInterface: 5,
			totalMethods:               5,
			want:                       StrategyLegacy,
		},
		{
			name:                       "complex_case_modern",
			fileCount:                  5,
			averageMethodsPerInterface: 15,
			totalMethods:               75,
			want:                       StrategyModern,
		},
		{
			name:                       "many_methods_modern",
			fileCount:                  2,
			averageMethodsPerInterface: 25,
			totalMethods:               50,
			want:                       StrategyModern,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRecommendedStrategy(tt.fileCount, tt.averageMethodsPerInterface, tt.totalMethods)
			if got != tt.want {
				t.Errorf("GetRecommendedStrategy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStrategyName(t *testing.T) {
	tests := []struct {
		strategy ParseStrategy
		want     string
	}{
		{StrategyLegacy, "Legacy Sequential Parser"},
		{StrategyModern, "Modern Concurrent Parser"},
		{StrategyAuto, "Adaptive Parser"},
		{ParseStrategy(999), "Unknown Strategy"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := GetStrategyName(tt.strategy)
			if got != tt.want {
				t.Errorf("GetStrategyName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLegacyParser_Basic(t *testing.T) {
	config := &ParserConfig{
		BuildTag:                "test",
		MaxConcurrentWorkers:    2,
		TypeResolutionTimeout:   5 * time.Second,
		EnableConcurrentLoading: false,
		EnableMethodConcurrency: false,
	}

	parser := NewLegacyParser(config)

	// Test basic functionality
	if parser.GetStrategy() != StrategyLegacy {
		t.Errorf("GetStrategy() = %v, want %v", parser.GetStrategy(), StrategyLegacy)
	}

	// Test validation with non-existent file
	ctx := context.Background()

	errors, warnings, err := parser.Validate(ctx, "nonexistent.go")
	if err == nil {
		t.Error("Validate() expected error for non-existent file")
	}

	if len(errors) == 0 {
		t.Error("Validate() expected errors for non-existent file")
	}

	// Test that warnings can be empty
	_ = warnings

	// Test cleanup
	err = parser.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestModernParser_Basic(t *testing.T) {
	config := &ParserConfig{
		BuildTag:                "test",
		MaxConcurrentWorkers:    4,
		TypeResolutionTimeout:   5 * time.Second,
		EnableConcurrentLoading: true,
		EnableMethodConcurrency: true,
	}

	parser := NewModernParser(config)

	// Test basic functionality
	if parser.GetStrategy() != StrategyModern {
		t.Errorf("GetStrategy() = %v, want %v", parser.GetStrategy(), StrategyModern)
	}

	// Ensure concurrent features are enabled
	actualConfig := parser.GetConfig()
	if !actualConfig.EnableConcurrentLoading {
		t.Error("ModernParser should enable concurrent loading")
	}

	if !actualConfig.EnableMethodConcurrency {
		t.Error("ModernParser should enable method concurrency")
	}

	// Test cleanup
	err := parser.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestAdaptiveParser_Basic(t *testing.T) {
	parser := NewAdaptiveParser(nil)

	// Test basic functionality
	if parser.GetStrategy() != StrategyAuto {
		t.Errorf("GetStrategy() = %v, want %v", parser.GetStrategy(), StrategyAuto)
	}

	// Test strategy determination
	tempFile, err := os.CreateTemp("", "test*.go")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Write small content
	_, err = tempFile.WriteString("package test\n")
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	// Test strategy determination for small file
	strategy := parser.determineStrategy(tempFile.Name(), "")
	if strategy != StrategyLegacy {
		t.Errorf("determineStrategy() for small file = %v, want %v", strategy, StrategyLegacy)
	}

	// Test cleanup
	err = parser.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestParseResult_ErrorMethods(t *testing.T) {
	// Test ParseError.Error() method
	parseError := ParseError{
		Code:     "TEST_ERROR",
		Message:  "test message",
		Location: "test.go:10",
	}

	expected := "test.go:10: test message"
	if parseError.Error() != expected {
		t.Errorf("ParseError.Error() = %v, want %v", parseError.Error(), expected)
	}

	// Test ParseError without location
	parseErrorNoLoc := ParseError{
		Code:    "TEST_ERROR",
		Message: "test message",
	}

	if parseErrorNoLoc.Error() != "test message" {
		t.Errorf("ParseError.Error() without location = %v, want %v", parseErrorNoLoc.Error(), "test message")
	}

	// Test ParseWarning.Error() method
	parseWarning := ParseWarning{
		Code:     "TEST_WARNING",
		Message:  "test warning",
		Location: "test.go:5",
	}

	expectedWarning := "test.go:5: test warning"
	if parseWarning.Error() != expectedWarning {
		t.Errorf("ParseWarning.Error() = %v, want %v", parseWarning.Error(), expectedWarning)
	}
}

func TestParserFactory_GetSupportedStrategies(t *testing.T) {
	factory := NewParserFactory(nil)
	strategies := factory.GetSupportedStrategies()

	expected := []ParseStrategy{StrategyLegacy, StrategyModern, StrategyAuto}
	if len(strategies) != len(expected) {
		t.Errorf("GetSupportedStrategies() length = %v, want %v", len(strategies), len(expected))
	}

	for i, strategy := range expected {
		if strategies[i] != strategy {
			t.Errorf("GetSupportedStrategies()[%d] = %v, want %v", i, strategies[i], strategy)
		}
	}
}
