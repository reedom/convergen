package model

import (
	"go/types"

	"github.com/reedom/convergen/pkg/util"
)

// Copier contains a helper function information.
type Copier struct {
	IsRoot      bool   // true means this copier refers the convergen method. false means it becomes an inner function.
	Name        string // name becomes a copier function's name.
	LHS         types.Type
	RHS         types.Type
	HandleCount int
}

func NewCopier(name string, lhs, rhs types.Type) *Copier {
	return &Copier{
		Name:        name,
		LHS:         lhs,
		RHS:         rhs,
		HandleCount: 1,
	}
}

func (h *Copier) MarkHandle(lhs, rhs types.Type) bool {
	canHandle :=
		types.AssignableTo(util.DerefPtr(lhs), util.DerefPtr(h.LHS)) &&
			types.AssignableTo(util.DerefPtr(rhs), util.DerefPtr(h.RHS))
	if !canHandle {
		return false
	}
	h.HandleCount += 1
	return true
}
