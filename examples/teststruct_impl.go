package examples

import "strings"

func (t *TestStruct) SqlSelectStatement(tableName string) string {
	columns := testStruct_Tag_Values[TestStruct_db]
	return "select " + strings.Join(columns.Strings(), ", ") + " from " + tableName
}

func (t *TestStruct) SqlInsertStatement(tableName string) (string, []interface{}) {

	columns := testStruct_Tag_Values[TestStruct_db]

	values := make([]interface{}, len(columns))
	colExpr := ""
	placeholderExp := ""

	for i, column := range columns {
		values[i] = t.FieldValueByTagValue(column)
		if i > 0 {
			colExpr += ", "
			placeholderExp += ", "
		}
		colExpr += string(column)
		placeholderExp += "?"
	}

	return "insert into " + tableName + " (" + colExpr + ")" + " values(" + placeholderExp + ")", values
}
