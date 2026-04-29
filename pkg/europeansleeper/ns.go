package europeansleeper

import (
	"context"
	"log"

	"github.com/meyskens/where-is-the-es/pkg/ns"
	"github.com/meyskens/where-is-the-es/pkg/traindata"
)

// EnhanceWithNS fills realtime arrival/departure information on trip stops
// whose UIC code is in the Dutch number range (UIC country prefix 84) using
// the NS Reisinformatie API. It is a no-op when the trip has no Dutch stops
// or no NS client is configured.
func EnhanceWithNS(ctx context.Context, client *ns.Client, trip *traindata.Trip) (int, error) {
	if client == nil || trip == nil {
		return 0, nil
	}

	hasNL := false
	dutchStops := 0
	for _, s := range trip.Stops {
		if isDutchUIC(s.StationUIC) {
			hasNL = true
			dutchStops++
		}
	}
	if !hasNL {
		return 0, nil
	}

	log.Println("Enhancing trip with NS for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"), "- Dutch stops:", dutchStops)

	nsTrip, err := client.GetTimetable(ctx, trip.TrainNumber, trip.Date)
	if err != nil {
		log.Println("Failed to fetch NS timetable for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"), ":", err)
		return 0, err
	}

	log.Println("Fetched", len(nsTrip.Stops), "NS stops for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"))

	nsByUIC := make(map[int]traindata.Stop, len(nsTrip.Stops))
	nsByName := make(map[string]traindata.Stop, len(nsTrip.Stops))
	nsEntries := make([]struct {
		normalized string
		stop       traindata.Stop
	}, 0, len(nsTrip.Stops))
	for _, s := range nsTrip.Stops {
		if s.StationUIC != 0 {
			nsByUIC[s.StationUIC] = s
		}
		if s.StationName != "" {
			norm := normalizeStationName(s.StationName)
			if norm != "" {
				if _, ok := nsByName[norm]; !ok {
					nsByName[norm] = s
				}
				nsEntries = append(nsEntries, struct {
					normalized string
					stop       traindata.Stop
				}{normalized: norm, stop: s})
			}
		}
	}

	enrichedStops := 0
	for i := range trip.Stops {
		if !isDutchUIC(trip.Stops[i].StationUIC) {
			continue
		}

		var s traindata.Stop
		var ok bool

		// Prefer matching by UIC code.
		if trip.Stops[i].StationUIC != 0 {
			s, ok = nsByUIC[trip.Stops[i].StationUIC]
		}
		if !ok {
			key := normalizeStationName(trip.Stops[i].StationName)
			s, ok = nsByName[key]
			if !ok {
				bestScore := 0.0
				bestName := ""
				for _, e := range nsEntries {
					score := stringSimilarity(key, e.normalized)
					if score > bestScore {
						bestScore = score
						bestName = e.stop.StationName
						s = e.stop
					}
				}
				if bestScore < 0.6 {
					log.Println("No NS name match for trip stop", trip.Stops[i].StationName, "(UIC", trip.Stops[i].StationUIC, ") on train", trip.TrainNumber, "best candidate:", bestName, "score:", bestScore)
					continue
				}
			}
		}

		if !s.RealArrivalTime.IsZero() {
			trip.Stops[i].RealArrivalTime = s.RealArrivalTime
		} else if !trip.Stops[i].ArrivalTime.IsZero() {
			trip.Stops[i].RealArrivalTime = trip.Stops[i].ArrivalTime
		}
		if !s.RealDepartureTime.IsZero() {
			trip.Stops[i].RealDepartureTime = s.RealDepartureTime
		} else if !trip.Stops[i].DepartureTime.IsZero() {
			trip.Stops[i].RealDepartureTime = trip.Stops[i].DepartureTime
		}
		if s.RealPlatform != "" {
			trip.Stops[i].RealPlatform = s.RealPlatform
		} else if trip.Stops[i].RealPlatform == "" {
			trip.Stops[i].RealPlatform = trip.Stops[i].Platform
		}
		if s.Cancelled {
			trip.Stops[i].Cancelled = true
		}
		trip.Stops[i].IsRealTime = true
		trip.Stops[i].DataSources = appendDataSource(trip.Stops[i].DataSources, traindata.DataSourceNS)
		trip.Stops[i].PrefferedDataSource = traindata.DataSourceNS
		enrichedStops++
	}

	log.Println("NS enrichment for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"), "- enriched", enrichedStops, "of", dutchStops, "Dutch stops")

	return enrichedStops, nil
}

// isDutchUIC reports whether a UIC station code belongs to the Netherlands
// (country code prefix 84, i.e. 84xxxxx).
func isDutchUIC(uic int) bool {
	return uic >= 8400000 && uic < 8500000
}
