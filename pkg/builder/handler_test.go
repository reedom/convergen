package builder

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"

	bmodel "github.com/reedom/convergen/v9/pkg/builder/model"
	gmodel "github.com/reedom/convergen/v9/pkg/generator/model"
	"github.com/reedom/convergen/v9/pkg/option"
)

func TestSkipHandler(t *testing.T) {
	// Create a new assignmentBuilder with a SkipFields option.
	ab := &assignmentBuilder{
		fset: token.NewFileSet(),
		opts: option.Options{
			SkipFields: []*option.PatternMatcher{
				func() *option.PatternMatcher {
					p, _ := option.NewPatternMatcher("ID", true)
					return p
				}(),
			},
		},
	}

	// Create a new SkipHandler.
	h := NewSkipHandler(ab)

	// Create a LHS node that should be skipped.
	lhs := bmodel.NewRootNode("dst.ID", types.Typ[types.Int])

	// Handle the assignment.
	a, err := h.Handle(lhs, nil, nil)

	// Assert that the assignment is a SkipField and the error is nil.
	assert.NoError(t, err)
	assert.IsType(t, &gmodel.SkipField{}, a)
	assert.Equal(t, "dst.ID", a.(*gmodel.SkipField).LHS)
}

func TestLiteralSetterHandler(t *testing.T) {
	// Create a new assignmentBuilder with a Literals option.
	ab := &assignmentBuilder{
		fset: token.NewFileSet(),
		opts: option.Options{
			Literals: []*option.LiteralSetter{
				option.NewLiteralSetter("Name", `"test"`, token.NoPos),
			},
		},
	}

	// Create a new LiteralSetterHandler.
	h := NewLiteralSetterHandler(ab)

	// Create a LHS node that should be set with a literal value.
	lhs := bmodel.NewRootNode("dst.Name", types.Typ[types.String])

	// Handle the assignment.
	a, err := h.Handle(lhs, nil, nil)

	// Assert that the assignment is a SimpleField and the error is nil.
	assert.NoError(t, err)
	assert.IsType(t, &gmodel.SimpleField{}, a)
	assert.Equal(t, "dst.Name", a.(*gmodel.SimpleField).LHS)
	assert.Equal(t, `"test"`, a.(*gmodel.SimpleField).RHS)
}
