package examples

import "strings"

func GetSqlDelete(idColumn string, placeholder string) string {
	return "DELETE FROM %s WHERE " + idColumn + "=" + placeholder
}

func GetSqlSelect(columns []string) string {
	return "SELECT " + strings.Join(columns, ", ") + " FROM %s"
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

	return "INSERT INTO %s (" + colExpr + ")" + " VALUES(" + placeholderExp + ")"
}

func GetSqlUpdate(columns []string, placeholder func(int) string) string {
	return "UPDATE %s " + GetSqlSetExpr(columns, placeholder)
}

func GetSqlSetExpr(columns []string, placeholder func(int) string) string {
	colExpr := ""

	for i, column := range columns {
		if i > 0 {
			colExpr += ", "
		}
		colExpr += string(column) + "=" + placeholder(i)
	}

	return "SET " + colExpr
}

func GetSqlUpdateByID(columns []string, idColumn string, placeholder func(int) string) string {
	return GetSqlUpdate(columns, placeholder) + " WHERE " + idColumn + "=" + placeholder(len(columns))
}

func GetPostgresUpsert(columns []string, idColumn string, placeholder func(int) string) string {
	return GetSqlInsert(columns, placeholder) + " ON CONFLICT (" + idColumn + ") DO UPDATE " + GetSqlSetExpr(columns, placeholder)
}
