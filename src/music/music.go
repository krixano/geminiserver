package music

import (
	"context"
	"time"
	"os"
	"path/filepath"
	"database/sql"
	"fmt"
	"strings"
	"net/url"

	"github.com/krixano/ponixserver/src/db"
	"github.com/pitr/gig"
)

var musicDirectory = "data/music/"
var userSongQuota = 500

var registerNotification = `# Ponix Music

You have selected a certificate that has not been registered yet. Please register here:

=> /music/register Register Page
=> /music/quota How the Quota System Works
`

func HandleMusic(g *gig.Gig) {
	conn := db.NewConn(db.MusicDB)
	conn.SetMaxOpenConns(500)
	conn.SetMaxIdleConns(3)
	conn.SetConnMaxLifetime(time.Hour * 4)
	//defer conn.Close() // TODO

	g.Handle("/music", func(c gig.Context) error {
		return c.NoContent(gig.StatusRedirectPermanent, "/music/")
	})

	g.Handle("/music/", func(c gig.Context) error {
		cert := c.Certificate()
		if cert == nil {
			return c.Gemini(`# Ponix Music

Welcome to the new Ponix Music Service, where you can upload a limited number of mp3s over Titan and listen to your private music library over Gemini.

Note: Remember to make sure your certificate is selected on this page if you've already registered.

In order to register, create and enable a client certificate and then head over to the register page:

=> /music/register Register Page
=> /music/quota How the Quota System Works
	`)
		} else {
			user, isRegistered := GetUser(conn, c.CertHash())
			if !isRegistered {
				return c.Gemini(registerNotification)
			} else {
				return getUserDashboard(c, conn, user)
			}
		}
	})

	g.Handle("/music/quota", func(c gig.Context) error {
		template := `# Ponix Music - How the Quota System Works

Each song adds to your quota 1 / the number of people who have uploaded that same song. If 3 people have uploaded the 3 same songs, only 1 song gets added to each person's quota (3 songs / 3 uploaders). However, if you are the only person who has uploaded a song, then 1 will be added to your quota (1 song / 1 uploader). The maximum quota that each user has is currently set to 1000.

If 38 people uploaded 1000 *unique* songs, that would fill up ~288.8 GB. But if 38 people uploaded the same 1000 songs, that would only be as if one person uploaded those 1000 songs, and that'd only be ~7.6 GB. And the quota added to each person is 1000 songs / 38 uploaders = 26.3
‎Which means that leaves each user with 973.7 more songs. If each user uploaded 973.7 *unique* songs, that would take up ~281 GB. 281 GB + the 7.6 GB for the 1000 non-unique songs would equal ~288.6 GB

The idea behind this system is to take into account duplicate uploads as being considered only 1 upload, because I use deduplication of song files to save space. By dividing an upload across the number of users that have uploaded that file, each user gets the same slice of quota, and the sum adds up to 1.

=> /music/ Back
`
		return c.Gemini(template)
	})

	g.Handle("/music/about", func(c gig.Context) error {
		template := `# About Ponix Music

This is a gemini capsule that allows users to upload their own mp3s (or oggs) to thier own private library (via Titan) and stream/download them via Gemini. A user's library is completely private. Nobody else can see the library, and songs are only streamable by the user that uploaded that song.

In order to save space, Ponix Music deduplicates songs by taking the hash of the audio contents. This is only done when the songs of multiple users are the *exact* same, and is done on upload of a song. Deduplication also has the benefit of lowering a user's quota. If the exact same song is in multiple users' libraries, the sum of the quotas for that song for each user adds up to 1. This is because the song is only stored once on the server. The quota is spread evenly between each of the users that have uploaded the song. The more users, the less quota each user has for that one song. You can find out more about how the quota system works below:

=> /music/quota How the Quota System Works
`
		return c.Gemini(template)
	})

	g.Handle("/music/random", func(c gig.Context) error {
		cert := c.Certificate()
		if cert == nil {
			return c.NoContent(gig.StatusClientCertificateRequired, "Please enable a certificate")
		} else {
			user, isRegistered := GetUser(conn, c.CertHash())
			if !isRegistered {
				return c.Gemini(registerNotification)
			} else {
				file, exists := GetRandomFileInUserLibray(conn, user.Id)
				if !exists {
					return c.NoContent(gig.StatusNotFound, "File not found. Your library may be empty")
				}
				openFile, err := os.Open(musicDirectory + file.Filename)
				if err != nil {
					panic(err)
				}
				err2 := c.Stream("audio/mpeg", openFile)
				openFile.Close()
				return err2
			}
		}
	})

	g.Handle("/music/*", func(c gig.Context) error {
		cert := c.Certificate()
		if cert == nil {
			return c.NoContent(gig.StatusClientCertificateRequired, "Please enable a certificate")
		} else {
			user, isRegistered := GetUser(conn, c.CertHash())
			if !isRegistered {
				return c.Gemini(registerNotification)
			} else {
				unescape, _ := url.PathUnescape(c.URL().EscapedPath())
				hash := strings.TrimSuffix(strings.Replace(unescape, "/music/", "", 1), filepath.Ext(unescape))

				file, exists := GetFileInUserLibrary_hash(conn, hash, user.Id)
				if !exists {
					return c.NoContent(gig.StatusNotFound, "File not found.")
				}

				openFile, err := os.Open(musicDirectory + file.Filename)
				if err != nil {
					panic(err)
				}
				err2 := c.Stream("audio/mpeg", openFile)
				openFile.Close()
				return err2
				//q := `SELECT COUNT(*) FROM uploads INNER JOIN library ON uploads.fileid=library.id WHERE uploads.memberid=? AND library.filename=?`
			}
		}
	})

	g.Handle("/music/albums", func(c gig.Context) error {
		cert := c.Certificate()
		if cert == nil {
			return c.NoContent(gig.StatusClientCertificateRequired, "Please enable a certificate")
		} else {
			user, isRegistered := GetUser(conn, c.CertHash())
			if !isRegistered {
				return c.Gemini(registerNotification)
			} else {
				albums := GetAlbumsInUserLibrary(conn, user.Id)

				var builder strings.Builder
				for _, album := range albums {
					fmt.Fprintf(&builder, "=> /music/artist/%s/%s %s - %s\n", url.PathEscape(album.Albumartist), url.PathEscape(album.Album), album.Album, album.Albumartist)
				}

				return c.Gemini(`# Ponix Music - %s
## Albums

%s
`, user.Username, builder.String())
			}
		}
	})

	g.Handle("/music/artists", func(c gig.Context) error {
		cert := c.Certificate()
		if cert == nil {
			return c.NoContent(gig.StatusClientCertificateRequired, "Please enable a certificate")
		} else {
			user, isRegistered := GetUser(conn, c.CertHash())
			if !isRegistered {
				return c.Gemini(registerNotification)
			} else {
				artists := GetArtistsInUserLibrary(conn, user.Id)

				var builder strings.Builder
				for _, artist := range artists {
					fmt.Fprintf(&builder, "=> /music/artist/%s %s\n", url.PathEscape(artist), artist)
				}

				return c.Gemini(`# Ponix Music - %s
## Artists

%s
`, user.Username, builder.String())
			}
		}
	})

	g.Handle("/music/artist/*", func(c gig.Context) error {
		cert := c.Certificate()
		if cert == nil {
			return c.NoContent(gig.StatusClientCertificateRequired, "Please enable a certificate")
		} else {
			user, isRegistered := GetUser(conn, c.CertHash())
			if !isRegistered {
				return c.Gemini(registerNotification)
			} else {
				unescape, _ := url.PathUnescape(c.URL().EscapedPath())
				p := strings.Split(strings.Replace(unescape, "/music/artist/", "", 1), "/")
				artist := p[0]
				album := ""
				if len(p) > 1 {
					album = p[1]
					return albumSongs(c, conn, user, artist, album)
				} else {
					return artistAlbums(c, conn, user, artist)
				}
			}
		}
	})

	g.Handle("/music/register", func(c gig.Context) error {
		cert := c.Certificate()

		if cert == nil {
			return c.NoContent(gig.StatusClientCertificateRequired, "Please enable a certificate")
		} else {
			query, err2 := c.QueryString()
			if err2 != nil {
				return err2
			} else if query == "" {
				return c.NoContent(gig.StatusInput, "Enter a username:")
			} else {
				// Do registration
				return registerUser(c, conn, query, c.CertHash())
			}
		}
	})

	g.Handle("/music/admin", func(c gig.Context) error {
		cert := c.Certificate()
		if cert == nil {
			return c.NoContent(gig.StatusClientCertificateRequired, "Please enable a certificate")
		} else {
			user, isRegistered := GetUser(conn, c.CertHash())
			if !isRegistered || !user.Is_staff {
				return c.NoContent(gig.StatusCertificateNotAuthorised, "Not authorized for this page")
			} else {
				return adminPage(c, conn, user)
			}
		}
	})
}

func registerUser(c gig.Context, conn *sql.DB, username string, certHash string) error {
	// Ensure user doesn't already exist
	row := conn.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM members WHERE certificate=?", certHash)

	var numRows int
	err := row.Scan(&numRows)
	if err != nil {
		panic(err)
	}
	if numRows < 1 {
		// Certificate doesn't already exist - Register User
		zone, _ := time.Now().Zone()
		conn.ExecContext(context.Background(), "INSERT INTO members (certificate, username, language, timezone, is_staff, is_active, date_joined) VALUES (?, ?, ?, ?, ?, ?, ?)", certHash, username, "en-US", zone, false, true, time.Now())

		user, _ := GetUser(conn, certHash)

		// Add default uploads
		defaultUploads := []int { 26, 31, 36 }

		for _, fileid := range defaultUploads {
			//conn.ExecContext(context.Background(), "INSERT INTO uploads (memberid, fileid, date_added) values (?, ?, ?)", user.id, fileid, time.Now())
			//conn.ExecContext(context.Background(), "UPDATE library SET uploadcount = uploadcount + 1 WHERE id=?", fileid)

			AddFileToUserLibrary(conn, fileid, user.Id, false)
		}
	}

	return c.NoContent(gig.StatusRedirectTemporary, "/music/")
}

func getUserDashboard(c gig.Context, conn *sql.DB, user MusicUser) error {
	template := `# Ponix Music - %s

Quota: %.2f / %d songs (%.1f%%)

=> /music/quota How the Quota System Works
=> /music/albums Albums
=> /music/artists Artists
=> /music/random Random Song
%s
`

	musicFiles := GetFilesInUserLibrary(conn, user.Id)

	var builder strings.Builder

	if user.Is_staff {
		fmt.Fprintf(&builder, "=> /music/admin Admin Dashboard\n")
	}
	fmt.Fprintf(&builder, "\n")
	
	for _, file := range musicFiles {
		artist := file.Composer
		if file.Composer == "" || (file.Genre != "Classical" && file.Genre != "String Quartets") {
			artist = file.Albumartist
		}
		fmt.Fprintf(&builder, "=> %s %s (%s)\n", url.PathEscape(file.Filename), file.Title, artist)
	}

	if len(musicFiles) == 0 {
		fmt.Fprintf(&builder, "Your music library is empty.")
	}

	return c.Gemini(template, user.Username, user.QuotaCount, userSongQuota, user.QuotaCount / float64(userSongQuota) * 100, builder.String())
}

func artistAlbums(c gig.Context, conn *sql.DB, user MusicUser, artist string) error {
	albums := GetAlbumsFromArtistInUserLibrary(conn, user.Id, artist)

	var builder strings.Builder
	for _, album := range albums {
		fmt.Fprintf(&builder, "=> /music/artist/%s/%s %s\n", url.PathEscape(album.Albumartist), url.PathEscape(album.Album), album.Album)
	}

	return c.Gemini(`# Ponix Music - %s
## Artist Albums: %s

%s
`, user.Username, artist, builder.String())
}

func albumSongs(c gig.Context, conn *sql.DB, user MusicUser, artist string, album string) error {
	musicFiles := GetFilesFromAlbumInUserLibrary(conn, user.Id, artist, album)

	var builder strings.Builder
	for _, file := range musicFiles {
		/*artist := file.Composer
		if file.Composer == "" {
			artist = file.Albumartist
		}*/
		fmt.Fprintf(&builder, "=> /music/%s %d. %s\n", url.PathEscape(file.Filename), file.Tracknumber, file.Title)
	}

	return c.Gemini(`# Ponix Music - %s
## Album: %s by %s

%s
`, user.Username, album, artist, builder.String())
}

func adminPage(c gig.Context, conn *sql.DB, user MusicUser) error {
	template := `# Ponix Music - Admin

Global Quota: %.2f / %.2f (%.1f%%)
User Quota Average: %.2f / %d (%.1f%%)
User Count: %d
Artist Count: %d
Album Count: %d
`

	globalQuotaCount := Admin_GetGlobalQuota(conn)
	userCount := Admin_UserCount(conn)
	avgUserQuotaCount := globalQuotaCount / float64(userCount)
	artistCount := Admin_ArtistCount(conn)
	albumCount := Admin_AlbumCount(conn)

	var globalSongQuota float64 = 58000 // 38666.67
	return c.Gemini(template, globalQuotaCount, globalSongQuota, globalQuotaCount / globalSongQuota * 100, avgUserQuotaCount, userSongQuota, avgUserQuotaCount / float64(userSongQuota) * 100, userCount, artistCount, albumCount)
}
