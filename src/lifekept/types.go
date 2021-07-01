package lifekept

import (
	"database/sql"
	"context"
	"time"
	"fmt"
)

type LifeKeptUser struct {
	Id int
	Username string
	Certificate string
	Language string
	Timezone *time.Location
	Is_staff bool
	Is_active bool
	Date_joined time.Time
}
type Collection struct {
	Id int
	Memberid int
	Name string
	Date_start time.Time
	Date_end time.Time
	Starred bool
	Date_created time.Time
}
type Bullet struct {
	Id int
	CollectionId int
	Parent int
	Content string
	Priority int
	Date_start time.Time
	Date_end time.Time
	Recurring string
	Date_created time.Time

	Children []Bullet
}

func GetUser(conn *sql.DB, certHash string) (LifeKeptUser, bool) {
	row := conn.QueryRowContext(context.Background(), "SELECT id, username, language, timezone, is_staff, is_active, date_joined FROM members WHERE certificate=?", certHash)

	var user LifeKeptUser
	var timezoneString string
	user.Certificate = certHash
	err := row.Scan(&user.Id, &user.Username, &user.Language, &timezoneString, &user.Is_staff, &user.Is_active, &user.Date_joined)
	if err == sql.ErrNoRows {
		return LifeKeptUser {}, false
	} else if err != nil {
		panic(err)
		return LifeKeptUser {}, false
	}

	user.Timezone, _ = time.LoadLocation(timezoneString)

	return user, true
}

func GetUserCollections(conn *sql.DB, userId int) []Collection {
	var collections []Collection
	rows, rows_err := conn.QueryContext(context.Background(), "SELECT id, memberid, name, date_start, date_end, starred, date_created FROM collections WHERE memberid=?", userId)
	if rows_err == nil {
		defer rows.Close()
		for rows.Next() {
			var collection Collection
			scan_err := rows.Scan(&collection.Id, &collection.Memberid, &collection.Name, &collection.Date_start, &collection.Date_end, &collection.Starred, &collection.Date_created)
			if scan_err == nil {
				collections = append(collections, collection)
			}
		}
	}

	return collections
}

func GetCollectionBullets(conn *sql.DB, collectionId int) []Bullet {
	var bullets []Bullet
	rows, rows_err := conn.QueryContext(context.Background(), "SELECT id, collectionid, parent, content, priority, date_start, date_end, recurring, date_created FROM bullets WHERE parent IS NULL AND collectionid=?", collectionId)
	if rows_err == nil {
		defer rows.Close()
		for rows.Next() {
			fmt.Print("Test")
			var bullet Bullet
			scan_err := rows.Scan(&bullet.Id, &bullet.CollectionId, &bullet.Parent, &bullet.Content, &bullet.Priority, &bullet.Date_start, &bullet.Date_end, &bullet.Recurring, &bullet.Date_created)
			if scan_err == nil {
				// Get children
				//bullet.Children = GetSubBullets(conn, bullet.Id)

				bullets = append(bullets, bullet)
			}
		}
	}

	return bullets
}

func GetSubBullets(conn *sql.DB, parentId int) []Bullet {
	var bullets []Bullet
	rows, rows_err := conn.QueryContext(context.Background(), "SELECT id, collectionid, parent, content, priority, date_start, date_end, recurring, date_created FROM bullets WHERE parent=?", parentId)
	if rows_err == nil {
		defer rows.Close()
		for rows.Next() {
			var bullet Bullet
			scan_err := rows.Scan(&bullet.Id, &bullet.CollectionId, &bullet.Parent, &bullet.Content, &bullet.Priority, &bullet.Date_start, &bullet.Date_end, &bullet.Recurring, &bullet.Date_created)
			if scan_err == nil {
				// Get children
				bullet.Children = GetSubBullets(conn, bullet.Id)

				bullets = append(bullets, bullet)
			}
		}
	}

	return bullets
}

func GetUserStarredCollections(conn *sql.DB, userId int) []Collection {
	var collections []Collection
	rows, rows_err := conn.QueryContext(context.Background(), "SELECT id, memberid, name, date_start, date_end, starred, date_created FROM collections WHERE memberid=? AND starred=true", userId)
	if rows_err == nil {
		defer rows.Close()
		for rows.Next() {
			var collection Collection
			scan_err := rows.Scan(&collection.Id, &collection.Memberid, &collection.Name, &collection.Date_start, &collection.Date_end, &collection.Starred, &collection.Date_created)
			if scan_err == nil {
				collections = append(collections, collection)
			}
		}
	}

	return collections
}

func AddCollectionToUser(conn *sql.DB, userId int, name string, date_start time.Time, date_end time.Time, starred bool) (Collection, bool) {
	row := conn.QueryRowContext(context.Background(), "INSERT INTO collections (memberid, name, date_start, date_end, starred, date_created) VALUES (?, ?, ?, ?, ?, ?) RETURNING id, date_created;", userId, name, date_start, date_end, starred, time.Now())
	var collection Collection
	err := row.Scan(&collection.Id, &collection.Date_created)
	if err == sql.ErrNoRows {
		return Collection {}, false
	} else if err != nil {
		panic(err)
		return Collection {}, false
	}

	collection.Name = name
	collection.Date_start = date_start
	collection.Date_end = date_end
	collection.Starred = starred

	return collection, true
}

func GetCollectionFromUser(conn *sql.DB, userId int, collectionId int) (Collection, bool) {
	row := conn.QueryRowContext(context.Background(), "SELECT FIRST 1 id, memberid, name, date_start, date_end, starred, date_created FROM collections WHERE memberid=? AND id=?", userId, collectionId)

	var collection Collection
	err := row.Scan(&collection.Id, &collection.Memberid, &collection.Name, &collection.Date_start, &collection.Date_end, &collection.Starred, &collection.Date_created)
	if err == sql.ErrNoRows {
		return Collection {}, false
	} else if err != nil {
		panic(err)
		return Collection {}, false
	}

	return collection, true
}

func AddBulletToCollection(conn *sql.DB, collectionId int, content string, priority int, date_start *time.Time, date_end *time.Time, recurring string) (Bullet, bool) {
	var bullet Bullet
	var row *sql.Row
	if date_start != nil && date_end != nil {
		row = conn.QueryRowContext(context.Background(), "INSERT INTO bullets (collectionid, content, priority, date_start, date_end, recurring, date_created) VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING id, date_created;", collectionId, content, priority, *date_start, *date_end, recurring, time.Now())
		bullet.Date_start = *date_start
		bullet.Date_end = *date_end
	} else if date_start == nil {
		row = conn.QueryRowContext(context.Background(), "INSERT INTO bullets (collectionid, content, priority, date_start, date_end, recurring, date_created) VALUES (?, ?, ?, NULL, NULL, ?, ?) RETURNING id, date_created;", collectionId, content, priority, recurring, time.Now())
	} else if date_end == nil {
		row = conn.QueryRowContext(context.Background(), "INSERT INTO bullets (collectionid, content, priority, date_start, date_end, recurring, date_created) VALUES (?, ?, ?, ?, NULL, ?, ?) RETURNING id, date_created;", collectionId, content, priority, *date_start, recurring, time.Now())
		bullet.Date_start = *date_start
	}

	err := row.Scan(&bullet.Id, &bullet.Date_created)
	if err == sql.ErrNoRows {
		return Bullet {}, false
	} else if err != nil {
		panic(err)
		return Bullet {}, false
	}

	bullet.Content = content
	bullet.Priority = priority
	bullet.Recurring = recurring

	return bullet, true
}

// Beginning of Day in a timezone
func BeginningOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
    return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

// End of Day in a timezone
func EndOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 11, 59, 59, 0, t.Location())
}
