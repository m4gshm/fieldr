// Code generated by 'fieldr'; DO NOT EDIT.

package get_set

import (
	"bytes"
	"example/sql_base"
	"time"
)

func (e *Entity[ID]) GetID() ID {
	if e != nil {
		if be := e.BaseEntity; be != nil {
			return be.ID
		}
	}

	var no ID
	return no
}

func (e *Entity[ID]) SetID(id ID) {
	if e != nil {
		if be := e.BaseEntity; be != nil {
			be.ID = id
		}
	}
}

func (e *Entity[ID]) GetCode() string {
	if e != nil {
		if be := e.BaseEntity; be != nil {
			if rcae := be.RefCodeAwareEntity; rcae != nil {
				if cae := rcae.CodeAwareEntity; cae != nil {
					return cae.Code
				}
			}
		}
	}

	var no string
	return no
}

func (e *Entity[ID]) SetCode(code string) {
	if e != nil {
		if be := e.BaseEntity; be != nil {
			if rcae := be.RefCodeAwareEntity; rcae != nil {
				if cae := rcae.CodeAwareEntity; cae != nil {
					cae.Code = code
				}
			}
		}
	}
}

func (e *Entity[ID]) GetForeignID() ID {
	if e != nil {
		if be := e.BaseEntity; be != nil {
			return be.foreignIDAwareEntity.ForeignID
		}
	}

	var no ID
	return no
}

func (e *Entity[ID]) SetForeignID(foreignID ID) {
	if e != nil {
		if be := e.BaseEntity; be != nil {
			be.foreignIDAwareEntity.ForeignID = foreignID
		}
	}
}

func (e *Entity[ID]) Metadata() struct {
	Schema  string
	Version int
} {
	if e != nil {
		return e.metadata
	}

	var no struct {
		Schema  string
		Version int
	}
	return no
}

func (e *Entity[ID]) SetMetadata(metadata struct {
	Schema  string
	Version int
}) {
	if e != nil {
		e.metadata = metadata
	}
}

func (e *Entity[ID]) GetNoDB() *NoDBFieldsEntity {
	if e != nil {
		return e.NoDB
	}

	var no *NoDBFieldsEntity
	return no
}

func (e *Entity[ID]) SetNoDB(noDB *NoDBFieldsEntity) {
	if e != nil {
		e.NoDB = noDB
	}
}

func (e *Entity[ID]) GetName() StringBasedType[string] {
	if e != nil {
		return e.name
	}

	var no StringBasedType[string]
	return no
}

func (e *Entity[ID]) SetName(name StringBasedType[string]) {
	if e != nil {
		e.name = name
	}
}

func (e *Entity[ID]) GetSurname() StringBasedAlias {
	if e != nil {
		return e.surname
	}

	var no StringBasedAlias
	return no
}

func (e *Entity[ID]) SetSurname(surname StringBasedAlias) {
	if e != nil {
		e.surname = surname
	}
}

func (e *Entity[ID]) GetValues() []int32 {
	if e != nil {
		return e.Values
	}

	var no []int32
	return no
}

func (e *Entity[ID]) SetValues(values []int32) {
	if e != nil {
		e.Values = values
	}
}

func (e *Entity[ID]) GetTs() []*time.Time {
	if e != nil {
		return e.Ts
	}

	var no []*time.Time
	return no
}

func (e *Entity[ID]) SetTs(ts []*time.Time) {
	if e != nil {
		e.Ts = ts
	}
}

func (e *Entity[ID]) GetVersioned() sql_base.VersionedEntity {
	if e != nil {
		return e.versioned
	}

	var no sql_base.VersionedEntity
	return no
}

func (e *Entity[ID]) SetVersioned(versioned sql_base.VersionedEntity) {
	if e != nil {
		e.versioned = versioned
	}
}

func (e *Entity[ID]) GetChannel() chan map[time.Time]string {
	if e != nil {
		return e.channel
	}

	var no chan map[time.Time]string
	return no
}

func (e *Entity[ID]) SetChannel(channel chan map[time.Time]string) {
	if e != nil {
		e.channel = channel
	}
}

func (e *Entity[ID]) GetSomeMap() map[StringBasedType[string]]bytes.Buffer {
	if e != nil {
		return e.someMap
	}

	var no map[StringBasedType[string]]bytes.Buffer
	return no
}

func (e *Entity[ID]) SetSomeMap(someMap map[StringBasedType[string]]bytes.Buffer) {
	if e != nil {
		e.someMap = someMap
	}
}

func (e *Entity[ID]) GetEmbedded() EmbeddedEntity {
	if e != nil {
		return e.Embedded
	}

	var no EmbeddedEntity
	return no
}

func (e *Entity[ID]) SetEmbedded(embedded EmbeddedEntity) {
	if e != nil {
		e.Embedded = embedded
	}
}

func (e *Entity[ID]) GetOldForeignID() *foreignIDAwareEntity[ID] {
	if e != nil {
		return e.OldForeignID
	}

	var no *foreignIDAwareEntity[ID]
	return no
}

func (e *Entity[ID]) SetOldForeignID(oldForeignID *foreignIDAwareEntity[ID]) {
	if e != nil {
		e.OldForeignID = oldForeignID
	}
}
