//go:build convergen

package postprocess

import (
	"github.com/reedom/convergen/tests/fixtures/data/domain"
	"github.com/reedom/convergen/tests/fixtures/data/model"
	_ "github.com/reedom/convergen/tests/fixtures/usecase/postprocess/local"
)

//go:generate go run github.com/reedom/convergen
type Convergen interface {
	// :preprocess PreDomainToModel
	// :postprocess PostDomainToModel
	DomainToModel(*domain.Pet) (*model.Pet, error)
	// :postprocess local.PostModelToDomain
	ModelToDomain(*model.Pet) (*domain.Pet, error)
}

func PreDomainToModel(lhs *model.Pet, rhs domain.Pet) {
}

func PostDomainToModel(lhs *model.Pet, rhs domain.Pet) error {
	return nil
}
