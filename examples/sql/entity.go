package sql

//go:fieldr -in ../sql_util/postgres.go -out entity.go -type Entity
//go:fieldr -constLen 100 -constReplace tableName="tableName"

//go:generate fieldr -GetFieldValuesByTag db -ref -excludeFields ID -name insertValues -compact
//go:generate fieldr -GetFieldValuesByTag db -ref -name values -compact
//go:generate fieldr -const sqlUpsert:_upsert -const sqlInsert:_insert -const sqlSelectByID:_selectByID
//go:generate fieldr -const sqlSelectByIDs:_selectByIDs -const sqlDeleteByID:_deleteByID

import (
	"database/sql"
	"time"

	pq "github.com/lib/pq"
)

type Entity struct {
	ID      int32     `db:"id" pk:"" json:"id,omitempty"`
	Name    string    `db:"name" json:"name,omitempty"`
	Surname string    `db:"surname" json:"surname,omitempty"`
	Values  []int32   `db:"values" json:"values,omitempty"`
	Ts      time.Time `db:"ts" json:"ts"`
}

const (
	sqlInsert = "INSERT INTO \"tableName\" (name,surname,values,ts) VALUES ($1,$2,$3,$4) RETURNING id"
	sqlUpsert = "INSERT INTO \"tableName\" (id,name,surname,values,ts) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (id) DO " +
		"UPDATE SET name=$2,surname=$3,values=$4,ts=$5 RETURNING id"
	sqlSelectByID  = "SELECT id,name,surname,values,ts FROM \"tableName\" WHERE id = $1"
	sqlSelectByIDs = "SELECT id,name,surname,values,ts FROM \"tableName\" WHERE id = ANY($1::int[])"
	sqlDeleteByID  = "DELETE FROM \"tableName\" WHERE id = $1"
)

func (v *Entity) insertValues() []interface{} {
	return []interface{}{&v.Name, &v.Surname, pq.Array(&v.Values), &v.Ts}
}

func (v *Entity) values() []interface{} {
	return []interface{}{&v.ID, &v.Name, &v.Surname, pq.Array(&v.Values), &v.Ts}
}

func GetByID(e RowQuerier, id int32) (*Entity, error) {
	row := e.QueryRow(sqlSelectByID, id)
	if err := row.Err(); err != nil {
		return nil, err
	}
	entity := new(Entity)
	if err := row.Scan(entity.values()...); err != nil {
		return nil, err
	}
	return entity, nil
}

func GetByIDs(e RowsQuerier, ids ...int32) ([]*Entity, error) {
	rows, err := e.Query(sqlSelectByIDs, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	result := make([]*Entity, 0, len(ids))
	for rows.Next() {
		if err = rows.Err(); err != nil {
			return nil, err
		}
		var entity Entity
		if err := rows.Scan(entity.values()...); err != nil {
			return nil, err
		}
		result = append(result, &entity)
	}
	return result, nil
}

func (v *Entity) Store(e RowQuerier) (int32, error) {
	var (
		columns []interface{}
		sqlOp   string
	)
	if v.ID == 0 {
		sqlOp = sqlInsert
		columns = v.insertValues()
	} else {
		sqlOp = sqlUpsert
		columns = v.values()
	}
	row := e.QueryRow(sqlOp, columns...)
	if err := row.Err(); err != nil {
		return -1, err
	}
	var newID int32
	if err := row.Scan(&newID); err != nil {
		return -1, err
	}
	if newID != v.ID {
		v.ID = newID
	}
	return newID, nil
}

func (v *Entity) Delete(e Execer) (bool, error) {
	exec, err := e.Exec(sqlDeleteByID, v.ID)
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

type RowsQuerier interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
}
