package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/reedom/convergen/v9/pkg/generator/model"
)

func TestGenerator_canUseStructLiteral(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		function *model.Function
		expected bool
		reason   string
	}{
		{
			name: "simple function - compatible",
			function: &model.Function{
				Name:        "ConvertUser",
				DstVarStyle: model.DstVarReturn,
				Assignments: []model.Assignment{
					model.SimpleField{LHS: "dst.ID", RHS: "src.ID", Error: false},
					model.SimpleField{LHS: "dst.Name", RHS: "src.Name", Error: false},
				},
			},
			expected: true,
			reason:   "",
		},
		{
			name: "function with preprocess - incompatible",
			function: &model.Function{
				Name:        "ConvertUser",
				DstVarStyle: model.DstVarReturn,
				PreProcess: &model.Manipulator{
					Name:     "PreProcess",
					IsSrcPtr: true,
					IsDstPtr: true,
					RetError: false,
				},
				Assignments: []model.Assignment{
					model.SimpleField{LHS: "dst.ID", RHS: "src.ID", Error: false},
				},
			},
			expected: false,
			reason:   "preprocess annotation requires imperative execution before struct creation",
		},
		{
			name: "function with postprocess - incompatible",
			function: &model.Function{
				Name:        "ConvertUser",
				DstVarStyle: model.DstVarReturn,
				PostProcess: &model.Manipulator{
					Name:     "PostProcess",
					IsSrcPtr: true,
					IsDstPtr: true,
					RetError: false,
				},
				Assignments: []model.Assignment{
					model.SimpleField{LHS: "dst.ID", RHS: "src.ID", Error: false},
				},
			},
			expected: false,
			reason:   "postprocess annotation requires imperative execution after struct creation",
		},
		{
			name: "function with style arg - incompatible",
			function: &model.Function{
				Name:        "ConvertUser",
				DstVarStyle: model.DstVarArg,
				Assignments: []model.Assignment{
					model.SimpleField{LHS: "dst.ID", RHS: "src.ID", Error: false},
				},
			},
			expected: false,
			reason:   ":style arg annotation modifies passed argument, incompatible with struct literal return",
		},
		{
			name: "function with error-returning assignment - incompatible",
			function: &model.Function{
				Name:        "ConvertUser",
				DstVarStyle: model.DstVarReturn,
				Assignments: []model.Assignment{
					model.SimpleField{LHS: "dst.ID", RHS: "src.ID", Error: false},
					model.SimpleField{LHS: "dst.Value", RHS: "src.GetValue()", Error: true},
				},
			},
			expected: false,
			reason:   "assignment 2 returns error, requires imperative error handling",
		},
		{
			name: "function with nested struct - incompatible",
			function: &model.Function{
				Name:        "ConvertUser",
				DstVarStyle: model.DstVarReturn,
				Assignments: []model.Assignment{
					model.SimpleField{LHS: "dst.ID", RHS: "src.ID", Error: false},
					model.NestStruct{
						InitExpr:      "dst.Profile = &Profile{}",
						NullCheckExpr: "src.Profile",
						Contents: []model.Assignment{
							model.SimpleField{LHS: "dst.Profile.Name", RHS: "src.Profile.Name", Error: false},
						},
					},
				},
			},
			expected: false,
			reason:   "assignment 2 contains nested struct with conditional logic",
		},
		{
			name: "function with slice assignment - incompatible",
			function: &model.Function{
				Name:        "ConvertUser",
				DstVarStyle: model.DstVarReturn,
				Assignments: []model.Assignment{
					model.SimpleField{LHS: "dst.ID", RHS: "src.ID", Error: false},
					model.SliceAssignment{
						LHS: "dst.Items",
						RHS: "src.Items",
						Typ: "[]string",
					},
				},
			},
			expected: false,
			reason:   "assignment 2 contains slice operations requiring loops and conditionals",
		},
		{
			name: "function with skip fields - compatible",
			function: &model.Function{
				Name:        "ConvertUser",
				DstVarStyle: model.DstVarReturn,
				Assignments: []model.Assignment{
					model.SimpleField{LHS: "dst.ID", RHS: "src.ID", Error: false},
					model.SkipField{LHS: "dst.Internal"},
					model.NoMatchField{LHS: "dst.Unknown"},
				},
			},
			expected: true,
			reason:   "",
		},
		{
			name: "function with mixed compatible and incompatible assignments",
			function: &model.Function{
				Name:        "ConvertUser",
				DstVarStyle: model.DstVarReturn,
				Assignments: []model.Assignment{
					model.SimpleField{LHS: "dst.ID", RHS: "src.ID", Error: false},
					model.SimpleField{LHS: "dst.Name", RHS: "src.Name", Error: false},
					model.SliceLoopAssignment{
						LHS: "dst.Tags",
						RHS: "src.Tags",
						Typ: "[]Tag",
					},
				},
			},
			expected: false,
			reason:   "assignment 3 contains slice operations requiring loops and conditionals",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{}

			// Test compatibility detection
			actual := g.canUseStructLiteral(tt.function)
			assert.Equal(t, tt.expected, actual, "canUseStructLiteral result mismatch")

			// Test fallback reason if incompatible
			if !tt.expected {
				reason := g.getFallbackReason(tt.function)
				assert.Equal(t, tt.reason, reason, "getFallbackReason result mismatch")
			}
		})
	}
}

func TestGenerator_canAssignInStructLiteral(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		assignment model.Assignment
		expected   bool
	}{
		{
			name: "simple field assignment - compatible",
			assignment: model.SimpleField{
				LHS:   "dst.Name",
				RHS:   "src.Name",
				Error: false,
			},
			expected: true,
		},
		{
			name: "error-returning assignment - incompatible",
			assignment: model.SimpleField{
				LHS:   "dst.Value",
				RHS:   "src.GetValue()",
				Error: true,
			},
			expected: false,
		},
		{
			name: "skip field - compatible (excluded from struct literal)",
			assignment: model.SkipField{
				LHS: "dst.Internal",
			},
			expected: true,
		},
		{
			name: "no match field - compatible (excluded from struct literal)",
			assignment: model.NoMatchField{
				LHS: "dst.Unknown",
			},
			expected: true,
		},
		{
			name: "nested struct - incompatible",
			assignment: model.NestStruct{
				InitExpr:      "dst.Profile = &Profile{}",
				NullCheckExpr: "src.Profile",
				Contents: []model.Assignment{
					model.SimpleField{LHS: "dst.Profile.Name", RHS: "src.Profile.Name", Error: false},
				},
			},
			expected: false,
		},
		{
			name: "slice assignment - incompatible",
			assignment: model.SliceAssignment{
				LHS: "dst.Items",
				RHS: "src.Items",
				Typ: "[]string",
			},
			expected: false,
		},
		{
			name: "slice loop assignment - incompatible",
			assignment: model.SliceLoopAssignment{
				LHS: "dst.Tags",
				RHS: "src.Tags",
				Typ: "[]Tag",
			},
			expected: false,
		},
		{
			name: "slice typecast assignment - incompatible",
			assignment: model.SliceTypecastAssignment{
				LHS:  "dst.IDs",
				RHS:  "src.IDs",
				Typ:  "[]int64",
				Cast: "int64",
			},
			expected: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{}
			actual := g.canAssignInStructLiteral(tt.assignment)
			assert.Equal(t, tt.expected, actual, "canAssignInStructLiteral result mismatch")
		})
	}
}

func TestGenerator_validateStructLiteralCompatibility(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		function    *model.Function
		expectError bool
		errorMsg    string
	}{
		{
			name: "compatible function with auto output style - no error",
			function: &model.Function{
				Name:        "ConvertUser",
				DstVarStyle: model.DstVarReturn,
				OutputStyle: model.OutputStyleAuto,
				Assignments: []model.Assignment{
					model.SimpleField{LHS: "dst.ID", RHS: "src.ID", Error: false},
				},
			},
			expectError: false,
		},
		{
			name: "compatible function with struct literal forced - no error",
			function: &model.Function{
				Name:        "ConvertUser",
				DstVarStyle: model.DstVarReturn,
				OutputStyle: model.OutputStyleStructLiteral,
				Assignments: []model.Assignment{
					model.SimpleField{LHS: "dst.ID", RHS: "src.ID", Error: false},
				},
			},
			expectError: false,
		},
		{
			name: "incompatible function with traditional output style - no error",
			function: &model.Function{
				Name:        "ConvertUser",
				DstVarStyle: model.DstVarArg,
				OutputStyle: model.OutputStyleTraditional,
				Assignments: []model.Assignment{
					model.SimpleField{LHS: "dst.ID", RHS: "src.ID", Error: false},
				},
			},
			expectError: false,
		},
		{
			name: "incompatible function with struct literal forced - error expected",
			function: &model.Function{
				Name:        "ConvertUser",
				DstVarStyle: model.DstVarArg,
				OutputStyle: model.OutputStyleStructLiteral,
				Assignments: []model.Assignment{
					model.SimpleField{LHS: "dst.ID", RHS: "src.ID", Error: false},
				},
			},
			expectError: true,
			errorMsg:    "method ConvertUser: method is forced to use struct literal but has incompatible features: :style arg annotation modifies passed argument, incompatible with struct literal return",
		},
		{
			name: "function with preprocess and forced struct literal - error expected",
			function: &model.Function{
				Name:        "ConvertUser",
				DstVarStyle: model.DstVarReturn,
				OutputStyle: model.OutputStyleStructLiteral,
				PreProcess: &model.Manipulator{
					Name:     "PreProcess",
					IsSrcPtr: true,
					IsDstPtr: true,
					RetError: false,
				},
				Assignments: []model.Assignment{
					model.SimpleField{LHS: "dst.ID", RHS: "src.ID", Error: false},
				},
			},
			expectError: true,
			errorMsg:    "method ConvertUser: method is forced to use struct literal but has incompatible features: preprocess annotation requires imperative execution before struct creation",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{}
			err := g.validateStructLiteralCompatibility(tt.function)

			if tt.expectError {
				assert.Error(t, err, "expected validation error")
				assert.Equal(t, tt.errorMsg, err.Error(), "error message mismatch")
			} else {
				assert.NoError(t, err, "unexpected validation error")
			}
		})
	}
}

func TestGenerator_isComplexAssignment(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		assignment model.Assignment
		expected   bool
	}{
		{
			name: "simple field assignment - not complex",
			assignment: model.SimpleField{
				LHS:   "dst.Name",
				RHS:   "src.Name",
				Error: false,
			},
			expected: false,
		},
		{
			name: "skip field - not complex",
			assignment: model.SkipField{
				LHS: "dst.Internal",
			},
			expected: false,
		},
		{
			name: "no match field - not complex",
			assignment: model.NoMatchField{
				LHS: "dst.Unknown",
			},
			expected: false,
		},
		{
			name: "nested struct - complex",
			assignment: model.NestStruct{
				InitExpr:      "dst.Profile = &Profile{}",
				NullCheckExpr: "src.Profile",
				Contents: []model.Assignment{
					model.SimpleField{LHS: "dst.Profile.Name", RHS: "src.Profile.Name", Error: false},
				},
			},
			expected: true,
		},
		{
			name: "slice assignment - complex",
			assignment: model.SliceAssignment{
				LHS: "dst.Items",
				RHS: "src.Items",
				Typ: "[]string",
			},
			expected: true,
		},
		{
			name: "slice loop assignment - complex",
			assignment: model.SliceLoopAssignment{
				LHS: "dst.Tags",
				RHS: "src.Tags",
				Typ: "[]Tag",
			},
			expected: true,
		},
		{
			name: "slice typecast assignment - complex",
			assignment: model.SliceTypecastAssignment{
				LHS:  "dst.IDs",
				RHS:  "src.IDs",
				Typ:  "[]int64",
				Cast: "int64",
			},
			expected: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{}
			actual := g.isComplexAssignment(tt.assignment)
			assert.Equal(t, tt.expected, actual, "isComplexAssignment result mismatch")
		})
	}
}

func TestGenerator_applyStructLiteralFallback(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		function         *model.Function
		requestedStyle   model.OutputStyle
		expectedStyle    model.OutputStyle
		expectedFallback bool
		description      string
	}{
		{
			name: "traditional style requested - no fallback",
			function: &model.Function{
				Name:        "ConvertUser",
				DstVarStyle: model.DstVarReturn,
				Assignments: []model.Assignment{
					model.SimpleField{LHS: "dst.ID", RHS: "src.ID", Error: false},
				},
			},
			requestedStyle:   model.OutputStyleTraditional,
			expectedStyle:    model.OutputStyleTraditional,
			expectedFallback: false,
			description:      "Traditional assignment is explicitly requested, no fallback needed",
		},
		{
			name: "struct literal forced on compatible function - no fallback",
			function: &model.Function{
				Name:        "ConvertUser",
				DstVarStyle: model.DstVarReturn,
				Assignments: []model.Assignment{
					model.SimpleField{LHS: "dst.ID", RHS: "src.ID", Error: false},
				},
			},
			requestedStyle:   model.OutputStyleStructLiteral,
			expectedStyle:    model.OutputStyleStructLiteral,
			expectedFallback: false,
			description:      "Compatible function can use forced struct literal",
		},
		{
			name: "struct literal forced on incompatible function - fallback applied",
			function: &model.Function{
				Name:        "ConvertUser",
				DstVarStyle: model.DstVarArg, // Incompatible with struct literal
				Assignments: []model.Assignment{
					model.SimpleField{LHS: "dst.ID", RHS: "src.ID", Error: false},
				},
			},
			requestedStyle:   model.OutputStyleStructLiteral,
			expectedStyle:    model.OutputStyleTraditional,
			expectedFallback: true,
			description:      "Incompatible function falls back to traditional assignment",
		},
		{
			name: "auto style on compatible function - use struct literal",
			function: &model.Function{
				Name:        "ConvertUser",
				DstVarStyle: model.DstVarReturn,
				Assignments: []model.Assignment{
					model.SimpleField{LHS: "dst.ID", RHS: "src.ID", Error: false},
					model.SimpleField{LHS: "dst.Name", RHS: "src.Name", Error: false},
				},
			},
			requestedStyle:   model.OutputStyleAuto,
			expectedStyle:    model.OutputStyleStructLiteral,
			expectedFallback: false,
			description:      "Auto mode chooses struct literal for compatible function",
		},
		{
			name: "auto style on incompatible function - fallback to traditional",
			function: &model.Function{
				Name:        "ConvertUser",
				DstVarStyle: model.DstVarReturn,
				Assignments: []model.Assignment{
					model.SimpleField{LHS: "dst.ID", RHS: "src.ID", Error: false},
					model.NestStruct{
						InitExpr:      "dst.Profile = &Profile{}",
						NullCheckExpr: "src.Profile",
						Contents: []model.Assignment{
							model.SimpleField{LHS: "dst.Profile.Name", RHS: "src.Profile.Name", Error: false},
						},
					},
				},
			},
			requestedStyle:   model.OutputStyleAuto,
			expectedStyle:    model.OutputStyleTraditional,
			expectedFallback: true,
			description:      "Auto mode falls back to traditional assignment for complex function",
		},
		{
			name: "unknown style - fallback to traditional",
			function: &model.Function{
				Name:        "ConvertUser",
				DstVarStyle: model.DstVarReturn,
				Assignments: []model.Assignment{
					model.SimpleField{LHS: "dst.ID", RHS: "src.ID", Error: false},
				},
			},
			requestedStyle:   model.OutputStyle("unknown"),
			expectedStyle:    model.OutputStyleTraditional,
			expectedFallback: true,
			description:      "Unknown style falls back to traditional assignment",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{}
			actualStyle, actualFallback := g.applyStructLiteralFallback(tt.function, tt.requestedStyle)

			assert.Equal(t, tt.expectedStyle, actualStyle, "output style mismatch")
			assert.Equal(t, tt.expectedFallback, actualFallback, "fallback detection mismatch")
		})
	}
}
