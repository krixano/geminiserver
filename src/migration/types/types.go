package types

import (
	"time"
	"database/sql"
	"github.com/krixano/ponixserver/src/db"
)

type Migration interface {
	Version() MigrationVersion
	Name() string
	DB() db.DBType
	Description() string
	Up(conn *sql.Tx) error
	Down(conn *sql.Tx) error
}

type MigrationVersion time.Time

func (v MigrationVersion) String() string {
	return time.Time(v).Format(time.RFC3339)
}

func (v MigrationVersion) Before(other MigrationVersion) bool {
	return time.Time(v).Before(time.Time(other))
}

func (v MigrationVersion) Equal(other MigrationVersion) bool {
	return time.Time(v).Equal(time.Time(other))
}

func (v MigrationVersion) IsZero() bool {
	return time.Time(v).IsZero()
}
