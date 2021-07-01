package migrations

import (
	"time"
	"database/sql"
	"context"

	"github.com/krixano/ponixserver/src/db"
	"github.com/krixano/ponixserver/src/migration/types"
)

func init() {
	registerMigration(LIBRARY_HASH{})
}

type LIBRARY_HASH struct{}

func (m LIBRARY_HASH) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 5, 24, 15, 42, 35, 0, time.UTC))
}

func (m LIBRARY_HASH) Name() string {
	return "LIBRARY_HASH"
}

func (m LIBRARY_HASH) DB() db.DBType {
	return db.MusicDB
}

func (m LIBRARY_HASH) Description() string {
	return "Create Library Index on filehash"
}

func (m LIBRARY_HASH) Up(tx *sql.Tx) error {
	q := `CREATE INDEX IDX_LIBRARY_HASH ON LIBRARY (FILEHASH);`
	_, err := tx.ExecContext(context.Background(), q)
	if err != nil {
		return err
	}

	return nil
}

func (m LIBRARY_HASH) Down(tx *sql.Tx) error {
	q := `DROP INDEX IDX_LIBRARY_HASH;`
	_, err := tx.ExecContext(context.Background(), q)
	if err != nil {
		return err
	}
	
	return nil
}
