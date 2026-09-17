package arenaways

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/astrocode-id/go-flaresolverr"
	"github.com/meyskens/where-is-the-es/pkg/traindata"
	"golang.org/x/net/html"
)

/*
Arenaways (https://www.arenaways.it) runs the ES in Italy.
Their train-status page is behind a Cloudflare wall, so we use FlareSolverr to fetch it.

The page renders a timeline of stops.
We parse the HTML and return stops, then the europeansleeper enhancement layer
matches them against the trip by station name.

Only Italian stations (UIC country prefix 83) are enriched.
*/

var (
	timeRegexp = regexp.MustCompile(`\b([0-2]?\d:[0-5]\d)\b`)
)

// ErrNoTrain is returned when Arenaways has no record of the requested train.
var ErrNoTrain = errors.New("no train found for given date")

// ArenaWaysFetcher scrapes the Arenaways train page via FlareSolverr.
type ArenaWaysFetcher struct {
	flareSolverr    *flaresolverr.Client
	flareSolverrURL string
}

// NewArenaWaysFetcher creates a new ArenaWaysFetcher using the given
// FlareSolverr base URL.
func NewArenaWaysFetcher(flareSolverrURL string) (*ArenaWaysFetcher, error) {
	c, err := flaresolverr.NewClient(flaresolverr.Config{
		BaseURL: flareSolverrURL,
	})
	if err != nil {
		return nil, err
	}
	return &ArenaWaysFetcher{
		flareSolverr:    c,
		flareSolverrURL: flareSolverrURL,
	}, nil
}

// FetchTimetable fetches the live timetable for the given train number
// from arenaways.it via FlareSolverr and parses it into stops.
func (f *ArenaWaysFetcher) FetchTimetable(trainNumber string) ([]traindata.Stop, error) {
	log.Println("Arenaways: fetching timetable for train", trainNumber)

	resp, err := f.flareSolverr.GetRaw(flaresolverr.GetParams{
		URL:        "https://www.arenaways.it/en/train/" + trainNumber,
		MaxTimeout: 60000,
	})
	if err != nil {
		log.Println("Arenaways: flaresolverr GET transport failed for train", trainNumber, ":", err)
		return nil, fmt.Errorf("flaresolverr GET: %w", err)
	}

	log.Printf("Arenaways: flaresolverr status=%q solution-status=%d message=%q version=%s for train %s",
		resp.Status, resp.Solution.Status, resp.Message, resp.Version, trainNumber)

	if string(resp.Status) != "ok" {
		return nil, fmt.Errorf("flaresolverr GET non-ok status=%q message=%q", resp.Status, resp.Message)
	}

	body, err := unwrapSolutionResponse(resp.Solution.Response)
	if err != nil {
		return nil, fmt.Errorf("unwrapping solution response: %w", err)
	}

	log.Println("Arenaways: received", len(body), "bytes for train", trainNumber)

	stops, err := f.ParseTimetable(body)
	if err != nil {
		preview := body
		if len(preview) > 500 {
			preview = preview[:500]
		}
		log.Println("Arenaways: parse failed for train", trainNumber, ":", err, "- response preview:", string(preview))
		return nil, fmt.Errorf("parse timetable: %w", err)
	}

	log.Println("Arenaways: parsed", len(stops), "stops for train", trainNumber)
	return stops, nil
}

// ParseTimetable parses the Arenaways train page HTML into a list of stops.
func (f *ArenaWaysFetcher) ParseTimetable(body []byte) ([]traindata.Stop, error) {
	z := html.NewTokenizer(bytes.NewReader(body))

	var stops []traindata.Stop

	// Walk the DOM looking for <div class="relative flex items-start"> —
	// each one is a station entry in the timeline.
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}

		if tt == html.StartTagToken {
			t := z.Token()
			if t.Data == "div" {
				isStopDiv := false
				for _, a := range t.Attr {
					if a.Key == "class" {
						cls := " " + a.Val + " "
						if strings.Contains(cls, " relative ") &&
							strings.Contains(cls, " flex ") &&
							strings.Contains(cls, " items-start ") {
							isStopDiv = true
							break
						}
					}
				}
				if isStopDiv {
					stop, err := f.parseStopDepth(z)
					if err != nil {
						return nil, err
					}
					if stop.StationName != "" {
						stops = append(stops, stop)
					}
				}
			}
		}
	}

	if len(stops) == 0 {
		return nil, ErrNoTrain
	}

	// The first stop has no arrival time, the last has no departure time.
	stops[0].ArrivalTime = time.Time{}
	stops[0].RealArrivalTime = time.Time{}
	stops[len(stops)-1].DepartureTime = time.Time{}
	stops[len(stops)-1].RealDepartureTime = time.Time{}

	return stops, nil
}

// parseStopDepth parses a single <div class="relative flex items-start">
// element by tracking div nesting depth. The Arenaways HTML lays out each
// stop as a sequence of <div class="mb-2 ..."> rows, each containing a
// scheduled-time <p class="text-sm">Arrivo:/Partenza:</p> followed by an
// optional real-time <p class="text-sm text-gray-500">Effettivo: ...</p>.
// The Effettivo row applies to the most recently seen Arrivo/Partenza.
func (f *ArenaWaysFetcher) parseStopDepth(z *html.Tokenizer) (traindata.Stop, error) {
	var stop traindata.Stop
	depth := 1 // we are already inside the stop div

	// lastField tracks which scheduled field the next "Effettivo" row
	// applies to: "arrival" or "departure".
	lastField := ""

	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}

		if tt == html.StartTagToken {
			t := z.Token()
			if t.Data == "div" {
				depth++
				continue
			}

			if t.Data == "p" {
				cls := ""
				for _, a := range t.Attr {
					if a.Key == "class" {
						cls = " " + a.Val + " "
						break
					}
				}

				// Station name: <p class="pb-6 font-semibold">NAME</p>
				if strings.Contains(cls, " pb-6 ") && strings.Contains(cls, " font-semibold ") {
					name, _ := readTextContent(z)
					stop.StationName = strings.TrimSpace(name)
					continue
				}

				// Real time: <p class="text-sm text-gray-500">Effettivo: ...</p>
				// The time value may be wrapped in a <span> child and may
				// be "-" when no realtime data is available yet.
				if strings.Contains(cls, " text-gray-500 ") {
					text, _ := readTextContent(z)
					text = strings.TrimSpace(strings.ReplaceAll(text, "\n", ""))
					if strings.HasPrefix(text, "Effettivo:") {
						val := strings.TrimSpace(strings.TrimPrefix(text, "Effettivo:"))
						if val != "" && val != "-" {
							t := parseTimeString(val)
							if !t.IsZero() {
								switch lastField {
								case "arrival":
									stop.RealArrivalTime = t
									stop.IsRealTime = true
								case "departure":
									stop.RealDepartureTime = t
									stop.IsRealTime = true
								}
							}
						}
					}
					continue
				}

				// Scheduled-time rows: <p class="text-sm">Arrivo: 16:55</p>
				if strings.Contains(cls, " text-sm ") {
					text, _ := readTextContent(z)
					text = strings.TrimSpace(strings.ReplaceAll(text, "\n", ""))
					text = strings.TrimSpace(text)

					if strings.HasPrefix(text, "Arrivo:") {
						stop.ArrivalTime = parseTimeString(stripLabel(text))
						stop.RealArrivalTime = stop.ArrivalTime
						lastField = "arrival"
					} else if strings.HasPrefix(text, "Partenza:") {
						stop.DepartureTime = parseTimeString(stripLabel(text))
						stop.RealDepartureTime = stop.DepartureTime
						lastField = "departure"
					}
					continue
				}
			}
		}

		if tt == html.EndTagToken && z.Token().Data == "div" {
			depth--
			if depth == 0 {
				break
			}
		}
	}

	return stop, nil
}

// stripLabel removes the leading label (e.g. "Arrivo:", "Partenza:")
// from a time string and returns the time portion.
func stripLabel(s string) string {
	idx := strings.Index(s, ":")
	if idx < 0 {
		return s
	}
	return strings.TrimSpace(s[idx+1:])
}

// parseTimeString extracts a HH:MM time from a string and returns it
// as a time.Time with only hour/minute set (date is zero).
func parseTimeString(s string) time.Time {
	match := timeRegexp.FindString(s)
	if match == "" {
		return time.Time{}
	}
	t, err := time.Parse("15:04", match)
	if err != nil {
		return time.Time{}
	}
	return t
}

// readTextContent reads all text content until the closing tag of the
// current element, following nested tags (e.g. <p>Effettivo: <span>16:56</span></p>).
func readTextContent(z *html.Tokenizer) (string, error) {
	var sb strings.Builder
	depth := 1
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt == html.TextToken {
			sb.Write(z.Text())
		}
		if tt == html.StartTagToken {
			depth++
		}
		if tt == html.EndTagToken {
			depth--
			if depth == 0 {
				break
			}
		}
	}
	return sb.String(), nil
}

// unwrapSolutionResponse converts the flaresolverr Solution.Response
// (json.RawMessage) into the actual page body bytes.
func unwrapSolutionResponse(raw []byte) ([]byte, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty solution response")
	}
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, err
		}
		return []byte(s), nil
	}
	return raw, nil
}
