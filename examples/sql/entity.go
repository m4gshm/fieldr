package sql

//go:generate fieldr -in ../util/const_template.go -out entity_sql.go -type Entity -GetFieldValuesByTag db -const sql_Upsert:_upsert -const sql_selectByID:_selectByID:tableName="tableName" -const sql_deleteByID:_deleteByID -const _updateByID -const _insert -const _pk -constLen 60 -constReplace tableName=TableName

import "time"

type Entity struct {
	ID      int       `db:"id"`
	Name    string    `db:"name"`
	Surname string    `db:"surname"`
	ts      time.Time `db:"ts"` //nolint
}

const TableName = "table" //nolint
