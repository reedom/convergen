//go:build convergen

package nocase

import (
	"github.com/reedom/convergen/tests/fixtures/usecase/nocase/model"
)

type ModelA struct {
	ID   uint64
	Name string
}

type ModelB struct {
	id   uint64
	name string
}

//go:generate go run github.com/reedom/convergen
type Convergen interface {
	// :case:off
	AtoB(*ModelA) *ModelB
	// :case:off
	BtoA(*ModelB) *ModelA
	// :case:off
	// UserToB demonstrates copy an external package type to internal.
	// It skips private fields (and getters) in the former type.
	UserToB(*model.User) *ModelB
	// :case:off
	// BtoUser demonstrates copy an internal to external package type.
	// It skips private fields (and getters) in the latter type.
	BtoUser(*ModelB) *model.User
}
