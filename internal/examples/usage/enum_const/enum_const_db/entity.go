package enum_const_db

//go:generate fieldr -debug -type Entity
//go:fieldr enum-const -name "join \"col\" field.name" -val "tag.db" -flat Versioned -type column -list . -ref-access .
//go:fieldr enum-const -name "join \"pk\" field.name" -val "tag.db" -include "notNil tag.pk" -type column -list pk

type Entity struct {
	BaseEntity
	Versioned *VersionedEntity
	Name string `db:"name"`
}

type BaseEntity struct {
	ID int32 `db:"id" pk:""`
}

type VersionedEntity struct {
	Version int64 `db:"version"`
}

