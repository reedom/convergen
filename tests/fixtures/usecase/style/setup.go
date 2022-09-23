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
	// :recv r
	// :style arg
	RcvToArg(pet *Pet) *model.Pet
	// :recv r
	// :style arg
	RcvToReturn(pet *Pet) *model.Pet
	// :recv r
	// :reverse
	// :style arg
	RevRcvFromArgVal(*Pet) model.Pet
	// :recv r
	// :reverse
	// :style arg
	RevRcvFromArgPtr(*Pet) (pet *model.Pet)
	// It is illegal to specify :recv:rev and :style return.
	//// :recv:rev m
	//// :style return
	//RcvFromReturn(*model.Pet) *Pet
}
