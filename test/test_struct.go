package test

import "time"

type TestStruct struct {
	ID     int       `db:"ID" json:"id"`
	Name   string    `db:"NAME" json:"name,omitempty"`
	NoJson string    `db:"NO_JSON"`
	ts     time.Time `db:"TS" json:"ts"`
}

//go:generate tag-constanter -type TestStruct -wrap -export -output test_struct_const.go
