package types

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
