package model

type User struct {
	id   uint64
	Name string
}

func (u *User) ID() uint64 {
	return u.id
}
