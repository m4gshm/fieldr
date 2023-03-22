package builder

//go:generate fieldr

//go:fieldr -type Entity builder -export all -deconstructor .
//go:fieldr -out entity_builder_noref.go builder -export all -name EntityBuilderVal -build-value -chain-value
//go:fieldr -out entity_builder_chainref_buildval.go builder -export all -name EntityBuilderChainRefBuildVal -build-value
//go:fieldr -out entity_tagged.go -out-build-tag integration builder -export all -deconstructor .
//go:fieldr -type EmbeddedEntity builder -build-value -chain-value -export all -deconstructor .

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

type CodeAwareEntity struct {
	Code string `db:"code" json:"code,omitempty"`
}

type ForeignIDAwareEntity[FiD any] struct {
	ForeignID FiD `db:"foreign_id" json:"foreignID,omitempty"`
}

type BaseEntity[ID any] struct {
	ID ID `db:"id" pk:"" json:"id,omitempty"`
	*CodeAwareEntity
	ForeignIDAwareEntity[ID]
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
