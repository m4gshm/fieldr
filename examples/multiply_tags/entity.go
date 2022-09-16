package gorm

import "time"

//go:generate fieldr -type Entity -out entity_fields.go
//go:fieldr enum-const -name "{{(join struct.name \"Col\" name) | snake | toUpper}}" -val "{{.gorm | rexp \"column:(\\\\w+),?\" | OR name | snake | up}}" -type EntityCol -func-list . -ref-access -val-access
//go:fieldr enum-const -export -val ".gorm | rexp \"column:(\\w+),?\" | OR name | snake | up"
//go:fieldr enum-const -export -val "rexp \"(?P<v>\\w+),?\" .json"
//go:fieldr enum-const -export -val "(OR (rexp \"column:(\\w+),?\" .gorm) (rexp \"(?P<v>\\w+),?\" .json))" -func-list gormOrJsonList
//go:field1 aggr-const -export -selector type == EntityCol -val ".const.name," -name EntityCols

type BaseEntity struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	UpdatedAt time.Time `json:"updateAt,omitempty"`
}

type Entity struct {
	BaseEntity
	ID        int       `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"column:NAME" json:"name"`
	Surname   string    `gorm:"column:SURNAME"`
	UpdatedAt time.Time `json:"updateAt,omitempty"`
}

func (e *Entity) Values() interface{} {
	s := entityCols()
	r := make([]interface{}, len(s))
	for i, c := range entityCols() {
		r[i] = c.val(e)
	}
	return r
}
