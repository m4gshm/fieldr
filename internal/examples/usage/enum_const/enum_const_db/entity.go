package enum_const_db

//go:generate fieldr -type Entity
//go:fieldr fields-to-consts -name "'col' + field.name" -val "tag.db" -flat Versioned -type column -list . -ref-access .
//go:fieldr fields-to-consts -name "'pk' + field.name" -val "tag.db" -include "tag.pk == 'true'" -type column -list pk

type Entity struct {
	BaseEntity
	Versioned *VersionedEntity
	Name      string `db:"name"`
}

type BaseEntity struct {
	ID int32 `db:"id" pk:"true"`
}

type VersionedEntity struct {
	Version int64 `db:"version"`
}
