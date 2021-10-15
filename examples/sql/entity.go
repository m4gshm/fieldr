package sql

//go:fieldr -in ../util/const_template.go -out entity.go -type Entity
//go:fieldr -constLen 60 -constReplace tableName=TableName

//go:generate fieldr -GetFieldValuesByTag db
//go:generate fieldr -const sql_Upsert:_upsert -const sql_selectByID:_selectByID:tableName="tableName"
//go:generate fieldr -const sql_deleteByID:_deleteByID -const _updateByID -const _insert -const _pk

import "time"

type Entity struct {
	ID      int       `db:"id"`
	Name    string    `db:"name"`
	Surname string    `db:"surname"`
	ts      time.Time `db:"ts"` //nolint
}

const (
	TableName  = "table" //nolint
	sql_Upsert = "INSERT INTO " + TableName + " (id,name,surname) VALUES ($1,$2,$3) " +
		"DO ON CONFLICT id UPDATE SET name=$2,surname=$3 RETURNING id" //nolint
	sql_selectByID     = "SELECT id,name,surname FROM tableName WHERE id = $1"               //nolint
	sql_deleteByID     = "DELETE FROM " + TableName + " WHERE id = $1"                       //nolint
	entity__updateByID = "UPDATE " + TableName + " SET name=$2,surname=$3 WHERE id = $1"     //nolint
	entity__insert     = "INSERT INTO " + TableName + " (id,name,surname) VALUES ($1,$2,$3)" //nolint
	entity__pk         = "id"                                                                //nolint
)

func (v *Entity) getFieldValuesByTagDb() []interface{} {
	return []interface{}{v.ID, v.Name, v.Surname}
}
