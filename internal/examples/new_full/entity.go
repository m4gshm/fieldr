package new_full

//go:generate fieldr -debug

//go:fieldr -type Entity new-full -return-value -flat -exclude excluded
//go:fieldr -type Entity new-full -name New2 -exclude excluded
//go:fieldr -type CodeAwareEntity new-full -no-inline -name NewCodeAware

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

type emptyInlined struct {
}

type CodeAwareEntity struct {
	*emptyInlined
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
	emptyInlined
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
	excluded     string
}

type EmbeddedEntity struct {
	Metadata string
}
