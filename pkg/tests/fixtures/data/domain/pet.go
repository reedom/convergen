package domain

type Pet struct {
	ID        uint
	Category  Category
	Name      string
	PhotoUrls []URL
	Status    PetStatus
}

type URL string
