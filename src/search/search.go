package search

import (
	"context"
	"database/sql"
	"strings"
	"fmt"
	"time"
	"net/url"

	"github.com/pitr/gig"
	"github.com/krixano/ponixserver/src/db"
)

/*
var searchQuery string = `SELECT r.id, r.url, r.urlhash, r.scheme, r.domainid, r.contenttype, r.charset, r.language, r.title, r.prompt, r.size, r.hash, r.feed, r.publishdate, r.indextime, r.album, r.artist, r.albumartist, r.composer, r.track, r.disc, r.copyright, r.crawlindex, r.date_added, r.hidden
FROM (
    SELECT internal.*, (SELECT SUM(RANK) FROM keywords WHERE pageid=internal.id AND (%%matches%%)) as searchrank
    FROM PAGES internal
) r
WHERE r.searchrank IS NOT NULL 
OR (lower(title) LIKE ? OR lower(artist) LIKE ? 
OR lower(album) LIKE ? OR lower(albumartist) LIKE ?)
ORDER BY r.searchrank DESC`
*/

var searchQueryExperimental string = `
SELECT FIRST 800 r.id, r.url, r.urlhash, r.scheme, r.domainid, r.contenttype, r.charset, r.language, r.title, r.prompt, r.size, r.hash, r.feed, r.publishdate, r.indextime, r.album, r.artist, r.albumartist, r.composer, r.track, r.disc, r.copyright, r.crawlindex, r.date_added, r.hidden
FROM (
    SELECT internal.*, (
            %%matches%%
        ) as searchrank
    FROM PAGES internal
) r
WHERE r.searchrank > 0
ORDER BY r.searchrank DESC
`

/*
var searchQueryExperimental2 string = `
SELECT replace(replace(r.url, 'gemini://', ''), 'gemini.', ''), r.*
FROM (
    SELECT internal.*, (
            (SELECT COUNT(*) * 2 FROM pages WHERE id=internal.id AND lower(pages.title) SIMILAR TO '(% |)gemini( %|)')
            + (SELECT COUNT(*) * 2 FROM pages WHERE id=internal.id AND lower(pages.title) SIMILAR TO '(% |)project( %|)')
            + (SELECT COUNT(*) * 4 FROM pages WHERE id=internal.id AND (lower(pages.title) SIMILAR TO '(% |)project gemini( %|)'))
            + (SELECT CASE WHEN MAX(RANK) IS NULL THEN 0.0 ELSE MAX(RANK) END FROM keywords WHERE pageid=internal.id AND (keywords.keyword LIKE 'project%' OR keywords.keyword LIKE 'gemini%'))
            + ((SELECT CASE WHEN MAX(RANK) IS NULL THEN 0.0 ELSE MAX(RANK) END FROM keywords WHERE pageid=internal.id AND keywords.keyword LIKE 'project gemini%'))
        ) as searchrank
    FROM PAGES internal
) r
WHERE r.searchrank > 0
  OR lower(r.artist) LIKE '%gemini%' 
OR lower(r.album) LIKE '%gemini%' OR lower(r.albumartist) LIKE '%gemini%'
ORDER BY r.searchrank DESC
`
*/

func HandleSearchEngine(g *gig.Gig) {
	conn := db.NewConn(db.SearchDB)
	/*conn.SetMaxOpenConns(500)
	conn.SetConnMaxIdleTime(0)
	conn.SetMaxIdleConns(6)
	conn.SetConnMaxLifetime(0)*/

	g.Handle("/searchengine", func(c gig.Context) error {
		return c.NoContent(gig.StatusRedirectPermanent, "/searchengine/")
	})
	g.Handle("/searchengine/", func(c gig.Context) error {
		row := conn.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM pages")
		pagesCount := 0
		row.Scan(&pagesCount)

		mimetypesList := getMimetypes(conn)
		var mimetypes strings.Builder
		for _, item := range mimetypesList {
			fmt.Fprintf(&mimetypes, "=> /searchengine/mimetype?%s %s (%d)\n", url.QueryEscape(item.mimetype), item.mimetype, item.count)
		}


		return c.Gemini(`# Ponix Search Engine

Currently Indexing...
Current Page Count: %d

=> /searchengine/search Search
=> /searchengine/recent 50 Most Recently Indexed
=> /searchengine/audio Indexed Audio Files
=> /searchengine/images Indexed Image Files

## Mimetypes
%s

## Support

Want to help support the project? Consider donating on the Patreon. The first goal is to get a server from a server hosting provider that could better support all of the projects I have planned.

=> https://www.patreon.com/krixano Patreon
`, pagesCount, mimetypes.String())
	})

	g.Handle("/searchengine/search", func(c gig.Context) error {
		query, err2 := c.QueryString()
		if err2 != nil {
			return err2
		} else if query == "" {
			return c.NoContent(gig.StatusInput, "Search Query:")
		} else {
			return handleSearch(c, conn, query)
		}
	})

	g.Handle("/searchengine/recent", func(c gig.Context) error {
		pages := getRecent(conn)

		var builder strings.Builder
		for _, page := range pages {
			artist := ""
			if page.AlbumArtist != "" {
				artist = "(" + page.AlbumArtist + ")"
			} else if page.Artist != "" {
				artist = "(" + page.Artist + ")"
			}
			if page.Title == "" {
				fmt.Fprintf(&builder, "=> %s %s %s\n", page.Url, page.Url, artist)
			} else {
				fmt.Fprintf(&builder, "=> %s %s %s\n", page.Url, page.Title, artist)
			}
		}

		return c.Gemini(`# 50 Most Recently Indexed

=> /searchengine/ Home
=> /searchengine/search Search

%s
`, builder.String())
	})

	g.Handle("/searchengine/audio", func(c gig.Context) error {
		pages := getAudioFiles(conn)

		var builder strings.Builder
		for _, page := range pages {
			artist := ""
			if page.AlbumArtist != "" {
				artist = "(" + page.AlbumArtist + ")"
			} else if page.Artist != "" {
				artist = "(" + page.Artist + ")"
			}
			if page.Title == "" {
				fmt.Fprintf(&builder, "=> %s %s %s\n", page.Url, page.Url, artist)
			} else {
				fmt.Fprintf(&builder, "=> %s %s %s\n", page.Url, page.Title, artist)
			}
		}

		return c.Gemini(`# Indexed Audio Files

=> /searchengine/ Home
=> /searchengine/search Search

%s
`, builder.String())
	})

	g.Handle("/searchengine/images", func(c gig.Context) error {
		pages := getImageFiles(conn)

		var builder strings.Builder
		for _, page := range pages {
			artist := ""
			if page.AlbumArtist != "" {
				artist = "(" + page.AlbumArtist + ")"
			} else if page.Artist != "" {
				artist = "(" + page.Artist + ")"
			}
			if page.Title == "" {
				fmt.Fprintf(&builder, "=> %s %s %s\n", page.Url, page.Url, artist)
			} else {
				fmt.Fprintf(&builder, "=> %s %s %s\n", page.Url, page.Title, artist)
			}
		}

		return c.Gemini(`# Indexed Image Files

=> /searchengine/ Home
=> /searchengine/search Search

%s
`, builder.String())
	})

	g.Handle("/searchengine/mimetype", func(c gig.Context) error {
		query, err2 := c.QueryString()
		if err2 != nil {
			return err2
		} else if query == "" {
			return c.NoContent(gig.StatusInput, "Mimetype:")
		} else {
			pages := getMimetypeFiles(conn, query)

			var builder strings.Builder
			for _, page := range pages {
				artist := ""
				if page.AlbumArtist != "" {
					artist = "(" + page.AlbumArtist + ")"
				} else if page.Artist != "" {
					artist = "(" + page.Artist + ")"
				}
				if page.Title == "" {
					fmt.Fprintf(&builder, "=> %s %s %s\n", page.Url, page.Url, artist)
				} else {
					fmt.Fprintf(&builder, "=> %s %s %s\n", page.Url, page.Title, artist)
				}
			}

			return c.Gemini(`# Indexed of Mimetype '%s'

=> /searchengine/ Home
=> /searchengine/search Search

%s
`, query, builder.String())
		}
	})
}

func handleSearch(c gig.Context, conn *sql.DB, query string) error {
	//likeQuery := "%" + query + "%"
	//likeQuery2 := queryLower + "%"

	// Escape single quotes ('test' => '''test''')
	queryFiltered := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(query, "\n", " "), "\r", ""), "'", "''")

	parts := strings.Split(queryFiltered, " ")
	var queryBuilder strings.Builder
	//fmt.Fprintf(&queryBuilder, "keywords.keyword LIKE '%s%%'", query)
	fmt.Fprintf(&queryBuilder, `(SELECT COUNT(*) * 10 FROM pages WHERE id=internal.id AND pages.title SIMILAR TO '(%% |)%s( %%|)')`, queryFiltered)
	for _, part := range parts {
		if part == "" {
			continue
		}
		/*if i != 0 {
			fmt.Fprintf(&queryBuilder, `+ `)
		}*/
		fmt.Fprintf(&queryBuilder, `+ (SELECT COUNT(*) * 10 FROM pages WHERE id=internal.id AND (pages.title SIMILAR TO '(%% |)%s( %%|)' OR pages.artist SIMILAR TO '(%% |)%s( %%|)' OR replace(pages.url, 'gemini', '') LIKE '%%%s%%')) `, part, part, part)
		fmt.Fprintf(&queryBuilder, `+ (SELECT CASE WHEN MAX(RANK) IS NULL THEN 0.0 ELSE MAX(RANK) END FROM keywords WHERE pageid=internal.id AND keywords.keyword LIKE '%s%%' AND RANK <= 100)`, part)
	}

	actualQuery := strings.Replace(searchQueryExperimental, `%%matches%%`, queryBuilder.String(), 1)
	//q := `SELECT id, url, urlhash, scheme, domainid, contenttype, charset, language, title, prompt, size, hash, feed, publishdate, indextime, album, artist, albumartist, composer, track, disc, copyright, crawlindex, date_added, hidden FROM pages WHERE lower(url) LIKE lower(?) OR lower(title) LIKE lower(?) OR lower(artist) LIKE lower(?) OR lower(album) LIKE lower(?) OR lower(albumartist) LIKE lower(?) OR id IN (SELECT keywords.pageid FROM keywords where lower(keywords.keyword) LIKE ?)`	

	//fmt.Printf("Query: %s", queryBuilder.String())

	fmt.Printf("%s\n", actualQuery)

	before := time.Now()
	rows, rows_err := conn.QueryContext(context.Background(), actualQuery)
	after := time.Now()
	timeTaken := after.Sub(before)
	fmt.Printf("Time taken: %v\n", timeTaken)

	var pages []Page = make([]Page, 0, 20)
	if rows_err == nil {
		defer rows.Close()
		for rows.Next() {
			var page pageNullable
			//var rank interface{}
			scan_err := rows.Scan(&page.Id, &page.Url, &page.UrlHash, &page.Scheme, &page.DomainId, &page.Content_type, &page.Charset, &page.Language, &page.Title, &page.Prompt, &page.Size, &page.Hash, &page.Feed, &page.PublishDate, &page.Index_time, &page.Album, &page.Artist, &page.AlbumArtist, &page.Composer, &page.Track, &page.Disc, &page.Copyright, &page.CrawlIndex, &page.Date_added, &page.Hidden)
			if scan_err == nil {
				pages = append(pages, scanPage(page))
			} else {
				panic(scan_err)
			}
		}
	} else {
		panic(rows_err)
	}

	var builder strings.Builder
	for _, page := range pages {
		artist := ""
		if page.AlbumArtist != "" {
			artist = "(" + page.AlbumArtist + ")"
		} else if page.Artist != "" {
			artist = "(" + page.Artist + ")"
		}
		if page.Title == "" {
			fmt.Fprintf(&builder, "=> %s %s %s\n", page.Url, page.Url, artist)
		} else {
			fmt.Fprintf(&builder, "=> %s %s %s\n", page.Url, page.Title, artist)
		}
	}

	return c.Gemini(`# Ponix Search Engine - Results

=> /searchengine/ Home
=> /searchengine/search New Search

Query: '%s'
Time Taken: %v

%s
`, query, timeTaken, builder.String())
}

func getRecent(conn *sql.DB) []Page {
	q := `SELECT FIRST 50 id, url, urlhash, scheme, domainid, contenttype, charset, language, title, prompt, size, hash, feed, publishdate, indextime, album, artist, albumartist, composer, track, disc, copyright, crawlindex, date_added, hidden FROM pages ORDER BY date_added DESC`

	rows, rows_err := conn.QueryContext(context.Background(), q)

	var pages []Page
	if rows_err == nil {
		defer rows.Close()
		for rows.Next() {
			var page pageNullable
			scan_err := rows.Scan(&page.Id, &page.Url, &page.UrlHash, &page.Scheme, &page.DomainId, &page.Content_type, &page.Charset, &page.Language, &page.Title, &page.Prompt, &page.Size, &page.Hash, &page.Feed, &page.PublishDate, &page.Index_time, &page.Album, &page.Artist, &page.AlbumArtist, &page.Composer, &page.Track, &page.Disc, &page.Copyright, &page.CrawlIndex, &page.Date_added, &page.Hidden)
			if scan_err == nil {
				pages = append(pages, scanPage(page))
			} else {
				panic(scan_err)
			}
		}
	}

	return pages
}

func getMimetypeFiles(conn *sql.DB, mimetype string) []Page {
	q := `SELECT id, url, urlhash, scheme, domainid, contenttype, charset, language, title, prompt, size, hash, feed, publishdate, indextime, album, artist, albumartist, composer, track, disc, copyright, crawlindex, date_added, hidden FROM pages WHERE contenttype=?`

	rows, rows_err := conn.QueryContext(context.Background(), q, mimetype)

	var pages []Page
	if rows_err == nil {
		defer rows.Close()
		for rows.Next() {
			var page pageNullable
			scan_err := rows.Scan(&page.Id, &page.Url, &page.UrlHash, &page.Scheme, &page.DomainId, &page.Content_type, &page.Charset, &page.Language, &page.Title, &page.Prompt, &page.Size, &page.Hash, &page.Feed, &page.PublishDate, &page.Index_time, &page.Album, &page.Artist, &page.AlbumArtist, &page.Composer, &page.Track, &page.Disc, &page.Copyright, &page.CrawlIndex, &page.Date_added, &page.Hidden)
			if scan_err == nil {
				pages = append(pages, scanPage(page))
			} else {
				panic(scan_err)
			}
		}
	}

	return pages
}

type MimetypeListItem struct {
	mimetype string
	count int
}

func getMimetypes(conn *sql.DB) []MimetypeListItem {
	var mimetypes []MimetypeListItem
	rows, rows_err := conn.QueryContext(context.Background(), "SELECT contenttype, COUNT(*) FROM pages GROUP BY contenttype ORDER BY COUNT(*) DESC")
	if rows_err == nil {
		defer rows.Close()
		for rows.Next() {
			var item MimetypeListItem
			scan_err := rows.Scan(&item.mimetype, &item.count)
			if scan_err == nil {
				mimetypes = append(mimetypes, item)
			} else {
				panic(scan_err)
			}
		}
	}

	return mimetypes
}

func getAudioFiles(conn *sql.DB) []Page {
	q := `SELECT id, url, urlhash, scheme, domainid, contenttype, charset, language, title, prompt, size, hash, feed, publishdate, indextime, album, artist, albumartist, composer, track, disc, copyright, crawlindex, date_added, hidden FROM pages WHERE contenttype IN ('audio/mpeg', 'audio/mp3', 'audio/ogg', 'audio/flac', 'audio/mid', 'audio/m4a', 'audio/x-flac')`

	rows, rows_err := conn.QueryContext(context.Background(), q)

	var pages []Page
	if rows_err == nil {
		defer rows.Close()
		for rows.Next() {
			var page pageNullable
			scan_err := rows.Scan(&page.Id, &page.Url, &page.UrlHash, &page.Scheme, &page.DomainId, &page.Content_type, &page.Charset, &page.Language, &page.Title, &page.Prompt, &page.Size, &page.Hash, &page.Feed, &page.PublishDate, &page.Index_time, &page.Album, &page.Artist, &page.AlbumArtist, &page.Composer, &page.Track, &page.Disc, &page.Copyright, &page.CrawlIndex, &page.Date_added, &page.Hidden)
			if scan_err == nil {
				pages = append(pages, scanPage(page))
			} else {
				panic(scan_err)
			}
		}
	}

	return pages
}

func getImageFiles(conn *sql.DB) []Page {
	q := `SELECT id, url, urlhash, scheme, domainid, contenttype, charset, language, title, prompt, size, hash, feed, publishdate, indextime, album, artist, albumartist, composer, track, disc, copyright, crawlindex, date_added, hidden FROM pages WHERE contenttype IN ('image/jpeg', 'image/jpg', 'image/png', 'image/gif', 'image/bmp', 'image/webp', 'image/svg+xml', 'image/vnd.mozilla.apng')`

	rows, rows_err := conn.QueryContext(context.Background(), q)

	var pages []Page
	if rows_err == nil {
		defer rows.Close()
		for rows.Next() {
			var page pageNullable
			scan_err := rows.Scan(&page.Id, &page.Url, &page.UrlHash, &page.Scheme, &page.DomainId, &page.Content_type, &page.Charset, &page.Language, &page.Title, &page.Prompt, &page.Size, &page.Hash, &page.Feed, &page.PublishDate, &page.Index_time, &page.Album, &page.Artist, &page.AlbumArtist, &page.Composer, &page.Track, &page.Disc, &page.Copyright, &page.CrawlIndex, &page.Date_added, &page.Hidden)
			if scan_err == nil {
				pages = append(pages, scanPage(page))
			} else {
				panic(scan_err)
			}
		}
	}

	return pages
}
