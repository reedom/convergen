package domain

type Pet struct {
	ID        uint
	Category  Category
	Name      string
	PhotoUrls []URL
	Status    PetStatus
}

type URL string

func NewURL(s string) URL {
	return URL(s)
}

func (u URL) String() string {
	return string(u)
}
