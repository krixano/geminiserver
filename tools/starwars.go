package main

import (
	"os"
	"log"
	"io/ioutil"
	"strings"
	"database/sql"
	"time"
	"context"
	// "mime"
	"fmt"

	"github.com/dhowden/tag"
	_ "github.com/nakagami/firebirdsql"
)


type ComicSeries struct {
	Id int
	Name string
	Oneoff bool
	Miniseries bool
	Crossover bool
	Date_added int
}

type ComicTPB struct {
	Id int
	Volume int
	Name string
	ComicSeriesId int
	TimelineDate string
	PublicationDate time.Time
	Date_added int
}

type ComicIssue struct {
	Id int
	Number int
	Name string
	Annual bool
	ComicSeriesId int
	ComicTPBId int
	TimelineDate string
	PublicationDate time.Time
	Publisher string // Marvel, IDW
	Date_added int
}

/*type ComicOmnibus struct {
	Id int
	Volume int
	Name string
	ComicSeriesId int
	TimelineDate string
	PublicationDate time.Time
	Date_added int
}*/

type BookSeries struct {
	Id int
	Name string
	Date_added int
}

type Book struct {
	Id int
	Number int
	Name string
	BookType string // Adult, YA, Junior, Young, ShortStory
	BookSeriesId int
	TimelineDate string
	PublicationDate time.Time
	Publisher string
	Date_added int
}

type Movie struct {
	Id int
	Name string
	TimelineDate string
	PublicationDate time.Time
	Date_added int
}

type TVShow struct {
	Id int
	Name string
	Date_added int
}

type TVShowEpisode struct {
	Id int
	Number int
	Season int
	Name string
	TvShowId int
	TimelineDate string
	PublicationDate time.Time
	Date_added int
}

func NewConn() *sql.DB {
	connectionString := "GEMINI:Blue132401!@localhost:3050/var/lib/firebird/3.0/data/starwars.fdb"
	conn, err := sql.Open("firebirdsql", connectionString)

	if err != nil {
		//panic(oops.New(err, "failed to connect to database"))
		panic(err)
	}

	return conn
}

func main() {
	conn := NewConn()
	defer conn.Close()

}
