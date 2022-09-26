package builder

//go:generate fieldr -type Entity

//go:fieldr builder

import (
	"bytes"
	t "time"

	"example/sql_base"
)

type StringBasedType string
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
	Name      StringBasedType  `db:"name" json:"name,omitempty"`
	Surname   StringBasedAlias `db:"surname" json:"surname,omitempty"`
	Values    []int32          `db:"values" json:"values,omitempty"`
	Ts        []*t.Time        `db:"ts" json:"ts"`
	Versioned sql_base.VersionedEntity
	Chan      chan map[t.Time]string
	SomeMap   map[StringBasedType]bytes.Buffer
}
