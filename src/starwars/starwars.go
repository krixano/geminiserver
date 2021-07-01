package starwars

import (
	"database/sql"
	"strings"
	"time"
	"fmt"

	"github.com/krixano/ponixserver/src/db"
	"github.com/pitr/gig"
)

func HandleStarWars(g *gig.Gig) {
	conn := db.NewConn(db.StarWarsDB)
	conn.SetMaxOpenConns(500)
	conn.SetMaxIdleConns(3)
	conn.SetConnMaxLifetime(time.Hour * 4)

	g.Handle("/starwars2", func(c gig.Context) error {
		return c.NoContent(gig.StatusRedirectPermanent, "/starwars2/")
	})

	g.Handle("/starwars2/", func(c gig.Context) error {
		return c.Gemini(`# StarWars Database

Welcome to the StarWars Database.

## Canon - Ordered By Timeline
=> /starwars2/timeline/movies Movies
=> /starwars2/timeline/shows TV Shows
=> /starwars2/timeline/comics Comics
=> /starwars2/timeline/bookseries Books
=> /starwars2/timeline/all All

## Canon - Ordered By Publication
=> /starwars2/publication/movies Movies
=> /starwars2/publication/comics Comics
`)
	})

	// Movies
	g.Handle("/starwars2/timeline/movies", func(c gig.Context) error {
		return handleMovies(c, conn, true)
	})
	g.Handle("/starwars2/publication/movies", func(c gig.Context) error {
		return handleMovies(c, conn, false)
	})

	// Movies GMICSV
	g.Handle("/starwars2/timeline/movies/csv", func(c gig.Context) error {
		return handleMoviesCSV(c, conn, true)
	})
	g.Handle("/starwars2/publication/movies/csv", func(c gig.Context) error {
		return handleMoviesCSV(c, conn, false)
	})

	g.Handle("/starwars2/timeline/shows", func(c gig.Context) error {
		shows := GetShows(conn)
		header, tableData := constructTableDataFromShows(shows)
		table := constructTable(header, tableData)

		var builder strings.Builder
		fmt.Fprintf(&builder, "```\n%s\n```\n\n", table)

		return c.Gemini(`# Star Wars Shows

=> /starwars2/ Home
=> /starwars2/timeline/shows/episodes/ Episodes

%s
`, builder.String())
	})

	g.Handle("/starwars2/timeline/comics", func(c gig.Context) error {
		fullSeries := GetComicSeries_Full(conn)
		miniseries := GetComicSeries_Miniseries(conn)
		crossovers := GetComicCrossovers(conn, true)
		oneshots := GetComicOneshots(conn, true)

		var builder strings.Builder
		fmt.Fprintf(&builder, "## Full Series\n")
		full_heading, full_data := constructTableDataFromSeries(fullSeries)
		full_table := constructTable(full_heading, full_data)
		fmt.Fprintf(&builder, "```\n%s```\n\n", full_table)
		
		fmt.Fprintf(&builder, "## Crossovers\n")
		crossovers_heading, crossovers_data := constructTableDataFromCrossover(crossovers)
		crossovers_table := constructTable(crossovers_heading, crossovers_data)
		fmt.Fprintf(&builder, "```\n%s```\n\n", crossovers_table)

		fmt.Fprintf(&builder, "## Miniseries\n")
		miniseries_heading, miniseries_data := constructTableDataFromSeries(miniseries)
		miniseries_table := constructTable(miniseries_heading, miniseries_data)
		fmt.Fprintf(&builder, "```\n%s```\n\n", miniseries_table)

		fmt.Fprintf(&builder, "## One-shots\n")
		oneshots_heading, oneshots_data := constructTableDataFromOneshots(oneshots)
		oneshots_table := constructTable(oneshots_heading, oneshots_data)
		fmt.Fprintf(&builder, "```\n%s```\n\n", oneshots_table)

		return c.Gemini(`# Star Wars Comics - Series'

=> /starwars2/ Home
=> /starwars2/timeline/comics/issues Issues
=> /starwars2/timeline/comics/tpbs TPBs

%s
`, builder.String())
	})

	g.Handle("/starwars2/publication/comics", func(c gig.Context) error {
		fullSeries := GetComicSeries_Full(conn)
		miniseries := GetComicSeries_Miniseries(conn)
		crossovers := GetComicCrossovers(conn, false)
		oneshots := GetComicOneshots(conn, false)

		var builder strings.Builder
		fmt.Fprintf(&builder, "## Full Series\n")
		full_heading, full_data := constructTableDataFromSeries(fullSeries)
		full_table := constructTable(full_heading, full_data)
		fmt.Fprintf(&builder, "```\n%s```\n\n", full_table)
		
		fmt.Fprintf(&builder, "## Crossovers\n")
		crossovers_heading, crossovers_data := constructTableDataFromCrossover(crossovers)
		crossovers_table := constructTable(crossovers_heading, crossovers_data)
		fmt.Fprintf(&builder, "```\n%s```\n\n", crossovers_table)

		fmt.Fprintf(&builder, "## Miniseries\n")
		miniseries_heading, miniseries_data := constructTableDataFromSeries(miniseries)
		miniseries_table := constructTable(miniseries_heading, miniseries_data)
		fmt.Fprintf(&builder, "```\n%s```\n\n", miniseries_table)

		fmt.Fprintf(&builder, "## One-shots\n")
		oneshots_heading, oneshots_data := constructTableDataFromOneshots(oneshots)
		oneshots_table := constructTable(oneshots_heading, oneshots_data)
		fmt.Fprintf(&builder, "```\n%s```\n\n", oneshots_table)

		return c.Gemini(`# Star Wars Comics - Series'

=> /starwars2/ Home
=> /starwars2/publication/comics/issues Issues
=> /starwars2/publication/comics/tpbs TPBs

%s
`, builder.String())
	})

	g.Handle("/starwars2/timeline/comics/tpbs", func(c gig.Context) error {
		tpbs := GetTPBs(conn, true)
		heading, data := constructTableDataFromTPBs(tpbs)
		table := constructTable(heading, data)

		var builder strings.Builder
		fmt.Fprintf(&builder, "```\n%s```\n\n", table)

		return c.Gemini(`# Star Wars Comics - TPBs

=> /starwars2/ Home
=> /starwars2/timeline/comics Comic Series'
=> /starwars2/timeline/comics/issues Issues
=> /starwars2/timeline/comics/tpbs TPBs

%s
`, builder.String())
	})

	g.Handle("/starwars2/publication/comics/tpbs", func(c gig.Context) error {
		tpbs := GetTPBs(conn, false)
		heading, data := constructTableDataFromTPBs(tpbs)
		table := constructTable(heading, data)

		var builder strings.Builder
		fmt.Fprintf(&builder, "```\n%s```\n\n", table)

		return c.Gemini(`# Star Wars Comics - TPBs

=> /starwars2/ Home
=> /starwars2/publication/comics Comic Series'
=> /starwars2/publication/comics/issues Issues
=> /starwars2/publication/comics/tpbs TPBs

%s
`, builder.String())
	})

	
	g.Handle("/starwars2/timeline/comics/issues", func(c gig.Context) error {
		issues := GetComicIssues(conn, true)
		heading, data := constructTableDataFromIssues(issues)
		table := constructTable(heading, data)

		var builder strings.Builder
		fmt.Fprintf(&builder, "```\n%s```\n\n", table)

		return c.Gemini(`# Star Wars Comics - Issues

=> /starwars2/ Home
=> /starwars2/timeline/comics Comic Series'
=> /starwars2/timeline/comics/issues Issues
=> /starwars2/timeline/comics/tpbs TPBs

%s
`, builder.String())
	})

	g.Handle("/starwars2/publication/comics/issues", func(c gig.Context) error {
		issues := GetComicIssues(conn, false)
		heading, data := constructTableDataFromIssues(issues)
		table := constructTable(heading, data)

		var builder strings.Builder
		fmt.Fprintf(&builder, "```\n%s```\n\n", table)

		return c.Gemini(`# Star Wars Comics - Issues

=> /starwars2/ Home
=> /starwars2/publication/comics Comic Series'
=> /starwars2/publication/comics/issues Issues
=> /starwars2/publication/comics/tpbs TPBs

%s
`, builder.String())
	})

	g.Handle("/starwars2/timeline/bookseries", func(c gig.Context) error {
		var builder strings.Builder

		series := GetBookSeries(conn)
		series_header, series_tableData := constructTableDataFromBookSeries(series)
		series_table := constructTable(series_header, series_tableData)
		fmt.Fprintf(&builder, "## Series'\n```\n%s\n```\n\n", series_table)

		standalones := GetBookStandalones(conn)
		standalones_header, standalones_tableData := constructTableDataFromBookStandalones(standalones)
		standalones_table := constructTable(standalones_header, standalones_tableData)
		fmt.Fprintf(&builder, "## Standalones\n```\n%s\n```\n\n", standalones_table)

		return c.Gemini(`# Star Wars Book Series'

=> /starwars2/ Home
=> /starwars2/timeline/bookseries Book Series'
=> /starwars2/timeline/books Books

%s
`, builder.String())
	})

	g.Handle("/starwars2/timeline/books", func(c gig.Context) error {
		books := GetBooks(conn)
		header, tableData := constructTableDataFromBooks(books)
		table := constructTable(header, tableData)

		var builder strings.Builder
		fmt.Fprintf(&builder, "```\n%s\n```\n\n", table)

		return c.Gemini(`# Star Wars Books

=> /starwars2/ Home
=> /starwars2/timeline/bookseries Book Series'
=> /starwars2/timeline/books Books

%s
`, builder.String())
	})
}

func handleMovies(c gig.Context, conn *sql.DB, timeline bool) error {
	movies := GetMovies(conn, timeline)
	header, tableData := constructTableDataFromMovies(movies)
	table := constructTable(header, tableData)

	var builder strings.Builder
	fmt.Fprintf(&builder, "```\n%s\n```\n", table)

	return c.Gemini(`# Star Wars Movies

=> /starwars2/ Home

%s
=> movies/csv CSV File
`, builder.String())
}

func handleMoviesCSV(c gig.Context, conn *sql.DB, timeline bool) error {
	movies := GetMovies(conn, timeline)
	header, tableData := constructTableDataFromMovies(movies)

	var builder strings.Builder
	for colNum, col := range header {
		fmt.Fprintf(&builder, "%s", col)
		if colNum < len(header) - 1 {
			fmt.Fprintf(&builder, ",")
		}
	}
	fmt.Fprintf(&builder, "\n")

	for _, row := range tableData {
		for colNum, col := range row {
			fmt.Fprintf(&builder, "%s", col)
			if colNum < len(row) - 1 {
				fmt.Fprintf(&builder, ",")
			}
		}
		fmt.Fprintf(&builder, "\n")
	}

	return c.Blob("text/csv", []byte(builder.String()))
}
