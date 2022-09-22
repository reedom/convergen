//go:build convergen
// +build convergen

package converter

import (
	"github.com/reedom/convergen/tests/fixtures/usecase/embedded/domain"
	"github.com/reedom/convergen/tests/fixtures/usecase/embedded/model"
)

//go:generate go run github.com/reedom/convergen
type Convergen interface {
	// :typecast
	DomainToModel(s *domain.Concrete) (d *model.Concrete)
	// :typecast
	ModelToDomain(*model.Concrete) (*domain.Concrete, error)
}
