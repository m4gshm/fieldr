package squirrel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSqlUpsert(t *testing.T) {

	entity := Entity{
		ID:      1,
		Name:    "test name",
		Surname: "test surname",
		ts:      time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC),
	}

	sql, values, err := entity.getSqlUpsert("test_table").ToSql()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []interface{}{entity.ID, entity.Name, entity.Surname, entity.Name, entity.Surname}, values)
	assert.Equal(t, "INSERT INTO test_table (ID,NAME,SURNAME) VALUES ($1,$2,$3) ON CONFLICT (ID) DO UPDATE   SET NAME = $4, SURNAME = $5", sql)
}

func TestSqlSelectByID(t *testing.T) {
	sql, values, err := getSqlSelectById("test_table", 100).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []interface{}{100}, values)
	assert.Equal(t, "SELECT ID, NAME, SURNAME FROM test_table WHERE ID = $1", sql)
}

func TestSqlDeleteByID(t *testing.T) {
	entity := Entity{
		ID:      1,
		Name:    "test name",
		Surname: "test surname",
		ts:      time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC),
	}

	sql, values, err := entity.getSqlDelete("test_table").ToSql()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []interface{}{1}, values)
	assert.Equal(t, "DELETE FROM test_table WHERE ID = $1", sql)
}
