package gemini

import (
	// "fmt"
	// "strings"
	"github.com/pitr/gig"
	// owm "github.com/briandowns/openweathermap"
)

var weatherApiKey = "6c331bc0de358dd4ad5dd24beeb4d262"

func handleDashboard(g *gig.Gig) {
	g.Handle("/dashboard", func(c gig.Context) error {
		return c.NoContent(gig.StatusRedirectTemporary, "/dashboard/")
	})
	g.Handle("/dashboard/", func(c gig.Context) error {
		return c.Gemini(`# Dashboard

=> geminispace.info/search Search (geminispace.info)
=> /dashboard/register Register
`)

		/*template := `# Dashboard %s

=> geminispace.info/search Search (geminispace.info)

## Weather Overview

## LifeKept - Today's Log

## Feed

## Bookmarks

`*/
	})
}
