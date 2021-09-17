package examples

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStruct_SqlStatements(t *testing.T) {

	tableName := "test_table"
	sqlSelect := testStruct.sqlSelectStatement(tableName)
	sqlUpsert, _ := testStruct.sqlUpsertStatement(tableName)
	sqlDelete, _ := testStruct.sqlDeleteStatement(tableName)

	assert.Equal(t, "SELECT ID, NAME, SURNAME, NO_JSON FROM "+tableName, sqlSelect)
	assert.Equal(t, "INSERT INTO "+tableName+" (ID, NAME, SURNAME, NO_JSON) VALUES($1, $2, $3, $4) ON CONFLICT (ID) DO UPDATE SET ID=$1, NAME=$2, SURNAME=$3, NO_JSON=$4", sqlUpsert)
	assert.Equal(t, "DELETE FROM test_table WHERE ID=$1", sqlDelete)

}
