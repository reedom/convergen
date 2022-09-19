//go:build convergen
// +build convergen

package converter

import (
	"github.com/reedom/convergen/tests/fixtures/data/domain"
	"github.com/reedom/convergen/tests/fixtures/data/model"
)

//go:generate go run github.com/reedom/convergen
type Convergen interface {
	DomainToModel(pet *domain.Pet) *model.Pet
	// :conv domain.NewPetStatusFromValue Status
	ModelToDomain(*model.Pet) (*domain.Pet, error)
}
