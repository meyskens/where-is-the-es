// Package sbbinternal provides a client for the SBB internal train info
// API at the cus-api-gateway. Unlike the public SBB GraphQL trip planner,
// this endpoint allows looking up a train directly by its number via
// GET {endpoint}/ws/fahrt/{trainNumber}, and returns real-time data
// including replacement train numbers coming from SBB Infra.
//
// The endpoint, Requestor-UUID and Referrer must be configured via options
// (typically from environment variables)
package sbbinternal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/meyskens/where-is-the-es/pkg/traindata"
)

const (
	userAgent = "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:153.0) Gecko/20100101 Firefox/153.0"
)

// Client is an SBB internal API client.
type Client struct {
	endpoint      string // base URL
	requestorUUID string // Requestor-UUID header value
	httpClient    *http.Client

	// rawJSON holds the most recent raw API response body (pretty-printed),
	// exposed via LastRawJSON so the debug interface can show the raw output.
	rawJSON string
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets a custom *http.Client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		c.httpClient = h
	}
}

// WithEndpoint overrides the gateway base URL.
func WithEndpoint(u string) Option {
	return func(c *Client) {
		c.endpoint = u
	}
}

// WithRequestorUUID sets the Requestor-UUID header value.
func WithRequestorUUID(u string) Option {
	return func(c *Client) {
		c.requestorUUID = u
	}
}

// NewClient creates a new SBB internal API client. The endpoint,
// requestorUUID and referrer are required for the API to return data.
func NewClient(opts ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// LastRawJSON returns the raw JSON body of the most recent API response
// (pretty-printed), or the empty string if no request has run yet. Used by
// the debug interface to surface the raw upstream response.
func (c *Client) LastRawJSON() string {
	return c.rawJSON
}

// --- Response types ---

type apiResponse struct {
	Betriebspunkt *betriebspunkt `json:"betriebspunkt"`
	ServerZeit    string         `json:"serverZeit"`
	Fahrten       []fahrt        `json:"fahrten"`
	GoneOffset    string         `json:"goneOffset"`
}

type betriebspunkt struct {
	Abk    string `json:"abk"`
	BezOff string `json:"bezOff"`
	BpUic  int    `json:"bpUic"`
	QosBit int    `json:"qosBitfeld"`
}

type fahrt struct {
	Ersatz    *ersatz   `json:"ersatz"`
	FahrtID   fahrtID   `json:"fahrtId"`
	FahrtInfo fahrtInfo `json:"fahrtInfo"`
	Zuglaeufe []zuglauf `json:"zuglaeufe"`
}

type ersatz struct {
	Fahrt  ersatzFahrt  `json:"fahrt"`
	Detail ersatzDetail `json:"detail"`
}

type ersatzFahrt struct {
	FahrtID      fahrtID      `json:"fahrtId"`
	FahrtInfo    fahrtInfo    `json:"fahrtInfo"`
	Haltestellen []haltstelle `json:"haltestellen"`
}

type ersatzDetail struct {
	Bp       betriebspunkt `json:"bp"`
	Richtung string        `json:"richtung"` // "NACH" or "VON"
}

type fahrtID struct {
	Betriebstag     string `json:"betriebstag"`
	FahrtBezeichner string `json:"fahrtBezeichner"`
}

type fahrtInfo struct {
	Tu        string `json:"tu"`
	VmArt     string `json:"vmArt"`
	VmGruppe  string `json:"vmGruppe"`
	Zugnummer int    `json:"zugnummer"`
}

type zuglauf struct {
	FahrtID      fahrtID      `json:"fahrtId"`
	FahrtInfo    fahrtInfo    `json:"fahrtInfo"`
	Haltestellen []haltstelle `json:"haltestellen"`
}

type haltstelle struct {
	Abk               string   `json:"abk"`
	Bezeichnung       string   `json:"bezeichnung"`
	BpUic             int      `json:"bpUic"`
	Abfahrtszeiten    *zeiten  `json:"abfahrtszeiten"`
	Ankunftszeiten    *zeiten  `json:"ankunftszeiten"`
	Ankunftsgleise    *gleise  `json:"ankunftsgleise"`
	Abfahrtsgleise    *gleise  `json:"abfahrtsgleise"`
	HalteCode         string   `json:"halteCode"`
	Flags             []string `json:"flags"`
	Ersatz            bool     `json:"ersatz"`
	Ausfallgrund      *grund   `json:"ausfallgrund"`
	Verspaetungsgrund *grund   `json:"verspaetungsgrund"`
}

type zeiten struct {
	Kb                string `json:"kb"`  // scheduled
	Ist               string `json:"ist"` // actual/real-time
	Erw               string `json:"erw"` // expected
	Verspaetung       int    `json:"verspaetung"`
	VerspaetungPrefix string `json:"verspaetungPrefix"`
	Status            string `json:"status"`       // AUSFALL, etc.
	EchtzeitInfo      string `json:"echtzeitInfo"` // KEINE_ECHTZEIT_INFO
}

type gleise struct {
	AnPerron       string `json:"anPerron"`
	AbPerron       string `json:"abPerron"`
	Aussteigeseite string `json:"aussteigeseite"`
	Ist            string `json:"ist"`
	Kb             string `json:"kb"`
	KbStatus       string `json:"kbStatus"`
}

type grund struct {
	De string `json:"de"`
	Fr string `json:"fr"`
	It string `json:"it"`
	ID string `json:"id"`
}

// GetTrainByNumber looks up a train by its number and converts the best
// available data into a traindata.Trip. When a fahrt has an `ersatz`
// object, its haltestellen are used because a train number change
// it has priority and carries real-time data.
func (c *Client) GetTrainByNumber(ctx context.Context, trainNumber string, date time.Time) (*traindata.Trip, error) {
	if c.endpoint == "" {
		return nil, fmt.Errorf("sbb-internal: endpoint not configured")
	}
	if trainNumber == "" {
		return nil, fmt.Errorf("sbb-internal: empty train number")
	}

	num := stripPrefix(trainNumber)

	url := fmt.Sprintf("%s/ws/fahrt/%s", c.endpoint, num)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("sbb-internal: create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Requestor-UUID", c.requestorUUID)

	referrer := "https://" + c.requestorUUID
	req.Header.Set("Referer", referrer+"/")
	req.Header.Set("Origin", referrer)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sbb-internal: request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("sbb-internal: read body: %w", err)
	}

	// Store pretty-printed raw JSON for the debug interface.
	if pretty, perr := prettyJSON(body); perr == nil {
		c.rawJSON = pretty
	} else {
		c.rawJSON = string(body)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sbb-internal: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("sbb-internal: parse response: %w", err)
	}

	// Select the best fahrt: prefer one with an ersatz (replacement) object
	// that matches our train number, as replacement traffic has priority.
	// Otherwise, prefer a fahrt with real-time data (ist times present).
	// Fall back to any fahrt matching the train number.
	var bestFahrt *fahrt
	bestScore := -1
	for i := range apiResp.Fahrten {
		f := &apiResp.Fahrten[i]
		if strconv.Itoa(f.FahrtInfo.Zugnummer) != num {
			continue
		}
		score := 0
		if f.Ersatz != nil {
			score += 100 // replacement traffic has priority
		}
		if hasRealtimeData(f) {
			score += 10
		}
		if score > bestScore {
			bestScore = score
			bestFahrt = f
		}
	}

	if bestFahrt == nil {
		return nil, ErrNotFound
	}

	// If the chosen fahrt has an ersatz, use its haltestellen (the
	// replacement train that is actually running, with real-time data).
	var haltestellen []haltstelle
	if bestFahrt.Ersatz != nil && len(bestFahrt.Ersatz.Fahrt.Haltestellen) > 0 {
		haltestellen = bestFahrt.Ersatz.Fahrt.Haltestellen
	} else if len(bestFahrt.Zuglaeufe) > 0 {
		haltestellen = bestFahrt.Zuglaeufe[0].Haltestellen
	}

	if len(haltestellen) == 0 {
		return nil, ErrNotFound
	}

	return haltestellenToTrip(haltestellen, trainNumber, date), nil
}

// hasRealtimeData checks whether any stop in the fahrt's zuglaeufe has
// real-time (ist) time data.
func hasRealtimeData(f *fahrt) bool {
	for _, zl := range f.Zuglaeufe {
		for _, h := range zl.Haltestellen {
			if h.Abfahrtszeiten != nil && h.Abfahrtszeiten.Ist != "" {
				return true
			}
			if h.Ankunftszeiten != nil && h.Ankunftszeiten.Ist != "" {
				return true
			}
		}
	}
	return false
}

// haltestellenToTrip converts SBB internal haltestellen into a traindata.Trip.
func haltestellenToTrip(haltestellen []haltstelle, trainNumber string, date time.Time) *traindata.Trip {
	trip := &traindata.Trip{
		TrainNumber: trainNumber,
		Date:        date,
		Stops:       make([]traindata.Stop, 0, len(haltestellen)),
	}

	for _, h := range haltestellen {
		stop := traindata.Stop{
			StationName:         h.Bezeichnung,
			StationUIC:          h.BpUic,
			DataSources:         []traindata.DataSource{traindata.DataSourceSBBInternal},
			PrefferedDataSource: traindata.DataSourceSBBInternal,
		}

		// Scheduled arrival from ankunftszeiten.kb
		if h.Ankunftszeiten != nil {
			stop.ArrivalTime = parseSBBTime(h.Ankunftszeiten.Kb)
			// Real-time arrival: prefer ist, then erw
			if h.Ankunftszeiten.Ist != "" {
				stop.RealArrivalTime = parseSBBTime(h.Ankunftszeiten.Ist)
				stop.IsRealTime = true
			} else if h.Ankunftszeiten.Erw != "" {
				stop.RealArrivalTime = parseSBBTime(h.Ankunftszeiten.Erw)
				stop.IsRealTime = true
			} else if !stop.ArrivalTime.IsZero() {
				stop.RealArrivalTime = stop.ArrivalTime
			}
		}

		// Scheduled departure from abfahrtszeiten.kb
		if h.Abfahrtszeiten != nil {
			stop.DepartureTime = parseSBBTime(h.Abfahrtszeiten.Kb)
			// Real-time departure: prefer ist, then erw
			if h.Abfahrtszeiten.Ist != "" {
				stop.RealDepartureTime = parseSBBTime(h.Abfahrtszeiten.Ist)
				stop.IsRealTime = true
			} else if h.Abfahrtszeiten.Erw != "" {
				stop.RealDepartureTime = parseSBBTime(h.Abfahrtszeiten.Erw)
				stop.IsRealTime = true
			} else if !stop.DepartureTime.IsZero() {
				stop.RealDepartureTime = stop.DepartureTime
			}
		}

		// Platform: prefer ist (real), fall back to kb (scheduled)
		if h.Ankunftsgleise != nil {
			stop.Platform = h.Ankunftsgleise.Kb
			if h.Ankunftsgleise.Ist != "" {
				stop.RealPlatform = h.Ankunftsgleise.Ist
			} else {
				stop.RealPlatform = h.Ankunftsgleise.Kb
			}
		}
		if h.Abfahrtsgleise != nil {
			if stop.Platform == "" {
				stop.Platform = h.Abfahrtsgleise.Kb
			}
			if h.Abfahrtsgleise.Ist != "" {
				stop.RealPlatform = h.Abfahrtsgleise.Ist
			} else if stop.RealPlatform == "" {
				stop.RealPlatform = h.Abfahrtsgleise.Kb
			}
		}

		// Cancelled: flags contains AUSFALL or status is AUSFALL
		for _, flag := range h.Flags {
			if flag == "AUSFALL" {
				stop.Cancelled = true
				stop.IsRealTime = true
				break
			}
		}
		if h.Ankunftszeiten != nil && h.Ankunftszeiten.Status == "AUSFALL" {
			stop.Cancelled = true
			stop.IsRealTime = true
		}
		if h.Abfahrtszeiten != nil && h.Abfahrtszeiten.Status == "AUSFALL" {
			stop.Cancelled = true
			stop.IsRealTime = true
		}

		trip.Stops = append(trip.Stops, stop)
	}

	return trip
}

// ErrNotFound is returned when no fahrt matching the train number is found.
var ErrNotFound = fmt.Errorf("sbb-internal: train not found")

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
// e.g. "EN401" -> "401".
func stripPrefix(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			return s[i:]
		}
	}
	return ""
}

// parseSBBTime parses an SBB RFC3339 timestamp (e.g.
// "2026-10-04T07:44:03+02:00"). It returns the zero time on failure.
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
