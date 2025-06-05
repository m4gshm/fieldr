package constructor

func NewEntity[ID any](
	Model *Model[ID],
	Name string,
) *Entity[ID] {
	return &Entity[ID]{
		Model: Model,
		Name:  Name,
	}
}
