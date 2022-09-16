package sql

//go:generate fieldr -type Entity -out entity_fields.go enum-const -name "{{ join \"col\" field.name }}" -val "tag.db" -type col -func-list . -val-access -ref-access -flat Versioned

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	"example/sql_base"

	pq "github.com/lib/pq"
)

type NoDBFieldsEntity struct {
	OldID int32
}

type BaseEntity struct {
	ID int32 `db:"id" pk:"" json:"id,omitempty"`
}

type Entity struct {
	BaseEntity
	NoDBFieldsEntity
	Name      string    `db:"name" json:"name,omitempty"`
	Surname   string    `db:"surname" json:"surname,omitempty"`
	Values    []int32   `db:"values" json:"values,omitempty"`
	Ts        time.Time `db:"ts" json:"ts"`
	Versioned sql_base.VersionedEntity
}

const tableName = "tableName"

var (
	sqlInsert      = initSqlInsert(tableName)
	sqlUpsert      = initSqlUpsert(tableName)
	sqlSelectByID  = initSqlSelectBy(tableName, string(colID)+"=$1")
	sqlSelectByIDs = initSqlSelectBy(tableName, string(colID)+"=ANY($1::int[])")
	sqlDeleteByID  = "DELETE FROM \"" + tableName + "\" WHERE " + string(colID) + " = $1"
)

func initSqlSelectBy(tableName, whereCondition string) string {
	columns := strings.Builder{}
	for i, c := range cols() {
		colName := string(c)
		if i > 0 {
			columns.WriteString(",")
		}
		columns.WriteString(colName)
	}
	return "SELECT " + columns.String() + " FROM \"" + tableName + "\" WHERE " + whereCondition
}

func initSqlInsert(tableName string) string {
	id := string(colID)

	columns := strings.Builder{}
	indexes := strings.Builder{}

	i := 0
	for _, c := range cols() {
		if c == colID {
			continue
		}
		colName := string(c)
		colIndex := strconv.Itoa(i + 1)
		if i > 0 {
			columns.WriteString(",")
			indexes.WriteString(",")
		}
		columns.WriteString(colName)
		indexes.WriteString("$" + colIndex)
		i++
	}
	return "INSERT INTO \"" + tableName + "\" (" + columns.String() + ") VALUES (" + indexes.String() + ") RETURNING " + id
}

func initSqlUpsert(tableName string) string {
	id := string(colID)

	columns := strings.Builder{}
	indexes := strings.Builder{}
	updatePairs := strings.Builder{}

	u := 1
	for i, c := range cols() {
		colName := string(c)
		colIndex := strconv.Itoa(i + 1)
		if i > 0 {
			columns.WriteString(",")
			indexes.WriteString(",")
		}
		columns.WriteString(colName)
		indexes.WriteString("$" + colIndex)

		if c != colID {
			if u > 1 {
				updatePairs.WriteString(",")
			}
			updatePairs.WriteString(colName + "=$" + colIndex)
			u++
		}
	}
	return "INSERT INTO \"" + tableName + "\" (" + columns.String() + ") VALUES (" + indexes.String() + ") ON CONFLICT (" + id + ") DO UPDATE SET " + updatePairs.String() + " RETURNING " + id
}

func (v *Entity) values() []interface{} {
	return v.valuesExcept()
}

func (v *Entity) valuesExcept(excepts ...col) []interface{} {
	exceptSet := map[col]struct{}{}
	for _, c := range excepts {
		exceptSet[c] = struct{}{}
	}

	cols := cols()
	r := make([]interface{}, 0, len(cols)-len(exceptSet))

	for _, c := range cols {
		if _, except := exceptSet[c]; !except {
			ref := c.ref(v)
			if c == colValues {
				ref = pq.Array(ref)
			}
			r = append(r, ref)
		}
	}

	return r
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
		if err = rows.Scan(entity.values()...); err != nil {
			return nil, err
		}
		result = append(result, &entity)
	}
	return result, nil
}

func (v *Entity) Store(e RowQuerier) (int32, error) {
	sqlOp := sqlUpsert
	columns := v.values()
	if v.ID == 0 {
		sqlOp = sqlInsert
		columns = v.valuesExcept(colID)
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
