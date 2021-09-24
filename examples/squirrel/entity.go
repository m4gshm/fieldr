package squirrel

import "time"

type Entity struct {
	ID      int       `db:"ID"`
	Name    string    `db:"NAME"`
	Surname string    `db:"SURNAME"`
	ts      time.Time `db:"TS"`
}

//go:generate fieldr -type Entity -output entity_fields.go -wrap -Strings -TagValuesMap -GetFieldValuesByTag
