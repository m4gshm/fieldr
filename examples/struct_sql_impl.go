package examples

import (
	"fmt"
	"strconv"
)

const placeholder = "?"

var (
	columns  = struct_TagValues[StructTag_db].Strings()
	idColumn = StructField_ID

	sqlSelect = GetSqlSelect(columns)
	sqlInsert = GetSqlInsert(columns, positionPlaceholder)
	sqlUpsert = GetPostgresUpsert(columns, string(idColumn), positionPlaceholder)
	sqlDelete = GetSqlDelete(string(idColumn), placeholder)
)

func (s *Struct) sqlSelectStatement(tableName string) string {
	return fmt.Sprintf(sqlSelect, tableName)
}

func (s *Struct) sqlUpsertStatement(tableName string) (string, []interface{}) {
	return fmt.Sprintf(sqlInsert, tableName), s.GetFieldValuesByTag(StructTag_db)
}

func (s *Struct) sqlDeleteStatement(tableName string) (string, interface{}) {
	return fmt.Sprintf(sqlDelete, tableName), s.GetFieldValue(idColumn)
}

func positionPlaceholder(index int) string {
	return placeholder + strconv.Itoa(index+1)
}
