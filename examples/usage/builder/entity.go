package builder

//go:generate fieldr -type Entity builder
type Entity struct {
	Id   int
	Name string
}
