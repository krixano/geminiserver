package search

import (
	"time"
)

type Page struct {
	Id int64
	Url string // fetchable_url, normalized_url
	UrlHash string
	Scheme string
	// Domain Domain // foreign key
	DomainId int64

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

type pageNullable struct {
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

func scanPage(page pageNullable) Page {
	var result Page
	result.Id = page.Id
	result.Url = page.Url
	result.UrlHash = page.UrlHash

	result.Scheme = page.Scheme
	if page.DomainId != nil {
		if _, ok := page.DomainId.(int64); ok {
			result.DomainId = int64(page.DomainId.(int64))
		} else {
			if _, ok2 := page.DomainId.(int); ok2 {
				result.DomainId = int64(page.DomainId.(int))
			}
		}
	} else {
		result.DomainId = -1
	}

	result.Content_type = page.Content_type
	result.Charset = page.Charset
	result.Language = page.Language

	result.Title = page.Title
	result.Prompt = page.Prompt
	result.Size = page.Size
	result.Hash = page.Hash
	result.Feed = page.Feed
	result.PublishDate = page.PublishDate
	result.Index_time = page.Index_time

	result.Album = page.Album
	result.Artist = page.Artist
	result.AlbumArtist = page.AlbumArtist
	result.Composer = page.Composer
	result.Track = page.Track
	result.Disc = page.Disc
	result.Copyright = page.Copyright
	result.CrawlIndex = page.CrawlIndex
	result.Date_added = page.Date_added

	result.Hidden = page.Hidden

	return result
}

