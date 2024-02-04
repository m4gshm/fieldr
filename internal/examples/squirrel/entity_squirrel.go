package squirrel

import (
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
)

var (
	tableName     = "table_name"
)

func GetEntity(db *sql.DB, id int) (*Entity, error) {
	if sql, values, err := getSqlSelectById(tableName, id).ToSql(); err != nil {
		return nil, err
	} else if r, err := db.Query(sql, values...); err != nil {
		return nil, err
	} else if r.Next() {
		e := &Entity{}
		if err := r.Scan(e.refs()...); err != nil {
			return nil, err
		}
		return e, nil
	}
	return nil, nil
}

func (e *Entity) Store(db *sql.DB) error {
	if sql, values, err := e.getSqlUpsert(tableName).ToSql(); err != nil {
		return err
	} else if r, err := db.Exec(sql, values...); err != nil {
		return err
	} else if rowsNum, err := r.RowsAffected(); err != nil {
		fmt.Printf("stored %d row", rowsNum)
	}
	return nil
}

func getSqlSelectById(table string, id int) sq.SelectBuilder {
	return sqlSelectWhere(table, cols(), idEqualTo(id))
}

func (e *Entity) getSqlUpsert(table string) sq.Sqlizer {
	return sqlUpsert(table, pk(), cols(), e.vals())
}

func (e *Entity) vals() []interface{} {
	cols := cols()
	r := make([]interface{}, len(cols))
	for i, c := range cols {
		r[i] = e.val(c)
	}
	return r
}

func (e *Entity) refs() []interface{} {
	cols := cols()
	r := make([]interface{}, len(cols))
	for i, c := range cols {
		r[i] = e.ref(c)
	}
	return r
}

func (e *Entity) getSqlDelete(table string) sq.Sqlizer {
	return sqlDeleteWhere(table, idEqualTo(e.ID))
}

func idEqualTo(id int) sq.Eq {
	return sq.Eq{string(colID): id}
}
