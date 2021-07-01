package starwars

import (
	"strings"
	"math"
	"fmt"
	"strconv"
	"github.com/rivo/uniseg"
)

func constructTableDataFromMovies(movies []Movie) ([]string, [][]string) {
	var rows [][]string
	for _, movie := range movies {
		cols := make([]string, 3)
		cols[0] = movie.Name
		year, aby := convertIntToStarWarsYear(movie.TimelineDate)
			if !aby {
				cols[1] = fmt.Sprintf("%d BBY", year)
			} else {
				cols[1] = fmt.Sprintf("%d ABY", year)
		}
		year, month, day := movie.PublicationDate.Date()
		cols[2] = fmt.Sprintf("%04d-%02d-%02d", year, month, day)

		rows = append(rows, cols)
	}
	return []string { "TITLE", "BBY/ABY", "PUBLICATION" }, rows
}

func constructTableDataFromShows(shows []TVShow) ([]string, [][]string) {
	var rows [][]string
	for _, show := range shows {
		cols := make([]string, 1)
		cols[0] = show.Name

		rows = append(rows, cols)
	}
	return []string { "TITLE" }, rows
}

func constructTableDataFromSeries(series []ComicSeries) ([]string, [][]string) {
	var rows [][]string
	for _, serie := range series {
		cols := make([]string, 4)
		cols[0] = serie.Name
		cols[1] = strconv.Itoa(serie.IssueCount)
		cols[2] = strconv.Itoa(serie.TPBCount)
		cols[3] = strconv.Itoa(serie.StartYear)

		rows = append(rows, cols)
	}
	return []string { "TITLE", "ISSUES", "TPBs", "YEAR" }, rows
}

func constructTableDataFromCrossover(tpbs []ComicTPB) ([]string, [][]string) {
	var rows [][]string
	for _, tpb := range tpbs {
		cols := make([]string, 2)
		cols[0] = tpb.Name
		year, month, day := tpb.PublicationDate.Date()
		cols[1] = fmt.Sprintf("%04d-%02d-%02d", year, month, day)

		rows = append(rows, cols)
	}
	return []string { "TITLE", "PUBLICATION" }, rows
}

func constructTableDataFromTPBs(tpbs []ComicTPB) ([]string, [][]string) {
	var rows [][]string
	for _, tpb := range tpbs {
		cols := make([]string, 5)
		if tpb.Series.Name != "" {
			cols[0] = fmt.Sprintf("%s (%d)", tpb.Series.Name, tpb.Series.StartYear)
		} else {
			cols[0] = "-"
		}
		cols[1] = strconv.Itoa(tpb.Volume)
		cols[2] = tpb.Name
		cols[3] = strconv.Itoa(tpb.IssueCount)
		year, month, day := tpb.PublicationDate.Date()
		cols[4] = fmt.Sprintf("%04d-%02d-%02d", year, month, day)

		rows = append(rows, cols)
	}
	return []string { "SERIES", "VOL", "TITLE", "ISSUES", "PUBLICATION" }, rows
}

func constructTableDataFromIssues(issues []ComicIssue) ([]string, [][]string) {
	var rows [][]string
	for _, issue := range issues {
		cols := make([]string, 6)
		if issue.Series.Name != "" {
			cols[0] = fmt.Sprintf("%s (%d)", issue.Series.Name, issue.Series.StartYear)
			cols[1] = strconv.Itoa(issue.Number)
			if issue.Annual {
				cols[1] = "A" + cols[1]
			}
		} else {
			cols[0] = "-"
			cols[1] = "-"
		}
		cols[2] = issue.Name

		year, aby := convertIntToStarWarsYear(issue.TimelineDate)
		if !aby {
				cols[3] = fmt.Sprintf("%d BBY", year)
			} else {
				cols[3] = fmt.Sprintf("%d ABY", year)
		}

		year, month, day := issue.PublicationDate.Date()
		cols[4] = fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		cols[5] = issue.Publisher

		rows = append(rows, cols)
	}
	return []string { "SERIES", "#", "TITLE", "BBY/ABY", "PUBLICATION", "PUBLISHER" }, rows
}

func constructTableDataFromOneshots(issues []ComicIssue) ([]string, [][]string) {
	var rows [][]string
	for _, issue := range issues {
		cols := make([]string, 4)
		cols[0] = issue.Name

		year, aby := convertIntToStarWarsYear(issue.TimelineDate)
		if !aby {
				cols[1] = fmt.Sprintf("%d BBY", year)
			} else {
				cols[1] = fmt.Sprintf("%d ABY", year)
		}

		year, month, day := issue.PublicationDate.Date()
		cols[2] = fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		cols[3] = issue.Publisher

		rows = append(rows, cols)
	}
	return []string { "TITLE", "BBY/ABY", "PUBLICATION", "PUBLISHER" }, rows
}

func constructTableDataFromBookSeries(series []BookSeries) ([]string, [][]string) {
	var rows [][]string
	for _, serie := range series {
		cols := make([]string, 2)
		cols[0] = serie.Name
		cols[1] = strconv.Itoa(serie.BookCount)

		rows = append(rows, cols)
	}
	return []string { "TITLE", "BOOKS" }, rows
}

func constructTableDataFromBooks(books []Book) ([]string, [][]string) {
	var rows [][]string
	for _, book := range books {
		cols := make([]string, 6)
		if book.Series.Name != "" {
			cols[0] = fmt.Sprintf("%s", book.Series.Name)
			cols[1] = strconv.Itoa(book.Number)
		} else {
			cols[0] = "-"
			cols[1] = "-"
		}

		cols[2] = book.Name
		cols[3] = book.Author

		year, aby := convertIntToStarWarsYear(book.TimelineDate)
		if !aby {
				cols[4] = fmt.Sprintf("%d BBY", year)
			} else {
				cols[4] = fmt.Sprintf("%d ABY", year)
		}

		year, month, day := book.PublicationDate.Date()
		cols[5] = fmt.Sprintf("%04d-%02d-%02d", year, month, day)

		rows = append(rows, cols)
	}
	return []string { "SERIES", "#", "TITLE", "AUTHOR", "BBY/ABY", "PUBLICATION" }, rows // TODO: Add Publisher?
}

func constructTableDataFromBookStandalones(books []Book) ([]string, [][]string) {
	var rows [][]string
	for _, book := range books {
		cols := make([]string, 4)

		cols[0] = book.Name
		cols[1] = book.Author

		year, aby := convertIntToStarWarsYear(book.TimelineDate)
		if !aby {
				cols[2] = fmt.Sprintf("%d BBY", year)
			} else {
				cols[2] = fmt.Sprintf("%d ABY", year)
		}

		year, month, day := book.PublicationDate.Date()
		cols[3] = fmt.Sprintf("%04d-%02d-%02d", year, month, day)

		rows = append(rows, cols)
	}
	return []string { "TITLE", "AUTHOR", "BBY/ABY", "PUBLICATION" }, rows // TODO: Add Publisher?
}

func constructTable(headingRow []string, data [][]string) string {
	if len(data) == 0 {
		return ""
	}
	cellLengthLimit := 37
	var builder strings.Builder

	// Get maximum length of each column and number of lines (for overflow)
	colLengths := make([]int, len(data[0]))
	rowLines := make([]int, len(data))
	for colNum, col := range headingRow {
		graphemeCount := uniseg.GraphemeClusterCount(col)
		if graphemeCount > colLengths[colNum] && graphemeCount <= cellLengthLimit {
			colLengths[colNum] = graphemeCount
		}
	}
	for rowNum, row := range data {
		for colNum, col := range row {
			graphemeCount := uniseg.GraphemeClusterCount(col)
			if graphemeCount > colLengths[colNum] && graphemeCount <= cellLengthLimit {
				colLengths[colNum] = graphemeCount
			}

			lines := int(math.Ceil(float64(graphemeCount) / float64(cellLengthLimit)))
			if lines > rowLines[rowNum] {
				rowLines[rowNum] = lines
			}
		}
	}

	// Construct heading row - First Line
	for colNum, _ := range headingRow {
		length := colLengths[colNum]

		if colNum == 0 {
			fmt.Fprintf(&builder, "╔═")
		} else {
			fmt.Fprintf(&builder, "╤═")
		}

		for i := 0; i < length + 1; i++ {
			fmt.Fprintf(&builder, "═")
		}
	}
	fmt.Fprintf(&builder, "╗\n")

	// Heading Row Contents
	for colNum, col := range headingRow {
		diff := colLengths[colNum] - len(col)

		if colNum == 0 {
			fmt.Fprintf(&builder, "║ ")
		} else {
			fmt.Fprintf(&builder, "│ ")
		}

		fmt.Fprintf(&builder, "%s", col)
		for i := 0; i < diff + 1; i++ {
			fmt.Fprintf(&builder, " ")
		}
	}
	fmt.Fprintf(&builder, "║\n")

	// Heading Row Bottom
	for colNum, _ := range headingRow {
		length := colLengths[colNum]

		if colNum == 0 {
			fmt.Fprintf(&builder, "╠═")
		} else {
			fmt.Fprintf(&builder, "╪═")
		}

		for i := 0; i < length + 1; i++ {
			fmt.Fprintf(&builder, "═")
		}
	}
	fmt.Fprintf(&builder, "╣\n")

	// Data
	for rowNum, row := range data { // TODO: I'm pretty sure this is very slow
		// Contents
		for rowLine := 1; rowLine <= rowLines[rowNum]; rowLine++ {
			for colNum, col := range row {
				if colNum == 0 {
					fmt.Fprintf(&builder, "║ ")
				} else {
					fmt.Fprintf(&builder, "│ ")
				}

				graphemeCount := 0
				graphemeIndex := 0
				gr := uniseg.NewGraphemes(col)

				start := (cellLengthLimit * rowLine) - cellLengthLimit - 1
				end := (cellLengthLimit * rowLine) - 1
				for gr.Next() {
					if graphemeIndex >= start && graphemeIndex < end {
						graphemeCount++
						runes := gr.Runes()
						fmt.Fprintf(&builder, "%s", string(runes))
					}
					graphemeIndex++
				}

				diff := colLengths[colNum] - graphemeCount

				/*if len(runes) > 0 {
					start := (cellLengthLimit * rowLine) - cellLengthLimit
					if start < len(runes) {
						end := (cellLengthLimit * rowLine)
						if end >= len(runes) {
							end = len(runes)
						}
						content := string(runes[start:end])
						fmt.Fprintf(&builder, "%s", content)
						diff = colLengths[colNum] - len(content)
					}
				}*/
				for i := 0; i < diff + 1; i++ {
					fmt.Fprintf(&builder, " ")
				}
			}
			fmt.Fprintf(&builder, "║\n")
		}

		// Bottom
		if rowNum == len(data) - 1 {
			for colNum, _ := range row {
				length := colLengths[colNum]

				if colNum == 0 {
					fmt.Fprintf(&builder, "╚═")
				} else {
					fmt.Fprintf(&builder, "╧═")
				}

				for i := 0; i < length + 1; i++ {
					fmt.Fprintf(&builder, "═")
				}
			}
			fmt.Fprintf(&builder, "╝\n")
		} else {
			for colNum, _ := range row {
				length := colLengths[colNum]

				if colNum == 0 {
					fmt.Fprintf(&builder, "╟─")
				} else {
					fmt.Fprintf(&builder, "┼─")
				}

				for i := 0; i < length + 1; i++ {
					fmt.Fprintf(&builder, "─")
				}
			}
			fmt.Fprintf(&builder, "╢\n")
		}
	}

	return builder.String()
}
