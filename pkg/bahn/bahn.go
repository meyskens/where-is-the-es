// Package bahn provides a client for the Deutsche Bahn RIS-Journeys API.
//
// It exposes helpers to look up a journey by train number and date, fetch
// the per-stop arrival/departure events for that journey and convert them
// into the project's traindata.Trip representation.
//
// The DB API requires two credentials passed as request headers:
//   - DB-Api-Key
//   - DB-Client-Id
//
// See: https://developers.deutschebahn.com/db-api-marketplace/apis/product/ris-journeys
package bahn

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/meyskens/where-is-the-es/pkg/traindata"
	"golang.org/x/time/rate"
)

const (
	defaultBaseURL = "https://apis.deutschebahn.com/db/apis/ris-journeys/v2"

	// defaultTransportTypes mirrors the example TS code and covers the
	// long-distance and regional categories DB exposes.
	defaultTransportTypes = "HIGH_SPEED_TRAIN,INTERCITY_TRAIN,INTER_REGIONAL_TRAIN,REGIONAL_TRAIN,CITY_TRAIN"
)

// EventType represents the kind of timetable event reported by the DB API.
type EventType string

const (
	EventArrival   EventType = "ARRIVAL"
	EventDeparture EventType = "DEPARTURE"
)

// Client is a Deutsche Bahn RIS-Journeys API client.
type Client struct {
	apiKey     string
	clientID   string
	baseURL    string
	httpClient *http.Client
	limiter    *rate.Limiter
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets a custom *http.Client (e.g. with a timeout or
// instrumented transport).
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.httpClient = h }
}

// WithBaseURL overrides the API base URL. Mostly useful for testing.
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") }
}

// NewClient creates a new DB API client. apiKey and clientID are required.
// Requests are rate limited to 1 per second by default to stay within the
// DB API marketplace quota.
func NewClient(apiKey, clientID string, opts ...Option) *Client {
	c := &Client{
		apiKey:     apiKey,
		clientID:   clientID,
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		limiter:    rate.NewLimiter(rate.Every(time.Second), 1),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// JourneyMatch is a journey returned by the journey search endpoint.
type JourneyMatch struct {
	JourneyID    string
	OperatorCode string
}

// Event is a single arrival or departure event at a stop on a journey.
type Event struct {
	StopPlace struct {
		EvaNumber string `json:"evaNumber"`
		Name      string `json:"name"`
	} `json:"stopPlace"`
	Platform          string    `json:"platform"`
	PlatformSchedule  string    `json:"platformSchedule"`
	Time              string    `json:"time"`
	TimeSchedule      string    `json:"timeSchedule"`
	Type              EventType `json:"type"`
	NoPassengerChange bool      `json:"noPassengerChange,omitempty"`
	Cancelled         bool      `json:"cancelled,omitempty"`
}

// Stop is a journey stop merged from arrival and departure events.
//
// Arrival/Departure are the scheduled (planned) times. ArrivalRealtime and
// DepartureRealtime are the realtime/predicted times reported by the DB API
// (the `time` field of the event), which may differ from the schedule when
// there is a delay.
type Stop struct {
	Name              string
	EvaNumber         string
	Arrival           time.Time
	Departure         time.Time
	Platform          string
	PlatformRealtime  string
	ArrivalRealtime   time.Time
	DepartureRealtime time.Time
	Cancelled         bool
}

// FindJourneys looks up journeys matching the given train number on the given date.
func (c *Client) FindJourneys(ctx context.Context, trainNumber string, date time.Time) ([]JourneyMatch, error) {
	num := stripPrefix(trainNumber)
	if num == "" {
		return nil, fmt.Errorf("db: empty train number")
	}

	q := url.Values{}
	q.Set("journeyNumber", num)
	//q.Set("transportTypes", defaultTransportTypes)
	q.Set("date", date.Format("2006-01-02"))

	endpoint := c.baseURL + "/find?" + q.Encode() + "&transportTypes=" + defaultTransportTypes
	fmt.Println("DB API FindJourneys endpoint:", endpoint)

	var raw struct {
		Journeys []struct {
			JourneyID string `json:"journeyID"`
			Info      struct {
				HeaderAdministration struct {
					OperatorCode string `json:"operatorCode"`
				} `json:"headerAdministration"`
			} `json:"info"`
		} `json:"journeys"`
	}

	if err := c.do(ctx, endpoint, &raw); err != nil {
		return nil, fmt.Errorf("db: find journeys: %w", err)
	}

	out := make([]JourneyMatch, 0, len(raw.Journeys))
	for _, j := range raw.Journeys {
		if j.Info.HeaderAdministration.OperatorCode != "ES" {
			continue
		}
		out = append(out, JourneyMatch{
			JourneyID:    j.JourneyID,
			OperatorCode: j.Info.HeaderAdministration.OperatorCode,
		})
	}
	return out, nil
}

// GetJourneyEvents fetches the arrival/departure events for the given journey.
func (c *Client) GetJourneyEvents(ctx context.Context, journeyID string) ([]Event, error) {
	if journeyID == "" {
		return nil, fmt.Errorf("db: empty journey ID")
	}

	endpoint := c.baseURL + "/" + url.PathEscape(journeyID)

	var raw struct {
		Events []Event `json:"events"`
	}
	if err := c.do(ctx, endpoint, &raw); err != nil {
		return nil, fmt.Errorf("db: get journey events: %w", err)
	}
	return raw.Events, nil
}

// GetStops returns the merged stop list (arrival + departure per station)
func (c *Client) GetStops(ctx context.Context, trainNumber string, date time.Time) ([]Stop, error) {
	journeys, err := c.FindJourneys(ctx, trainNumber, date)
	if err != nil {
		return nil, fmt.Errorf("error in FindJourneys: %w", err)
	}

	if len(journeys) == 0 {
		return nil, fmt.Errorf("db: no matching journey for train %s on %s", trainNumber, date.Format("2006-01-02"))
	}
	match := &journeys[0]

	events, err := c.GetJourneyEvents(ctx, match.JourneyID)
	if err != nil {
		return nil, fmt.Errorf("error in GetJourneyEvents: %w", err)
	}
	return GroupEvents(events), nil
}

// GetTimetable returns a traindata.Trip for the given train number and date.
func (c *Client) GetTimetable(ctx context.Context, trainNumber string, date time.Time) (*traindata.Trip, error) {
	stops, err := c.GetStops(ctx, trainNumber, date)
	if err != nil {
		return nil, err
	}

	trip := &traindata.Trip{
		TrainNumber: trainNumber,
		Date:        date.Truncate(24 * time.Hour),
		Stops:       make([]traindata.Stop, 0, len(stops)),
	}

	for _, s := range stops {
		uic, _ := strconv.Atoi(s.EvaNumber)
		trip.Stops = append(trip.Stops, traindata.Stop{
			StationName:         s.Name,
			StationUIC:          uic,
			ArrivalTime:         s.Arrival,
			DepartureTime:       s.Departure,
			Platform:            s.Platform,
			RealPlatform:        s.PlatformRealtime,
			DataSources:         []traindata.DataSource{traindata.DataSourceDB},
			PrefferedDataSource: traindata.DataSourceDB,
			Cancelled:           s.Cancelled,
		})
	}

	return trip, nil
}

// GroupEvents merges the flat event list returned by the DB API into one
// stop entry per station, preserving the order of first appearance.
func GroupEvents(events []Event) []Stop {
	type slot struct {
		idx  int
		stop Stop
	}
	byEva := make(map[string]*slot, len(events))
	order := make([]string, 0, len(events))

	for _, ev := range events {
		eva := ev.StopPlace.EvaNumber
		s, ok := byEva[eva]
		if !ok {
			s = &slot{idx: len(order), stop: Stop{
				Name:      ev.StopPlace.Name,
				EvaNumber: eva,
			}}
			byEva[eva] = s
			order = append(order, eva)
		}
		scheduled := parseDBTime(ev.TimeSchedule)
		realtime := parseDBTime(ev.Time)
		switch ev.Type {
		case EventArrival:
			s.stop.Arrival = scheduled
			s.stop.ArrivalRealtime = realtime
			if s.stop.Platform == "" && ev.PlatformSchedule != "" {
				s.stop.Platform = ev.PlatformSchedule
			}
			if s.stop.PlatformRealtime == "" && ev.Platform != "" {
				s.stop.PlatformRealtime = ev.Platform
			}
		case EventDeparture:
			s.stop.Departure = scheduled
			s.stop.DepartureRealtime = realtime
			if ev.PlatformSchedule != "" {
				s.stop.Platform = ev.PlatformSchedule
			} else if s.stop.Platform == "" && ev.Platform != "" {
				s.stop.Platform = ev.Platform
			}
			if ev.Platform != "" {
				s.stop.PlatformRealtime = ev.Platform
			}
		}
		if ev.Cancelled {
			s.stop.Cancelled = true
		}
	}

	out := make([]Stop, len(order))
	for i, eva := range order {
		out[i] = byEva[eva].stop
	}
	return out
}

// HTTPError is returned when the DB API responds with a non-2xx status.
type HTTPError struct {
	StatusCode int
	Headers    http.Header
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("unexpected status %d: headers=%v body=%q", e.StatusCode, e.Headers, e.Body)
}

func (c *Client) do(ctx context.Context, endpoint string, out any) error {
	if c.limiter != nil {
		if err := c.limiter.Wait(ctx); err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("DB-Api-Key", c.apiKey)
	req.Header.Set("DB-Client-Id", c.clientID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Headers:    resp.Header,
			Body:       strings.TrimSpace(string(body)),
		}
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// stripPrefix removes any leading non-digit characters from a train number,
// e.g. "ES302" -> "302".
func stripPrefix(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			return s[i:]
		}
	}
	return ""
}

// parseDBTime parses an ISO-8601 timestamp returned by the DB API. It returns
// the zero time on parse failure or empty input.
func parseDBTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
