package gorm

import "time"

type Entity struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"column:NAME" json:"name"`
	Surname   string    `gorm:"column:SURNAME" json:"_surname"`
	UpdatedAt time.Time `json:"updateAt,omitempty"`
}
