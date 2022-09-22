package domain

type Base struct {
	ID      int64
	created int64
}

func (b *Base) Created() int64 {
	return b.created
}

func (b *Base) SetCreated(v int64) {
	b.created = v
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
