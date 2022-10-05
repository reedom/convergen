package option

import (
	"go/token"
	"go/types"
)

type Manipulator struct {
	Func     types.Object
	DstSide  types.Type
	SrcSide  types.Type
	RetError bool
	Pos      token.Pos
}
