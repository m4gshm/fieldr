package new_full

//go:generate fieldr -debug

//go:fieldr -type Entity new-full -return-value -flat -exclude excluded
//go:fieldr -type Entity -out . new-full -name New2 -exclude excluded
//go:fieldr -type CodeAwareEntity new-full -no-inline -name NewCodeAware

import (
	"bytes"
	c "cmp"
	"example/sql_base"
	t "time"
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

type E[ID, FID any] struct {
	ID ID
	*RefCodeAwareEntity
	foreignIDAwareEntity[FID]
}

type Entity2[ID any] = *E[ID, ID]

type Entity[ID c.Ordered, FID any] struct {
	*E[ID, FID]
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

func New2[ID c.Ordered, FID any](
	e *E[ID, FID],
	metadata struct {
		Schema  string
		Version int
	},
	noDB NoDBFieldsEntity,
	name StringBasedType[string],
	surname StringBasedAlias,
	values []int32,
	ts []*t.Time,
	versioned sql_base.VersionedEntity,
	channel chan map[t.Time]string,
	someMap map[StringBasedType[string]]bytes.Buffer,
	embedded EmbeddedEntity,
	oldForeignID *foreignIDAwareEntity[ID],
) *Entity[ID, FID] {
	return &Entity[ID, FID]{
		E:            e,
		metadata:     metadata,
		NoDB:         noDB,
		name:         name,
		surname:      surname,
		Values:       values,
		Ts:           ts,
		versioned:    versioned,
		channel:      channel,
		someMap:      someMap,
		Embedded:     embedded,
		OldForeignID: oldForeignID,
	}
}
