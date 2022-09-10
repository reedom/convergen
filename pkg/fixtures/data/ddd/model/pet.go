package model

type Pet struct {
	ID           uint64   `storage:"id"`
	CategoryID   uint64   `storage:"categoryId"`
	CategoryName string   `storage:"categoryName"`
	Name         string   `storage:"name"`
	PhotoUrls    []string `storage:"photoUrls"`
	Status       string   `storage:"status"`
}
