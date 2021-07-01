package migrations

import (
	"time"
	"database/sql"
	"context"

	"github.com/krixano/ponixserver/src/db"
	"github.com/krixano/ponixserver/src/migration/types"
)

func init() {
	registerMigration(SearchInitial{})
}

type SearchInitial struct{}

func (m SearchInitial) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 6, 8, 14, 2, 37, 0, time.UTC))
}

func (m SearchInitial) Name() string {
	return "SearchInitial"
}

func (m SearchInitial) DB() db.DBType {
	return db.SearchDB
}

func (m SearchInitial) Description() string {
	return "Initial Tables for Search DB"
}

func (m SearchInitial) Up(tx *sql.Tx) error {
	_, err := tx.ExecContext(context.Background(), `
	CREATE TABLE domains (
		id bigint generated by default as identity primary key,
		domain character varying(1015) NOT NULL COLLATE UNICODE_CI,
		title character varying(250) NOT NULL COLLATE UNICODE_CI,
		parentdomainid bigint references domains,
		crawlIndex integer,
		date_added timestamp NOT NULL
	);
	`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(context.Background(), `
	CREATE TABLE pages (
		id bigint generated by default as identity primary key,
		url character varying(1020) NOT NULL COLLATE UNICODE_CI,
		urlhash character varying(250) NOT NULL,
		scheme character varying(50) NOT NULL COLLATE UNICODE_CI,
		domainid bigint references domains,

		contenttype character varying(250) COLLATE UNICODE,
		charset character varying(250) COLLATE UNICODE,
		language character varying(250) COLLATE UNICODE,

		title character varying(250) NOT NULL COLLATE UNICODE_CI,
		prompt character varying(250) NOT NULL COLLATE UNICODE_CI,
		size integer NOT NULL,
		hash character varying(250) NOT NULL,
		feed boolean,
		publishdate timestamp NOT NULL,
		indextime timestamp NOT NULL,

		album character varying(250) NOT NULL COLLATE UNICODE_CI,
		artist character varying(250) NOT NULL COLLATE UNICODE_CI,
		albumartist character varying(250) NOT NULL COLLATE UNICODE_CI,
		composer character varying(250) NOT NULL COLLATE UNICODE_CI,
		track integer,
		disc integer,
		copyright character varying(250) NOT NULL COLLATE UNICODE_CI,
		crawlIndex integer,
		date_added timestamp NOT NULL,
		hidden boolean
	);
	`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(context.Background(), `
	CREATE TABLE keywords (
		id bigint generated by default as identity primary key,
		pageid bigint references pages,
		keyword character varying(250) NOT NULL COLLATE UNICODE_CI,
		rank float,
		crawlIndex integer,
		date_added timestamp NOT NULL
	);
	`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(context.Background(), `
	CREATE TABLE links (
		id bigint generated by default as identity primary key,
		pageid_from bigint references pages,
		pageid_to bigint references pages,
		name character varying(250) COLLATE UNICODE_CI,
		crosshost boolean,
		crawlIndex integer,
		date_added timestamp NOT NULL
	);
	`)
	if err != nil {
		return err
	}

	return nil
}

func (m SearchInitial) Down(tx *sql.Tx) error {
	panic("Implement me")
}