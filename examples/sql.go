package examples

import (
	"strings"
)

func GetSqlDelete(idColumn StructField, placeholder string) string {
	return "delete from %s where " + string(idColumn) + "=" + placeholder
}

func GetSqlSelect(columns StructTagValues) string {
	return "select " + strings.Join(columns.Strings(), ", ") + " from %s"
}

func GetSqlInsert(columns StructTagValues, placeholder func(int) string) string {
	colExpr := ""
	placeholderExp := ""

	for i, column := range columns {
		if i > 0 {
			colExpr += ", "
			placeholderExp += ", "
		}
		colExpr += string(column)
		placeholderExp += placeholder(i)
	}

	return "insert into %s (" + colExpr + ")" + " values(" + placeholderExp + ")"
}

func GetSqlUpdate(columns StructTagValues, placeholder func(int) string) string {
	return "update %s " + GetSqlSetExpr(columns, placeholder)
}

func GetSqlSetExpr(columns StructTagValues, placeholder func(int) string) string {
	colExpr := ""

	for i, column := range columns {
		if i > 0 {
			colExpr += ", "
		}
		colExpr += string(column) + "=" + placeholder(i)
	}

	return "set " + colExpr
}

func GetSqlUpdateByID(columns StructTagValues, idColumn StructField, placeholder func(int) string) string {
	return GetSqlUpdate(columns, placeholder) + " where " + string(idColumn) + "=" + placeholder(len(columns))
}

func GetPostgresUpsert(columns StructTagValues, idColumn StructField, placeholder func(int) string) string {
	return GetSqlInsert(columns, placeholder) + " on conflict (" + string(idColumn) + ") do update " + GetSqlSetExpr(columns, placeholder)
}
