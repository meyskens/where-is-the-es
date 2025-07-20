package europeansleeper

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

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

func FetchTimetable(trainNumber string, date time.Time) (*traindata.Trip, error) {
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

	for i := 0; i < len(tt.Stops); i++ {
		// set the dates for arrival and departure times to the given date
		// if next day, set the date to the next day
		var targetDate time.Time
		if tt.Stops[i].NextDay {
			targetDate = date.AddDate(0, 0, 1) // next day
		} else {
			targetDate = date
		}

		// Combine the date with the parsed time (hour and minute)
		if !tt.Stops[i].ArrivalTime.IsZero() {
			tt.Stops[i].ArrivalTime = time.Date(
				targetDate.Year(), targetDate.Month(), targetDate.Day(),
				tt.Stops[i].ArrivalTime.Hour(), tt.Stops[i].ArrivalTime.Minute(), 0, 0,
				targetDate.Location(),
			)
		}

		tt.Stops[i].DepartureTime = time.Date(
			targetDate.Year(), targetDate.Month(), targetDate.Day(),
			tt.Stops[i].DepartureTime.Hour(), tt.Stops[i].DepartureTime.Minute(), 0, 0,
			targetDate.Location(),
		)
	}

	return tt, nil
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
