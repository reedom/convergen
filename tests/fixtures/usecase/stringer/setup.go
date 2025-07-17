//go:build convergen
// +build convergen

package stringer

import (
	"github.com/reedom/convergen/v8/tests/fixtures/data/model"
	"github.com/reedom/convergen/v8/tests/fixtures/usecase/stringer/local"
)

//go:generate go run github.com/reedom/convergen
type Convergen interface {
	// :stringer
	// :getter
	LocalToModel(pet *local.Pet) *model.Pet
}
