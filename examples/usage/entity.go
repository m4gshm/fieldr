package usage

//go:generate fieldr -type Entity enum-const -val .json -list jsons
type Entity struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}
