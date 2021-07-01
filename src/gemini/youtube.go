package gemini

import (
	"context"
	"fmt"
	"html"
	//"log"
	"strings"

	ytd "github.com/kkdai/youtube/v2"
	"github.com/pitr/gig"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
	"github.com/krixano/ponixserver/src/config"
)

var (
	youtubeAPIKey = config.YoutubeApiKey
	maxResults    = int64(25) /*flag.Int64("max-results", 25, "Max YouTube results")*/
)

func handleYoutube(g *gig.Gig) {
	// Create Youtube Service
	service, err1 := youtube.NewService(context.Background(), option.WithAPIKey(youtubeAPIKey))
	if err1 != nil {
		//log.Fatalf("Error creating new Youtube client: %v", err1)
		panic(err1)
	}

	g.Handle("/youtube", func(c gig.Context) error {
		return c.Gemini(`# Ponix YouTube Proxy

Welcome to the Ponix YouTube Proxy for Gemini!

=> /youtube/search Search
=> / Ponix Home
=> gemini://kwiecien.us/gemcast/20210425.gmi See This Proxy Featured on Gemini Radio
`)
	})

	g.Handle("/cgi-bin/youtube.cgi", func(c gig.Context) error {
		return c.NoContent(gig.StatusRedirectPermanent, "/youtube")
	})

	g.Handle("/youtube/search/:page", func(c gig.Context) error {
		query, err2 := c.QueryString()

		if err2 != nil {
			return err2
		} else if query == "" {
			return c.NoContent(gig.StatusInput, "Search Query:")
		} else {
			rawQuery := c.URL().RawQuery
			return searchYoutube(c, service, query, rawQuery, c.Param("page"))
		}
	})

	g.Handle("/youtube/search", func(c gig.Context) error {
		query, err2 := c.QueryString()

		if err2 != nil {
			return err2
		} else if query == "" {
			return c.NoContent(gig.StatusInput, "Search Query:")
		} else {
			rawQuery := c.URL().RawQuery
			return searchYoutube(c, service, query, rawQuery, "")
			/*return c.Gemini(fmt.Sprintf(, )*/
		}
	})

	handleVideoPage(g, service)
	handleVideoDownload(g, service)
	handleChannelPage(g, service)
	handlePlaylistPage(g, service)
}

func searchYoutube(c gig.Context, service *youtube.Service, query string, rawQuery string, currentPage string) error {
	template := `# Search

=> /youtube Home
=> /youtube/search New Search
%s`

	var call *youtube.SearchListCall
	if currentPage != "" {
		call = service.Search.List([]string{"snippet"}).Q(query).MaxResults(maxResults).PageToken(currentPage)
	} else {
		call = service.Search.List([]string{"snippet"}).Q(query).MaxResults(maxResults)
	}

	response, err := call.Do()
	if err != nil {
		//log.Fatalf("Error: %v", err)
		panic(err)
	}

	var builder strings.Builder
	if response.PrevPageToken != "" {
		fmt.Fprintf(&builder, "=> /youtube/search/%s?%s Previous Page\n", response.PrevPageToken, rawQuery)
	}
	if response.NextPageToken != "" {
		fmt.Fprintf(&builder, "=> /youtube/search/%s?%s Next Page\n", response.NextPageToken, rawQuery)
	}
	fmt.Fprintf(&builder, "\n## %d/%d Results for '%s'\n\n", response.PageInfo.ResultsPerPage, response.PageInfo.TotalResults, query)

	for _, item := range response.Items {
		switch item.Id.Kind {
		case "youtube#video":
			fmt.Fprintf(&builder, "=> /youtube/video/%s Video: %s\nUploaded by %s\n\n", item.Id.VideoId, html.UnescapeString(item.Snippet.Title), html.UnescapeString(item.Snippet.ChannelTitle))
		case "youtube#channel":
			fmt.Fprintf(&builder, "=> /youtube/channel/%s Channel: %s\n\n", item.Id.ChannelId, html.UnescapeString(item.Snippet.Title))
		case "youtube#playlist":
			fmt.Fprintf(&builder, "=> /youtube/playlist/%s Playlist: %s\n\n", item.Id.PlaylistId, html.UnescapeString(item.Snippet.Title))
		}
	}

	return c.Gemini(template, builder.String())
}

func handleVideoPage(g *gig.Gig, service *youtube.Service) {
	g.Handle("/youtube/video/:id", func(c gig.Context) error {
		call := service.Videos.List([]string{"id", "snippet"}).Id(c.Param("id")).MaxResults(1)
		response, err := call.Do()
		if err != nil {
			//log.Fatalf("Error: %v", err) // TODO
			panic(err)
		}

		video := response.Items[0]

		return c.Gemini(`# Video: %s

=> /youtube/downloadVideo/%s Download Video
=> https://youtube.com/watch?v=%s On YouTube

## Description
%s
=> /youtube/channel/%s Uploaded by %s
`, html.UnescapeString(video.Snippet.Title), video.Id, video.Id, html.UnescapeString(video.Snippet.Description), video.Snippet.ChannelId, html.UnescapeString(video.Snippet.ChannelTitle))
	})
}

func filterYT(fl ytd.FormatList, test func(ytd.Format) bool) (ret []ytd.Format) {
	for _, format := range fl {
		if test(format) {
			ret = append(ret, format)
		}
	}
	return
}

func handleVideoDownload(g *gig.Gig, service *youtube.Service) {
	g.Handle("/youtube/downloadVideo/:id", func(c gig.Context) error {
		client := ytd.Client{}
		video, err := client.GetVideo(c.Param("id"))
		if err != nil {
			panic(err)
		}

		//format := video.Formats.AudioChannels(2).FindByQuality("medium")
		video.Formats.Sort()

		formats := filterYT(video.Formats, func(format ytd.Format) bool {
			return format.AudioChannels != 0
		})

		fmt.Printf("Format: %v\n", formats)

		format := ytd.FormatList(formats).FindByQuality("medium")
		//format := video.Formats.AudioChannels(2).FindByQuality("hd1080")
		//.DownloadSeparatedStreams(ctx, "", video, "hd1080", "mp4")
		//resp, err := client.GetStream(video, format)

		rc, _, err := client.GetStream(video, format)
		if err != nil {
			return c.Gemini("Error: Video Not Found\n%v", err)
		}
		err2 := c.Stream(format.MimeType, rc)
		rc.Close()

		//url, err := client.GetStreamURL(video, format)



		return err2
	})
}

func handleChannelPage(g *gig.Gig, service *youtube.Service) {
	// Channel Home
	g.Handle("/youtube/channel/:id", func(c gig.Context) error {
		template := `# Channel: %s

=> /youtube/channel/%s/videos All Videos
=> /youtube/channel/%s/playlists Playlists
=> /youtube/channel/%s/communityposts Community Posts
=> /youtube/channel/%s/activity Gemini Sub Activity Feed

## About
%s

## Recent Videos

=> /youtube/channel/%s/videos All Videos
`
		call := service.Channels.List([]string{"id", "snippet", "contentDetails"}).Id(c.Param("id")).MaxResults(1)
		response, err := call.Do()
		if err != nil {
			//log.Fatalf("Error: %v", err) // TODO
			panic(err)
		}

		channel := response.Items[0]

		return c.Gemini(template, html.UnescapeString(channel.Snippet.Title), channel.Id, channel.Id, channel.Id, channel.Id, html.UnescapeString(channel.Snippet.Description), channel.Id)
	})

	// Channel Playlists
	g.Handle("/youtube/channel/:id/playlists/:page", func(c gig.Context) error {
		return getChannelPlaylists(c, service, c.Param("id"), c.Param("page"))
	})
	g.Handle("/youtube/channel/:id/playlists", func(c gig.Context) error {
		return getChannelPlaylists(c, service, c.Param("id"), "")
	})

	// Channel Videos/Uploads
	g.Handle("/youtube/channel/:id/videos/:page", func(c gig.Context) error {
		return getChannelVideos(c, service, c.Param("id"), c.Param("page"))
	})
	g.Handle("/youtube/channel/:id/videos", func(c gig.Context) error {
		return getChannelVideos(c, service, c.Param("id"), "")
	})

	g.Handle("/youtube/channel/:id/activity", func(c gig.Context) error {
		return getChannelActivity(c, service, c.Param("id"))
	})
}

func getChannelPlaylists(c gig.Context, service *youtube.Service, channelId string, currentPage string) error {
	template := `# Playlists for '%s'

=> /youtube/channel/%s ChannelPage
%s`

	var call *youtube.PlaylistsListCall
	if currentPage != "" {
		call = service.Playlists.List([]string{"id", "snippet"}).ChannelId(channelId).MaxResults(50).PageToken(currentPage)
	} else {
		call = service.Playlists.List([]string{"id", "snippet"}).ChannelId(channelId).MaxResults(50)
	}
	response, err := call.Do()
	if err != nil {
		//log.Fatalf("Error: %v", err)
		panic(err)
	}

	var builder strings.Builder
	if response.PrevPageToken != "" {
		fmt.Fprintf(&builder, "=> /youtube/channel/%s/playlists/%s Previous Page\n", channelId, response.PrevPageToken)
	}
	if response.NextPageToken != "" {
		fmt.Fprintf(&builder, "=> /youtube/channel/%s/playlists/%s Next Page\n", channelId, response.NextPageToken)
	}
	fmt.Fprintf(&builder, "\n")

	for _, item := range response.Items {
		fmt.Fprintf(&builder, "=> /youtube/playlist/%s %s\n", item.Id, html.UnescapeString(item.Snippet.Title))
	}

	return c.Gemini(template, html.UnescapeString(response.Items[0].Snippet.ChannelTitle), response.Items[0].Snippet.ChannelId, builder.String())
}

func getChannelVideos(c gig.Context, service *youtube.Service, channelId string, currentPage string) error {
	template := `# Uploads for '%s'

=> /youtube/channel/%s Channel Page
%s`

	call := service.Channels.List([]string{"id", "snippet", "contentDetails"}).Id(channelId).MaxResults(1)
	response, err := call.Do()
	if err != nil {
		//log.Fatalf("Error: %v", err)
		panic(err)
	}

	channel := response.Items[0]
	uploadsPlaylistId := channel.ContentDetails.RelatedPlaylists.Uploads

	var call2 *youtube.PlaylistItemsListCall
	if currentPage != "" {
		call2 = service.PlaylistItems.List([]string{"id", "snippet"}).PlaylistId(uploadsPlaylistId).MaxResults(25).PageToken(currentPage)
	} else {
		call2 = service.PlaylistItems.List([]string{"id", "snippet"}).PlaylistId(uploadsPlaylistId).MaxResults(25)
	}
	response2, err2 := call2.Do()
	if err2 != nil {
		//log.Fatalf("Error: %v", err)
		panic(err)
	}

	var builder strings.Builder
	if response2.PrevPageToken != "" {
		fmt.Fprintf(&builder, "=> /youtube/channel/%s/videos/%s Previous Page\n", channelId, response2.PrevPageToken)
	}
	if response2.NextPageToken != "" {
		fmt.Fprintf(&builder, "=> /youtube/channel/%s/videos/%s Next Page\n", channelId, response2.NextPageToken)
	}
	fmt.Fprintf(&builder, "\n")

	for _, item := range response2.Items {
		date := strings.Split(item.Snippet.PublishedAt, "T")[0]
		fmt.Fprintf(&builder, "=> /youtube/video/%s %s %s\n", item.Snippet.ResourceId.VideoId, date, html.UnescapeString(item.Snippet.Title))
	}

	return c.Gemini(template, html.UnescapeString(channel.Snippet.Title), channel.Id, builder.String())
}

func getChannelActivity(c gig.Context, service *youtube.Service, channelId string) error {
	template := `# Activity for '%s'

%s`

	call := service.Channels.List([]string{"id", "snippet", "contentDetails"}).Id(channelId).MaxResults(1)
	response, err := call.Do()
	if err != nil {
		//log.Fatalf("Error: %v", err)
		panic(err)
	}

	channel := response.Items[0]
	uploadsPlaylistId := channel.ContentDetails.RelatedPlaylists.Uploads

	call2 := service.PlaylistItems.List([]string{"id", "snippet"}).PlaylistId(uploadsPlaylistId).MaxResults(100) // TODO
	response2, err2 := call2.Do()
	if err2 != nil {
		//log.Fatalf("Error: %v", err)
		panic(err)
	}

	var builder strings.Builder
	for _, item := range response2.Items {
		date := strings.Split(item.Snippet.PublishedAt, "T")[0]
		fmt.Fprintf(&builder, "=> /youtube/video/%s %s %s\n", item.Snippet.ResourceId.VideoId, date, html.UnescapeString(item.Snippet.Title))
	}

	return c.Gemini(template, html.UnescapeString(channel.Snippet.Title), builder.String())
}

func handlePlaylistPage(g *gig.Gig, service *youtube.Service) {
	g.Handle("/youtube/playlist/:id/:page", func(c gig.Context) error {
		return getPlaylistVideos(c, service, c.Param("id"), c.Param("page"))
	})
	g.Handle("/youtube/playlist/:id", func(c gig.Context) error {
		return getPlaylistVideos(c, service, c.Param("id"), "")
	})
}

func getPlaylistVideos(c gig.Context, service *youtube.Service, playlistId string, currentPage string) error {
	template := `# Playlist: %s

=> /youtube/channel/%s Created by %s
%s`

	call_pl := service.Playlists.List([]string{"id", "snippet"}).Id(playlistId).MaxResults(1)
	response_pl, err_pl := call_pl.Do()
	if err_pl != nil {
		//log.Fatalf("Error: %v", err_pl)
		panic(err_pl)
	}

	playlist := response_pl.Items[0]
	playlistTitle := playlist.Snippet.Title

	var call *youtube.PlaylistItemsListCall
	if currentPage != "" {
		call = service.PlaylistItems.List([]string{"id", "snippet"}).PlaylistId(playlistId).MaxResults(50).PageToken(currentPage)
	} else {
		call = service.PlaylistItems.List([]string{"id", "snippet"}).PlaylistId(playlistId).MaxResults(50)
	}
	response, err := call.Do()
	if err != nil {
		//log.Fatalf("Error: %v", err)
		panic(err)
	}

	var builder strings.Builder
	if response.PrevPageToken != "" {
		fmt.Fprintf(&builder, "=> /youtube/playlist/%s/%s Previous Page\n", playlistId, response.PrevPageToken)
	}
	if response.NextPageToken != "" {
		fmt.Fprintf(&builder, "=> /youtube/playlist/%s/%s Next Page\n", playlistId, response.NextPageToken)
	}
	fmt.Fprintf(&builder, "\n")

	for _, item := range response.Items {
		fmt.Fprintf(&builder, "=> /youtube/video/%s %s\nUploaded by %s\n\n", item.Snippet.ResourceId.VideoId, html.UnescapeString(item.Snippet.Title), html.UnescapeString(item.Snippet.ChannelTitle))
	}

	return c.Gemini(template, playlistTitle, playlist.Snippet.ChannelId, html.UnescapeString(playlist.Snippet.ChannelTitle), builder.String())
}
