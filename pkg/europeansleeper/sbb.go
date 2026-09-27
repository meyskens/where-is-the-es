package europeansleeper

import (
	"context"
	"errors"
	"log"
	"strconv"

	"github.com/meyskens/where-is-the-es/pkg/sbb"
	"github.com/meyskens/where-is-the-es/pkg/traindata"
)

// EnhanceWithSBB fills realtime arrival/departure information on trip stops
// whose UIC code is in the Swiss number range (UIC country prefix 85, i.e.
// 85xxxxx) using the SBB public GraphQL trip planner.
//
// Because SBB does not expose a "look up train by number" endpoint, the
// enhancer runs a trip search from the journey's overall origin to its
// overall destination (NOT just the Swiss section). The SBB planner only
// finds through-trains when the search endpoints are stops the train
// actually runs between; searching only the Swiss sub-section fails because
// the train continues beyond Switzerland. Once the matching leg is found its
// full stop list is fetched via serviceJourneyById and only the Swiss stops
// are used for enrichment.
//
// It is a no-op when the trip has no Swiss stops or no SBB client is
// configured. It returns the number of enriched stops, the raw stops fetched
// from SBB, and any error.
func EnhanceWithSBB(ctx context.Context, client *sbb.Client, trip *traindata.Trip) (int, []traindata.Stop, error) {
	if client == nil || trip == nil {
		return 0, nil, nil
	}

	// Count Swiss stops and remember the first one (used as the search
	// anchor time). We do NOT restrict the search to the Swiss section.
	swissStops := 0
	var firstSwiss *traindata.Stop
	for i := range trip.Stops {
		if isSwissUIC(trip.Stops[i].StationUIC) {
			swissStops++
			if firstSwiss == nil {
				firstSwiss = &trip.Stops[i]
			}
		}
	}
	if swissStops == 0 {
		return 0, nil, nil
	}
	// SBB needs two endpoints to form a trip search; if the journey has
	// only one stop overall there is nothing to search for.
	if len(trip.Stops) < 2 {
		return 0, nil, nil
	}

	log.Println("Enhancing trip with SBB for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"), "- Swiss stops:", swissStops)

	// Search the full journey: first stop -> last stop. SBB will find the
	// through-train (e.g. EN 400 Milano -> Bruxelles) even though we only
	// enrich the Swiss stops from the result.
	origin := &trip.Stops[0]
	destination := &trip.Stops[len(trip.Stops)-1]

	// Anchor the SBB trip search on the train's overall departure time
	// (when it leaves its origin), NOT when it reaches Switzerland. For
	// overnight trains like EN 401 (Bruxelles 19:06 -> Milano 11:40 next
	// day) the train left its origin ~11h before it enters Switzerland;
	// anchoring on the first Swiss stop misses the actual departure
	// window and the SBB planner finds nothing. Fall back to the first
	// Swiss stop, then the trip date, when the origin has no scheduled
	// departure time.
	departure := trip.Date
	if !origin.DepartureTime.IsZero() {
		departure = origin.DepartureTime
	} else if !origin.ArrivalTime.IsZero() {
		departure = origin.ArrivalTime
	} else if firstSwiss != nil {
		if !firstSwiss.DepartureTime.IsZero() {
			departure = firstSwiss.DepartureTime
		} else if !firstSwiss.ArrivalTime.IsZero() {
			departure = firstSwiss.ArrivalTime
		}
	}

	sbbTrip, err := client.GetTimetable(ctx, trip.TrainNumber, strconv.Itoa(origin.StationUIC), strconv.Itoa(destination.StationUIC), departure)
	if err != nil {
		if errors.Is(err, sbb.ErrNotFound) {
			log.Println("SBB: no matching train for", trip.TrainNumber, "between", origin.StationName, "and", destination.StationName)
			return 0, nil, nil
		}
		log.Println("Failed to fetch SBB timetable for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"), ":", err)
		return 0, nil, err
	}

	log.Println("Fetched", len(sbbTrip.Stops), "SBB stops for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"))

	sbbByUIC := make(map[int]traindata.Stop, len(sbbTrip.Stops))
	sbbByName := make(map[string]traindata.Stop, len(sbbTrip.Stops))
	sbbEntries := make([]struct {
		normalized string
		stop       traindata.Stop
	}, 0, len(sbbTrip.Stops))
	for _, s := range sbbTrip.Stops {
		if s.StationUIC != 0 {
			sbbByUIC[s.StationUIC] = s
		}
		if s.StationName != "" {
			norm := normalizeStationName(s.StationName)
			if norm != "" {
				if _, ok := sbbByName[norm]; !ok {
					sbbByName[norm] = s
				}
				sbbEntries = append(sbbEntries, struct {
					normalized string
					stop       traindata.Stop
				}{normalized: norm, stop: s})
			}
		}
	}

	enrichedStops := 0
	for i := range trip.Stops {
		if !isSwissUIC(trip.Stops[i].StationUIC) {
			continue
		}

		var s traindata.Stop
		var ok bool

		// Prefer matching by UIC code.
		if trip.Stops[i].StationUIC != 0 {
			s, ok = sbbByUIC[trip.Stops[i].StationUIC]
		}
		if !ok {
			key := normalizeStationName(trip.Stops[i].StationName)
			s, ok = sbbByName[key]
			if !ok {
				bestScore := 0.0
				bestName := ""
				for _, e := range sbbEntries {
					score := stringSimilarity(key, e.normalized)
					if score > bestScore {
						bestScore = score
						bestName = e.stop.StationName
						s = e.stop
					}
				}
				if bestScore < 0.6 {
					log.Println("No SBB name match for trip stop", trip.Stops[i].StationName, "(UIC", trip.Stops[i].StationUIC, ") on train", trip.TrainNumber, "best candidate:", bestName, "score:", bestScore)
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
		if !s.ArrivalTime.IsZero() {
			trip.Stops[i].ArrivalTime = s.ArrivalTime
		}
		if !s.DepartureTime.IsZero() {
			trip.Stops[i].DepartureTime = s.DepartureTime
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
		trip.Stops[i].DataSources = appendDataSource(trip.Stops[i].DataSources, traindata.DataSourceSBB)
		trip.Stops[i].PrefferedDataSource = traindata.DataSourceSBB
		enrichedStops++
	}

	log.Println("SBB enrichment for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"), "- enriched", enrichedStops, "of", swissStops, "Swiss stops")

	return enrichedStops, sbbTrip.Stops, nil
}

// isSwissUIC reports whether a UIC station code belongs to Switzerland
// (country code prefix 85, i.e. 85xxxxx).
func isSwissUIC(uic int) bool {
	return uic >= 8500000 && uic < 8600000
}
