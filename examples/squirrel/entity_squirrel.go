package squirrel

import (
	sq "github.com/Masterminds/squirrel"
)

var (
	pkColumn      = string(colID)
	dbColumnNames = strings(cols())
)

func getSqlSelectById(table string, id int) sq.SelectBuilder {
	return sqlSelectWhere(table, dbColumnNames, idEqualTo(id))
}

func (e *Entity) getSqlUpsert(table string) sq.Sqlizer {
	return sqlUpsert(table, pkColumn, dbColumnNames, e.vals())
}

func (e *Entity) vals() []interface{} {
	cols := cols()
	r := make([]interface{}, len(cols))
	for i, c := range cols {
		r[i] = c.val(e)
	}
	return r
}

func (e *Entity) refs() []interface{} {
	cols := cols()
	r := make([]interface{}, len(cols))
	for i, c := range cols {
		r[i] = c.ref(e)
	}
	return r
}

func (e *Entity) getSqlDelete(table string) sq.Sqlizer {
	return sqlDeleteWhere(table, idEqualTo(e.ID))
}

func idEqualTo(id int) sq.Eq {
	return sq.Eq{string(colID): id}
}
