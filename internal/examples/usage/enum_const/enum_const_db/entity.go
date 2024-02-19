package enum_const_db

//go:generate fieldr -type Entity
//go:fieldr enum-const -name "'col' + field.name" -val "tag.db" -flat Versioned -type column -list . -ref-access .
//go:fieldr enum-const -name "'pk' + field.name" -val "tag.db" -include "tag.pk != nil" -type column -list pk

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

