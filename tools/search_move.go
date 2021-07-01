package main

import (
	"context"
	"database/sql"
	"time"
	_ "github.com/nakagami/firebirdsql"
)

type Domain struct {
	Id int64
	Domain string
	Port int
	//ParentDomain Domain // ForeignKey
	ParentDomainId int64
	//Robots string // contents of robots.txt?
	Date_added time.Time
}

// Domains can have multiple schemes
/*type DomainScheme struct {
	Id int
	//Domain Domain
	DomainId int
	Scheme string
}*/

type Page struct {
	Id int64
	Url string // fetchable_url, normalized_url
	UrlHash string
	Scheme string
	// Domain Domain // foreign key
	DomainId interface{}

	Content_type string
	Charset string
	Language string

	Title string // Used for text/gemini and text/markdown files with page titles
	// content []u8 // TODO
	Prompt string // For input prompt urls
	Size int // bytes
	Hash string
	Feed bool // rss, atom, or gmisub
	PublishDate time.Time // Used if linked from a feed, or if audio/video with year tag
	Index_time time.Time

	// Audio/Video-only info
	Album string
	Artist string
	AlbumArtist string
	Composer string
	Track int
	Disc int
	Copyright string
	CrawlIndex int
	Date_added time.Time

	Hidden bool
}

type Keyword struct {
	Id int64
	PageId int64
	Keyword string
	Rank float64
	CrawlIndex int
	Date_added time.Time
}

type Link struct {
	Id int64
	FromPageId int64
	ToPageId int64
	Name string
	Cross_host bool
	Date_added time.Time
}

func NewConnOld() *sql.DB {
	connectionString := "GEMINI:Blue132401!@localhost:3050/var/lib/firebird/3.0/data/search.fdb"
	conn, err := sql.Open("firebirdsql", connectionString)

	if err != nil {
		//panic(oops.New(err, "failed to connect to database"))
		panic(err)
	}

	return conn
}

func NewConnNew() *sql.DB {
	connectionString := "GEMINI:Blue132401!@10.0.0.20:3050/var/lib/firebird/3.0/data/search.fdb"
	conn, err := sql.Open("firebirdsql", connectionString)

	if err != nil {
		//panic(oops.New(err, "failed to connect to database"))
		panic(err)
	}

	return conn
}

func main() {
	connOld := NewConnOld()
	connNew := NewConnNew()

	//movePages(connOld, connNew)
	moveKeywords(connOld, connNew)
}

func movePages(connOld *sql.DB, connNew *sql.DB) {
	q := `SELECT id, url, urlhash, scheme, contenttype, charset, language, title, prompt, size, hash, feed, publishdate, indextime, album, artist, albumartist, composer, track, disc, copyright, crawlIndex, date_added, hidden FROM PAGES`
	rows, rows_err := connOld.QueryContext(context.Background(), q)

	if rows_err == nil {
		defer rows.Close()
		for rows.Next() {
			var page Page
			scan_err := rows.Scan(&page.Id, &page.Url, &page.UrlHash, &page.Scheme, &page.Content_type, &page.Charset, &page.Language, &page.Title, &page.Prompt, &page.Size, &page.Hash, &page.Feed, &page.PublishDate, &page.Index_time, &page.Album, &page.Artist, &page.AlbumArtist, &page.Composer, &page.Track, &page.Disc, &page.Copyright, &page.CrawlIndex, &page.Date_added, &page.Hidden)
			if scan_err == nil {
				q2 := `INSERT INTO pages (url, urlhash, scheme, contenttype, charset, language, title, prompt, size, hash, feed, publishdate, indextime, album, artist, albumartist, composer, track, disc, copyright, crawlIndex, date_added, hidden) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'false')`
				_, err := connNew.ExecContext(context.Background(), q2, page.Url, page.UrlHash, page.Scheme, page.Content_type, page.Charset, page.Language, page.Title, page.Prompt, page.Size, page.Hash, page.Feed, page.PublishDate, page.Index_time, page.Album, page.Artist, page.AlbumArtist, page.Composer, page.Track, page.Disc, page.Copyright, page.CrawlIndex, page.Date_added)
				if err != nil {
					panic(err)
				}
			} else {
				panic(scan_err)
			}
		}
	}
}

func moveKeywords(connOld *sql.DB, connNew *sql.DB) {
	q := `SELECT * FROM KEYWORDS`
	rows, rows_err := connOld.QueryContext(context.Background(), q)

	if rows_err == nil {
		defer rows.Close()
		for rows.Next() {
			var keyword Keyword
			scan_err := rows.Scan(&keyword.Id, &keyword.PageId, &keyword.Keyword, &keyword.Rank, &keyword.CrawlIndex, &keyword.Date_added)
			if scan_err == nil {
				q2 := `INSERT INTO keywords (pageid, keyword, rank, crawlindex, date_added) VALUES (?, ?, ?, ?, ?)`
				_, err := connNew.ExecContext(context.Background(), q2, keyword.PageId, keyword.Keyword, keyword.Rank, keyword.CrawlIndex, keyword.Date_added)
				if err != nil {
					panic(err)
				}
			} else {
				panic(scan_err)
			}
		}
	}
}
