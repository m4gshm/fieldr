package sql

import "time"

type Entity struct {
	ID      int       `db:"id"`
	Name    string    `db:"name"`
	Surname string    `db:"surname"`
	ts      time.Time `db:"ts"` //nolint
}

//go:generate fieldr -type Entity -src ../util/const_template.go -output entity_sql.go -const _upsert:sql_Upsert -const _selectByID:sql_selectByID -const _deleteByID:sql_deleteByID -const _updateByID -const _insert -const _pk

const tableName = "table" //nolint
