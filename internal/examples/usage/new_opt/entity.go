package new_opt

//go:generate fieldr -type Entity new-opt -return-value
type Entity[ID any] struct {
	*Model[ID]
	Name string
}

type Model[ID any] struct {
	ID        ID
	CreatedAt int64
	UpdatedAt int64
}
