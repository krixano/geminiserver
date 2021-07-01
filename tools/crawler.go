package main

import (
	"database/sql"
	"context"
	"time"
	"mime"
	"io"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"fmt"
	"errors"
	"bytes"
	neturl "net/url"

	rake "github.com/afjoseph/RAKE.Go"
	gemini "git.sr.ht/~adnano/go-gemini"
	_ "github.com/nakagami/firebirdsql"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/temoto/robotstxt"
	"github.com/dhowden/tag"
	"github.com/rivo/uniseg"

	// "golang.org/x/text/encoding/ianaindex"
    // "golang.org/x/text/transform"
)

type Robots struct {
	robots *robotstxt.RobotsData
	indexerGroup *robotstxt.Group
	ponixSearchGroup *robotstxt.Group
}

var crawlIndex = 1
var robotsMap cmap.ConcurrentMap

func initializeCrawler() {
	robotsMap = cmap.New()
}

type Domain struct {
	Id int
	Domain string
	Port int
	//ParentDomain Domain // ForeignKey
	ParentDomainId int
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
	Id int
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
	Id int
	//Page Page
	PageId int
	Key string
	Rank int
	Date_added time.Time
}

type Link struct {
	Id int
	//From Page
	FromPageId int
	//To Page
	ToPageId int
	Name string
	Cross_host bool
	Date_added time.Time
}

// Pages to crawl // TODO
type CrawlLog struct {
	url string
}

func NewConn() *sql.DB {
	connectionString := "GEMINI:Blue132401!@localhost:3050/var/lib/firebird/3.0/data/search.fdb"
	conn, err := sql.Open("firebirdsql", connectionString)

	if err != nil {
		//panic(oops.New(err, "failed to connect to database"))
		panic(err)
	}

	return conn
}

type CrawlContext struct {
	client gemini.Client
	resp *gemini.Response
	currentURL *neturl.URL
	currentRobots Robots
	domainsCrawled map[string]struct{}
	urlsCrawled map[string]struct{}

	urlsToCrawl map[string]struct{} // bool is whether robots.txt should be checked or not

	dbConn *sql.DB
}

var NotGeminiError = errors.New("Not a gemini url")
var NotAllowedError = errors.New("Not allowed by robots.txt")

func (c *CrawlContext) GetRobotsTxt(host string) Robots {
	robotsData := Robots{}
	ctx := /*context.WithTimeout(*/context.Background()//, )
	resp, err := c.client.Get(ctx, host + "robots.txt")
	defer resp.Body.Close()

	// Defaults
	robotsStr := "User-agent: *\nAllow: /"
	robotsData.robots, _ = robotstxt.FromString(robotsStr)

	if err != nil || resp.Status != gemini.StatusSuccess {
		// Insert empty robots content
		fmt.Printf("Robots: %s\n%s\n\n", host + "robots.txt", string(robotsStr))
	} else {
		data, read_err := io.ReadAll(resp.Body)
		if read_err == nil {
			robotsData.robots, _ = robotstxt.FromBytes(data)
			fmt.Printf("Robots: %s\n%s\n\n", host + "robots.txt", string(data))
		}
	}

	robotsData.indexerGroup = robotsData.robots.FindGroup("indexer")
	robotsData.ponixSearchGroup = robotsData.robots.FindGroup("ponixsearch")

	robotsMap.Set(host, robotsData)
	return robotsData
}

func (c *CrawlContext) Get(url string) (*gemini.Response, error) {
	c.currentURL, _ = neturl.Parse(url)
	if c.currentURL.Scheme != "gemini" {
		return nil, NotGeminiError
	}

	// Get host
	host := ""
	if c.currentURL.Port() == "" || c.currentURL.Port() == "1965" {
		host = c.currentURL.Scheme + "://" + c.currentURL.Hostname() + "/"
	} else {
		host = c.currentURL.Scheme + "://" + c.currentURL.Hostname() + ":" + c.currentURL.Port() + "/"
	}

	// Check if host is in robotsMap. If not, get robots.txt. If so, check if allowed to crawl, and return if not.
	if r, ok := robotsMap.Get(host); ok {
		allow := r.(Robots).indexerGroup.Test(c.currentURL.Path)
		if !allow {
			return nil, NotAllowedError
			/*allow = r.(Robots).ponixSearchGroup.Test(c.currentURL.Path)
			if !allow {
			}*/
		}
		c.currentRobots = r.(Robots)
	} else {
		// Get robots.txt and insert into map if exists
		r := c.GetRobotsTxt(host)
		allow := r.indexerGroup.Test(c.currentURL.Path)
		if !allow {
			return nil, NotAllowedError
			/*allow = r.ponixSearchGroup.Test(c.currentURL.Path)
			if !allow {
			}*/
		}
		c.currentRobots = r
	}

	if !c.addDomain(c.currentURL.Hostname()) {
		c.addUrl(host)
	}
	c.removeUrl(url)
	resp, err := c.client.Get(context.Background(), url)
	c.resp = resp
	return resp, err
}

func (c *CrawlContext) addUrl(url string) {
	var exists = struct{}{}
	c.urlsToCrawl[url] = exists
}

// Returns true if already existed
func (c *CrawlContext) addDomain(domain string) bool {
	var exists = struct{}{}
	_, preexists := c.domainsCrawled[domain]
	c.domainsCrawled[domain] = exists
	return preexists
}

func (c *CrawlContext) removeUrl(url string) {
	var exists = struct{}{}
	delete(c.urlsToCrawl, url)
	c.urlsCrawled[url] = exists
}

func (c *CrawlContext) removeDomain(domain string) {
	delete(c.domainsCrawled, domain)
}

func (c *CrawlContext) getNextUrl() string {
	for k, _ := range c.urlsToCrawl {
		return k
	}

	return ""
}

// TODO: Flushes out the domainsCrawled map into the db if gets to a certain size
func (c *CrawlContext) flush() {
	for _, domain := range c.domainsCrawled {
		// Check if exists in db, then update or insert
		row := c.dbConn.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM domains WHERE domain=?", domain)
		count := 0
		err := row.Scan(&count)
		if err != sql.ErrNoRows && err != nil { // TODO
			continue
		}
		if err == sql.ErrNoRows || count <= 0 {
			_, execErr := c.dbConn.ExecContext(context.Background(), "INSERT INTO domains (domain, crawlIndex, date_added) VALUES (?, ?, ?)", domain, crawlIndex, time.Now().UTC())
			if execErr != nil {
				panic(err)
			}
		} else if count > 0 {
			_, execErr := c.dbConn.ExecContext(context.Background(), "UPDATE domains SET crawlIndex=? WHERE domain=?", crawlIndex, domain)
			if execErr != nil {
				panic(err)
			}
		}
	}
}

func newCrawlContext() CrawlContext {
	url, _ := neturl.Parse("gemini://gemini.circumlunar.space/")
	return CrawlContext { gemini.Client{}, nil, url, Robots{}, make(map[string]struct{}), make(map[string]struct{}), make(map[string]struct{}), NewConn() }
}

func main() {
	initializeCrawler()
	crawl()
}

func crawl() {
	crawlCtx := newCrawlContext()
	//crawlCtx.addUrl("gemini://gemini.circumlunar.space/")
	crawlCtx.addUrl("gemini://pon.ix.tc/")

	for i := 0; i < 2900; i++ {
		//fmt.Printf("\n")
		nextUrl := crawlCtx.getNextUrl()
		if nextUrl == "" {
			break
		}

		resp, err := crawlCtx.Get(nextUrl)
		//meta := resp.Meta
		if err != nil {
			//panic(err)
			continue
		}
		status := resp.Status

		//crawlCtx := CrawlContext{ resp, "gemini://gemini.circumlunar.space/" }

		//fmt.Printf("Status: %d\n", status)
		defer resp.Body.Close()
		switch status {
		case gemini.StatusInput: handleInput(crawlCtx)
		case gemini.StatusSensitiveInput: // TODO
		case gemini.StatusSuccess: handleSuccess(crawlCtx)
		case gemini.StatusRedirect: handleRedirect(crawlCtx, false)
		case gemini.StatusPermanentRedirect: handleRedirect(crawlCtx, true)
		case gemini.StatusTemporaryFailure: continue
		case gemini.StatusServerUnavailable: continue
		case gemini.StatusCGIError: // TODO
		case gemini.StatusProxyError: // TODO
		case gemini.StatusSlowDown: // TODO
		case gemini.StatusPermanentFailure: continue
		case gemini.StatusNotFound: continue
		case gemini.StatusGone: continue
		case gemini.StatusProxyRequestRefused: continue
		case gemini.StatusBadRequest: continue
		case gemini.StatusCertificateRequired: continue
		case gemini.StatusCertificateNotAuthorized: continue
		case gemini.StatusCertificateNotValid: continue
		}

		sleepDuration, _ := time.ParseDuration("10ms")
		time.Sleep(sleepDuration)
	}

	crawlCtx.flush()

	fmt.Printf("\n%v", crawlCtx.urlsToCrawl)

	crawlCtx.dbConn.Close()
}

func handleInput(ctx CrawlContext) {

}

func handleRedirect(ctx CrawlContext, permanent bool) {
	meta := ctx.resp.Meta
	url, _ := ctx.currentURL.Parse(meta)
	if _, ok := ctx.urlsCrawled[url.String()]; ok {
		return
	}
	ctx.addUrl(url.String())
}

func handleSuccess(ctx CrawlContext) {
	//status := ctx.resp.Status
	meta := ctx.resp.Meta

	mediatype, params, _ := mime.ParseMediaType(meta)
	var charset string = ""
	if _, ok := params["charset"]; ok {
		charset = params["charset"]
	}
	var language string = ""
	if _, ok := params["lang"]; ok {
		language = params["lang"]
	}

	if mediatype == "text/gemini" {
		var geminiTitle string = ""
		text, _ := gemini.ParseText(ctx.resp.Body)
		textStr := text.String()
		size := len(textStr)

		hasher := sha256.New()
		hasher.Write([]byte(textStr))
		hashStr := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

		update := true
		// TODO: Check if hash has changed in database. If so, set update to true so that the information in the db
		// updates. Otherwise, just get links
		//q := `SELECT FIRST 1 id, url, urlhash, scheme, domainid, contenttype, charset, language, title, prompt, size, hash, feed, publishdate, indextime, album, artist, albumartist, composer, track, disc, copyright, crawlindex, date_added, hidden FROM pages WHERE url=?`
		//row := ctx.dbConn.QueryRowContext(context.Background(), q, ctx.currentURL.String())
		//var page pageNullable
		//row.Scan(&result.Id, &result.Url, &result.UrlHash, &result.Scheme, &result.DomainId, &result.Content_type, &result.Charset, &result.Language, &result.Title, &result.Prompt, &result.Size, &result.Hash, &result.Feed, &result.PublishDate, &result.Index_time, &result.Album, &result.Artist, &result.AlbumArtist, &result.Composer, &result.Track, &result.Disc, &result.Copyright, &result.CrawlIndex, &result.Date_added, &result.Hidden)
		/*if err != sql.ErrNoRows && err != nil { // TODO
			update = true
		}*/

		/*if  {
			update = true
		} else {
			update = false
		}*/

		// Go through gemini document to get keywords, title, and links
		var strippedTextBuilder strings.Builder
		keywordsMap := make(map[string]float64)
		links := make([]gemini.LineLink, 0)
		for _, line := range text {
			switch v := line.(type) {
				case gemini.LineHeading1: {
					if update {
						fmt.Fprintf(&strippedTextBuilder, "%s\n", string(v))
						if geminiTitle == "" {
							geminiTitle = string(v)
						}
						if _, ok := keywordsMap[string(v)]; ok {
							keywordsMap[string(v)] += 3
						} else {
							keywordsMap[string(v)] = 3
						}
					}
				}
				case gemini.LineHeading2: {
					if update {
						fmt.Fprintf(&strippedTextBuilder, "%s\n", string(v))
						if _, ok := keywordsMap[string(v)]; ok {
							keywordsMap[string(v)] += 2
						} else {
							keywordsMap[string(v)] = 2
						}
					}
				}
				case gemini.LineHeading3: {
					if update {
						fmt.Fprintf(&strippedTextBuilder, "%s\n", string(v))
						if _, ok := keywordsMap[string(v)]; ok {
							keywordsMap[string(v)] += 1
						} else {
							keywordsMap[string(v)] = 1
						}
					}
				}
				case gemini.LineLink: {
					//fmt.Fprintf(&strippedTextBuilder, "%s\n", v.Name)
					// v.URL, v.Name
					links = append(links, v)
				}
				case gemini.LineListItem: {
					if update {
						fmt.Fprintf(&strippedTextBuilder, "%s\n", v.String())
					}
				}
				case gemini.LinePreformattedText: {
					if update {
						fmt.Fprintf(&strippedTextBuilder, "%s\n", string(v))
					}
				}
				case gemini.LineQuote: {
					if update {
						fmt.Fprintf(&strippedTextBuilder, "%s\n", string(v))
					}
				}
				case gemini.LineText: {
					if update {
						fmt.Fprintf(&strippedTextBuilder, "%s\n", string(v))
					}
				}
			}
		}

		// Get keywords. Only needed if update is true
		if update {
			keywords := rake.RunRake(strippedTextBuilder.String())
			//fmt.Printf("Keywords: \n")
			for _, keyword := range keywords {
				//fmt.Printf("%s -> %f\n", keyword.Key, keyword.Value)
				if _, ok := keywordsMap[keyword.Key]; ok {
					keywordsMap[keyword.Key] += keyword.Value
				} else {
					keywordsMap[keyword.Key] = keyword.Value
				}
			}
			//fmt.Printf("\n\n")
		}

		// Update the entry in the db if needed.
		if update {
			urlString := ctx.currentURL.String()
			urlHasher := sha256.New()
			urlHasher.Write([]byte(urlString))
			urlHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			page := Page { 0, urlString, urlHash, ctx.currentURL.Scheme, 0, mediatype, charset, language, geminiTitle, "", size, hashStr, false, time.Time{}, time.Now().UTC(), "", "", "", "", 0, 0, "", crawlIndex, time.Now().UTC(), false }
			page = addPageToDb(ctx, page)

			for keyword, rank := range keywordsMap {
				graphemeCount := uniseg.GraphemeClusterCount(keyword)
				if len(keyword) <= 2 || graphemeCount > 250 {
					continue
				}
				// TODO: skip if rank == 0?
				addKeywordToDb(ctx, page.Id, keyword, rank)
			}
		} else { // TODO: Temporary - update UrlHash
			urlString := ctx.currentURL.String()
			urlHasher := sha256.New()
			urlHasher.Write([]byte(urlString))
			urlHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			q := `UPDATE pages SET urlhash=? WHERE url=?`
			_, update_err := ctx.dbConn.ExecContext(context.Background(), q, urlHash, urlString)
			if update_err != nil {
				panic(update_err)
			}
		}

		//fmt.Printf("Title: %s; Status: %d; Size: %d; Hash: %s\n\n", geminiTitle, status, size, hashStr)

		//fmt.Printf("Links: \n")
		for _, link := range links {
			url, _ := ctx.currentURL.Parse(link.URL)
			if _, ok := ctx.urlsCrawled[url.String()]; ok {
				continue
			}
			if ctx.currentURL.Hostname() == url.Hostname() && ctx.currentURL.Port() == url.Port() && ctx.currentURL.Scheme == url.Scheme {
				allow := ctx.currentRobots.indexerGroup.Test(url.Path)
				if !allow {
					continue
					/*allow = ctx.currentRobots.ponixSearchGroup.Test(url.Path)
					if !allow {
						fmt.Printf("A link not allowed\n")
						continue
					}*/
				}
				ctx.addUrl(url.String())
			} else if url.Scheme == "gemini" {
				ctx.addUrl(url.String())
			}
			//fmt.Printf("%s\n", url.Path)
		}
	} else if mediatype == "text/plain" || mediatype == "text/markdown" {
		textBytes, _ := io.ReadAll(ctx.resp.Body)
		textStr := string(textBytes)
		size := len(textBytes)
		keywords := rake.RunRake(textStr)

		hasher := sha256.New()
		hasher.Write([]byte(textStr))
		hashStr := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

		//fmt.Printf("Status: %d; Size: %d; Hash: %s; Keywords: %v;\n", status, size, hashStr, keywords)

		urlString := ctx.currentURL.String()
		urlHasher := sha256.New()
		urlHasher.Write([]byte(urlString))
		urlHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

		page := Page { 0, urlString, urlHash, ctx.currentURL.Scheme, 0, mediatype, charset, language, "", "", size, hashStr, false, time.Time{}, time.Now().UTC(), "", "", "", "", 0, 0, "", crawlIndex, time.Now().UTC(), false }
		page = addPageToDb(ctx, page)

		for _, candidate := range keywords {
			/*graphemeCount := uniseg.GraphemeClusterCount(keyword)
			if len(keyword) <= 2 || graphemeCount > 250 {
				continue
			}*/
			// TODO: Skip if keyword == 0?
			addKeywordToDb(ctx, page.Id, candidate.Key, candidate.Value)
		}
	} else if mediatype == "audio/mpeg" {
		p := make([]byte, 1024 * 1024 * 72) // Note: Takes very long to download on a slow server
		size, err := io.ReadFull(ctx.resp.Body, p)
		if err == io.ErrUnexpectedEOF || err == io.EOF {
			m, _ := tag.ReadFrom(bytes.NewReader(p[:size]))

			hasher := sha256.New()
			hasher.Write(p[:size])
			hashStr := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			//fmt.Printf("Title: %s; Hash: %s\n", m.Title(), hashStr)
			track, _ := m.Track()
			disc, _ := m.Disc()

			urlString := ctx.currentURL.String()
			urlHasher := sha256.New()
			urlHasher.Write([]byte(urlString))
			urlHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			page := Page { 0, urlString, urlHash, ctx.currentURL.Scheme, 0, mediatype, charset, language, m.Title(), "", size, hashStr, false, time.Time{}, time.Now().UTC(), m.Album(), m.Artist(), m.AlbumArtist(), m.Composer(), track, disc, "", crawlIndex, time.Now().UTC(), false }
			addPageToDb(ctx, page)
		} else {
			return
		}
	} else if mediatype == "audio/ogg" {
		p := make([]byte, 1024 * 1024 * 72) // Note: Takes very long to download on a slow server
		size, err := io.ReadFull(ctx.resp.Body, p)
		if err == io.ErrUnexpectedEOF || err == io.EOF {
			m, _ := tag.ReadFrom(bytes.NewReader(p[:size]))

			hasher := sha256.New()
			hasher.Write(p[:size])
			hashStr := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			//fmt.Printf("Title: %s; Hash: %s\n", m.Title(), hashStr)
			track, _ := m.Track()
			disc, _ := m.Disc()

			urlString := ctx.currentURL.String()
			urlHasher := sha256.New()
			urlHasher.Write([]byte(urlString))
			urlHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			page := Page { 0, urlString, urlHash, ctx.currentURL.Scheme, 0, mediatype, charset, language, m.Title(), "", size, hashStr, false, time.Time{}, time.Now().UTC(), m.Album(), m.Artist(), m.AlbumArtist(), m.Composer(), track, disc, "", crawlIndex, time.Now().UTC(), false }
			addPageToDb(ctx, page)
		} else {
			return
		}
	} else {
		p := make([]byte, 1024 * 1024 * 72) // Note: Takes very long to download on a slow server
		size, err := io.ReadFull(ctx.resp.Body, p)
		if err == io.ErrUnexpectedEOF || err == io.EOF {
			hasher := sha256.New()
			hasher.Write(p[:size])
			hashStr := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			urlString := ctx.currentURL.String()
			urlHasher := sha256.New()
			urlHasher.Write([]byte(urlString))
			urlHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			page := Page { 0, urlString, urlHash, ctx.currentURL.Scheme, 0, mediatype, charset, language, "", "", size, hashStr, false, time.Time{}, time.Now().UTC(), "", "", "", "", 0, 0, "", crawlIndex, time.Now().UTC(), false }
			addPageToDb(ctx, page)
		}
	}
}

func addPageToDb(ctx CrawlContext, page Page) Page {
	// Check if exists in db, then update or insert
	row := ctx.dbConn.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM pages WHERE url=?", page.Url)
	count := 0
	err := row.Scan(&count)
	if err != sql.ErrNoRows && err != nil { // TODO
		panic(err)
		return Page {}
	}
	if err == sql.ErrNoRows || count <= 0 {
		_, err := ctx.dbConn.ExecContext(context.Background(), "INSERT INTO pages (url, urlhash, scheme, contenttype, charset, language, title, prompt, size, hash, feed, publishdate, indextime, album, artist, albumartist, composer, track, disc, copyright, crawlIndex, date_added, hidden) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'false')", page.Url, page.UrlHash, page.Scheme, page.Content_type, page.Charset, page.Language, page.Title, page.Prompt, page.Size, page.Hash, page.Feed, page.PublishDate, time.Now().UTC(), page.Album, page.Artist, page.AlbumArtist, page.Composer, page.Track, page.Disc, page.Copyright, crawlIndex, time.Now().UTC())
		if err != nil {
			panic(err)
		}
	} else if count > 0 {
		_, err := ctx.dbConn.ExecContext(context.Background(), "UPDATE pages SET urlhash=?, scheme=?, contenttype=?, charset=?, language=?, title=?, prompt=?, size=?, hash=?, feed=?, publishdate=?, indextime=?, album=?, artist=?, albumartist=?, composer=?, track=?, disc=?, copyright=?, crawlIndex=? WHERE url=?", page.UrlHash, page.Scheme, page.Content_type, page.Charset, page.Language, page.Title, page.Prompt, page.Size, page.Hash, page.Feed, page.PublishDate, time.Now().UTC(), page.Album, page.Artist, page.AlbumArtist, page.Composer, page.Track, page.Disc, page.Copyright, crawlIndex, page.Url)
		if err != nil {
			panic(err)
		}
	}

	// Get the page
	var result Page
	row2 := ctx.dbConn.QueryRowContext(context.Background(), "SELECT FIRST 1 id, url, urlhash, scheme, domainid, contenttype, charset, language, title, prompt, size, hash, feed, publishdate, indextime, album, artist, albumartist, composer, track, disc, copyright, crawlindex, date_added, hidden FROM pages WHERE url=?", page.Url)
	row2.Scan(&result.Id, &result.Url, &result.UrlHash, &result.Scheme, &result.DomainId, &result.Content_type, &result.Charset, &result.Language, &result.Title, &result.Prompt, &result.Size, &result.Hash, &result.Feed, &result.PublishDate, &result.Index_time, &result.Album, &result.Artist, &result.AlbumArtist, &result.Composer, &result.Track, &result.Disc, &result.Copyright, &result.CrawlIndex, &result.Date_added, &result.Hidden)
	return result
}

func addKeywordToDb(ctx CrawlContext, pageId int, keyword string, rank float64) {
	// Check if exists in db, then update or insert
	row := ctx.dbConn.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM keywords WHERE pageid=? AND keyword=?", pageId, keyword)
	count := 0
	err := row.Scan(&count)
	if err != sql.ErrNoRows && err != nil { // TODO
		panic(err)
		return
	}
	if err == sql.ErrNoRows || count <= 0 {
		_, err := ctx.dbConn.ExecContext(context.Background(), "INSERT INTO keywords (keyword, pageid, rank, crawlindex, date_added) VALUES (?, ?, ?, ?, ?)", keyword, pageId, rank, crawlIndex, time.Now().UTC())
		if err != nil {
			panic(err)
		}
	} else if count > 0 {
		_, err := ctx.dbConn.ExecContext(context.Background(), "UPDATE keywords SET rank=?, crawlindex=? WHERE pageid=? AND keyword=?", rank, crawlIndex, pageId, keyword)
		if err != nil {
			panic(err)
		}
	}
}
