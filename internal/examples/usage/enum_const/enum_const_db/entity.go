package enum_const_db

//go:generate fieldr -debug -type Entity
//go:fieldr enum-const -name "{{ join \"col\" field.name }}" -val "tag.db" -flat Versioned -type column -list . -ref-access .

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

