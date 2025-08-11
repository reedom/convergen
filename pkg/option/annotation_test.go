package option

import (
	"testing"

	"github.com/reedom/convergen/v9/pkg/generator/model"
)

func TestValidOpsIntf_NoStructLiteral(t *testing.T) {
	// Test that no-struct-literal is a valid interface-level annotation
	if _, exists := ValidOpsIntf["no-struct-literal"]; !exists {
		t.Error("no-struct-literal should be a valid interface-level annotation")
	}
}

func TestValidOpsMethod_NoStructLiteral(t *testing.T) {
	// Test that no-struct-literal is a valid method-level annotation
	if _, exists := ValidOpsMethod["no-struct-literal"]; !exists {
		t.Error("no-struct-literal should be a valid method-level annotation")
	}
}

func TestOptions_DefaultValues(t *testing.T) {
	// Test that new options have proper default values
	opts := NewOptions()

	// Test that NewOptions sets expected defaults
	if opts.Style != model.DstVarReturn {
		t.Errorf("NewOptions() should set Style to DstVarReturn, got %v", opts.Style)
	}

	if opts.Rule != model.MatchRuleName {
		t.Errorf("NewOptions() should set Rule to MatchRuleName, got %v", opts.Rule)
	}

	if !opts.ExactCase {
		t.Error("NewOptions() should set ExactCase to true")
	}
}

func TestOptions_ShouldSkip_Unchanged(t *testing.T) {
	// Test that existing functionality is not affected
	opts := NewOptions()

	// Should not skip fields that don't match any pattern
	if opts.ShouldSkip("TestField") {
		t.Error("ShouldSkip should return false for fields with no skip patterns")
	}

	// Add a skip pattern and test it works
	matcher, err := NewPatternMatcher("SkipThis", true)
	if err != nil {
		t.Fatalf("Failed to create pattern matcher: %v", err)
	}
	opts.SkipFields = append(opts.SkipFields, matcher)

	if !opts.ShouldSkip("SkipThis") {
		t.Error("ShouldSkip should return true for fields matching skip patterns")
	}

	if opts.ShouldSkip("KeepThis") {
		t.Error("ShouldSkip should return false for fields not matching skip patterns")
	}
}

func TestOptions_CompareFieldName_Unchanged(t *testing.T) {
	// Test that existing functionality is not affected
	opts := NewOptions()

	// Test exact case (default)
	if !opts.CompareFieldName("Test", "Test") {
		t.Error("CompareFieldName should return true for exact matches with ExactCase=true")
	}

	if opts.CompareFieldName("Test", "test") {
		t.Error("CompareFieldName should return false for case mismatches with ExactCase=true")
	}

	// Test case insensitive
	opts.ExactCase = false
	if !opts.CompareFieldName("Test", "test") {
		t.Error("CompareFieldName should return true for case mismatches with ExactCase=false")
	}

	if !opts.CompareFieldName("TEST", "test") {
		t.Error("CompareFieldName should return true for case mismatches with ExactCase=false")
	}
}
