package examples

import "time"

type Struct struct {
	ID     int       `db:"ID" json:"id"`
	Name   string    `db:"NAME" json:"name,omitempty"`
	NoJson string    `db:"NO_JSON" json:"-"`
	ts     time.Time `db:"TS"`
}

//go:generate fieldr -type Struct -wrap -export -output struct_util.go
