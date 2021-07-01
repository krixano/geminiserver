package db

import (
	//"context"
	"database/sql"

	"github.com/krixano/ponixserver/src/config"
	"github.com/krixano/ponixserver/src/oops"
	_ "github.com/nakagami/firebirdsql"
	// "github.com/jackc/pgx/v4"
)

type DBType string
const (
	MusicDB = "Music"
	LifeKeptDB = "LifeKept"
	StarWarsDB = "StarWars"
	SearchDB = "Search"
)

func StringToDB(s string) DBType {
	if s == "Music" || s == "music" {
		return MusicDB
	} else if s == "LifeKept" || s == "lifekept" {
		return LifeKeptDB
	} else if s == "StarWars" || s == "starwars" {
		return StarWarsDB
	} else if s == "Search" || s == "search" {
		return SearchDB
	} else {
		return MusicDB
	}
}

func NewConn(database DBType) *sql.DB {
	var conn *sql.DB
	var err error
	if database == MusicDB {
		conn, err = sql.Open("firebirdsql", config.MusicConfig.Firebird.DSN())
	} else if database == LifeKeptDB {
		conn, err = sql.Open("firebirdsql", config.LifeKeptConfig.Firebird.DSN())
	} else if database == StarWarsDB {
		conn, err = sql.Open("firebirdsql", config.StarWarsConfig.Firebird.DSN())
	} else if database == SearchDB {
		conn, err = sql.Open("firebirdsql", config.SearchConfig.Firebird.DSN())
	}

	if err != nil {
		panic(oops.New(err, "failed to connect to database"))
	} else if err := conn.Ping(); err != nil {
		panic(err)
	}

	return conn
}


/*
func NewConnPool2(minConns, maxConns int32) *pgxpool.Pool {
	config, err := pgxpool.ParseConfig(config.SearchConfig2.Postgres.DSN())

	config.MinConns = minConns
	config.MaxConns = maxConns

	conn, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		panic(oops.New(err, "failed to create database connection pool"))
	}

	return conn
}
*/
