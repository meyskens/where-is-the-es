package europeansleeper

import (
	"log"

	"github.com/meyskens/where-is-the-es/pkg/arenaways"
	"github.com/meyskens/where-is-the-es/pkg/traindata"
)

// EnhanceWithArenaways fills realtime arrival/departure information on
// trip stops whose UIC code is in the Italian number range (UIC country
// prefix 83) using the Arenaways train-status page. It is a no-op when
// the trip has no Italian stops or no ArenaWaysFetcher is configured.
func EnhanceWithArenaways(fetcher *arenaways.ArenaWaysFetcher, trip *traindata.Trip) (int, error) {
	if fetcher == nil || trip == nil {
		return 0, nil
	}

	hasIT := false
	italianStops := 0
	for _, s := range trip.Stops {
		if isItalianUIC(s.StationUIC) {
			hasIT = true
			italianStops++
		}
	}
	if !hasIT {
		return 0, nil
	}

	log.Println("Enhancing trip with Arenaways for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"), "- Italian stops:", italianStops)

	stops, err := fetcher.FetchTimetable(trip.TrainNumber)
	if err != nil {
		log.Println("Failed to fetch Arenaways timetable for train", trip.TrainNumber, ":", err)
		return 0, err
	}

	log.Println("Fetched", len(stops), "Arenaways stops for train", trip.TrainNumber)

	type arenaEntry struct {
		normalized string
		stop       traindata.Stop
	}
	arenaByName := make(map[string]traindata.Stop, len(stops))
	arenaEntries := make([]arenaEntry, 0, len(stops))
	for _, s := range stops {
		if s.StationName == "" {
			continue
		}
		norm := normalizeStationName(s.StationName)
		if norm == "" {
			continue
		}
		if _, ok := arenaByName[norm]; !ok {
			arenaByName[norm] = s
		}
		arenaEntries = append(arenaEntries, arenaEntry{normalized: norm, stop: s})
	}

	enrichedStops := 0
	for i := range trip.Stops {
		if !isItalianUIC(trip.Stops[i].StationUIC) {
			continue
		}

		key := normalizeStationName(trip.Stops[i].StationName)
		s, ok := arenaByName[key]
		if !ok {
			bestScore := 0.0
			bestName := ""
			for _, e := range arenaEntries {
				score := stringSimilarity(key, e.normalized)
				if score > bestScore {
					bestScore = score
					bestName = e.stop.StationName
					s = e.stop
				}
			}
			if bestScore < 0.6 {
				log.Println("No Arenaways name match for trip stop", trip.Stops[i].StationName, "(UIC", trip.Stops[i].StationUIC, ") on train", trip.TrainNumber, "best candidate:", bestName, "score:", bestScore)
				continue
			}
		}

		// Arenaways times are parsed as time-of-day only (year 0001-01-01
		// UTC). Anchor them to the date of the trip stop's planned time
		// so we end up with a real instant — same approach as NMBS.
		if !s.RealArrivalTime.IsZero() {
			trip.Stops[i].RealArrivalTime = combineDateTime(trip.Stops[i].ArrivalTime, s.RealArrivalTime)
		} else if !trip.Stops[i].ArrivalTime.IsZero() {
			trip.Stops[i].RealArrivalTime = trip.Stops[i].ArrivalTime
		}
		if !s.RealDepartureTime.IsZero() {
			trip.Stops[i].RealDepartureTime = combineDateTime(trip.Stops[i].DepartureTime, s.RealDepartureTime)
		} else if !trip.Stops[i].DepartureTime.IsZero() {
			trip.Stops[i].RealDepartureTime = trip.Stops[i].DepartureTime
		}

		trip.Stops[i].IsRealTime = true
		trip.Stops[i].Cancelled = s.Cancelled
		trip.Stops[i].DataSources = appendDataSource(trip.Stops[i].DataSources, traindata.DataSourceArenaways)
		trip.Stops[i].PrefferedDataSource = traindata.DataSourceArenaways
		enrichedStops++
	}

	log.Println("Arenaways enrichment for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"), "- enriched", enrichedStops, "of", italianStops, "Italian stops")

	return enrichedStops, nil
}

// isItalianUIC reports whether a UIC station code belongs to Italy
// (country code prefix 83, i.e. 83xxxxx).
func isItalianUIC(uic int) bool {
	return uic >= 8300000 && uic < 8400000
}
