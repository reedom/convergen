package model

import (
	"github.com/reedom/convergen/v8/tests/fixtures/usecase/embedded/model/types"
)

type Concrete struct {
	types.Base
	Name       string
	NestedData Nest
}

type Nest struct {
	types.Base
	NestedDataSub NestSub
}

type NestSub struct {
	types.Base
	// ID shadows the Base.ID. Also, it's a differently typed field.
	ID string
}
