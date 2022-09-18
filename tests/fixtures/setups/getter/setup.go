//go:build convergen
// +build convergen

package getter

import (
	"github.com/reedom/convergen/tests/fixtures/data/ddd/domain"
	"github.com/reedom/convergen/tests/fixtures/data/ddd/model"
)

//go:generate go run github.com/reedom/convergen
type Convergen interface {
	// DomainToModel copies domain.Pet to model.Pet.
	DomainToModel(pet *domain.Pet) *model.Pet
}
