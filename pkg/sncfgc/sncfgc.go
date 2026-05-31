// Package sncfgc provides a client for the SNCF Gares & Connexions API.
//
// It exposes a helper to fetch the journey for a given train number, UIC station code,
// and date, converting it into the project's traindata.Trip representation.
//
// The SNCF Gares & Connexions API requires a subscription key passed as a request header:
//   - Ocp-Apim-Subscription-Key
package sncfgc

import (
	"context"
	"encoding/json"
	"errors"
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
	defaultBaseURL = "https://garesetconnexions-online.azure-api.net/API/V3/PIV"
)

// Client is an SNCF Gares & Connexions API client.
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

// NewClient creates a new SNCF Gares & Connexions API client. subscriptionKey is required.
// Requests are rate limited to 1 per second by default to stay within the API quota.
func NewClient(subscriptionKey string, opts ...Option) *Client {
	c := &Client{
		subscriptionKey: subscriptionKey,
		baseURL:         defaultBaseURL,
		httpClient:      &http.Client{Timeout: 30 * time.Second, Transport: http.DefaultTransport},
		limiter:         rate.NewLimiter(rate.Every(time.Second), 1),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// TrainDetailsResponse is the JSON response from the SNCF Gares & Connexions TrainDetails endpoint.
type TrainDetailsResponse struct {
	ShortTermInformations []interface{} `json:"shortTermInformations"`
	Composition           interface{}   `json:"composition"`
	ListeReservations     interface{}   `json:"listeReservations"`
	ListServicesABord     []struct {
		UIC     string `json:"uic"`
		Libelle string `json:"libelle"`
		URL     string `json:"url"`
	} `json:"listServicesABord"`
	Brand          string `json:"brand"`
	StationName    string `json:"stationName"`
	StopoverNumber string `json:"stopoverNumber"`
	IsLeft         bool   `json:"isLeft"`
	TrainLength    *int   `json:"trainLength"`
	JourneyStop    []struct {
		ScheduledTime     string `json:"scheduledTime"`
		ActualTime        string `json:"actualTime"`
		InformationStatus struct {
			TrainStatus string `json:"trainStatus"`
			EventLevel  string `json:"eventLevel"`
			Delay       *int   `json:"delay"`
		} `json:"informationStatus"`
		Platform struct {
			Track           string `json:"track"`
			IsTrackactive   bool   `json:"isTrackactive"`
			TrackGroupTitle string `json:"trackGroupTitle"`
			TrackGroupValue string `json:"trackGroupValue"`
			BackgroundColor string `json:"backgroundColor"`
			TrackPosition   string `json:"trackPosition"`
		} `json:"platform"`
		StatusModification    interface{}   `json:"statusModification"`
		ShortTermInformations []interface{} `json:"shortTermInformations"`
		StationName           string        `json:"stationName"`
		Downtime              *int          `json:"downtime"`
		TheoreticalOccupancy  interface{}   `json:"theoreticalOccupancy"`
		RealOccupancy         interface{}   `json:"realOccupancy"`
		UIC                   string        `json:"uic"`
		IsNewOrigin           bool          `json:"isNewOrigin"`
		IsNewDestination      bool          `json:"isNewDestination"`
	} `json:"JourneyStop"`
	Direction     string `json:"direction"`
	TrainNumber   string `json:"trainNumber"`
	ScheduledTime string `json:"scheduledTime"`
	ActualTime    string `json:"actualTime"`
	TrainType     string `json:"trainType"`
	TrainMode     string `json:"trainMode"`
	Platform      struct {
		Track           string `json:"track"`
		IsTrackactive   bool   `json:"isTrackactive"`
		TrackGroupTitle string `json:"trackGroupTitle"`
		TrackGroupValue string `json:"trackGroupValue"`
		BackgroundColor string `json:"backgroundColor"`
		TrackPosition   string `json:"trackPosition"`
	} `json:"platform"`
	InformationStatus struct {
		TrainStatus string `json:"trainStatus"`
		EventLevel  string `json:"eventLevel"`
		Delay       *int   `json:"delay"`
	} `json:"informationStatus"`
	Traffic struct {
		Origin         string `json:"origin"`
		Destination    string `json:"destination"`
		OldOrigin      string `json:"oldOrigin"`
		OldDestination string `json:"oldDestination"`
		EventStatus    string `json:"eventStatus"`
		EventLevel     string `json:"eventLevel"`
	} `json:"traffic"`
	StatusModification interface{} `json:"statusModification"`
	TrafficDetailsURL  string      `json:"TrafficDetailsUrl"`
	UIC                string      `json:"uic"`
	MissionCode        interface{} `json:"missionCode"`
	TrainLine          interface{} `json:"trainLine"`
	IsGL               bool        `json:"isGL"`
	Presentation       struct {
		ColorCode     string `json:"colorCode"`
		TextColorCode string `json:"textColorCode"`
	} `json:"presentation"`
	Stops            []string    `json:"stops"`
	AlternativeMeans interface{} `json:"alternativeMeans"`
}

// HTTPError is returned when the SNCF API responds with a non-2xx status.
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

// GetTimetable returns a traindata.Trip for the given train number, UIC station code, and date.
// The isDeparture parameter determines whether to look for departures or arrivals.
func (c *Client) GetTimetable(ctx context.Context, trainNumber string, uicStation string, date time.Time, isDeparture bool) (*traindata.Trip, error) {
	if trainNumber == "" {
		return nil, fmt.Errorf("sncfgc: empty train number")
	}
	if uicStation == "" {
		return nil, fmt.Errorf("sncfgc: empty UIC station code")
	}

	// Format departureDateTime as ISO 8601 with URL encoding
	departureDateTime := date.Format(time.RFC3339)

	q := url.Values{}
	q.Set("departureDateTime", departureDateTime)
	q.Set("isDeparture", strconv.FormatBool(isDeparture))
	q.Set("trainNumber", trainNumber)
	q.Set("uic", uicStation)

	endpoint := c.baseURL + "/TrainDetails?" + q.Encode()

	var raw []TrainDetailsResponse
	if err := c.do(ctx, endpoint, &raw); err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == 404 {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("sncfgc: get train details: %w", err)
	}

	if len(raw) == 0 {
		return nil, fmt.Errorf("sncfgc: no train details found for train %s", trainNumber)
	}

	// Use the first train details (usually there's only one)
	trainDetails := raw[0]

	if len(trainDetails.JourneyStop) == 0 {
		return nil, fmt.Errorf("sncfgc: no stops found for train %s", trainNumber)
	}

	trip := &traindata.Trip{
		TrainNumber: trainNumber,
		Stops:       make([]traindata.Stop, 0, len(trainDetails.JourneyStop)),
		Date:        date,
	}

	for _, s := range trainDetails.JourneyStop {
		uic, _ := strconv.Atoi(s.UIC)

		stop := traindata.Stop{
			StationName:         s.StationName,
			StationUIC:          uic,
			DataSources:         []traindata.DataSource{traindata.DataSourceSNCFGC},
			PrefferedDataSource: traindata.DataSourceSNCFGC,
		}

		// Parse scheduled and actual times
		stop.ArrivalTime = parseSNCFCGTime(s.ScheduledTime)
		stop.RealArrivalTime = parseSNCFCGTime(s.ActualTime)
		stop.DepartureTime = parseSNCFCGTime(s.ScheduledTime)
		stop.RealDepartureTime = parseSNCFCGTime(s.ActualTime)

		// If actualTime is null but delay is set, calculate real time from scheduled + delay
		if stop.RealArrivalTime.IsZero() && s.InformationStatus.Delay != nil && *s.InformationStatus.Delay > 0 {
			stop.RealArrivalTime = stop.ArrivalTime.Add(time.Duration(*s.InformationStatus.Delay) * time.Minute)
		}
		if stop.RealDepartureTime.IsZero() && s.InformationStatus.Delay != nil && *s.InformationStatus.Delay > 0 {
			stop.RealDepartureTime = stop.DepartureTime.Add(time.Duration(*s.InformationStatus.Delay) * time.Minute)
		}

		// Add downtime to departure time (downtime is the seconds the train waits in station, we think a mistranslation of dwelltime)
		// If there's a delay, subtract the delay from the downtime
		// If remaining downtime > 0, train leaves punctually; otherwise it leaves immediately
		if s.Downtime != nil && *s.Downtime > 0 {
			remainingDowntime := *s.Downtime
			if s.InformationStatus.Delay != nil && *s.InformationStatus.Delay > 0 {
				// Delay is in minutes, downtime is in seconds - convert delay to seconds
				remainingDowntime = *s.Downtime - (*s.InformationStatus.Delay * 60)
			}
			if remainingDowntime > 0 {
				stop.DepartureTime = stop.DepartureTime.Add(time.Duration(remainingDowntime) * time.Second)
				stop.RealDepartureTime = stop.RealDepartureTime.Add(time.Duration(remainingDowntime) * time.Second)
			}
			// If remainingDowntime <= 0, train leaves immediately (no downtime added)
		}

		// Platform information
		stop.Platform = s.Platform.Track
		if s.Platform.IsTrackactive {
			stop.RealPlatform = s.Platform.Track
		}

		// Delay information
		if s.InformationStatus.Delay != nil && *s.InformationStatus.Delay > 0 {
			stop.IsRealTime = true
		}

		// Check if actual time differs from scheduled
		if !stop.RealArrivalTime.IsZero() && !stop.ArrivalTime.IsZero() && !stop.RealArrivalTime.Equal(stop.ArrivalTime) {
			stop.IsRealTime = true
		}
		if !stop.RealDepartureTime.IsZero() && !stop.DepartureTime.IsZero() && !stop.RealDepartureTime.Equal(stop.DepartureTime) {
			stop.IsRealTime = true
		}

		trip.Stops = append(trip.Stops, stop)
	}

	return trip, nil
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
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "MaGare/4.13.0 (com.sncf.AppliGares.app; build:202602271029; iOS 26.5.0) Alamofire/5.10.2")
	req.Header.Set("Accept-Language", "en")
	req.Header.Set("Connection", "keep-alive")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read the full body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Headers:    resp.Header,
			Body:       strings.TrimSpace(string(body)),
		}
	}
	return json.Unmarshal(body, out)
}

// parseSNCFCGTime parses SNCF Gares & Connexions time strings (RFC3339 format).
func parseSNCFCGTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
