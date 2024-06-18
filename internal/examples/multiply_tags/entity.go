package multiply_tags

import "time"

//go:generate fieldr -type Entity -out entity_fields.go
//go:fieldr fields-to-consts -name "toUpper(snake(join(struct.name, \"Col\", name)))" -val "up(snake(OR(rexp(`column:(\\w+),?`, tag.gorm), name)))" -type EntityCol -list . -ref-access . -val-access . -nolint -flat Upd -flat Upd2 -flat Upd3
//go:fieldr fields-to-consts -export -val "OR(rexp(`column:(\\w+),?`, tag.gorm), name) | snake() | up()"
//go:fieldr fields-to-consts -export -val "rexp(`(?P<v>\\w+),?`, tag.json)"
//go:fieldr fields-to-consts -export -val "OR(rexp(`column:(\\w+),?`, tag.gorm), rexp(`(?P<v>\\w+),?`, tag.json))" -list gormOrJsonList

type BaseEntity struct {
	ID int `gorm:"primaryKey" json:"id"`
}

type UpdateableEntity struct {
	UpdatedAt time.Time `json:"updateAt,omitempty"`
}
type UpdateableEntity2 struct {
	UpdatedAt2 time.Time `json:"updateAt2,omitempty"`
}

type UpdateableEntity3 struct {
	UpdatedAt3 time.Time `json:"updateAt2,omitempty"`
}

type UpdateableEntityRef *UpdateableEntity
type UpdateableEntityRef3 **UpdateableEntity3

type Entity struct {
	*BaseEntity
	// ID        int       `gorm:"primaryKey" json:"id"`
	Name    string `gorm:"column:NAME" json:"name"`
	Surname string `gorm:"column:SURNAME"`
	// UpdatedAt time.Time `json:"updateAt,omitempty"`
	Upd  ***UpdateableEntityRef
	Upd2 ****UpdateableEntity2
	Upd3 UpdateableEntityRef3
}

func (e *Entity) Values() interface{} {
	s := entityCols()
	r := make([]interface{}, len(s))
	for i, c := range entityCols() {
		r[i] = e.val(c)
	}
	return r
}
