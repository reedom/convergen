package domain

import (
	"testing"
	"time"
)

func TestInterfaceOptions_NoStructLiteralFields(t *testing.T) {
	opts := &InterfaceOptions{
		Style:           StyleCamelCase,
		NoStructLiteral: true,
		ForceStructLit:  false,
	}

	if !opts.NoStructLiteral {
		t.Error("NoStructLiteral field should be accessible and settable")
	}

	if opts.ForceStructLit {
		t.Error("ForceStructLit field should default to false")
	}

	// Test setting ForceStructLit
	opts.ForceStructLit = true
	if !opts.ForceStructLit {
		t.Error("ForceStructLit field should be settable to true")
	}
}

func TestMethodOptions_NoStructLiteralFields(t *testing.T) {
	opts := &MethodOptions{
		Style:            StyleCamelCase,
		NoStructLiteral:  true,
		ForceStructLit:   false,
		ConcurrencyLevel: 1,
		TimeoutDuration:  time.Second,
	}

	if !opts.NoStructLiteral {
		t.Error("NoStructLiteral field should be accessible and settable")
	}

	if opts.ForceStructLit {
		t.Error("ForceStructLit field should default to false")
	}

	// Test setting ForceStructLit
	opts.ForceStructLit = true
	if !opts.ForceStructLit {
		t.Error("ForceStructLit field should be settable to true")
	}

	// Ensure existing fields still work
	if opts.ConcurrencyLevel != 1 {
		t.Error("ConcurrencyLevel should be preserved")
	}

	if opts.TimeoutDuration != time.Second {
		t.Error("TimeoutDuration should be preserved")
	}
}

func TestVariableStyle_String(t *testing.T) {
	// Test that existing enum functionality is preserved
	tests := []struct {
		style    VariableStyle
		expected string
	}{
		{StyleCamelCase, "camelCase"},
		{StyleSnakeCase, "snake_case"},
		{StylePascalCase, "PascalCase"},
	}

	for _, test := range tests {
		if result := test.style.String(); result != test.expected {
			t.Errorf("VariableStyle.String() = %q, want %q", result, test.expected)
		}
	}
}

func TestMatchRule_String(t *testing.T) {
	// Test that existing enum functionality is preserved
	tests := []struct {
		rule     MatchRule
		expected string
	}{
		{MatchByName, "name"},
		{MatchByType, "type"},
		{MatchByTag, "tag"},
	}

	for _, test := range tests {
		if result := test.rule.String(); result != test.expected {
			t.Errorf("MatchRule.String() = %q, want %q", result, test.expected)
		}
	}
}

func TestConversionType_String(t *testing.T) {
	// Test that existing enum functionality is preserved
	tests := []struct {
		convType ConversionType
		expected string
	}{
		{ConversionDirect, "direct"},
		{ConversionCast, "cast"},
		{ConversionMethod, "method"},
		{ConversionCustom, "custom"},
		{ConversionLiteral, "literal"},
	}

	for _, test := range tests {
		if result := test.convType.String(); result != test.expected {
			t.Errorf("ConversionType.String() = %q, want %q", result, test.expected)
		}
	}
}

func TestErrorHandlingMethod_String(t *testing.T) {
	// Test that existing enum functionality is preserved
	tests := []struct {
		method   ErrorHandlingMethod
		expected string
	}{
		{ErrorHandlingNone, "none"},
		{ErrorHandlingReturn, "return"},
		{ErrorHandlingPanic, "panic"},
		{ErrorHandlingLog, "log"},
	}

	for _, test := range tests {
		if result := test.method.String(); result != test.expected {
			t.Errorf("ErrorHandlingMethod.String() = %q, want %q", result, test.expected)
		}
	}
}

func TestChannelDirection_String(t *testing.T) {
	// Test that existing enum functionality is preserved
	tests := []struct {
		direction ChannelDirection
		expected  string
	}{
		{ChannelBidirectional, "bidirectional"},
		{ChannelSendOnly, "send-only"},
		{ChannelReceiveOnly, "receive-only"},
	}

	for _, test := range tests {
		if result := test.direction.String(); result != test.expected {
			t.Errorf("ChannelDirection.String() = %q, want %q", result, test.expected)
		}
	}
}
