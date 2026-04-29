// Package ns provides a client for the NS (Nederlandse Spoorwegen)
// Reisinformatie API.
//
// It exposes a helper to fetch the journey for a given train number and
// convert it into the project's traindata.Trip representation.
//
// The NS API requires a subscription key passed as a request header:
//   - Ocp-Apim-Subscription-Key
//
// See: https://apiportal.ns.nl/
package ns

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
	defaultBaseURL = "https://gateway.apiportal.ns.nl/reisinformatie-api/api/v2"
)

// Client is an NS Reisinformatie API client.
type Client struct {
	subscriptionKey string
	baseURL         string
	httpClient      *http.Client
	limiter         *rate.Limiter
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

// NewClient creates a new NS API client. subscriptionKey is required.
// Requests are rate limited to 1 per second by default to stay within the
// NS API quota.
func NewClient(subscriptionKey string, opts ...Option) *Client {
	c := &Client{
		subscriptionKey: subscriptionKey,
		baseURL:         defaultBaseURL,
		httpClient:      &http.Client{Timeout: 30 * time.Second},
		limiter:         rate.NewLimiter(rate.Every(time.Second), 1),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// journeyResponse is the top-level response from the NS journey endpoint.
type journeyResponse struct {
	Payload struct {
		Stops []stop `json:"stops"`
	} `json:"payload"`
}

// stop represents a single stop on a journey.
type stop struct {
	ID       string `json:"id"`
	StopInfo struct {
		Name        string `json:"name"`
		UICCode     string `json:"uicCode"`
		CountryCode string `json:"countryCode"`
	} `json:"stop"`
	Status     string    `json:"status"`
	Arrivals   []arrival `json:"arrivals"`
	Departures []departure
}

// arrival represents an arrival event at a stop.
type arrival struct {
	PlannedTime   string `json:"plannedTime"`
	ActualTime    string `json:"actualTime"`
	PlannedTrack  string `json:"plannedTrack"`
	ActualTrack   string `json:"actualTrack"`
	Cancelled     bool   `json:"cancelled"`
	CrowdForecast string `json:"crowdForecast"`
}

// departure represents a departure event at a stop.
type departure struct {
	PlannedTime   string `json:"plannedTime"`
	ActualTime    string `json:"actualTime"`
	PlannedTrack  string `json:"plannedTrack"`
	ActualTrack   string `json:"actualTrack"`
	Cancelled     bool   `json:"cancelled"`
	CrowdForecast string `json:"crowdForecast"`
}

// GetTimetable returns a traindata.Trip for the given train number and date.
// The date parameter is optional; pass a zero time to omit it from the query.
func (c *Client) GetTimetable(ctx context.Context, trainNumber string, date time.Time) (*traindata.Trip, error) {
	num := stripPrefix(trainNumber)
	if num == "" {
		return nil, fmt.Errorf("ns: empty train number")
	}

	q := url.Values{}
	q.Set("train", num)
	q.Set("omitCrowdForecast", "false")
	if !date.IsZero() {
		q.Set("dateTime", date.Format(time.RFC3339))
	}

	endpoint := c.baseURL + "/journey?" + q.Encode()

	var raw journeyResponse
	if err := c.do(ctx, endpoint, &raw); err != nil {
		return nil, fmt.Errorf("ns: get journey: %w", err)
	}

	if len(raw.Payload.Stops) == 0 {
		return nil, fmt.Errorf("ns: no stops found for train %s", trainNumber)
	}

	trip := &traindata.Trip{
		TrainNumber: trainNumber,
		Stops:       make([]traindata.Stop, 0, len(raw.Payload.Stops)),
	}

	for _, s := range raw.Payload.Stops {
		uic, _ := strconv.Atoi(s.StopInfo.UICCode)

		stop := traindata.Stop{
			StationName:         s.StopInfo.Name,
			StationUIC:          uic,
			DataSources:         []traindata.DataSource{traindata.DataSourceNS},
			PrefferedDataSource: traindata.DataSourceNS,
		}

		// Arrivals
		if len(s.Arrivals) > 0 {
			arr := s.Arrivals[0]
			stop.ArrivalTime = parseNSTime(arr.PlannedTime)
			stop.RealArrivalTime = parseNSTime(arr.ActualTime)
			stop.Platform = arr.PlannedTrack
			stop.RealPlatform = arr.ActualTrack
			stop.Cancelled = arr.Cancelled
			if arr.ActualTime != "" {
				stop.IsRealTime = true
			}
		}

		// Departures
		if len(s.Departures) > 0 {
			dep := s.Departures[0]
			stop.DepartureTime = parseNSTime(dep.PlannedTime)
			stop.RealDepartureTime = parseNSTime(dep.ActualTime)
			if stop.Platform == "" {
				stop.Platform = dep.PlannedTrack
			}
			if stop.RealPlatform == "" {
				stop.RealPlatform = dep.ActualTrack
			}
			if dep.Cancelled {
				stop.Cancelled = true
			}
			if dep.ActualTime != "" {
				stop.IsRealTime = true
			}
		}

		trip.Stops = append(trip.Stops, stop)
	}

	return trip, nil
}

// HTTPError is returned when the NS API responds with a non-2xx status.
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
	req.Header.Set("Ocp-Apim-Subscription-Key", c.subscriptionKey)

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

// nsTimeLayout is the timestamp layout used by the NS API (RFC3339 without
// the colon in the timezone offset, e.g. "2026-04-29T06:06:00+0200").
const nsTimeLayout = "2006-01-02T15:04:05-0700"

// parseNSTime parses a timestamp returned by the NS API. It returns the
// zero time on parse failure or empty input.
func parseNSTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(nsTimeLayout, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
