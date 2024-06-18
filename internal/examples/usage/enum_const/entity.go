package enum_const

//go:generate fieldr -type Entity fields-to-consts -val tag.json -list jsons
type Entity struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}
