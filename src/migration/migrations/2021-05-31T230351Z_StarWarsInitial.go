package migrations

import (
	"time"
	"context"
	"database/sql"

	"github.com/krixano/ponixserver/src/db"
	"github.com/krixano/ponixserver/src/migration/types"
)

func init() {
	registerMigration(StarWarsInitial{})
}

type StarWarsInitial struct{}

func (m StarWarsInitial) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 5, 31, 23, 3, 51, 0, time.UTC))
}

func (m StarWarsInitial) Name() string {
	return "StarWarsInitial"
}

func (m StarWarsInitial) DB() db.DBType {
	return db.StarWarsDB
}

func (m StarWarsInitial) Description() string {
	return "Initial migration for Star Wars Database"
}

func (m StarWarsInitial) Up(tx *sql.Tx) error {
	_, err := tx.ExecContext(context.Background(), `
	CREATE TABLE comicseries (
		id integer generated by default as identity primary key,
		name character varying(250) NOT NULL,
		miniseries boolean NOT NULL,
		startyear int,
		date_added timestamp NOT NULL
	);
	`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(context.Background(), `
	CREATE TABLE comictpbs (
		id integer generated by default as identity primary key,
		volume integer,
		name character varying(250) NOT NULL,
		crossover boolean NOT NULL,
		
		comicseriesid integer references comicseries,

		timelinedate integer,
		publicationdate timestamp NOT NULL,
		date_added timestamp NOT NULL
	);
	`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(context.Background(), `
	CREATE TABLE comicissues (
		id integer generated by default as identity primary key,
		number integer,
		name character varying(250) NOT NULL,
		annual boolean NOT NULL,

		comicseriesid integer references comicseries,
		comictpbid integer references comictpbs,

		timelinedate integer,
		publicationdate timestamp NOT NULL,
		publisher character varying(250),
		date_added timestamp NOT NULL
	);
	`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(context.Background(), `
	CREATE TABLE bookseries (
		id integer generated by default as identity primary key,
		name character varying(250) NOT NULL,
		
		date_added timestamp NOT NULL
	);
	`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(context.Background(), `
	CREATE TABLE books (
		id integer generated by default as identity primary key,
		number integer,
		name character varying(250) NOT NULL,
		booktype character varying(250) NOT NULL,
		author character varying(250) NOT NULL,

		bookseriesid integer references bookseries,

		timelinedate integer,
		publicationdate timestamp NOT NULL,
		publisher character varying(250),
		date_added timestamp NOT NULL
	);
	`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(context.Background(), `
	CREATE TABLE movies (
		id integer generated by default as identity primary key,
		name character varying(250) NOT NULL,

		timelinedate integer,
		publicationdate timestamp NOT NULL,
		productioncompany character varying(250),
		distributor character varying(250),
		date_added timestamp NOT NULL
	);
	`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(context.Background(), `
	CREATE TABLE tvshows (
		id integer generated by default as identity primary key,
		name character varying(250) NOT NULL,
		
		date_added timestamp NOT NULL
	);
	`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(context.Background(), `
	CREATE TABLE tvshowepisodes (
		id integer generated by default as identity primary key,
		number integer,
		season integer,
		name character varying(250) NOT NULL,

		tvshowid integer references tvshows,
		
		timelinedate integer,
		publicationdate timestamp NOT NULL,
		date_added timestamp NOT NULL
	);
	`)
	if err != nil {
		return err
	}

	return nil
}

func (m StarWarsInitial) Down(tx *sql.Tx) error {
	panic("Dangerous")
}
