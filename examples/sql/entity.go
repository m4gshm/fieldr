package sql

import "time"

type Entity struct {
	ID      int       `db:"id"`
	Name    string    `db:"name"`
	Surname string    `db:"surname"`
	ts      time.Time `db:"ts"` //nolint
}

//go:generate fieldr -type Entity -src ../util/const_template.go -out entity_sql.go -constLen 60 -const sql_Upsert:_upsert -const sql_selectByID:_selectByID:tableName="tableName" -const sql_deleteByID:_deleteByID -const _updateByID -const _insert -const _pk

const tableName = "table" //nolint
