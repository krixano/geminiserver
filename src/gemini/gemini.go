package gemini

import (
	"io"
	"io/ioutil"
	"os"

	"github.com/pitr/gig"
	"github.com/spf13/cobra"
	"github.com/krixano/ponixserver/src/lifekept"
	"github.com/krixano/ponixserver/src/music"
	"github.com/krixano/ponixserver/src/starwars"
	"github.com/krixano/ponixserver/src/search"
)

var GeminiCommand = &cobra.Command{
	Short: "Run the Gemini site",
	Run: func(cmd *cobra.Command, args []string) {
		f, _ := os.Create("access.log")
		gig.DefaultWriter = io.MultiWriter(f, os.Stdout)

		// Gig instance
		g := gig.Default()

		g.File("Poetry.pdf", "Poetry.pdf")
		g.File("/poetry.gpub", "./poetry.gpub")
		g.Handle("/poetry.gpub", func(c gig.Context) error {
			fileContents, err := ioutil.ReadFile("poetry.gpub")
			if err != nil {
				return err
			}

			return c.Blob("application/gpub+zip", fileContents)
		})

		// Routes
		g.Handle("/robots.txt", func(c gig.Context) error {
			return c.Text(`
User-agent: archiver
Disallow: /youtube
Disallow: /github/repo/*/files
Disallow: /searchengine/
User-agent: indexer
Disallow: /youtube
Disallow: /github/repo/*/files
Disallow: /searchengine/
User-agent: researcher
Disallow: /youtube
Disallow: /github/repo/*/files
Disallow: /searchengine/
`)
		})
		g.Handle("/favicon.txt", func(c gig.Context) error {
			return c.Text(`🧮`)
		})

		g.File("/", "./index.gmi")
		g.File("/proxies.gmi", "./proxies.gmi")
		g.File("/capsules.gmi", "./capsules.gmi")
		g.File("/feed.gmi", "./feed.gmi")
		g.File("/subscriptions.gmi", "./subscriptions.gmi")
		g.File("/comitium.json", "./comitium.json")
		g.File("/usefulrepos.gmi", "./usefulrepos.gmi")
		g.File("/GeminiHistoryMustReads.gmi", "./GeminiHistoryMustReads.gmi")

		g.File("/odin", "./odin/index.gmi")
		g.Static("/odin", "./odin")

		g.Static("/ponics", "./ponics")
		g.Static("/starwars", "./starwars")

		g.Handle("/~krixano", func(c gig.Context) error {
			return c.NoContent(gig.StatusRedirectTemporary, "/~krixano/")
		})
		g.Static("/~krixano", "./krixano")

		lifekept.HandleLifeKept(g)
		music.HandleMusic(g)
		starwars.HandleStarWars(g)
		search.HandleSearchEngine(g)
		handleDevlog(g)
		handleYoutube(g)
		handleGithub(g)
		handleWeather(g)

		// Start server on PORT or default port
		g.Run("cert.pem", "key.pem")
	},
}
