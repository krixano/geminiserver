package gemini

import (
	"context"
	"fmt"
	"mime"
	"path"
	"strconv"
	"strings"

	"github.com/krixano/ponixserver/src/config"
	"github.com/google/go-github/v35/github"
	"github.com/pitr/gig"
	"golang.org/x/oauth2"
)

var apiToken = config.GithubToken

func handleGithub(g *gig.Gig) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: apiToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	g.Handle("/github", func(c gig.Context) error {
		return c.Gemini(`# Ponix Github Proxy

Welcome to the Ponix Github proxy!

=> /github/search Search Repos
`)
	})

	g.Handle("/github/search", func(c gig.Context) error {
		query, err2 := c.QueryString()
		if err2 != nil {
			return err2
		} else if query == "" {
			return c.NoContent(gig.StatusInput, "Search Query:")
		} else {
			return handleGithubSearch(ctx, c, client, query, "")
		}
	})
	g.Handle("/github/search/:page", func(c gig.Context) error {
		query, err2 := c.QueryString()
		if err2 != nil {
			return err2
		} else if query == "" {
			return c.NoContent(gig.StatusInput, "Search Query:")
		} else {
			return handleGithubSearch(ctx, c, client, query, c.Param("page"))
		}
	})

	g.Handle("/github/repo/:id", func(c gig.Context) error {
		id := c.Param("id")
		template := `# Repo: %s

%s

SSH Url: %s
HTML Url: %s
Homepage: %s
License: %s

=> /github/repo/%d/b Branches

## Contents - Branch %s

%s
`
		id_int, err1 := strconv.Atoi(id)
		if err1 != nil {
			panic(err1)
		}

		repository, _, err2 := client.Repositories.GetByID(ctx, int64(id_int))
		if err2 != nil {
			panic(err2)
		}

		// TODO: README, README.md, readme.md
		/*opts := &github.RepositoryContentGetOptions{}
		readmeContents, _, _, err3 := client.Repositories.GetContents(ctx, repository.GetOwner().GetLogin(), repository.GetName(), "README.md", opts)
		readmeContents_str := ""
		var err4 error
		if err3 != nil {
			readmeContents, _, _, err3 = client.Repositories.GetContents(ctx, *repository.GetOwner().Login, repository.GetName(), "readme.md", opts)
			if err3 == nil {
				readmeContents_str, err4 = readmeContents.GetContent()
			}
		} else {
			readmeContents_str, err4 = readmeContents.GetContent()
		}
		if err4 != nil {
			panic(err4)
		}*/

		rootContents, _ := getRepoContents(ctx, client, repository, "")
		return c.Gemini(template, repository.GetFullName(), repository.GetDescription(), repository.GetSSHURL(), repository.GetHTMLURL(), repository.GetHomepage(), repository.GetLicense(), repository.GetID(), repository.GetDefaultBranch(), rootContents)
	})

	g.Handle("/github/repo/:id/b", func(c gig.Context) error {
		id := c.Param("id")

		id_int, err1 := strconv.Atoi(id)
		if err1 != nil {
			panic(err1)
		}

		repository, _, err2 := client.Repositories.GetByID(ctx, int64(id_int))
		if err2 != nil {
			panic(err2)
		}

		template := `# Repo: %s - Branches

%s`

		opts2 := &github.BranchListOptions{}
		branches, _, err3 := client.Repositories.ListBranches(ctx, repository.GetOwner().GetLogin(), repository.GetName(), opts2)
		if err3 != nil {
			panic(err3)
		}

		var builder strings.Builder
		for _, branch := range branches {
			fmt.Fprintf(&builder, "=> /github/repo/%d/b/%s %s\n", repository.GetID(), branch.GetName(), branch.GetName())
		}

		return c.Gemini(template, repository.GetFullName(), builder.String())
	})

	g.Handle("/github/repo/:id/files/*", func(c gig.Context) error {
		id := c.Param("id")
		route := fmt.Sprintf("/github/repo/%s/files", id)
		p := strings.Replace(c.URL().EscapedPath(), route, "", 1)

		id_int, err1 := strconv.Atoi(id)
		if err1 != nil {
			panic(err1)
		}

		repository, _, err2 := client.Repositories.GetByID(ctx, int64(id_int))
		if err2 != nil {
			panic(err2)
		}

		template := `# Repo Contents: %s - Branch %s

Path: %s

..
%s
`
		contents, isFile := getRepoContents(ctx, client, repository, p)
		if isFile {
			if strings.HasSuffix(p, ".gmi") || strings.HasSuffix(p, ".gemini") {
				return c.Gemini(contents)
			} else if strings.HasSuffix(p, ".md") {
				return c.Blob("text/markdown", []byte(contents))
			} else if strings.HasSuffix(p, ".rss") || strings.HasSuffix(p, ".atom") {
				return c.Blob("text/rss", []byte(contents))
			} else if strings.HasSuffix(p, ".gpub") {
				return c.Blob("application/gpub+zip", []byte(contents))
			} else {
				extension := path.Ext(p)
				mimeType := mime.TypeByExtension(extension)
				if mimeType == "" {
					//return c.Text(contents)
					return c.Blob("text/plain", []byte(contents))
				} else {
					return c.Blob(mimeType, []byte(contents))
				}
			}
		}
		return c.Gemini(template, repository.GetFullName(), repository.GetDefaultBranch(), p, contents)
	})
}

func getRepoContents(ctx context.Context, client *github.Client, repository *github.Repository, path string) (string, bool) {
	opts := &github.RepositoryContentGetOptions{}
	fileContents, dirContents, _, err := client.Repositories.GetContents(ctx, repository.GetOwner().GetLogin(), repository.GetName(), path, opts)
	if err != nil {
		panic(err) // TODO
	}

	var builder strings.Builder
	if dirContents != nil {
		for _, v := range dirContents {
			fmt.Fprintf(&builder, "=> /github/repo/%d/files/%s %s\n", repository.GetID(), v.GetPath(), v.GetName())
		}
		return builder.String(), false
	}
	if fileContents != nil {
		c, _ := fileContents.GetContent()
		return c, true
	}
	if dirContents == nil && fileContents == nil {
		fmt.Fprintf(&builder, "Not found")
		return builder.String(), false
	}

	return builder.String(), false
}

func handleGithubSearch(ctx context.Context, c gig.Context, client *github.Client, query string, page string) error {
	template := `# Github Search

=> /github/search New Search

%s`

	opts := &github.SearchOptions{}
	result, _, err := client.Search.Repositories(ctx, query, opts)
	if err != nil {
		panic(err)
	}

	var builder strings.Builder
	for _, repository := range result.Repositories {
		fmt.Fprintf(&builder, "=> /github/repo/%d %s\n%s\n\n", repository.GetID(), repository.GetFullName(), repository.GetDescription())
	}

	return c.Gemini(template, builder.String())
}
