package option

import (
	"go/token"
	"go/types"
)

// Manipulator represents a manipulator that manipulates the source and destination types.
type Manipulator struct {
	Func           types.Object // Func represents the function object that this manipulator invokes.
	DstSide        types.Type   // DstSide is the type expression of the destination side.
	SrcSide        types.Type   // SrcSide is the type expression of the source side.
	AdditionalArgs []types.Type // AdditionalArgs is the type expressions of the additional arguments.
	RetError       bool         // RetError indicates whether the manipulator returns an error or not.
	Pos            token.Pos    // Pos represents the position of the manipulator in the source code.
}
