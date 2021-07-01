package migrations

import (
	"time"
	"database/sql"
	"context"

	"github.com/krixano/ponixserver/src/db"
	"github.com/krixano/ponixserver/src/migration/types"
)

func init() {
	registerMigration(Uploads_Index{})
}

type Uploads_Index struct{}

func (m Uploads_Index) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 5, 24, 15, 18, 24, 0, time.UTC))
}

func (m Uploads_Index) Name() string {
	return "Uploads_Index"
}

func (m Uploads_Index) DB() db.DBType {
	return db.MusicDB
}

func (m Uploads_Index) Description() string {
	return "Create Uploads Index"
}

func (m Uploads_Index) Up(tx *sql.Tx) error {
	q := `CREATE UNIQUE INDEX IDX_UPLOADS_IDS ON UPLOADS (MEMBERID, FILEID);`
	_, err := tx.ExecContext(context.Background(), q)
	if err != nil {
		return err
	}

	return nil
}

func (m Uploads_Index) Down(tx *sql.Tx) error {
	q := `DROP INDEX IDX_UPLOADS_IDS;`
	_, err := tx.ExecContext(context.Background(), q)
	if err != nil {
		return err
	}
	
	return nil
}
