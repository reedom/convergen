//go:build convergen
// +build convergen

package converter

import (
	"github.com/reedom/convergen/tests/fixtures/data/domain"
	"github.com/reedom/convergen/tests/fixtures/data/model"
)

//go:generate go run github.com/reedom/convergen
type Convergen interface {
	// :conv fromDomainCategory Category
	// :conv urlsToStrings PhotoUrls
	DomainToModel(*domain.Pet) *model.Pet
	// :conv toDomainCategory Category
	// :conv stringsToURLs PhotoUrls
	// :conv domain.NewPetStatusFromValue Status
	ModelToDomain(*model.Pet) (*domain.Pet, error)
}

func urlsToStrings(list []domain.URL) []string {
	ret := make([]string, len(list))
	for i, url := range list {
		ret[i] = url.String()
	}
	return ret
}

func stringsToURLs(list []string) []domain.URL {
	ret := make([]domain.URL, len(list))
	for i, s := range list {
		ret[i] = domain.NewURL(s)
	}
	return ret
}

func fromDomainCategory(cat domain.Category) model.Category {
	return model.Category{
		CategoryID: uint64(cat.ID),
		Name:       cat.Name,
	}
}

func toDomainCategory(cat model.Category) domain.Category {
	return domain.Category{
		ID:   uint(cat.CategoryID),
		Name: cat.Name,
	}
}
