package sql

//go:fieldr -in ../util/const_template.go -out entity.go -type Entity
//go:fieldr -constLen 60 -constReplace tableName="tableName"

//go:generate fieldr -GetFieldValuesByTag db -ref -excludeFields ID -name insertValues
//go:generate fieldr -GetFieldValuesByTag db -ref -name values
//go:generate fieldr -const sql_Upsert:_upsert -const sql_Insert:_insert
//go:generate fieldr -const sql_selectByID:_selectByID -const sql_deleteByID:_deleteByID

import (
	"database/sql"
	"time"
)

type Entity struct {
	ID      int       `db:"id" pk:""`
	Name    string    `db:"name"`
	Surname string    `db:"surname"`
	Ts      time.Time `db:"ts"`
}

const (
	TableName  = "tableName" //nolint
	sql_Insert = "INSERT INTO \"tableName\" (name,surname,ts) VALUES ($1,$2,$3) " +
		"RETURNING id"
	sql_Upsert = "INSERT INTO \"tableName\" (id,name,surname,ts) VALUES " +
		"($1,$2,$3,$4) ON CONFLICT (id) DO UPDATE SET name=$2,surname=$3,ts=$4 " +
		"RETURNING id" //nolint
	sql_selectByID = "SELECT id,name,surname,ts FROM \"tableName\" WHERE id = $1"
	sql_deleteByID = "DELETE FROM \"tableName\" WHERE id = $1"
)

func (v *Entity) insertValues() []interface{} {
	return []interface{}{&v.Name, &v.Surname, &v.Ts}
}

func (v *Entity) values() []interface{} {
	return []interface{}{
		&v.ID,
		&v.Name,
		&v.Surname,
		&v.Ts,
	}
}

func GetByID(e *sql.DB, id int) (*Entity, error) {
	row := e.QueryRow(sql_selectByID, id)
	if err := row.Err(); err != nil {
		return nil, err
	}
	entity := new(Entity)
	if err := row.Scan(entity.values()...); err != nil {
		return nil, err
	}
	return entity, nil
}

func (v *Entity) Store(e RowQuerier) (int, error) {
	var (
		columns []interface{}
		sqlOp   string
	)
	if v.ID == 0 {
		sqlOp = sql_Insert
		columns = v.insertValues()
	} else {
		sqlOp = sql_Upsert
		columns = v.values()
	}
	row := e.QueryRow(sqlOp, columns...)
	if err := row.Err(); err != nil {
		return -1, err
	}
	var newID int
	if err := row.Scan(&newID); err != nil {
		return -1, err
	}
	if newID != v.ID {
		v.ID = newID
	}
	return newID, nil
}

func (v *Entity) Delete(e Execer) (bool, error) {
	exec, err := e.Exec(sql_deleteByID, v.ID)
	if err != nil {
		return false, err
	}
	rowsAffected, err := exec.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

type Execer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

type RowQuerier interface {
	QueryRow(string, ...interface{}) *sql.Row
}
