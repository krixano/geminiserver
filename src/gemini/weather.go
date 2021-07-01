package gemini

import (
	"fmt"
	"strings"
	"github.com/pitr/gig"
	owm "github.com/briandowns/openweathermap"
	"github.com/krixano/ponixserver/src/config"
)

var apiKey = config.WeatherApiKey

func handleWeather(g *gig.Gig) {
	g.Handle("/weather", func(c gig.Context) error {
		return c.NoContent(gig.StatusRedirectTemporary, "/weather/")
	})
	g.Handle("/weather/", func(c gig.Context) error {
		return c.Gemini(`# Weather

=> /weather/imperial/location Get Weather By Location (Imperial)
=> /weather/metric/location Get Weather By Location (Metric)

=> / Ponix Home
=> https://openweathermap.org Powered by openweathermap.org
`)
	})

	g.Handle("/weather/imperial/location", func(c gig.Context) error {
		query, err2 := c.QueryString()
		if err2 != nil {
			return err2
		} else if query == "" {
			return c.NoContent(gig.StatusInput, "Enter a Location ('City, Country' or 'City, State, Country'):")
		} else {
			return handleLocation(c, query, true)
		}
	})

	g.Handle("/weather/metric/location", func(c gig.Context) error {
		query, err2 := c.QueryString()
		if err2 != nil {
			return err2
		} else if query == "" {
			return c.NoContent(gig.StatusInput, "Enter a Location ('City, Country' or 'City, State, Country'):")
		} else {
			return handleLocation(c, query, false)
		}
	})
}

func handleLocation(c gig.Context, query string, imperial bool) error {
	var err error
	var w *owm.CurrentWeatherData
	var coords owm.Coordinates
	// var forecast *owm.ForecastWeatherData

	var temp string
	var feelsLike string
	var windSpeed string
	var rain string
	var snow string

	if imperial {
		w, err = owm.NewCurrent("F", "EN", apiKey)
		w.CurrentByName(query)
		coords = w.GeoPos
		if err != nil {
			panic(err)
		}
		/*forecast, err = owm.NewForecast("5", "F", "EN", apiKey)
		if err != nil {
			panic(err)
		}
		forecast.DailyByCoordinates(&coords, 5)*/

		temp = fmt.Sprintf("%.1f°F (%.1f°F/%.1f°F)", w.Main.Temp, w.Main.TempMax, w.Main.TempMin)
		feelsLike = fmt.Sprintf("%.1f°F", w.Main.FeelsLike)
		windSpeed = fmt.Sprintf("%.1f mph", w.Wind.Speed)
	} else {
		w, err = owm.NewCurrent("C", "EN", apiKey)
		w.CurrentByName(query)
		coords = w.GeoPos
		if err != nil {
			panic(err)
		}
		/*forecast, err = owm.NewForecast("5", "C", "EN", apiKey)
		if err != nil {
			panic(err)
		}
		forecast.DailyByCoordinates(&coords, 5)*/

		temp = fmt.Sprintf("%.1f°C (%.1f°C/%.1f°C)", w.Main.Temp, w.Main.TempMax, w.Main.TempMin)
		feelsLike = fmt.Sprintf("%.1f°C", w.Main.FeelsLike)
		windSpeed = fmt.Sprintf("%.1f m/s (%.1f Km/h)", w.Wind.Speed, w.Wind.Speed * 3.6)
	}

	var weatherDescriptionBuilder strings.Builder
	for _, v := range w.Weather {
		fmt.Fprintf(&weatherDescriptionBuilder, "%s: %s\n", v.Main, v.Description)
	}
	rain = fmt.Sprintf("%.1f mm/h", w.Rain.OneH)
	snow = fmt.Sprintf("%.1f mm/h", w.Snow.OneH)

	// UV Info
	var uv *owm.UV
	var uvString string
	uv, err = owm.NewUV(apiKey)
	if err != nil {
		panic(err)
	}

	err = uv.Current(&coords)
	if err != nil {
		panic(err)
	}

	uvInfo, info_err := uv.UVInformation()
	if info_err != nil {
		panic(info_err)
	}
	uvString = fmt.Sprintf("%.1f (%s)", uvInfo[0].UVIndex[0], uvInfo[0].Risk)

	// Forecast
	// forecast5 := forecast.ForecastWeatherJson.(*owm.Forecast5WeatherData)
	// list := forecast5.List

	return c.Gemini(`# Current Weather - %s, %s
## Current
%s
Temp: %s
Feels Like: %s
Humidity: %d%%
Wind Speed: %s
Max UV: %s

### Precipitation Within an Hour
Rain: %s
Snow: %s

## Forecast
Coming soon...

=> /weather/ Weather Home
=> https://openweathermap.org Powered by openweathermap.org
`, w.Name, w.Sys.Country, weatherDescriptionBuilder.String(), temp, feelsLike, w.Main.Humidity, windSpeed, uvString, rain, snow)
}
