package examples

import (
	"fmt"
	"strconv"
)

const pgPlaceholderPrefix = "$"

var (
	columns  = struct_TagValues[StructTag_db].Strings()
	idColumn = StructField_ID

	sqlSelect = GetSqlSelect(columns)
	sqlUpsert = GetPostgresUpsert(columns, string(idColumn), postgresPlaceholder)
	sqlDelete = GetSqlDelete(string(idColumn), postgresPlaceholder(0))
)

var postgresPlaceholder = func(i int) string {
	return positionPlaceholder(pgPlaceholderPrefix, i)
}

func (s *Struct) sqlSelectStatement(tableName string) string {
	return fmt.Sprintf(sqlSelect, tableName)
}

func (s *Struct) sqlUpsertStatement(tableName string) (string, []interface{}) {
	return fmt.Sprintf(sqlUpsert, tableName), s.GetFieldValuesByTag(StructTag_db)
}

func (s *Struct) sqlDeleteStatement(tableName string) (string, interface{}) {
	return fmt.Sprintf(sqlDelete, tableName), s.GetFieldValue(idColumn)
}

func positionPlaceholder(prefix string, index int) string {
	return prefix + strconv.Itoa(index+1)
}
