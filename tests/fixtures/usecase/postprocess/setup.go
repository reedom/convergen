//go:build convergen

package postprocess

import (
	"github.com/reedom/convergen/tests/fixtures/data/domain"
	"github.com/reedom/convergen/tests/fixtures/data/model"
)

//go:generate go run github.com/reedom/convergen
type Convergen interface {
	// :postprocess PostDomainToModel
	DomainToModel(*domain.Pet) *model.Pet
	ModelToDomain(*model.Pet) *domain.Pet
}

func PostDomainToModel(lhs *model.Pet, rhs domain.Pet) {

}
