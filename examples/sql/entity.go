package squirrel

import "time"

type Entity struct {
	ID      int       `db:"id"`
	Name    string    `db:"name"`
	Surname string    `db:"surname"`
	ts      time.Time `db:"ts"` //nolint
}

//go:generate fieldr -type Entity -output entity_sql.go -const _upsert -const _selectByID -const _deleteByID -const _pk

const tableName = "table"
