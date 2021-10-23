//go:build postgres

package sql

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
)

func Test_CRUD(t *testing.T) {

	db, err := sql.Open("pgx", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if dErr := db.Close(); dErr != nil {
			t.Fatalf("goose: failed to close DB: %v\n", dErr)
		}
	}()

	if err = goose.Up(db, "."); err != nil {
		t.Fatalf("goose up err; %v", err)
		return
	}

	var (
		ts      = time.Now().UTC()
		surname = "Surname"
		name    = "Name"
	)
	e := &Entity{Name: name, Surname: surname, Ts: ts}
	var newID int
	newID, err = e.Store(db)
	if err != nil {
		t.Fatal(err)
		return
	}
	assert.Equal(t, newID, e.ID)
	id := e.ID
	assert.NotEqual(t, 0, id)

	name = "new name"
	e.Name = name

	_, err = e.Store(db)
	if err != nil {
		t.Fatal(err)
		return
	}

	byID, err := GetByID(db, id)
	if err != nil {
		t.Fatal(err)
		return
	}

	assert.Equal(t, *e, *byID)
	assert.Equal(t, name, byID.Name)
	assert.Equal(t, surname, byID.Surname)
	assert.Equal(t, ts, byID.Ts)

	deleted, err := byID.Delete(db)
	if err != nil {
		t.Fatal(err)
		return
	}
	assert.True(t, deleted, "deleted")

	deleted, err = byID.Delete(db)
	if err != nil {
		t.Fatal(err)
		return
	}
	assert.False(t, deleted, "nothing to delete")

	defer func() {
		if dErr := goose.Down(db, "."); dErr != nil {
			t.Fatalf("goose down err; %v", dErr)
		}
	}()
}
