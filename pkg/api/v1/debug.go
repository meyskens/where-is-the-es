package v1

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/meyskens/where-is-the-es/pkg/traindata"
)

// DebugStop is a generic, JSON-friendly representation of a single stop as
// returned by one data source. It intentionally mirrors the fields of
// traindata.Stop so every source can be displayed in the same table.
// The Relevant flag indicates whether the stop was actually used in the
// merged timetable (passed the country/name match) or discarded.
type DebugStop struct {
	StationName         string   `json:"stationName"`
	StationUIC          int      `json:"stationUIC"`
	ArrivalTime         string   `json:"arrivalTime"`
	DepartureTime       string   `json:"departureTime"`
	Platform            string   `json:"platform"`
	RealPlatform        string   `json:"realPlatform"`
	RealArrivalTime     string   `json:"realArrivalTime"`
	RealDepartureTime   string   `json:"realDepartureTime"`
	IsRealTime          bool     `json:"isRealTime"`
	Cancelled           bool     `json:"cancelled"`
	Relevant            bool     `json:"relevant"`
	DataSources         []string `json:"dataSources"`
	PrefferedDataSource string   `json:"prefferedDataSource"`
}

// DebugSourceResult is the generic envelope returned by every debug endpoint.
type DebugSourceResult struct {
	Source      string      `json:"source"`
	TrainNumber string      `json:"trainNumber"`
	Date        string      `json:"date"`
	Available   bool        `json:"available"`
	Found       bool        `json:"found"`
	Error       string      `json:"error,omitempty"`
	Stops       []DebugStop `json:"stops,omitempty"`
}

func (a *APIV1) registerDebugRoutes(e *echo.Echo) {
	e.GET("/api/debug/sources", a.debugSources)

	// Return cached data for a given train from a single data source.
	e.GET("/api/debug/:source/:trainNumber", a.debugSource)
	e.GET("/api/debug/:source/:trainNumber/:date", a.debugSource)
}

// sourceToDataSource maps the debug source ID to the traindata.DataSource
// constant used in stop.DataSources. The "es-base" and "es" sources are
// handled specially — they return the full base / calculated trip
// respectively rather than filtering by DataSources.
var sourceToDataSource = map[string]traindata.DataSource{
	"bahn":      traindata.DataSourceDB,
	"ns":        traindata.DataSourceNS,
	"nmbs":      traindata.DataSourceNMBS,
	"arenaways": traindata.DataSourceArenaways,
	"grapper":   traindata.DataSourceSZ,
	"sncfgc":    traindata.DataSourceSNCFGC,
}

// sourceInfo describes a data source for the /api/debug/sources listing.
type sourceInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Available bool   `json:"available"`
}

func (a *APIV1) debugSources(c echo.Context) error {
	sources := []sourceInfo{
		{ID: "es-base", Name: "European Sleeper (base timetable)", Available: true},
		{ID: "es", Name: "European Sleeper (calculated)", Available: true},
		{ID: "bahn", Name: "Deutsche Bahn", Available: a.bahnClient != nil},
		{ID: "ns", Name: "NS", Available: a.nsClient != nil},
		{ID: "nmbs", Name: "NMBS/SNCB Delay Certificate", Available: a.nmbsFetcher != nil},
		{ID: "arenaways", Name: "Arenaways", Available: a.arenawaysFetcher != nil},
		{ID: "grapper", Name: "GRAPP (CD/SŽ)", Available: a.grapperClient != nil},
		{ID: "sncfgc", Name: "SNCF Gares & Connexions", Available: a.sncfgcClient != nil},
	}
	return c.JSON(http.StatusOK, sources)
}

func (a *APIV1) debugSource(c echo.Context) error {
	source := c.Param("source")
	trainNumber := c.Param("trainNumber")
	dateStr := c.Param("date")

	if trainNumber == "" {
		return c.JSON(http.StatusBadRequest, DebugSourceResult{Source: source, Error: "missing trainNumber"})
	}

	// Validate the source ID and determine which cache to read from.
	isESBase := source == "es-base"
	isES := source == "es"
	ds, isRealtimeSource := sourceToDataSource[source]
	if !isESBase && !isES && !isRealtimeSource {
		return c.JSON(http.StatusBadRequest, DebugSourceResult{Source: source, TrainNumber: trainNumber, Error: "unknown data source: " + source})
	}

	// Resolve the date — same logic as the /api/v1/timetable endpoint.
	var dateStrResolved string
	if dateStr == "" || dateStr == "next" {
		nextDate, found := a.findNextDeparture(trainNumber)
		if !found {
			// Fall back to today so the debug page can still show something.
			dateStrResolved = time.Now().Format("2006-01-02")
		} else {
			dateStrResolved = nextDate
		}
	} else {
		dateStrResolved = dateStr
	}

	service := Service{TrainNumber: trainNumber, Date: dateStrResolved}

	// es-base reads from the base timetable cache; everything else reads
	// from the calculated/enhanced timetable cache.
	var trip *traindata.Trip
	var ok bool
	if isESBase {
		trip, ok = a.baseTimetableCache[service]
	} else {
		trip, ok = a.timetableCache[service]
	}

	result := DebugSourceResult{
		Source:      source,
		TrainNumber: trainNumber,
		Date:        dateStrResolved,
		Available:   true,
		Found:       ok,
	}

	if !ok {
		result.Error = "no cached timetable for train " + trainNumber + " on " + dateStrResolved
		return c.JSON(http.StatusOK, result)
	}

	// es-base and es return the full trip (all stops are relevant).
	// Realtime sources return the raw stops from rawStopsCache, with a
	// Relevant flag indicating whether each stop was actually used in the
	// merged timetable (matched the country + name filter).
	if isESBase || isES {
		result.Stops = stopsToDebugStops(trip.Stops, true)
	} else {
		rawStops, hasRaw := a.rawStopsCache[service]
		if !hasRaw {
			// Fall back to filtering the merged trip if no raw cache exists.
			filtered := make([]traindata.Stop, 0, len(trip.Stops))
			for _, s := range trip.Stops {
				for _, d := range s.DataSources {
					if d == ds {
						filtered = append(filtered, s)
						break
					}
				}
			}
			result.Stops = stopsToDebugStops(filtered, true)
		} else {
			raw, ok := rawStops[source]
			if !ok {
				result.Stops = []DebugStop{}
				// Serve the cached fetch error (if any) so the debug
				// interface can show why no stops are available.
				if errs, hasErrs := a.rawStopsErrorCache[service]; hasErrs {
					if errMsg, hasErr := errs[source]; hasErr {
						result.Error = errMsg
					}
				}
			} else {
				result.Stops = rawStopsToDebugStops(raw, trip, ds)
			}
		}
	}

	return c.JSON(http.StatusOK, result)
}

// rawStopsToDebugStops converts raw stops from a data source into DebugStops,
// computing the Relevant flag by checking whether each raw stop matches a
// trip stop whose DataSources include the given source.
func rawStopsToDebugStops(raw []traindata.Stop, trip *traindata.Trip, ds traindata.DataSource) []DebugStop {
	// Build a set of station names that were successfully matched for this
	// source in the merged trip.
	relevantNames := make(map[string]bool)
	for _, ts := range trip.Stops {
		for _, d := range ts.DataSources {
			if d == ds {
				relevantNames[normalizeStationName(ts.StationName)] = true
				break
			}
		}
	}

	out := make([]DebugStop, 0, len(raw))
	for _, s := range raw {
		relevant := relevantNames[normalizeStationName(s.StationName)]
		out = append(out, debugStopFromStop(s, relevant))
	}
	return out
}

// stopsToDebugStops converts trip stops into the generic DebugStop JSON form.
// All stops are marked as relevant (they are part of the trip).
func stopsToDebugStops(stops []traindata.Stop, relevant bool) []DebugStop {
	out := make([]DebugStop, 0, len(stops))
	for _, s := range stops {
		out = append(out, debugStopFromStop(s, relevant))
	}
	return out
}

// debugStopFromStop converts a single traindata.Stop to a DebugStop with the
// given Relevant flag.
func debugStopFromStop(s traindata.Stop, relevant bool) DebugStop {
	return DebugStop{
		StationName:         s.StationName,
		StationUIC:          s.StationUIC,
		ArrivalTime:         formatDebugTime(s.ArrivalTime),
		DepartureTime:       formatDebugTime(s.DepartureTime),
		Platform:            s.Platform,
		RealPlatform:        s.RealPlatform,
		RealArrivalTime:     formatDebugTime(s.RealArrivalTime),
		RealDepartureTime:   formatDebugTime(s.RealDepartureTime),
		IsRealTime:          s.IsRealTime,
		Cancelled:           s.Cancelled,
		Relevant:            relevant,
		DataSources:         traindata.DataSourcesToStrings(s.DataSources),
		PrefferedDataSource: s.PrefferedDataSource.String(),
	}
}

func formatDebugTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// normalizeStationName lowercases and strips to alphanumerics for matching.
// This mirrors the logic in pkg/europeansleeper used to match stops by name.
func normalizeStationName(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}
