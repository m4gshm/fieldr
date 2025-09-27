package new_full

//go:generate fieldr -type Entity new-full
type Entity[ID any] struct {
	*Model[ID]
	Name string
}

type Model[ID any] struct {
	ID        ID
	CreatedAt int64
	UpdatedAt int64
}
