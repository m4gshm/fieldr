package sql

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(Up00001, Down00002)
}

func Up00001(tx *sql.Tx) error {
	query := `create table "tableName"
(
	id serial constraint table_name_pk primary key,
	name text,
	surname text,
	values int[],
	ts timestamp
);`
	if _, err := tx.Exec(query); err != nil {
		return err
	}
	return nil
}

func Down00002(tx *sql.Tx) error {
	_, err := tx.Exec(`drop table "tableName";`)
	if err != nil {
		return err
	}
	return nil
}
