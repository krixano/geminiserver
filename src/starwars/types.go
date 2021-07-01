package starwars

import (
	"time"
)

// Franchise

type ComicSeries struct {
	Id int
	Name string
	Oneoff bool
	Miniseries bool
	StartYear int
	Date_added time.Time

	TPBCount int
	IssueCount int
	// AnnualCount int
}

type comicSeriesNullable struct {
	Id interface{}
	Name interface{}
	Oneoff interface{}
	Miniseries interface{}
	StartYear interface{}
	Date_added interface{}
}

func comicSeriesScan(series comicSeriesNullable) ComicSeries {
	var result ComicSeries
	if series.Id != nil {
		result.Id = int(series.Id.(int32))
	}
	if series.Name != nil {
		result.Name = string(series.Name.([]uint8))
	}
	if series.Oneoff != nil {
		result.Oneoff = series.Oneoff.(bool)
	}
	if series.Miniseries != nil {
		result.Miniseries = series.Miniseries.(bool)
	}
	if series.StartYear != nil {
		result.StartYear = int(series.StartYear.(int32))
	}
	if series.Date_added != nil {
		result.Date_added = series.Date_added.(time.Time)
	}

	return result
}

type ComicTPB struct {
	Id int
	Volume int
	Name string
	Crossover bool

	ComicSeriesId int // Optional
	
	TimelineDate int // NOTE: Only used if TPB doesn't consist of individual issues
	PublicationDate time.Time
	Date_added time.Time

	Series ComicSeries
	IssueCount int
}

type ComicIssue struct {
	Id int
	Number int
	Name string
	Annual bool
	
	ComicSeriesId int
	ComicTPBId int
	ComicCrossoverId int

	TimelineDate int
	PublicationDate time.Time
	Publisher string // Marvel, IDW
	Date_added time.Time

	Series ComicSeries
	//TPB ComicTPB // TODO
}

/*type ComicOmnibus struct {
	Id int
	Volume int
	Name string
	ComicSeriesId int
	TimelineDate string
	PublicationDate time.Time
	Date_added int
}*/

type BookSeries struct {
	Id int
	Name string
	Date_added time.Time

	BookCount int
}

type bookSeriesNullable struct {
	Id interface{}
	Name interface{}
	Date_added interface{}
}


func bookSeriesScan(serie bookSeriesNullable) BookSeries {
	var result BookSeries
	if serie.Id != nil {
		result.Id = int(serie.Id.(int32))
	}
	if serie.Name != nil {
		result.Name = string(serie.Name.([]uint8))
	}
	if serie.Date_added != nil {
		result.Date_added = serie.Date_added.(time.Time)
	}

	return result
}

type Book struct {
	Id int
	Number int
	Name string
	BookType string // Adult, YA, Junior, Young, ShortStory
	Author string
	BookSeriesId int
	TimelineDate int
	PublicationDate time.Time
	Publisher string
	Date_added time.Time

	Series BookSeries
}

type bookNullable struct {
	Id interface{}
	Number interface{}
	Name interface{}
	BookType interface{}
	Author interface{}
	BookSeriesId interface{}
	TimelineDate interface{}
	PublicationDate interface{}
	Publisher interface{}
	Date_added interface{}
}

func bookScan(book bookNullable) Book {
	var result Book
	if book.Id != nil {
		result.Id = int(book.Id.(int32))
	}
	if book.Number != nil {
		result.Number = int(book.Number.(int32))
	}
	if book.Name != nil {
		result.Name = string(book.Name.([]uint8))
	}
	if book.BookType != nil {
		result.BookType = string(book.BookType.([]uint8))
	}
	if book.Author != nil {
		result.Author = string(book.Author.([]uint8))
	}
	if book.BookSeriesId != nil {
		result.BookSeriesId = int(book.BookSeriesId.(int32))
	}
	if book.TimelineDate != nil {
		result.TimelineDate = int(book.TimelineDate.(int32))
	}
	if book.PublicationDate != nil {
		result.PublicationDate = book.PublicationDate.(time.Time)
	}
	if book.Publisher != nil {
		result.Publisher = string(book.Publisher.([]uint8))
	}
	if book.Date_added != nil {
		result.Date_added = book.Date_added.(time.Time)
	}

	return result
}

type Movie struct {
	Id int
	Name string
	TimelineDate int
	PublicationDate time.Time
	ProductionCompany string
	Distributor string
	Date_added time.Time
}

type TVShow struct {
	Id int
	Name string
	Date_added time.Time
}

type TVShowEpisode struct {
	Id int
	Number int
	Season int
	Name string
	TvShowId int
	TimelineDate int
	PublicationDate time.Time
	Date_added time.Time
}

// -1 = 0 BBY; 0 = 0 ABY
// (-timeline + 1) * -1 = BBY; BBY - 1 = -timeline
func convertStarWarsYearToInt(year int, aby bool) int {
	if !aby {
		return year - 1
	} else {
		return year
	}
}

func convertIntToStarWarsYear(timeline int) (int, bool) {
	if timeline < 0 { // BBY
		return (timeline + 1) * -1, false
	} else { // ABY
		return timeline, true
	}
}
