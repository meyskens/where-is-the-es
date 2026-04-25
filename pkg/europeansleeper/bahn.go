package europeansleeper

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/meyskens/where-is-the-es/pkg/bahn"
	"github.com/meyskens/where-is-the-es/pkg/traindata"
)

// EnhanceWithDB fills realtime arrival/departure information on trip stops
// whose UIC code is in the German number range (UIC country prefix 80) using
// the Deutsche Bahn RIS-Journeys API. It is a no-op when the trip has no
// German stops.
func EnhanceWithDB(ctx context.Context, client *bahn.Client, trip *traindata.Trip) (int, error) {
	if client == nil || trip == nil {
		return 0, nil
	}

	hasDE := false
	germanStops := 0
	for _, s := range trip.Stops {
		if isGermanUIC(s.StationUIC) {
			hasDE = true
			germanStops++
		}
	}
	if !hasDE {
		return 0, nil
	}

	stops, err := client.GetStops(ctx, trip.TrainNumber, trip.Date)
	if err != nil {
		// Overnight trains may be indexed by DB under a neighbouring
		// calendar day (the day boarding starts vs. the day they cross
		// into Germany). On a 406 ("no journey found") retry with +1
		// then -1 day before giving up.
		var httpErr *bahn.HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotAcceptable {
			for _, offset := range []int{1, -1} {
				retryDate := trip.Date.AddDate(0, 0, offset)
				log.Println("No journey found for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"), "- retrying with", retryDate.Format("2006-01-02"))
				stops, err = client.GetStops(ctx, trip.TrainNumber, retryDate)
				if err == nil {
					break
				}
				log.Println("Retry failed for train", trip.TrainNumber, "on date", retryDate.Format("2006-01-02"), ":", err)
			}
		}
		if err != nil {
			return 0, err
		}
	}

	type dbEntry struct {
		normalized string
		stop       bahn.Stop
	}
	dbByName := make(map[string]bahn.Stop, len(stops))
	dbEntries := make([]dbEntry, 0, len(stops))
	for _, s := range stops {
		if s.Name == "" {
			continue
		}
		norm := normalizeStationName(s.Name)
		if norm == "" {
			continue
		}
		if _, ok := dbByName[norm]; !ok {
			dbByName[norm] = s
		}
		dbEntries = append(dbEntries, dbEntry{normalized: norm, stop: s})
	}

	enrichedStops := 0
	for i := range trip.Stops {
		if !isGermanUIC(trip.Stops[i].StationUIC) {
			continue
		}

		key := normalizeStationName(trip.Stops[i].StationName)
		s, ok := dbByName[key]
		if !ok {
			bestScore := 0.0
			bestName := ""
			for _, e := range dbEntries {
				score := stringSimilarity(key, e.normalized)
				if score > bestScore {
					bestScore = score
					bestName = e.stop.Name
					s = e.stop
				}
			}
			if bestScore < 0.6 {
				log.Println("No DB name match for trip stop", trip.Stops[i].StationName, "(UIC", trip.Stops[i].StationUIC, ") on train", trip.TrainNumber, "best candidate:", bestName, "score:", bestScore)
				continue
			}
		}

		if !s.ArrivalRealtime.IsZero() {
			trip.Stops[i].RealArrivalTime = s.ArrivalRealtime
		} else if !trip.Stops[i].ArrivalTime.IsZero() {
			trip.Stops[i].RealArrivalTime = trip.Stops[i].ArrivalTime
		}
		if !s.Arrival.IsZero() {
			trip.Stops[i].ArrivalTime = s.Arrival
		}
		if !s.DepartureRealtime.IsZero() {
			trip.Stops[i].RealDepartureTime = s.DepartureRealtime
		} else if !trip.Stops[i].DepartureTime.IsZero() {
			trip.Stops[i].RealDepartureTime = trip.Stops[i].DepartureTime
		}
		if !s.Departure.IsZero() {
			trip.Stops[i].DepartureTime = s.Departure
		}
		if s.Platform != "" {
			trip.Stops[i].Platform = s.Platform
		}
		if s.PlatformRealtime != "" {
			trip.Stops[i].RealPlatform = s.PlatformRealtime
		} else if trip.Stops[i].RealPlatform == "" {
			trip.Stops[i].RealPlatform = trip.Stops[i].Platform
		}
		if s.Cancelled {
			trip.Stops[i].Cancelled = true
		}
		trip.Stops[i].IsRealTime = true
		trip.Stops[i].DataSources = appendDataSource(trip.Stops[i].DataSources, traindata.DataSourceDB)
		trip.Stops[i].PrefferedDataSource = traindata.DataSourceDB
		enrichedStops++
	}

	return enrichedStops, nil
}

// isGermanUIC reports whether a UIC station code belongs to Germany
// (country code prefix 80, i.e. 80xxxxx).
func isGermanUIC(uic int) bool {
	return uic >= 8000000 && uic < 8100000
}

func appendDataSource(sources []traindata.DataSource, src traindata.DataSource) []traindata.DataSource {
	for _, s := range sources {
		if s == src {
			return sources
		}
	}
	return append(sources, src)
}
