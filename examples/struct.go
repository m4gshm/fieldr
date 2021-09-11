package examples

import "time"

type Struct struct {
	ID     int       `db:"ID" json:"id"`
	Name   string    `db:"NAME" json:"name,omitempty"`
	NoJson string    `db:"NO_JSON"`
	ts     time.Time `db:"TS" json:"ts"`
}

//go:generate fieldr -type Struct -export -output struct_util.go
