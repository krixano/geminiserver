package lifekept

import (
	"context"
	"database/sql"
	"time"
	"fmt"
	"strings"
	"strconv"

	"github.com/krixano/ponixserver/src/db"
	"github.com/pitr/gig"
)

var registerNotification = `# LifeKept

You have selected a certificate that has not been registered yet. Please register here:

=> /lifekept/register Register Page
`

func HandleLifeKept(g *gig.Gig) {
	conn := db.NewConn(db.LifeKeptDB)
	conn.SetMaxOpenConns(500)
	conn.SetMaxIdleConns(3)
	conn.SetConnMaxLifetime(time.Hour * 4)

	g.Handle("/lifekept", func(c gig.Context) error {
		return c.NoContent(gig.StatusRedirectPermanent, "/lifekept/")
	})

	g.Handle("/lifekept/", func(c gig.Context) error {
		cert := c.Certificate()
		if cert == nil {
			return c.Gemini(`# LifeKept

Welcome to LifeKept. This is a Bullet Journal capsule for Gemini.

Note: Remember to make sure your certificate is selected on this page if you've already registered.

In order to register, create and enable a client certificate and then head over to the register page:

=> /lifekept/register Register Page
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

	g.Handle("/lifekept/collection/", func(c gig.Context) error {
		cert := c.Certificate()

		if cert == nil {
			return c.NoContent(gig.StatusClientCertificateRequired, "Please enable a certificate")
		} else {
			user, isRegistered := GetUser(conn, c.CertHash())
			if !isRegistered {
				return c.Gemini(registerNotification)
			} else {
				collections := GetUserCollections(conn, user.Id)
				var builder strings.Builder
				for _, collection := range collections {
					fmt.Fprintf(&builder, "=> /lifekept/collection/%d %s\n", collection.Id, collection.Name)
				}

				return c.Gemini(`# LifeKept

=> /lifekept/ Dashboard

## Collections
%s
`, builder.String())
			}
		}
	})

	g.Handle("/lifekept/collection/:id", func(c gig.Context) error {
		cert := c.Certificate()

		if cert == nil {
			return c.NoContent(gig.StatusClientCertificateRequired, "Please enable a certificate")
		} else {
			user, isRegistered := GetUser(conn, c.CertHash())
			if !isRegistered {
				return c.Gemini(registerNotification)
			} else {
				id, _ := strconv.Atoi(c.Param("id"))
				collection, exists := GetCollectionFromUser(conn, user.Id, id)
				if !exists {
					return c.NoContent(gig.StatusNotFound, "Not found")
				}

				bullets := GetCollectionBullets(conn, collection.Id)
				fmt.Print(bullets)
				var builder strings.Builder
				for _, bullet := range bullets {
					buildBulletString(&builder, bullet, 0)
				}

				return c.Gemini(`# LifeKept

=> /lifekept/ Dashboard
=> /lifekept/collection/ Index

=> /lifekept/collection/%d/star Star
=> /lifekept/collection/%d/range Set Date Range
=> /lifekept/collection/%d/delete Delete

## Collection: %s
%s

=> /lifekept/collection/%d/createnote Add Note
=> /lifekept/collection/%d/createtask Add Task
=> /lifekept/collection/%d/createevent Add Event
`, collection.Id, collection.Id, collection.Id, collection.Name, builder.String(), collection.Id, collection.Id, collection.Id)
			}
		}
	})

	g.Handle("/lifekept/collection/:id/createnote", func(c gig.Context) error {
		cert := c.Certificate()

		if cert == nil {
			return c.NoContent(gig.StatusClientCertificateRequired, "Please enable a certificate")
		} else {
			user, isRegistered := GetUser(conn, c.CertHash())
			if !isRegistered {
				return c.Gemini(registerNotification)
			} else {
				query, err2 := c.QueryString()
				if err2 != nil {
					return err2
				} else if query == "" {
					return c.NoContent(gig.StatusInput, "Enter content for new note:")
				} else {
					collectionId, _ := strconv.Atoi(c.Param("id"))
					_, exists := GetCollectionFromUser(conn, user.Id, collectionId)
					if !exists {
						return c.NoContent(gig.StatusNotFound, "Not found")
					}
					AddBulletToCollection(conn, collectionId, query, 0, nil, nil, "")

					return c.NoContent(gig.StatusRedirectTemporary, fmt.Sprintf("/lifekept/collection/%d", collectionId))
				}
			}
		}
	})

	g.Handle("/lifekept/collection/create", func(c gig.Context) error {
		cert := c.Certificate()

		if cert == nil {
			return c.NoContent(gig.StatusClientCertificateRequired, "Please enable a certificate")
		} else {
			user, isRegistered := GetUser(conn, c.CertHash())
			if !isRegistered {
				return c.Gemini(registerNotification)
			} else {
				query, err2 := c.QueryString()
				if err2 != nil {
					return err2
				} else if query == "" {
					return c.NoContent(gig.StatusInput, "Enter a name for the collection:")
				} else {
					now := time.Now().In(user.Timezone)
					today := BeginningOfDay(now).UTC()
					today_end := EndOfDay(now).UTC()
					collection, _ := AddCollectionToUser(conn, user.Id, query, today, today_end, false)

					return c.NoContent(gig.StatusRedirectTemporary, fmt.Sprintf("/lifekept/collection/%d", collection.Id))
				}
			}
		}
	})

	g.Handle("/lifekept/register", func(c gig.Context) error {
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
}

func buildBulletString(builder *strings.Builder, bullet Bullet, indent int) {
	for i := 0; i < indent; i++ {
		fmt.Fprintf(builder, "  ")
	}
	fmt.Fprintf(builder, "* %s\n", bullet.Content)

	for _, bullet := range bullet.Children {
		buildBulletString(builder, bullet, indent + 1)
	}
}

func getUserDashboard(c gig.Context, conn *sql.DB, user LifeKeptUser) error {
	template := `# LifeKept

=> /lifekept/collection/ Index
=> /lifekept/ Future Log
=> /lifekept/ Upcoming
=> /lifekept/ Settings
=> /lifekept/ Help

=> /lifekept/collection/create Create Collection

## Today's Log
* Test Note

=> /lifekept/ Create Note
=> /lifekept/ Create Task
=> /lifekept/ Create Event

## This Month's Log
* Test Note

=> /lifekept/ Create Note
=> /lifekept/ Create Task
=> /lifekept/ Create Event

## Starred Collections
%s
`

	starredCollections := GetUserStarredCollections(conn, user.Id)
	var builder strings.Builder
	for _, collection := range starredCollections {
		fmt.Fprintf(&builder, "=> /lifekept/collection/%d %s\n", collection.Id, collection.Name)
	}

	return c.Gemini(template)
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
		location := time.Now().Location()
		conn.ExecContext(context.Background(), "INSERT INTO members (certificate, username, language, timezone, is_staff, is_active, date_joined) VALUES (?, ?, ?, ?, ?, ?, ?)", certHash, username, "en-US", location, false, true, time.Now())

		//user, _ := GetUser(conn, certHash) // TODO
	}

	return c.NoContent(gig.StatusRedirectTemporary, "/lifekept/")
}

