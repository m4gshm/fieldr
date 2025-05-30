package squirrel

import (
	"example/sql_base"
	"time"
)

type Entity struct {
	ID        int       `db:"ID" pk:"true"`
	Name      string    `db:"NAME"`
	Surname   string    `db:"SURNAME"`
	ts        time.Time `db:"TS"` //private excluded
	Versioned sql_base.VersionedEntity
}

type Entity2 struct {
	ID        int       `db:"ID"`
	Name      string    `db:"NAME"`
	Surname   string    `db:"SURNAME"`
	ts        time.Time `db:"TS"` //private excluded
	Versioned sql_base.VersionedEntity
}

//go:generate fieldr

//go:fieldr -type Entity -out entity_fields.go
//go:fieldr fields-to-consts -name "join('col', field.name)" -val "tag.db" -type Col -list . -val-access . -ref-access . -field-name-access . -flat Versioned
//go:fieldr fields-to-consts -name "join('pk', field.name)" -val "tag.db" -include "tag.pk == 'true'" -type Col -list pk

//go:fieldr -out entity_fields_alternative.go
//go:fieldr fields-to-consts -name "join('AlterCol', field.name)" -val "tag.db" -type Col -not-declare-type -list ACols -val-access Aval -ref-access Aref -exclude Versioned

//go:fieldr -type Entity2 -out entity_fields2.go
//go:fieldr fields-to-consts -name "join('col2', field.name)" -val "tag.db" -type Col2 -list . -val-access . -ref-access . -exclude Surname -exclude Versioned
