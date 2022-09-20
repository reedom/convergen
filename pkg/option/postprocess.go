package option

import (
	"go/token"
	"go/types"
)

type Postprocess struct {
	Func         types.Object
	DstSide      types.Type
	SrcSide      types.Type
	ReturnsError bool
	Pos          token.Pos
}
