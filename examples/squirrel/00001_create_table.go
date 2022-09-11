package squirrel

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

var (
	createTableSql = `create table ` + tableName + `
(
	id int constraint test_table_pk primary key,
	name text,
	surname text,
	values int[]
)`
)

func init() {
	goose.AddMigration(Up00001, Down00002)
}

func Up00001(tx *sql.Tx) error {
	if _, err := tx.Exec(createTableSql); err != nil {
		return err
	}
	return nil
}

func Down00002(tx *sql.Tx) error {
	_, err := tx.Exec(`drop table "` + tableName + `";`)
	if err != nil {
		return err
	}
	return nil
}
