package new_opt

//go:generate fieldr -debug

//go:fieldr -type Entity new-opt -required ID -flat

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

type E[ID any] struct {
	ID ID
	*RefCodeAwareEntity
	foreignIDAwareEntity[ID]
}

type Entity2[ID any] = *E[ID]

type Entity[ID any] struct {
	*E[ID]
	metadata struct {
		Schema  string
		Version int
	}
	NoDB         NoDBFieldsEntity
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
