package examples

import "time"

type Struct struct {
	ID       int       `db:"ID" json:"id"`
	Name     string    `db:"NAME" json:"name,omitempty"`
	Surname  string    `db:"SURNAME" json:"surname,omitempty"`
	NoJson   string    `db:"NO_JSON" json:"-"`
	noExport string    `json:"no_export"` //nolint
	ts       time.Time `db:"TS"`
	NoTag    string
}

//go:generate fieldr -type Struct -wrap -export -output struct_util.go
