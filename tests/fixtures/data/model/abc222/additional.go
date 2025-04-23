package abc222

type AdditionalItem123 struct {
	Name string          `json:"name"`
	List []Additional321 `storage:"list"`
}

type Additional321 struct {
	Name string `json:"name"`
}
