package domain

type Pet struct {
	id        uint
	category  Category
	name      string
	photoUrls []URL
	status    PetStatus
}

func NewPet(id uint, cat Category, name string, photoUrls []URL, status PetStatus) *Pet {
	return &Pet{
		id:        id,
		category:  cat,
		name:      name,
		photoUrls: photoUrls,
		status:    status,
	}
}

func (p *Pet) ID() uint {
	return p.id
}

func (p *Pet) Category() Category {
	return p.category
}

func (p *Pet) Name() string {
	return p.name
}

func (p *Pet) PhotoUrls() []URL {
	return p.photoUrls
}

func (p *Pet) Status() PetStatus {
	return p.status
}

func (p *Pet) Rename(v string) {
	p.name = v
}

func (p *Pet) SetCategory(v Category) {
	p.category = v
}

type URL string
