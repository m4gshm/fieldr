package builder

//go:generate fieldr -type Entity builder -deconstructor .
type Entity struct {
	Id   int
	Name string
}
