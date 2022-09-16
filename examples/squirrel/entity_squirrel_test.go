package squirrel

import (
	"example/sql_base"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSqlUpsert(t *testing.T) {

	entity := Entity{
		ID:        1,
		Name:      "test name",
		Surname:   "test surname",
		ts:        time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC),
		Versioned: sql_base.VersionedEntity{Version: 111},
	}

	sql, values, err := entity.getSqlUpsert("test_table").ToSql()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []interface{}{entity.ID, entity.Name, entity.Surname, entity.Versioned.Version, entity.Name, entity.Surname, entity.Versioned.Version}, values)
	assert.Equal(t, "INSERT INTO test_table (ID,NAME,SURNAME,version) VALUES ($1,$2,$3,$4) ON CONFLICT (ID) DO UPDATE   SET NAME = $5, SURNAME = $6, version = $7", sql)
}

func TestSqlSelectByID(t *testing.T) {
	buildr := getSqlSelectById("test_table", 100)
	if sql, values, err := buildr.ToSql(); err != nil {
		t.Fatal(err)
	} else {
		assert.Equal(t, []interface{}{100}, values)
		assert.Equal(t, "SELECT ID, NAME, SURNAME, version FROM test_table WHERE ID = $1", sql)
	}
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
