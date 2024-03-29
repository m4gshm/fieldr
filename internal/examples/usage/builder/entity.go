package builder

//go:generate fieldr -type Entity builder -deconstructor .
type Entity[ID any] struct {
	*Model[ID]
	Name string
}

type Model[ID any] struct {
	ID        ID
	CreatedAt int64
	UpdatedAt int64
}
