package model

type Pet struct {
	ID        uint     `storage:"id"`
	Category  Category `storage:"category"`
	Name      string   `storage:"name"`
	PhotoUrls []string `storage:"photoUrls"`
	Status    string   `storage:"status"`
}

type Category struct {
	ID   uint64 `storage:"id"`
	Name string `storage:"name"`
}
