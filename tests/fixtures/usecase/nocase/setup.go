//go:build convergen

package nocase

type ModelA struct {
	ID uint64
}

type ModelB struct {
	id uint64
}

//go:generate go run github.com/reedom/convergen
type Convergen interface {
	// :case:off
	AtoB(*ModelA) *ModelB
	// :case:off
	BtoA(*ModelB) *ModelA
}
