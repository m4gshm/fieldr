package squirrel

import (
	sq "github.com/Masterminds/squirrel"
)

var (
	pkColumn      = string(entityTagValueDbID)
	dbColumnNames = entityTagValuesDb.strings()
)

func getSqlSelectById(table string, id int) sq.Sqlizer {
	return sqlSelectWhere(table, dbColumnNames, idEqualTo(id))
}

func (e *Entity) getSqlUpsert(table string) sq.Sqlizer {
	return sqlUpsert(table, pkColumn, dbColumnNames, e.getFieldValuesByTagDb())
}

func (e *Entity) getSqlDelete(table string) sq.Sqlizer {
	return sqlDeleteWhere(table, idEqualTo(e.ID))
}

func idEqualTo(id int) sq.Eq {
	return sq.Eq{string(entityTagValueDbID): id}
}
