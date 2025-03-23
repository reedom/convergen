//go:build convergen

package postprocess

import (
	"github.com/reedom/convergen/tests/fixtures/data/domain"
	"github.com/reedom/convergen/tests/fixtures/data/model"
	_ "github.com/reedom/convergen/tests/fixtures/usecase/postprocess/local"
)

type A struct {
	Value int
}

type B struct {
	Value int
	Arg0  uint
	Arg1  string
}

type AdditionalArgs struct {
	Arg1 string
}

//go:generate go run github.com/reedom/convergen
type Convergen interface {
	// :preprocess PreDomainToModel
	// :postprocess PostDomainToModel
	DomainToModel(*domain.Pet) (*model.Pet, error)
	// :postprocess local.PostModelToDomain
	ModelToDomain(*model.Pet) (*domain.Pet, error)
	// :preprocess PreAToB
	// :postprocess PostAToB
	AToB(*A, uint, AdditionalArgs) *B
}

func PreDomainToModel(lhs *model.Pet, rhs domain.Pet) {
}

func PostDomainToModel(lhs *model.Pet, rhs domain.Pet) error {
	return nil
}

func PreAToB(lhs *B, rhs A, arg0 uint, arg1 AdditionalArgs) {

}

func PostAToB(lhs *B, rhs A, arg0 uint, arg1 AdditionalArgs) {

}
