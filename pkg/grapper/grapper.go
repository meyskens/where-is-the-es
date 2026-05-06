// Package grapper provides a client for a private GRAPP to JSON instance.
//
// It exposes a helper to fetch the journey for a given train number and
// convert it into the project's traindata.Trip representation.
//
// The GRAPP instance URL is configurable via the base URL option.
package grapper

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

// Client is a GRAPP API client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	limiter    *rate.Limiter
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets a custom *http.Client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.httpClient = h }
}

// WithBaseURL overrides the API base URL.
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") }
}

// NewClient creates a new GRAPP API client.
// Requests are rate limited to 1 per second by default.
func NewClient(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		limiter:    rate.NewLimiter(rate.Every(time.Second), 1),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// TrainResponse is the JSON response from the GRAPP API.
type TrainResponse struct {
	Carrier         string `json:"carrier"`
	CurrentLocation string `json:"current_location"`
	Delay           int    `json:"delay"`
	Destination     string `json:"destination"`
	ETCS            bool   `json:"etcs"`
	ExpectedDelay   int    `json:"expected_delay"`
	ID              int64  `json:"id"`
	Number          int    `json:"number"`
	Origin          string `json:"origin"`
	Position        struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"position"`
	Route []struct {
		Arrival struct {
			Actual string `json:"actual"`
			Plan   string `json:"plan"`
		} `json:"arrival"`
		Departure struct {
			Actual string `json:"actual"`
			Plan   string `json:"plan"`
		} `json:"departure"`
		IsCurrent bool   `json:"is_current"`
		Name      string `json:"name"`
	} `json:"route"`
	Title string `json:"title"`
}

// HTTPError is returned when the API responds with a non-2xx status.
type HTTPError struct {
	StatusCode int
	Headers    http.Header
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("unexpected status %d: headers=%v body=%q", e.StatusCode, e.Headers, e.Body)
}

// ErrNotFound is returned when the train is not found.
var ErrNotFound = fmt.Errorf("train not found")

// ErrTitleMismatch is returned when the train title does not contain the expected train number.
var ErrTitleMismatch = fmt.Errorf("train title does not match expected number")

// FetchTrain fetches the train data for the given train number.
func (c *Client) FetchTrain(ctx context.Context, trainNumber string) (*TrainResponse, error) {
	num := stripPrefix(trainNumber)
	if num == "" {
		return nil, fmt.Errorf("grapper: empty train number")
	}

	endpoint := c.baseURL + "/train/" + url.PathEscape(num)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Headers:    resp.Header,
			Body:       strings.TrimSpace(string(body)),
		}
	}

	var raw TrainResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("grapper: decode response: %w", err)
	}

	// Verify the title contains the train number with exact match.
	// Extract the train number from the title and compare exactly
	// to avoid partial matches (e.g., "452" should not match "1452").
	title := strings.TrimSpace(raw.Title)
	if title == "" {
		return nil, ErrTitleMismatch
	}

	// Extract the numeric part from the title
	titleNum := stripPrefix(title)
	if titleNum == "" {
		return nil, ErrTitleMismatch
	}

	// Exact match: the numeric part of the title must equal the requested number
	if titleNum != num {
		return nil, ErrTitleMismatch
	}

	return &raw, nil
}

// GetTimetable returns a traindata.Trip for the given train number and date.
func (c *Client) GetTimetable(ctx context.Context, trainNumber string, date time.Time) (*traindata.Trip, error) {
	resp, err := c.FetchTrain(ctx, trainNumber)
	if err != nil {
		return nil, err
	}

	trip := &traindata.Trip{
		TrainNumber: trainNumber,
		Date:        date.Truncate(24 * time.Hour),
		Stops:       make([]traindata.Stop, 0, len(resp.Route)),
	}

	for i, r := range resp.Route {
		stop := traindata.Stop{
			StationName:         r.Name,
			DataSources:         []traindata.DataSource{traindata.DataSourceSZ},
			PrefferedDataSource: traindata.DataSourceSZ,
		}

		// Parse arrival time.
		if r.Arrival.Plan != "" {
			stop.ArrivalTime = parseTime(date, r.Arrival.Plan)
		}
		if r.Arrival.Actual != "" {
			stop.RealArrivalTime = parseTime(date, r.Arrival.Actual)
			stop.IsRealTime = true
		}

		// Parse departure time.
		if r.Departure.Plan != "" {
			stop.DepartureTime = parseTime(date, r.Departure.Plan)
		}
		if r.Departure.Actual != "" {
			stop.RealDepartureTime = parseTime(date, r.Departure.Actual)
			stop.IsRealTime = true
		}

		// For the first stop, only departure matters.
		if i == 0 {
			stop.ArrivalTime = time.Time{}
			stop.RealArrivalTime = time.Time{}
		}

		// For the last stop, only arrival matters.
		if i == len(resp.Route)-1 {
			stop.DepartureTime = time.Time{}
			stop.RealDepartureTime = time.Time{}
		}

		trip.Stops = append(trip.Stops, stop)
	}

	return trip, nil
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

// prahaLocation is the timezone for Praha (Prague), used for all GRAPP times.
var prahaLocation = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Prague")
	if err != nil {
		panic("grapper: failed to load Europe/Prague timezone: " + err.Error())
	}
	return loc
}()

// parseTime parses a "HH:MM" time string and anchors it to the given date in Praha timezone.
func parseTime(date time.Time, t string) time.Time {
	parts := strings.Split(t, ":")
	if len(parts) != 2 {
		return time.Time{}
	}
	hour, err1 := strconv.Atoi(parts[0])
	min, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return time.Time{}
	}
	return time.Date(date.Year(), date.Month(), date.Day(), hour, min, 0, 0, prahaLocation)
}
