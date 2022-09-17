//go:build convergen

package simple

import (
	"github.com/reedom/convergen/tests/fixtures/data/domain"
	"github.com/reedom/convergen/tests/fixtures/data/model"
)

//go:generate go run github.com/reedom/convergen
type Convergen interface {
	// convergen:opt:style arg
	ArgToArg(pet *domain.Pet) *model.Pet
	// convergen:opt:style return
	ArgToReturn(pet *domain.Pet) *model.Pet
	// convergen:rcv r
	// convergen:opt:style arg
	RecvToArg(pet *domain.Pet) *model.Pet
	// convergen:rcv r
	// convergen:opt:style arg
	RecvToReturn(pet *domain.Pet) *model.Pet
}
