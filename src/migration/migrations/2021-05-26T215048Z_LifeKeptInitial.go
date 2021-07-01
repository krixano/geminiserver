package migrations

import (
	"time"
	"database/sql"
	"context"

	"github.com/krixano/ponixserver/src/db"
	"github.com/krixano/ponixserver/src/migration/types"
)

func init() {
	registerMigration(LifeKeptInitial{})
}

type LifeKeptInitial struct{}

func (m LifeKeptInitial) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 5, 26, 21, 50, 48, 0, time.UTC))
}

func (m LifeKeptInitial) Name() string {
	return "LifeKeptInitial"
}

func (m LifeKeptInitial) DB() db.DBType {
	return db.LifeKeptDB
}

func (m LifeKeptInitial) Description() string {
	return "Initial for LifeKept DB"
}

func (m LifeKeptInitial) Up(tx *sql.Tx) error {
	_, err := tx.ExecContext(context.Background(), `
	CREATE TABLE members (
		id integer generated by default as identity primary key,
		certificate character varying(2000) NOT NULL,
		username character varying(150) NOT NULL,
		language character varying(10) NOT NULL,
		timezone character varying(255) NOT NULL,
		is_staff boolean NOT NULL,
		is_active boolean NOT NULL,
		date_joined timestamp NOT NULL
	);
	`)
	if err != nil {
		return err
	}

	return nil
}

func (m LifeKeptInitial) Down(tx *sql.Tx) error {
	panic("Not allowed")
}