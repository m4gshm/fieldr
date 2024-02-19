package enum_const

//go:generate fieldr -type Entity enum-const -val tag.json -list jsons
type Entity struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}
