package sql

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	sqldblogger "github.com/simukti/sqldb-logger"
	"github.com/simukti/sqldb-logger/logadapter/zerologadapter"
	"github.com/stretchr/testify/assert"
)

func Test_CRUD(t *testing.T) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	env := "POSTGRES_TEST_DSN"
	dsn := os.Getenv(env)
	// dsn = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	if dsn == "" {
		t.Skip("set '" + env + "' to run this test")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}

	db = sqldblogger.OpenDriver(dsn, db.Driver(), zerologadapter.New(log.Logger))
	if err := db.Ping(); err != nil {
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
		ts      = time.Now().Round(time.Second).UTC()
		surname = "Surname"
		name    = "Name"
	)
	e := &Entity{Name: name, Surname: surname, Ts: ts, Values: []int32{1, 2, 3, 4}}
	var newID int32
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

	byIDs, err := GetByIDs(db, id, -1)
	if err != nil {
		t.Fatal(err)
		return
	}

	assert.Equal(t, *e, *(byIDs[0]))

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
