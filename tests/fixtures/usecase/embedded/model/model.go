package model

type Base struct {
	ID      int64
	Created int64
}

type Concrete struct {
	Base
	Name       string
	NestedData Nest
}

type Nest struct {
	Base
	NestedDataSub NestSub
}

type NestSub struct {
	Base
	// ID shadows the Base.ID. Also, it's a differently typed field.
	ID string
}
