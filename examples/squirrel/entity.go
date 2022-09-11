package squirrel

import "time"

type Entity struct {
	ID      int       `db:"ID"`
	Name    string    `db:"NAME"`
	Surname string    `db:"SURNAME"`
	ts      time.Time `db:"TS"`
}

//go:generate fieldr -type Entity -out entity_fields.go enum-const -name "{{ join \"col\" field.name }}" -val "tag.db" -type Col -val-accessor -ref-accessor
