package examples

import "strings"

func GetSqlDelete(idColumn string, placeholder string) string {
	return "delete from %s where " + idColumn + "=" + placeholder
}

func GetSqlSelect(columns []string) string {
	return "select " + strings.Join(columns, ", ") + " from %s"
}

func GetSqlInsert(columns []string, placeholder func(int) string) string {
	colExpr := ""
	placeholderExp := ""

	for i, column := range columns {
		if i > 0 {
			colExpr += ", "
			placeholderExp += ", "
		}
		colExpr += column
		placeholderExp += placeholder(i)
	}

	return "insert into %s (" + colExpr + ")" + " values(" + placeholderExp + ")"
}

func GetSqlUpdate(columns []string, placeholder func(int) string) string {
	return "update %s " + GetSqlSetExpr(columns, placeholder)
}

func GetSqlSetExpr(columns []string, placeholder func(int) string) string {
	colExpr := ""

	for i, column := range columns {
		if i > 0 {
			colExpr += ", "
		}
		colExpr += string(column) + "=" + placeholder(i)
	}

	return "set " + colExpr
}

func GetSqlUpdateByID(columns []string, idColumn string, placeholder func(int) string) string {
	return GetSqlUpdate(columns, placeholder) + " where " + idColumn + "=" + placeholder(len(columns))
}

func GetPostgresUpsert(columns []string, idColumn string, placeholder func(int) string) string {
	return GetSqlInsert(columns, placeholder) + " on conflict (" + idColumn + ") do update " + GetSqlSetExpr(columns, placeholder)
}
