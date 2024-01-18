package sql_base

type VersionedEntity struct {
	Version int64 `db:"version" json:"version"`
}
