package gorm

import "time"

//go:generate fieldr -type Entity -out entity_fields.go -export -enum-field-const "{{(join type.name \"Col\" field.name) | snake | toUpper}}={{.gorm | rexp \"column:(\\\\w+),?\" | or field.name | snake | toUpper}}" -enum-field-const "rexp \"(?P<v>\\w+),?\" .json"


type Entity struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"column:NAME" json:"name"`
	Surname   string    `gorm:"column:SURNAME" json:"surname"`
	UpdatedAt time.Time `json:"updateAt,omitempty"`
}