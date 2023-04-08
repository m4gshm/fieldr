package get_set

//go:generate fieldr

//go:fieldr -type Entity get-set

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
	Code string
}

type RefCodeAwareEntity struct {
	*CodeAwareEntity
}

type foreignIDAwareEntity[FiD any] struct {
	ForeignID FiD
}

type BaseEntity[ID any] struct {
	ID ID
	*RefCodeAwareEntity
	foreignIDAwareEntity[ID]
}

type Entity[ID any] struct {
	*BaseEntity[ID]
	metadata
	NoDB         *NoDBFieldsEntity
	name         StringBasedType[string]
	surname      StringBasedAlias
	Values       []int32
	Ts           []*t.Time
	versioned    sql_base.VersionedEntity
	channel      chan map[t.Time]string
	someMap      map[StringBasedType[string]]bytes.Buffer
	Embedded     EmbeddedEntity
	OldForeignID *foreignIDAwareEntity[ID]
}

type EmbeddedEntity struct {
	Metadata string
}

type metadata struct {
	Schema  string
	Version int
}
