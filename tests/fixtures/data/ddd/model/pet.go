package model

type Pet struct {
	ID        uint     `storage:"id"`
	Category  Category `storage:"category"`
	Name      string   `storage:"name"`
	PhotoUrls []string `storage:"photoUrls"`
	Status    string   `storage:"status"`
}

type Category struct {
	CategoryID uint64 `storage:"categoryId"`
	Name       string `storage:"name"`
}
