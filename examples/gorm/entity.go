package gorm

import "time"

//go:generate fieldr -type Entity -out entity_fields.go -enum-field-const ".gorm | rexp \"column:(\\w+),?\" | or field.name | snake | toUpper"

type Entity struct {
	ID        int    `gorm:"column:ID,primaryKey"`
	Name      string `gorm:"column:NAME"`
	Surname   string `gorm:"column:SURNAME"`
	UpdatedAt time.Time
}
