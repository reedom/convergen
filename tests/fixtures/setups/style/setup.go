//go:build convergen

package style

import (
	"github.com/reedom/convergen/tests/fixtures/data/model"
)

type Pet struct {
	ID        uint64         `storage:"id"`
	Category  model.Category `storage:"category"`
	Name      string         `storage:"name"`
	PhotoUrls []string       `storage:"photoUrls"`
	Status    string         `storage:"status"`
}

//go:generate go run github.com/reedom/convergen
type Convergen interface {
	// :style arg
	ArgToArg(pet *Pet) *model.Pet
	// :style return
	ArgToReturn(pet *Pet) *model.Pet
	// :rcv r
	// :style arg
	RcvToArg(pet *Pet) *model.Pet
	// :rcv r
	// :style arg
	RcvToReturn(pet *Pet) *model.Pet
}
