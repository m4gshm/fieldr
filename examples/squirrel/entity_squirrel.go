package squirrel

import (
	sq "github.com/Masterminds/squirrel"
)

var (
	pkColumn      = entityTagValue_db_ID
	dbTag         = entityTag_db
	dbColumnNames = entity_TagValues_db.strings()
)

func getSqlSelectById(table string, id int) sq.Sqlizer {
	return sqlSelectWhere(table, dbColumnNames, idEqualTo(id))
}

func (e *Entity) getSqlUpsert(table string) sq.Sqlizer {
	return sqlUpsert(table, string(pkColumn), dbColumnNames, e.getFieldValuesByTag(dbTag))
}

func (e *Entity) getSqlDelete(table string) sq.Sqlizer {
	return sqlDeleteWhere(table, idEqualTo(e.ID))
}

func idEqualTo(id int) sq.Eq {
	return sq.Eq{string(entityTagValue_db_ID): id}
}
