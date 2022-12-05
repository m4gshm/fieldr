package builder

//go:generate fieldr -debug

//go:fieldr -type Entity builder -export all
//go:fieldr -out entity_tagged.go -out-build-tag integration builder -export all
//go:fieldr -type EmbeddedEntity builder -build-value -chain-value -export all

import (
	"bytes"
	t "time"

	"example/sql_base"
)

type StringBasedType[s string] string
type StringBasedAlias = string

type NoDBFieldsEntity struct {
	OldID int32
}

type BaseEntity[ID any] struct {
	ID ID `db:"id" pk:"" json:"id,omitempty"`
}

type Entity[ID any] struct {
	*BaseEntity[ID]
	NoDB      *NoDBFieldsEntity
	Name      StringBasedType[string] `db:"name" json:"name,omitempty"`
	Surname   StringBasedAlias        `db:"surname" json:"surname,omitempty"`
	Values    []int32                 `db:"values" json:"values,omitempty"`
	Ts        []*t.Time               `db:"ts" json:"ts"`
	Versioned sql_base.VersionedEntity
	Chan      chan map[t.Time]string
	SomeMap   map[StringBasedType[string]]bytes.Buffer
	Embedded  EmbeddedEntity
}

type EmbeddedEntity struct {
	Metadata string
}
