package squirrel

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/pressly/goose/v3"

	// "github.com/rs/zerolog"
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
	dsn = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
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
	}

	var (
		surname = "Surname"
		name    = "Name"
		id      = 1
		e       = &Entity{ID: id, Name: name, Surname: surname}
	)
	if err = e.Store(db); err != nil {
		t.Fatal(err)
	}

	name = "new name"
	e.Name = name
	if err = e.Store(db); err != nil {
		t.Fatal(err)
	}

	if byID, err := GetEntity(db, id); err != nil {
		t.Fatal(err)
	} else {
		assert.Equal(t, *e, *byID)
		assert.Equal(t, name, byID.Name)
		assert.Equal(t, surname, byID.Surname)
	}

	defer func() {
		if dErr := goose.Down(db, "."); dErr != nil {
			t.Fatalf("goose down err; %v", dErr)
		}
	}()
}
