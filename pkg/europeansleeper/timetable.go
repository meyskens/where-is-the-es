package europeansleeper

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	edlib "github.com/hbollon/go-edlib"
	"github.com/meyskens/where-is-the-es/pkg/traindata"
	"golang.org/x/net/html"
)

/*
 <div class="bg-white flex-2 padding-l rounded" id="453">
        <h3 class="text-center">Train ES 453</h3>
        <table class="margin-auto text-center">
            <tr>
                <td>Bruxelles-Midi</td>
                <td><span class="material-symbols-outlined padding-x">train</span></td>
                <td>Prague hl.n. (main station)</td>
            </tr>
            <tr>
                <td>Wed 07 May 2025</td>
                <td></td>
                <td>Thu 08 May 2025</td>
            </tr>
        </table>
        <div class="margin-auto stops">
                                            <div class="flex margin-top">
                    <b>
                                                    19:22                                            </b>
                    <span class="flex-col stop">
                        Bruxelles-Midi                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    20:02                                            </b>
                    <span class="flex-col stop">
                        Antwerpen-Centraal                                                    <i class="text-dark-lavender text-s">Arrival 19:58</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    20:44                                            </b>
                    <span class="flex-col stop">
                        Roosendaal                                                    <i class="text-dark-lavender text-s">Arrival 20:41</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    21:22                                            </b>
                    <span class="flex-col stop">
                        Rotterdam Centraal                                                    <i class="text-dark-lavender text-s">Arrival 21:19</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    21:42                                            </b>
                    <span class="flex-col stop">
                        Den Haag HS                                                    <i class="text-dark-lavender text-s">Arrival 21:40</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    22:34                                            </b>
                    <span class="flex-col stop">
                        Amsterdam Centraal                                                    <i class="text-dark-lavender text-s">Arrival 22:28</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    23:13                                            </b>
                    <span class="flex-col stop">
                        Amersfoort Centraal                                                    <i class="text-dark-lavender text-s">Arrival 23:08</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    23:52                                            </b>
                    <span class="flex-col stop">
                        Deventer                                                    <i class="text-dark-lavender text-s">Arrival 23:48</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    06:20                                            </b>
                    <span class="flex-col stop">
                        Berlin Hauptbahnhof                                                    <i class="text-dark-lavender text-s">Arrival 06:16</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    06:29                                            </b>
                    <span class="flex-col stop">
                        Berlin Ostbahnhof                                                    <i class="text-dark-lavender text-s">Arrival 06:27</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    08:54                                            </b>
                    <span class="flex-col stop">
                        Dresden Hbf                                                    <i class="text-dark-lavender text-s">Arrival 08:50</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    09:23                                            </b>
                    <span class="flex-col stop">
                        Bad Schandau                                                    <i class="text-dark-lavender text-s">Arrival 09:21</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    09:46                                            </b>
                    <span class="flex-col stop">
                        Decin hl.n.                                                    <i class="text-dark-lavender text-s">Arrival 09:41</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    11:24                                            </b>
                    <span class="flex-col stop stop-last">
                        Prague hl.n. (main station)                                            </span>
                </div>
                    </div>
    </div>
        <div class="bg-white flex-2 padding-l rounded" id="452">
        <h3 class="text-center">Train ES 452</h3>
        <table class="margin-auto text-center">
            <tr>
                <td>Prague hl.n. (main station)</td>
                <td><span class="material-symbols-outlined padding-x">train</span></td>
                <td>Bruxelles-Midi</td>
            </tr>
            <tr>
                <td>Thu 08 May 2025</td>
                <td></td>
                <td>Fri 09 May 2025</td>
            </tr>
        </table>
        <div class="margin-auto stops">
                                            <div class="flex margin-top">
                    <b>
                                                    18:02                                            </b>
                    <span class="flex-col stop">
                        Prague hl.n. (main station)                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    19:38                                            </b>
                    <span class="flex-col stop">
                        Decin hl.n.                                                    <i class="text-dark-lavender text-s">Arrival 19:36</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    19:55                                            </b>
                    <span class="flex-col stop">
                        Bad Schandau                                                    <i class="text-dark-lavender text-s">Arrival 19:53</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    20:30                                            </b>
                    <span class="flex-col stop">
                        Dresden Hbf                                                    <i class="text-dark-lavender text-s">Arrival 20:24</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    23:59                                            </b>
                    <span class="flex-col stop">
                        Berlin Ostbahnhof                                                    <i class="text-dark-lavender text-s">Arrival 23:57</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    00:11                                            </b>
                    <span class="flex-col stop">
                        Berlin Hauptbahnhof                                                    <i class="text-dark-lavender text-s">Arrival 00:08</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    05:12                                            </b>
                    <span class="flex-col stop">
                        Deventer                                                    <i class="text-dark-lavender text-s">Arrival 05:09</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    05:48                                            </b>
                    <span class="flex-col stop">
                        Amersfoort Centraal                                                    <i class="text-dark-lavender text-s">Arrival 05:46</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    06:31                                            </b>
                    <span class="flex-col stop">
                        Amsterdam Centraal                                                    <i class="text-dark-lavender text-s">Arrival 06:26</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    07:12                                            </b>
                    <span class="flex-col stop">
                        Den Haag HS                                                    <i class="text-dark-lavender text-s">Arrival 07:10</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    07:37                                            </b>
                    <span class="flex-col stop">
                        Rotterdam Centraal                                                    <i class="text-dark-lavender text-s">Arrival 07:27</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    08:16                                            </b>
                    <span class="flex-col stop">
                        Roosendaal                                                    <i class="text-dark-lavender text-s">Arrival 08:11</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    08:47                                            </b>
                    <span class="flex-col stop">
                        Antwerpen-Centraal                                                    <i class="text-dark-lavender text-s">Arrival 08:43</i>
                                            </span>
                </div>
                                            <div class="flex margin-top">
                    <b>
                                                    09:27                                            </b>
                    <span class="flex-col stop stop-last">
                        Bruxelles-Midi                                            </span>
                </div>
                    </div>
    </div>
*/

func FetchTimetable(trainNumber string, date time.Time, tcURL string) (*traindata.Trip, error) {
	resp, err := http.PostForm("https://europeansleeper.eu/timetable/run", url.Values{
		"departure-date-sql": {date.Format("2006-01-02")},
	})

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	tt, err := parseTimetable(trainNumber, resp.Body)
	if err != nil {
		return nil, err
	}

	// Load Amsterdam timezone
	amsterdamTz, err := time.LoadLocation("Europe/Amsterdam")
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(tt.Stops); i++ {
		// set the dates for arrival and departure times to the given date
		// if next day, set the date to the next day
		var targetDate time.Time
		if tt.Stops[i].NextDay {
			targetDate = date.AddDate(0, 0, 1) // next day
		} else {
			targetDate = date
		}

		// Combine the date with the parsed time (hour and minute) in Amsterdam timezone
		if !tt.Stops[i].ArrivalTime.IsZero() {
			tt.Stops[i].ArrivalTime = time.Date(
				targetDate.Year(), targetDate.Month(), targetDate.Day(),
				tt.Stops[i].ArrivalTime.Hour(), tt.Stops[i].ArrivalTime.Minute(), 0, 0,
				amsterdamTz,
			)
		}

		tt.Stops[i].DepartureTime = time.Date(
			targetDate.Year(), targetDate.Month(), targetDate.Day(),
			tt.Stops[i].DepartureTime.Hour(), tt.Stops[i].DepartureTime.Minute(), 0, 0,
			amsterdamTz,
		)
	}

	// the last station the departure time is the arrival time
	if len(tt.Stops) > 0 {
		tt.Stops[len(tt.Stops)-1].ArrivalTime = tt.Stops[len(tt.Stops)-1].DepartureTime
		tt.Stops[len(tt.Stops)-1].DepartureTime = time.Time{}
	}

	tt.Date = date.Truncate(24 * time.Hour) // Set the date of the trip to the given date

	if tcURL != "" {
		enhanceTimetableData(tcURL, tt, trainNumber)
	}

	return tt, nil
}

func enhanceTimetableData(tcURL string, trip *traindata.Trip, trainNum string) {
	resp, err := http.Get(tcURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	tc := TCResponse{}
	if err := json.Unmarshal(data, &tc); err != nil {
		return
	}

	svcIdx := -1
	for i := range tc.Services {
		if tc.Services[i].TrainNumber == "ES"+trainNum &&
			tc.Services[i].DepartureDate == trip.Date.Format("2006-01-02") {
			svcIdx = i
			break
		}
	}
	if svcIdx == -1 {
		return
	}

	type tcEntry struct {
		normalized string
		uic        int
	}
	tcEntries := make([]tcEntry, 0, len(tc.Services[svcIdx].Timetable.Stops))
	uicByName := map[string]int{}
	for _, tcStop := range tc.Services[svcIdx].Timetable.Stops {
		if tcStop.StationUIC == 0 {
			continue
		}
		norm := normalizeStationName(tcStop.Station)
		uicByName[norm] = tcStop.StationUIC
		tcEntries = append(tcEntries, tcEntry{normalized: norm, uic: tcStop.StationUIC})
	}

	for j := range trip.Stops {
		key := normalizeStationName(trip.Stops[j].StationName)
		if uic, ok := uicByName[key]; ok {
			trip.Stops[j].StationUIC = uic
			continue
		}
		// Fuzzy match: find the TC stop with the highest similarity above
		// a threshold. This handles cases where the TC and HTML sources
		// spell stations differently (extra/missing words, accents, etc.).
		bestUIC := 0
		bestScore := 0.0
		for _, e := range tcEntries {
			score := stringSimilarity(key, e.normalized)
			if score > bestScore {
				bestScore = score
				bestUIC = e.uic
			}
		}
		if bestScore >= 0.7 {
			trip.Stops[j].StationUIC = bestUIC
		}
	}
}

// stringSimilarity returns a value in [0, 1] expressing how similar two
// normalized station names are. It combines a substring containment check
// (so a name fully contained in another scores highly even if much shorter)
// with the Jaro-Winkler similarity from go-edlib.
func stringSimilarity(a, b string) float64 {
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1
	}
	// Substring containment is a strong signal: e.g. "praguehlavninadrazi"
	// vs "praguehlavninadrazimainstation".
	if strings.Contains(a, b) || strings.Contains(b, a) {
		shorter, longer := len(a), len(b)
		if shorter > longer {
			shorter, longer = longer, shorter
		}
		return 0.8 + 0.2*float64(shorter)/float64(longer)
	}
	return float64(edlib.JaroWinklerSimilarity(a, b))
}

var stationNameSubstitutions = []struct {
	pattern *regexp.Regexp
	to      string
}{
	// Typo in the TC response: "Ostbahnnhof" -> "Ostbahnhof".
	{regexp.MustCompile(`(?i)ostbahnnhof`), "ostbahnhof"},
	// Czech spellings used interchangeably with English in different sources.
	{regexp.MustCompile(`(?i)\bpraha\b`), "prague"},
	// Abbreviations -> full form. Use word boundaries so "cs" doesn't match
	// inside other words.
	{regexp.MustCompile(`(?i)\bhbf\b`), "hauptbahnhof"},
	{regexp.MustCompile(`(?i)\bcs\b`), "centraal"},
	{regexp.MustCompile(`(?i)\bhs\b`), "holland spoor"},
	{regexp.MustCompile(`(?i)\bhl\.?\s*n\.?`), "hlavni nadrazi"},
	{regexp.MustCompile(`(?i)\bhln\b`), "hlavni nadrazi"},
	// "(main station)" annotation found in the timetable HTML.
	{regexp.MustCompile(`(?i)\bmain\s*station\b`), "hlavni nadrazi"},
}

// normalizeStationName produces a comparable form of a station name so that
// stations spelled differently in the timetable HTML and the TC API still
// match. It lowercases, expands known abbreviations to a canonical full form,
// then strips to alphanumerics.
func normalizeStationName(s string) string {
	s = strings.ToLower(s)
	for _, sub := range stationNameSubstitutions {
		s = sub.pattern.ReplaceAllString(s, sub.to)
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func parseTimetable(trainNumber string, body io.Reader) (*traindata.Trip, error) {
	stops := []traindata.Stop{}

	z := html.NewTokenizer(body)

	// look for <h3>Train ES $TRAIN_NUM</h3>

	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			return nil, errors.New("failed to parse train timetable")
		}

		if tt == html.StartTagToken {
			t := z.Token()
			if t.Data == "h3" {
				z.Next()
				if strings.Contains(string(z.Text()), "Train ES "+trainNumber) {
					break
				}
			}
		}
	}

	// look for <span class="flex-col stop">

	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt == html.StartTagToken {
			t := z.Token()
			if t.Data == "b" {
				stop, err := parseStop(z)
				if err != nil {
					return nil, err
				}
				if stop != nil {
					stops = append(stops, *stop)
				}
			}
			if t.Data == "h3" {
				break
			}
		}
	}

	return &traindata.Trip{
		TrainNumber: trainNumber,
		Stops:       stops,
	}, nil
}

func parseStop(z *html.Tokenizer) (*traindata.Stop, error) {
	tt := z.Next()
	if tt == html.ErrorToken {
		return nil, errors.New("failed to parse stop")
	}
	departureTime := string(z.Text())

	for {
		tt = z.Next()
		if tt == html.ErrorToken {
			return nil, errors.New("failed to parse stop")
		}
		if tt == html.StartTagToken {
			break
		}
	}
	t := z.Token()
	if t.Data != "span" {
		return nil, errors.New("expected span tag")
	}
	tt = z.Next()
	if tt == html.ErrorToken {
		return nil, errors.New("failed to parse stop")
	}
	stationName := string(z.Text())
	tt = z.Next()

	if tt == html.ErrorToken {
		return nil, errors.New("failed to parse stop")
	}

	arrivalTime := ""

	// parse optional arrival time
	if tt == html.StartTagToken {
		t := z.Token()
		if t.Data == "i" {
			tt = z.Next()
			if tt == html.ErrorToken {
				return nil, errors.New("failed to parse stop")
			}
			if tt != html.TextToken {
				return nil, errors.New("expected text token")
			}
			arrivalTime = string(z.Text())
		}
	}

	stationName = strings.ReplaceAll(stationName, "\n", "")
	stationName = strings.TrimSpace(stationName)

	departureTime = strings.ReplaceAll(departureTime, "\n", "")
	departureTime = strings.TrimSpace(departureTime)

	arrivalTime = strings.ReplaceAll(arrivalTime, "\n", "")
	arrivalTime = strings.ReplaceAll(arrivalTime, "Arrival ", "")
	arrivalTime = strings.TrimSpace(arrivalTime)

	// parse time
	departureTimeParsed, err := time.Parse("15:04", departureTime)
	if err != nil {
		return nil, err
	}

	var arrivalTimeParsed time.Time
	if arrivalTime != "" {
		arrivalTimeParsed, err = time.Parse("15:04", arrivalTime)
		if err != nil {
			return nil, err
		}
	}
	// check if the stop is next day
	nextDay := false
	// let's assume if the train arrives before 15:00, it is next day
	if departureTimeParsed.Hour() < 15 {
		nextDay = true
	}

	return &traindata.Stop{
		StationName:   stationName,
		ArrivalTime:   arrivalTimeParsed,
		DepartureTime: departureTimeParsed,
		NextDay:       nextDay,
	}, nil
}
