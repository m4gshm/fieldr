package squirrel

import (
	"example/sql_base"
	"time"
)

type Entity struct {
	ID        int       `db:"ID"`
	Name      string    `db:"NAME"`
	Surname   string    `db:"SURNAME"`
	ts        time.Time `db:"TS"` //private excluded
	Versioned sql_base.VersionedEntity
}

//go:generate fieldr -type Entity -out entity_fields.go enum-const -name "{{ join \"col\" field.name }}" -val "tag.db" -type Col -list . -val-access -ref-access -flat Versioned
