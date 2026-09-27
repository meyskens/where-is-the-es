// Package sbb provides a client for the SBB public
// GraphQL trip-search endpoint at https://graphql.www.sbb.ch/.
//
// For SBB we use the trip planner: we give it an
// origin, a destination and a departure window and it returns candidate
// trips, and look for the European Sleeper in here.

// The endpoint is the same one used by the SBB web shop and requires the
// `apollographql-client-*` headers to be set. No API key is needed.
package sbb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/machinebox/graphql"
	"github.com/meyskens/where-is-the-es/pkg/traindata"
	"golang.org/x/time/rate"
)

const (
	defaultEndpoint = "https://graphql.www.sbb.ch/"

	apolloClientName    = "sbb-webshop-home"
	apolloClientVersion = "18.2.2"
	apolloClientOrigin  = "https://www.sbb.ch"

	userAgent = "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:153.0) Gecko/20100101 Firefox/153.0"
)

// tripsQuery is the GraphQL operation used to search for trips. It requests
// only the fields we need to locate a specific train and extract its stops
// and realtime information.
const tripsQuery = `
query Trips($input: TripInput!, $pagingCursor: String, $language: LanguageEnum!) {
  trips(tripInput: $input, pagingCursor: $pagingCursor, language: $language) {
    trips {
      id
      legs {
        id
        __typename
        duration
        ... on PTRideLeg {
          start { id name }
          end { id name }
          arrival {
            time
            delay
            delayText
            quayFormatted
            quayChanged
          }
          departure {
            time
            delay
            delayText
            quayFormatted
            quayChanged
          }
          serviceJourney {
            id
            stopPoints {
              place { id name }
              occupancy { firstClass secondClass }
              stopStatus
              stopStatusFormatted
              delayUndefined
            }
            serviceProducts {
              name
              number
              vehicleMode
              vehicleSubModeShortName
              routeIndexFrom
              routeIndexTo
            }
            direction
            serviceAlteration {
              cancelled
              partiallyCancelled
              redirected
              delayText
            }
          }
        }
      }
    }
    paginationCursor { previous next }
  }
}
`

// serviceJourneyQuery is the GraphQL operation used to fetch the full
// per-stop arrival/departure data for a single service journey. The trips
// query only exposes times at the leg boundaries; this query returns every
// stop point of the journey with its own arrival and departure events.
const serviceJourneyQuery = `
query getServiceJourneyById($id: NonEmptyString!, $language: LanguageEnum!) {
  serviceJourneyById(id: $id, language: $language) {
    id
    stopPoints {
      stopStatus
      stopStatusFormatted
      requestStop
      delayUndefined
      arrival {
        time
        delay
        delayText
        quayFormatted
        quayChanged
        quayChangedText
      }
      departure {
        time
        delay
        delayText
        quayFormatted
        quayChanged
        quayChangedText
      }
      occupancy { firstClass secondClass }
      place { id name }
      forBoarding
      forAlighting
    }
  }
}
`

// Client is an SBB GraphQL trip-search client.
type Client struct {
	endpoint   string
	httpClient *http.Client
	graphql    *graphql.Client
	limiter    *rate.Limiter

	// rawTripsJSON holds the most recent raw trips-query response body
	// (pretty-printed). It is exposed via LastRawTripsJSON so the debug
	// interface can show the raw SBB output when a train is not found.
	rawTripsJSON string
}

// LastRawTripsJSON returns the raw JSON body of the most recent trips
// query (pretty-printed), or the empty string if no trips query has run
// yet. Used by the debug interface to surface the raw SBB response when a
// train cannot be matched.
func (c *Client) LastRawTripsJSON() string {
	return c.rawTripsJSON
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets a custom *http.Client (e.g. with a timeout or
// instrumented transport).
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		c.httpClient = h
		c.graphql = graphql.NewClient(c.endpoint, graphql.WithHTTPClient(h))
	}
}

// WithEndpoint overrides the GraphQL endpoint. Mostly useful for testing.
func WithEndpoint(u string) Option {
	return func(c *Client) {
		c.endpoint = strings.TrimRight(u, "/")
		c.graphql = graphql.NewClient(c.endpoint, graphql.WithHTTPClient(c.httpClient))
	}
}

// NewClient creates a new SBB GraphQL client
func NewClient(opts ...Option) *Client {
	c := &Client{
		endpoint:   defaultEndpoint,
		httpClient: &http.Client{Timeout: 120 * time.Second},
		limiter:    rate.NewLimiter(rate.Every(time.Second), 1),
	}
	for _, opt := range opts {
		opt(c)
	}
	// Wrap the HTTP transport so we can capture the raw trips-query
	// response body for the debug interface (e.g. when no matching
	// train is found).
	base := c.httpClient.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	c.httpClient.Transport = &recordingTransport{base: base, client: c}
	if c.graphql == nil {
		c.graphql = graphql.NewClient(c.endpoint, graphql.WithHTTPClient(c.httpClient))
	}
	return c
}

// placeInput is a typed place reference inside a TripInput.
type placeInput struct {
	Type  string `json:"type"`  // always "ID"
	Value string `json:"value"` // the UIC station code as a string
}

// timeInput is the departure/arrival time specification.
type timeInput struct {
	Date string `json:"date"` // "2006-01-02"
	Time string `json:"time"` // "15:04"
	Type string `json:"type"` // "DEPARTURE" or "ARRIVAL"
}

// tripInput mirrors the TripInput GraphQL input type used by the SBB trips
// query. The field set matches the request the SBB web shop sends.
type tripInput struct {
	Places                  []placeInput `json:"places"`
	Time                    timeInput    `json:"time"`
	IncludeEconomic         bool         `json:"includeEconomic"`
	DirectConnection        bool         `json:"directConnection"`
	IncludeAccessibility    string       `json:"includeAccessibility"`
	IncludeNoticeAttributes []any        `json:"includeNoticeAttributes"`
	IncludeTransportModes   []string     `json:"includeTransportModes"`
	IncludeUnsharp          bool         `json:"includeUnsharp"`
	Occupancy               string       `json:"occupancy"`
	WalkSpeed               int          `json:"walkSpeed"`
}

// tripsResponse is the contents of the GraphQL `data` field returned by the
// SBB trips query
type tripsResponse struct {
	Trips struct {
		Trips            []sbbTrip `json:"trips"`
		PaginationCursor struct {
			Previous string `json:"previous"`
			Next     string `json:"next"`
		} `json:"paginationCursor"`
	} `json:"trips"`
}

// serviceJourneyResponse is the contents of the GraphQL `data` field returned
// by the serviceJourneyById query
type serviceJourneyResponse struct {
	ServiceJourneyByID *sbbServiceJourneyDetail `json:"serviceJourneyById"`
}

// sbbServiceJourneyDetail is the full service-journey object returned by
// serviceJourneyById.
type sbbServiceJourneyDetail struct {
	ID         string         `json:"id"`
	StopPoints []sbbStopPoint `json:"stopPoints"`
}

type sbbTrip struct {
	ID   string   `json:"id"`
	Legs []sbbLeg `json:"legs"`
}

// sbbLeg is a single leg of a trip
type sbbLeg struct {
	ID             string              `json:"id"`
	Typename       string              `json:"__typename"`
	Duration       int                 `json:"duration"`
	Start          sbbStopPlace        `json:"start"`
	End            sbbStopPlace        `json:"end"`
	Arrival        sbbArrivalDeparture `json:"arrival"`
	Departure      sbbArrivalDeparture `json:"departure"`
	ServiceJourney sbbServiceJourney   `json:"serviceJourney"`
}

func (l sbbLeg) isRideLeg() bool { return l.Typename == "PTRideLeg" }

type sbbStopPlace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type sbbArrivalDeparture struct {
	Time          string `json:"time"`
	Delay         int    `json:"delay"`
	DelayText     string `json:"delayText"`
	QuayFormatted string `json:"quayFormatted"`
	QuayChanged   bool   `json:"quayChanged"`
}

type sbbServiceJourney struct {
	ID                string               `json:"id"`
	StopPoints        []sbbStopPoint       `json:"stopPoints"`
	ServiceProducts   []sbbServiceProduct  `json:"serviceProducts"`
	Direction         string               `json:"direction"`
	ServiceAlteration sbbServiceAlteration `json:"serviceAlteration"`
}

type sbbStopPoint struct {
	Place               sbbStopPlace         `json:"place"`
	Occupancy           sbbOccupancy         `json:"occupancy"`
	StopStatus          string               `json:"stopStatus"`
	StopStatusFormatted string               `json:"stopStatusFormatted"`
	DelayUndefined      bool                 `json:"delayUndefined"`
	Arrival             *sbbArrivalDeparture `json:"arrival"`
	Departure           *sbbArrivalDeparture `json:"departure"`
	ForBoarding         bool                 `json:"forBoarding"`
	ForAlighting        bool                 `json:"forAlighting"`
}

type sbbOccupancy struct {
	FirstClass  string `json:"firstClass"`
	SecondClass string `json:"secondClass"`
}

type sbbServiceProduct struct {
	Name                    string `json:"name"`
	Number                  string `json:"number"`
	VehicleMode             string `json:"vehicleMode"`
	VehicleSubModeShortName string `json:"vehicleSubModeShortName"`
	RouteIndexFrom          int    `json:"routeIndexFrom"`
	RouteIndexTo            int    `json:"routeIndexTo"`
}

type sbbServiceAlteration struct {
	Cancelled          bool   `json:"cancelled"`
	PartiallyCancelled bool   `json:"partiallyCancelled"`
	Redirected         bool   `json:"redirected"`
	DelayText          string `json:"delayText"`
}

// GetTimetable searches the SBB trip planner for a train matching trainNumber
// departing from originUIC towards destUIC around the given departure time,
// and converts the matching leg into a traindata.Trip.
func (c *Client) GetTimetable(ctx context.Context, trainNumber, originUIC, destUIC string, departure time.Time) (*traindata.Trip, error) {
	num := stripPrefix(trainNumber)
	if num == "" {
		return nil, fmt.Errorf("sbb: empty train number")
	}
	if originUIC == "" || destUIC == "" {
		return nil, fmt.Errorf("sbb: origin and destination UIC are required")
	}

	// Search from 30 minutes before the planned departure so the train we
	// want is inside the result window.
	searchTime := departure.Add(-30 * time.Minute)

	input := tripInput{
		Places: []placeInput{
			{Type: "ID", Value: originUIC},
			{Type: "ID", Value: destUIC},
		},
		Time: timeInput{
			Date: searchTime.Format("2006-01-02"),
			Time: searchTime.Format("15:04"),
			Type: "DEPARTURE",
		},
		IncludeEconomic:      false,
		DirectConnection:     false,
		IncludeAccessibility: "NONE",
		IncludeTransportModes: []string{
			"HIGH_SPEED_TRAIN", "INTERCITY", "INTERREGIO", "REGIO",
			"URBAN_TRAIN", "SPECIAL_TRAIN", "SHIP", "BUS",
			"TRAMWAY", "CABLEWAY_GONDOLA_CHAIRLIFT_FUNICULAR", // why not let's have fun
		},
		IncludeUnsharp: false,
		Occupancy:      "ALL",
		WalkSpeed:      100,
	}

	var resp tripsResponse
	if err := c.run(ctx, tripsQuery, map[string]any{
		"input":        input,
		"pagingCursor": nil,
		"language":     "DE",
	}, &resp); err != nil {
		return nil, fmt.Errorf("sbb: search trips: %w", err)
	}

	leg, err := findLegByTrainNumber(resp.Trips.Trips, num)
	if err != nil {
		// Attach the raw trips JSON to the not-found error so callers can
		// surface it in the debug interface. We keep ErrNotFound as the
		// sentinel so errors.Is still works.
		return nil, fmt.Errorf("%w (raw trips response: %s)", err, c.rawTripsJSON)
	}

	// The trips query only returns arrival/departure times at the leg
	// boundaries. Fetch the full per-stop data via serviceJourneyById so
	// every intermediate stop gets its own arrival/departure event.
	detail, err := c.GetServiceJourney(ctx, leg.ServiceJourney.ID)
	if err != nil {
		// Fall back to the leg-level data if the detail lookup fails
		// (e.g. the journey id has expired). The leg still carries the
		// boundary times and the full stop-point list.
		return legToTrip(leg, nil, trainNumber, departure), nil
	}

	return legToTrip(leg, detail, trainNumber, departure), nil
}

// GetServiceJourney fetches the full per-stop arrival/departure data for a
// single SBB service journey by its id (the `serviceJourney.id` returned by
// the trips query).
func (c *Client) GetServiceJourney(ctx context.Context, journeyID string) (*sbbServiceJourneyDetail, error) {
	if journeyID == "" {
		return nil, fmt.Errorf("sbb: empty service journey id")
	}

	var resp serviceJourneyResponse
	if err := c.run(ctx, serviceJourneyQuery, map[string]any{
		"id":       journeyID,
		"language": "DE",
	}, &resp); err != nil {
		return nil, fmt.Errorf("sbb: get service journey: %w", err)
	}
	if resp.ServiceJourneyByID == nil {
		return nil, ErrNotFound
	}
	return resp.ServiceJourneyByID, nil
}

// findLegByTrainNumber scans the first page of trip results for a PTRideLeg
// whose service-product number matches num.
func findLegByTrainNumber(trips []sbbTrip, num string) (sbbLeg, error) {
	for _, t := range trips {
		for _, l := range t.Legs {
			if !l.isRideLeg() {
				continue
			}
			for _, p := range l.ServiceJourney.ServiceProducts {
				if stripPrefix(p.Number) == num {
					return l, nil
				}
			}
		}
	}
	return sbbLeg{}, ErrNotFound
}

// legToTrip converts a matched SBB ride leg into a traindata.Trip.
func legToTrip(leg sbbLeg, detail *sbbServiceJourneyDetail, trainNumber string, date time.Time) *traindata.Trip {
	legStartUIC, _ := strconv.Atoi(leg.Start.ID)
	legEndUIC, _ := strconv.Atoi(leg.End.ID)
	departTime := parseSBBTime(leg.Departure.Time)
	arriveTime := parseSBBTime(leg.Arrival.Time)
	departDelay := time.Duration(leg.Departure.Delay) * time.Minute
	arriveDelay := time.Duration(leg.Arrival.Delay) * time.Minute

	// Prefer the detailed stop list from serviceJourneyById; it has the
	// same ordering as the leg stopPoints but with full arrival/departure
	// events on every stop.
	stopPoints := leg.ServiceJourney.StopPoints
	if detail != nil && len(detail.StopPoints) > 0 {
		stopPoints = detail.StopPoints
	}

	trip := &traindata.Trip{
		TrainNumber: trainNumber,
		Date:        date,
		Stops:       make([]traindata.Stop, 0, len(stopPoints)),
	}

	for _, sp := range stopPoints {
		uic, _ := strconv.Atoi(sp.Place.ID)
		stop := traindata.Stop{
			StationName:         sp.Place.Name,
			StationUIC:          uic,
			DataSources:         []traindata.DataSource{traindata.DataSourceSBB},
			PrefferedDataSource: traindata.DataSourceSBB,
		}

		// Per-stop arrival/departure from the detail query.
		if sp.Arrival != nil {
			stop.ArrivalTime = parseSBBTime(sp.Arrival.Time)
			if sp.Arrival.Delay != 0 {
				stop.RealArrivalTime = stop.ArrivalTime.Add(time.Duration(sp.Arrival.Delay) * time.Minute)
				stop.IsRealTime = true
			} else if !stop.ArrivalTime.IsZero() {
				stop.RealArrivalTime = stop.ArrivalTime
			}
			stop.Platform = sp.Arrival.QuayFormatted
			stop.RealPlatform = sp.Arrival.QuayFormatted
			if sp.Arrival.QuayChanged {
				stop.IsRealTime = true
			}
		}
		if sp.Departure != nil {
			stop.DepartureTime = parseSBBTime(sp.Departure.Time)
			if sp.Departure.Delay != 0 {
				stop.RealDepartureTime = stop.DepartureTime.Add(time.Duration(sp.Departure.Delay) * time.Minute)
				stop.IsRealTime = true
			} else if !stop.DepartureTime.IsZero() {
				stop.RealDepartureTime = stop.DepartureTime
			}
			if stop.Platform == "" {
				stop.Platform = sp.Departure.QuayFormatted
			}
			if stop.RealPlatform == "" {
				stop.RealPlatform = sp.Departure.QuayFormatted
			}
			if sp.Departure.QuayChanged {
				stop.IsRealTime = true
			}
		}

		// If there was no detail query, fall back to the leg boundary
		// times for the leg start (departure) and end (arrival) stops.
		if detail == nil {
			if uic == legStartUIC && !departTime.IsZero() && stop.DepartureTime.IsZero() {
				stop.DepartureTime = departTime
				stop.RealDepartureTime = departTime.Add(departDelay)
				stop.Platform = leg.Departure.QuayFormatted
				stop.RealPlatform = leg.Departure.QuayFormatted
				if leg.Departure.Delay != 0 || leg.Departure.QuayChanged {
					stop.IsRealTime = true
				}
			}
			if uic == legEndUIC && !arriveTime.IsZero() && stop.ArrivalTime.IsZero() {
				stop.ArrivalTime = arriveTime
				stop.RealArrivalTime = arriveTime.Add(arriveDelay)
				if stop.Platform == "" {
					stop.Platform = leg.Arrival.QuayFormatted
				}
				if stop.RealPlatform == "" {
					stop.RealPlatform = leg.Arrival.QuayFormatted
				}
				if leg.Arrival.Delay != 0 || leg.Arrival.QuayChanged {
					stop.IsRealTime = true
				}
			}
		}

		if leg.ServiceJourney.ServiceAlteration.Cancelled {
			stop.Cancelled = true
			stop.IsRealTime = true
		}

		trip.Stops = append(trip.Stops, stop)
	}

	return trip
}

// run executes a GraphQL request with the SBB-specific Apollo headers and
// rate limiting.
func (c *Client) run(ctx context.Context, query string, vars map[string]any, out any) error {
	if c.limiter != nil {
		if err := c.limiter.Wait(ctx); err != nil {
			return err
		}
	}

	req := graphql.NewRequest(query)
	for k, v := range vars {
		req.Var(k, v)
	}

	req.Header.Set("Accept", "application/graphql-response+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("apollographql-client-name", apolloClientName)
	req.Header.Set("apollographql-client-origin", apolloClientOrigin)
	req.Header.Set("apollographql-client-version", apolloClientVersion)
	req.Header.Set("apollographql-client-time", time.Now().Format("02.01.2006 15:04:05.000 -0700"))
	req.Header.Set("Origin", "https://www.sbb.ch")
	req.Header.Set("Referer", "https://www.sbb.ch/")

	return c.graphql.Run(ctx, req, out)
}

// ErrNotFound is returned when no leg matching the train number is found in
// the SBB trip results.
var ErrNotFound = fmt.Errorf("sbb: train not found")

// recordingTransport wraps an http.RoundTripper and stores the last response
// body on the owning Client so the debug interface can surface the raw SBB
// trips JSON when no matching train is found.
type recordingTransport struct {
	base   http.RoundTripper
	client *Client
}

func (r *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := r.base.RoundTrip(req)
	if err != nil || resp == nil {
		return resp, err
	}
	body, readErr := io.ReadAll(resp.Body)
	resp.Body.Close()
	if readErr != nil {
		return nil, readErr
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	if r.client != nil {
		if pretty, perr := prettyJSON(body); perr == nil {
			r.client.rawTripsJSON = pretty
		} else {
			r.client.rawTripsJSON = string(body)
		}
	}
	return resp, nil
}

// prettyJSON re-indents a JSON document for human-readable debug output.
func prettyJSON(b []byte) (string, error) {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return "", err
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// stripPrefix removes any leading non-digit characters from a train number,
// e.g. "EN400" -> "400".
func stripPrefix(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			return s[i:]
		}
	}
	return ""
}

// parseSBBTime parses an SBB RFC3339 timestamp (e.g.
// "2026-09-27T17:44:00+02:00"). It returns the zero time on failure.
func parseSBBTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
