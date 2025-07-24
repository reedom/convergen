package builder

import (
	bmodel "github.com/reedom/convergen/v8/pkg/builder/model"
	gmodel "github.com/reedom/convergen/v8/pkg/generator/model"
	"github.com/reedom/convergen/v8/pkg/logger"
)

// AssignmentHandler defines the interface for a handler in the chain of responsibility
// for generating an assignment between two variables.
type AssignmentHandler interface {
	SetNext(handler AssignmentHandler)
	Handle(lhs, rhs bmodel.Node, additionalArgs []bmodel.Node) (gmodel.Assignment, error)
}

// next is a helper struct for chaining AssignmentHandlers.
type next struct {
	nextHandler AssignmentHandler
}

// SetNext sets the next handler in the chain.
func (h *next) SetNext(handler AssignmentHandler) {
	h.nextHandler = handler
}

// SkipHandler handles the `:skip` notation.
type SkipHandler struct {
	next
	ab *assignmentBuilder
}

// NewSkipHandler creates a new SkipHandler.
func NewSkipHandler(ab *assignmentBuilder) *SkipHandler {
	return &SkipHandler{ab: ab}
}

// Handle checks if the LHS field should be skipped.
func (h *SkipHandler) Handle(lhs, rhs bmodel.Node, additionalArgs []bmodel.Node) (gmodel.Assignment, error) {
	if h.ab.opts.ShouldSkip(lhs.MatcherExpr()) {
		logger.Printf("%v: skip %v", h.ab.fset.Position(h.ab.methodPos), lhs.AssignExpr())
		return &gmodel.SkipField{LHS: lhs.AssignExpr()}, nil
	}

	if h.nextHandler != nil {
		return h.nextHandler.Handle(lhs, rhs, additionalArgs)
	}
	return nil, nil
}

// LiteralSetterHandler handles the `:literal` notation.
type LiteralSetterHandler struct {
	next
	ab *assignmentBuilder
}

// NewLiteralSetterHandler creates a new LiteralSetterHandler.
func NewLiteralSetterHandler(ab *assignmentBuilder) *LiteralSetterHandler {
	return &LiteralSetterHandler{ab: ab}
}

// Handle checks if the LHS field should be set with a literal value.
func (h *LiteralSetterHandler) Handle(lhs, rhs bmodel.Node, additionalArgs []bmodel.Node) (gmodel.Assignment, error) {
	for _, setter := range h.ab.opts.Literals {
		if setter.Dst().Match(lhs.MatcherExpr(), true) {
			return &gmodel.SimpleField{LHS: lhs.AssignExpr(), RHS: setter.Literal()}, nil
		}
	}

	if h.nextHandler != nil {
		return h.nextHandler.Handle(lhs, rhs, additionalArgs)
	}
	return nil, nil
}
