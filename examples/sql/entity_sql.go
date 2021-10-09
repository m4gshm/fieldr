// Code generated by 'fieldr -type Entity -src ../util/const_template.go -out entity_sql.go -const sql_Upsert:_upsert -const sql_selectByID:_selectByID:tableName="tableName" -const sql_deleteByID:_deleteByID -const _updateByID -const _insert -const _pk -constLen 60 -constReplace tableName=TableName'; DO NOT EDIT.

package sql

const (
	sql_Upsert = "INSERT INTO " + TableName + " (id,name,surname) VALUES ($1,$2,$3) " +
		"DO ON CONFLICT id UPDATE SET name=$2,surname=$3 RETURNING id"
	sql_selectByID     = "SELECT id,name,surname FROM tableName WHERE id = $1"
	sql_deleteByID     = "DELETE FROM " + TableName + " WHERE id = $1"
	entity__updateByID = "UPDATE " + TableName + " SET name=$2,surname=$3 WHERE id = $1"
	entity__insert     = "INSERT INTO " + TableName + " (id,name,surname) VALUES ($1,$2,$3)"
	entity__pk         = "id"
)
