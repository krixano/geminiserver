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

type MusicFile struct {
	id int
	filehash string
	filename string
	mimetype string
	title string
	album string
	artist string
	albumartist string
	composer string
	genre string
	releaseyear int
	tracknumber int
	discnumber int
	date_added time.Time
}

func NewConn() *sql.DB {
	connectionString := "GEMINI:Blue132401!@localhost:3050/var/lib/firebird/3.0/data/music.fdb"
	conn, err := sql.Open("firebirdsql", connectionString)

	if err != nil {
		//panic(oops.New(err, "failed to connect to database"))
		panic(err)
	}

	return conn
}

func main() {
	files, err := ioutil.ReadDir(".")
	if err != nil {
	    log.Fatal(err)
	}

	for _, fi := range files {
		if !strings.HasSuffix(fi.Name(), ".mp3") {
			continue;
		}

		f, _ := os.Open(fi.Name())
		fmt.Print(fi.Name() + "\n")
		m, tag_err := tag.ReadFrom(f)
		if tag_err != nil {
			log.Fatal(err)
		}

		hash, _ := tag.Sum(f)
		newName := hash + ".mp3" // TODO
		if newName != fi.Name() {
			//log.Print("test")
			os.Rename(fi.Name(), newName)
			fmt.Print("Renamed\n")
		} else {
			//log.Print(hash)
			fmt.Print("No need for rename\n")
		}

		conn := NewConn()
		defer conn.Close()

		// Check if already exists in database
		musicFile, exists := getFileInLibrary(conn, hash)
		if !exists {
			fmt.Print("Doesn't exist. Adding to library\n")
			musicFile, exists = addFileToLibrary(conn, hash, m)
		}

		// Add to user's account
		addFileToUser(conn, musicFile.id, 1)

		fmt.Printf("\n")
	}
}

func mimetypeAndExtensionFromFiletype(t tag.FileType) (string, string) {
	switch t {
	case tag.MP3: return "audio/mpeg", ".mp3"
	case tag.M4A: return "audio/mp4", ".m4a"
	case tag.M4B: return "audio/m4b", ".m4b"
	case tag.M4P: return "audio/m4p", ".m4p"
	case tag.ALAC: return "audio/m4a", ".m4a"
	case tag.FLAC: return "audio/flac", ".flac"
	case tag.OGG: return "audio/ogg", ".ogg"
	case tag.DSF: return "audio/x-dsf", ".dsf"
	default: return "application/octet-stream", ""
	}
}

func getFileInLibrary(conn *sql.DB, hash string) (MusicFile, bool) {
	query := `SELECT FIRST 1 library.id, library.filehash, library.filename, library.mimetype, library.title, library.album, library.artist, library.albumartist, library.composer, library.genre, library.releaseyear, library.tracknumber, library.discnumber, library.date_added FROM library WHERE filehash=?`

	row := conn.QueryRowContext(context.Background(), query, hash)
	
	var file MusicFile
	err := row.Scan(&file.id, &file.filehash, &file.filename, &file.mimetype, &file.title, &file.album, &file.artist, &file.albumartist, &file.composer, &file.genre, &file.releaseyear, &file.tracknumber, &file.discnumber, &file.date_added)
	if err == sql.ErrNoRows {
		return MusicFile{}, false
	}
	return file, true
}

func addFileToLibrary(conn *sql.DB, hash string, m tag.Metadata) (MusicFile, bool) {
	query := `INSERT INTO library (FILEHASH, FILENAME, MIMETYPE, TITLE, ALBUM, ARTIST, ALBUMARTIST, COMPOSER, GENRE, RELEASEYEAR, TRACKNUMBER, DISCNUMBER, UPLOADCOUNT, DATE_ADDED) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	trackNumber, _ := m.Track()
	discNumber, _ := m.Disc()
	mtype, ext := mimetypeAndExtensionFromFiletype(m.FileType())

    conn.ExecContext(context.Background(), query, hash, hash + ext, mtype, m.Title(), m.Album(), m.Artist(), m.AlbumArtist(), m.Composer(), m.Genre(), m.Year(), trackNumber, discNumber, 0, time.Now())

    return getFileInLibrary(conn, hash)
}

func getFileInUserLibrary(conn *sql.DB, musicFileId int, userId int) (MusicFile, bool) {
	query := `SELECT FIRST 1 library.id, library.filehash, library.filename, library.mimetype, library.title, library.album, library.artist, library.albumartist, library.composer, library.genre, library.releaseyear, library.tracknumber, library.discnumber, library.date_added FROM uploads INNER JOIN library ON uploads.fileid=library.id WHERE library.id=? AND uploads.memberid=?`

	row := conn.QueryRowContext(context.Background(), query, musicFileId, userId)
	
	var file MusicFile
	err := row.Scan(&file.id, &file.filehash, &file.filename, &file.mimetype, &file.title, &file.album, &file.artist, &file.albumartist, &file.composer, &file.genre, &file.releaseyear, &file.tracknumber, &file.discnumber, &file.date_added)
	if err == sql.ErrNoRows {
		return MusicFile{}, false
	}
	return file, true
}

func addFileToUser(conn *sql.DB, musicFileId int, userId int) {
	// Check first if user already has the file
	_, exists := getFileInUserLibrary(conn, musicFileId, userId)

	if !exists {
		conn.ExecContext(context.Background(), "INSERT INTO uploads (memberid, fileid, date_added) values (?, ?, ?)", userId, musicFileId, time.Now())
		conn.ExecContext(context.Background(), "UPDATE library SET uploadcount = uploadcount + 1 WHERE id=?", musicFileId)
	} else {
		fmt.Print("Already exists in user's library\n")
	}
}
