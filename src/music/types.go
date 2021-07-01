package music

import (
	"database/sql"
	"time"
	"context"
	"github.com/dhowden/tag"
)

type MusicUser struct {
	Id int
	Username string
	Certificate string
	Language string
	Timezone string
	Is_staff bool
	Is_active bool
	Date_joined time.Time

	QuotaCount float64
}
type MusicFile struct {
	Id int
	Filehash string
	Filename string
	Mimetype string
	Title string
	Album string
	Artist string
	Albumartist string
	Composer string
	Genre string
	Releaseyear int
	Tracknumber int
	Discnumber int
	Date_added time.Time
}
type MusicAlbum struct {
	Album string
	Albumartist string
}


func GetFileInLibrary_hash(conn *sql.DB, hash string) (MusicFile, bool) {
	query := `SELECT FIRST 1 library.id, library.filehash, library.filename, library.mimetype, library.title, library.album, library.artist, library.albumartist, library.composer, library.genre, library.releaseyear, library.tracknumber, library.discnumber, library.date_added FROM library WHERE filehash=?`

	row := conn.QueryRowContext(context.Background(), query, hash)
	
	var file MusicFile
	err := row.Scan(&file.Id, &file.Filehash, &file.Filename, &file.Mimetype, &file.Title, &file.Album, &file.Artist, &file.Albumartist, &file.Composer, &file.Genre, &file.Releaseyear, &file.Tracknumber, &file.Discnumber, &file.Date_added)
	if err == sql.ErrNoRows {
		return MusicFile{}, false
	}
	return file, true
}

func GetFileInLibrary_id(conn *sql.DB, fileid int) (MusicFile, bool) {
	query := `SELECT FIRST 1 library.id, library.filehash, library.filename, library.mimetype, library.title, library.album, library.artist, library.albumartist, library.composer, library.genre, library.releaseyear, library.tracknumber, library.discnumber, library.date_added FROM library WHERE id=?`

	row := conn.QueryRowContext(context.Background(), query, fileid)
	
	var file MusicFile
	err := row.Scan(&file.Id, &file.Filehash, &file.Filename, &file.Mimetype, &file.Title, &file.Album, &file.Artist, &file.Albumartist, &file.Composer, &file.Genre, &file.Releaseyear, &file.Tracknumber, &file.Discnumber, &file.Date_added)
	if err == sql.ErrNoRows {
		return MusicFile{}, false
	}
	return file, true
}

func AddFileToLibrary(conn *sql.DB, hash string, m tag.Metadata) (MusicFile, bool) {
	query := `INSERT INTO library (FILEHASH, FILENAME, MIMETYPE, TITLE, ALBUM, ARTIST, ALBUMARTIST, COMPOSER, GENRE, RELEASEYEAR, TRACKNUMBER, DISCNUMBER, UPLOADCOUNT, DATE_ADDED) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	trackNumber, _ := m.Track()
	discNumber, _ := m.Disc()
    conn.ExecContext(context.Background(), query, hash, hash + ".mp3", "audio/mpeg", m.Title(), m.Album(), m.Artist(), m.AlbumArtist(), m.Composer(), m.Genre(), m.Year(), trackNumber, discNumber, 0, time.Now())

    return GetFileInLibrary_hash(conn, hash)
}

func GetFileInUserLibrary(conn *sql.DB, musicFileId int, userId int) (MusicFile, bool) {
	query := `SELECT FIRST 1 library.id, library.filehash, library.filename, library.mimetype, library.title, library.album, library.artist, library.albumartist, library.composer, library.genre, library.releaseyear, library.tracknumber, library.discnumber, library.date_added FROM uploads INNER JOIN library ON uploads.fileid=library.id WHERE library.id=? AND uploads.memberid=?`

	row := conn.QueryRowContext(context.Background(), query, musicFileId, userId)
	
	var file MusicFile
	err := row.Scan(&file.Id, &file.Filehash, &file.Filename, &file.Mimetype, &file.Title, &file.Album, &file.Artist, &file.Albumartist, &file.Composer, &file.Genre, &file.Releaseyear, &file.Tracknumber, &file.Discnumber, &file.Date_added)
	if err == sql.ErrNoRows || err != nil {
		return MusicFile{}, false
	}
	return file, true
}

func GetFileInUserLibrary_hash(conn *sql.DB, musicFileHash string, userId int) (MusicFile, bool) {
	query := `SELECT FIRST 1 library.id, library.filehash, library.filename, library.mimetype, library.title, library.album, library.artist, library.albumartist, library.composer, library.genre, library.releaseyear, library.tracknumber, library.discnumber, library.date_added FROM uploads INNER JOIN library ON uploads.fileid=library.id WHERE library.filehash=? AND uploads.memberid=?`

	row := conn.QueryRowContext(context.Background(), query, musicFileHash, userId)
	
	var file MusicFile
	err := row.Scan(&file.Id, &file.Filehash, &file.Filename, &file.Mimetype, &file.Title, &file.Album, &file.Artist, &file.Albumartist, &file.Composer, &file.Genre, &file.Releaseyear, &file.Tracknumber, &file.Discnumber, &file.Date_added)
	if err == sql.ErrNoRows || err != nil {
		return MusicFile{}, false
	}
	return file, true
}

func AddFileToUserLibrary(conn *sql.DB, musicFileId int, userId int, check bool) {
	// Check first if user already has the file
	exists := false
	if check {
		_, exists = GetFileInUserLibrary(conn, musicFileId, userId)
	}

	if !exists {
		conn.ExecContext(context.Background(), "INSERT INTO uploads (memberid, fileid, date_added) values (?, ?, ?)", userId, musicFileId, time.Now())
		conn.ExecContext(context.Background(), "UPDATE library SET uploadcount = uploadcount + 1 WHERE id=?", musicFileId)
	}
}

func GetRandomFileInUserLibray(conn *sql.DB, userId int) (MusicFile, bool) {
	query := `SELECT FIRST 1 library.id, library.filehash, library.filename, library.mimetype, library.title, library.album, library.artist, library.albumartist, library.composer, library.genre, library.releaseyear, library.tracknumber, library.discnumber, library.date_added FROM uploads INNER JOIN library ON uploads.fileid=library.id WHERE uploads.memberid=? ORDER BY RAND()`
	row := conn.QueryRowContext(context.Background(), query, userId)

	var file MusicFile
	err := row.Scan(&file.Id, &file.Filehash, &file.Filename, &file.Mimetype, &file.Title, &file.Album, &file.Artist, &file.Albumartist, &file.Composer, &file.Genre, &file.Releaseyear, &file.Tracknumber, &file.Discnumber, &file.Date_added)
	if err == sql.ErrNoRows || err != nil {
		return MusicFile{}, false
	}
	return file, true
}

// ---------

func GetUser(conn *sql.DB, certHash string) (MusicUser, bool) {
	row := conn.QueryRowContext(context.Background(), "SELECT id, username, language, timezone, is_staff, is_active, date_joined FROM members WHERE certificate=?", certHash)

	var user MusicUser
	user.Certificate = certHash
	err := row.Scan(&user.Id, &user.Username, &user.Language, &user.Timezone, &user.Is_staff, &user.Is_active, &user.Date_joined)
	if err == sql.ErrNoRows {
		return MusicUser {}, false
	} else if err != nil {
		panic(err)
		return MusicUser {}, false
	}

	// Get user quota
	quotaCount := GetUserQuota(conn, user.Id)
	user.QuotaCount = quotaCount

	return user, true
}

func GetFilesInUserLibrary(conn *sql.DB, userId int) []MusicFile {
	var musicFiles []MusicFile
	rows, rows_err := conn.QueryContext(context.Background(), "SELECT library.id, library.filehash, library.filename, library.mimetype, library.title, library.album, library.artist, library.albumartist, library.composer, library.genre, library.releaseyear, library.tracknumber, library.discnumber, library.date_added FROM uploads INNER JOIN library ON uploads.fileid=library.id WHERE uploads.memberid=? ORDER BY library.albumartist ASC, library.releaseyear DESC, library.album ASC, library.discnumber ASC, library.tracknumber ASC", userId)
	if rows_err == nil {
		defer rows.Close()
		for rows.Next() {
			var file MusicFile
			scan_err := rows.Scan(&file.Id, &file.Filehash, &file.Filename, &file.Mimetype, &file.Title, &file.Album, &file.Artist, &file.Albumartist, &file.Composer, &file.Genre, &file.Releaseyear, &file.Tracknumber, &file.Discnumber, &file.Date_added)
			if scan_err == nil {
				musicFiles = append(musicFiles, file)
			}
		}
	}

	return musicFiles
}

func GetArtistsInUserLibrary(conn *sql.DB, userId int) []string {
	var artists []string
	rows, rows_err := conn.QueryContext(context.Background(), "SELECT DISTINCT library.albumartist FROM uploads INNER JOIN library ON uploads.fileid=library.id WHERE uploads.memberid=? ORDER By library.albumartist ASC", userId)
	if rows_err == nil {
		defer rows.Close()
		for rows.Next() {
			var artist string
			scan_err := rows.Scan(&artist)
			if scan_err == nil {
				artists = append(artists, artist)
			}
		}
	}

	return artists
}

func GetAlbumsInUserLibrary(conn *sql.DB, userId int) []MusicAlbum {
	var albums []MusicAlbum
	rows, rows_err := conn.QueryContext(context.Background(), "SELECT DISTINCT library.album, library.albumartist FROM uploads INNER JOIN library ON uploads.fileid=library.id WHERE uploads.memberid=? ORDER BY library.albumartist, library.releaseyear DESC, library.album ASC", userId)
	if rows_err == nil {
		defer rows.Close()
		for rows.Next() {
			var album MusicAlbum
			scan_err := rows.Scan(&album.Album, &album.Albumartist)
			if scan_err == nil {
				albums = append(albums, album)
			}
		}
	}

	return albums
}

func GetAlbumsFromArtistInUserLibrary(conn *sql.DB, userId int, albumArtist string) []MusicAlbum {
	var albums []MusicAlbum
	rows, rows_err := conn.QueryContext(context.Background(), "SELECT DISTINCT library.album, library.albumartist FROM uploads INNER JOIN library ON uploads.fileid=library.id WHERE uploads.memberid=? AND library.albumartist=? ORDER BY library.releaseyear DESC, library.album ASC", userId, albumArtist)
	if rows_err == nil {
		defer rows.Close()
		for rows.Next() {
			var album MusicAlbum
			scan_err := rows.Scan(&album.Album, &album.Albumartist)
			if scan_err == nil {
				albums = append(albums, album)
			}
		}
	}

	return albums
}

func GetFilesFromAlbumInUserLibrary(conn *sql.DB, userId int, albumArtist string, album string) []MusicFile {
	var musicFiles []MusicFile
	rows, rows_err := conn.QueryContext(context.Background(), "SELECT library.id, library.filehash, library.filename, library.mimetype, library.title, library.album, library.artist, library.albumartist, library.composer, library.genre, library.releaseyear, library.tracknumber, library.discnumber, library.date_added FROM uploads INNER JOIN library ON uploads.fileid=library.id WHERE uploads.memberid=? AND albumartist=? AND album=? ORDER BY library.discnumber, library.tracknumber ASC", userId, albumArtist, album)
	if rows_err == nil {
		defer rows.Close()
		for rows.Next() {
			var file MusicFile
			scan_err := rows.Scan(&file.Id, &file.Filehash, &file.Filename, &file.Mimetype, &file.Title, &file.Album, &file.Artist, &file.Albumartist, &file.Composer, &file.Genre, &file.Releaseyear, &file.Tracknumber, &file.Discnumber, &file.Date_added)
			if scan_err == nil {
				musicFiles = append(musicFiles, file)
			}
		}
	}

	return musicFiles
}

func GetUserQuota(conn *sql.DB, userId int) float64 {
	q := `SELECT COUNT(*) / (CAST(SUM(library.uploadcount) as float) / COUNT(*) * 1.0) FROM uploads INNER JOIN library ON uploads.fileid=library.id WHERE uploads.memberid=? GROUP BY library.uploadcount`
	rows, rows_err := conn.QueryContext(context.Background(), q, userId)
	if rows_err == nil {
		defer rows.Close()
		quotaCount := 0.0
		for rows.Next() {
			var i float64
			scan_err := rows.Scan(&i)
			if scan_err == nil {
				quotaCount += i
			}
		}

		return quotaCount
	}

	return 0
}

// ------ Admin stuff

func Admin_GetGlobalQuota(conn *sql.DB) float64 {
	q := `SELECT COUNT(DISTINCT filehash) FROM library`
	row := conn.QueryRowContext(context.Background(), q)
	var quotaCount float64
	err := row.Scan(&quotaCount)
	if err != nil {
		panic(err)
		return 0
	}

	return quotaCount
}

func Admin_UserCount(conn *sql.DB) int {
	q := `SELECT COUNT(DISTINCT id) FROM members`
	row := conn.QueryRowContext(context.Background(), q)
	var userCount int
	err := row.Scan(&userCount)
	if err != nil {
		panic(err)
		return 0
	}

	return userCount
}

func Admin_ArtistCount(conn *sql.DB) int {
	q := `SELECT COUNT(DISTINCT albumartist) FROM library`
	row := conn.QueryRowContext(context.Background(), q)
	var count int
	err := row.Scan(&count)
	if err != nil {
		panic(err)
		return 0
	}

	return count
}

func Admin_AlbumCount(conn *sql.DB) int {
	q := `SELECT COUNT(DISTINCT album || albumartist) FROM library`
	row := conn.QueryRowContext(context.Background(), q)
	var count int
	err := row.Scan(&count)
	if err != nil {
		panic(err)
		return 0
	}

	return count
}
