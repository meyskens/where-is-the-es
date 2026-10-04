package europeansleeper

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/meyskens/where-is-the-es/pkg/sbbinternal"
	"github.com/meyskens/where-is-the-es/pkg/traindata"
)

// EnhanceWithSBBInternal fills realtime arrival/departure information on
// trip stops using the SBB internal journey API.
//
// This data source has PRIORITY over the public SBB enhancer.
func EnhanceWithSBBInternal(ctx context.Context, client *sbbinternal.Client, trip *traindata.Trip) (int, []traindata.Stop, error) {
	if client == nil || trip == nil {
		return 0, nil, nil
	}

	// Count Swiss stops. The internal API returns the full journey, but
	// we only enrich Swiss stops (UIC 85xxxxx) to match the public SBB
	// enhancer's scope.
	swissStops := 0
	for i := range trip.Stops {
		if isSwissUIC(trip.Stops[i].StationUIC) {
			swissStops++
		}
	}
	if swissStops == 0 {
		return 0, nil, nil
	}

	log.Println("Enhancing trip with SBB Internal for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"), "- Swiss stops:", swissStops)

	sbbTrip, err := client.GetTrainByNumber(ctx, trip.TrainNumber, trip.Date)
	if err != nil {
		if errors.Is(err, sbbinternal.ErrNotFound) {
			log.Println("SBB Internal: no matching train for", trip.TrainNumber)
			return 0, nil, fmt.Errorf("%w: no matching train for %s", sbbinternal.ErrNotFound, trip.TrainNumber)
		}
		log.Println("Failed to fetch SBB Internal timetable for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"), ":", err)
		return 0, nil, err
	}

	log.Println("Fetched", len(sbbTrip.Stops), "SBB Internal stops for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"))

	// Build lookup maps by UIC and normalized name for matching.
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
					log.Println("No SBB Internal name match for trip stop", trip.Stops[i].StationName, "(UIC", trip.Stops[i].StationUIC, ") on train", trip.TrainNumber, "best candidate:", bestName, "score:", bestScore)
					continue
				}
			}
		}

		// SBB Internal has priority over SBB public. Overwrite the
		// realtime fields regardless of whether SBB public already set
		// them.
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
		trip.Stops[i].DataSources = appendDataSource(trip.Stops[i].DataSources, traindata.DataSourceSBBInternal)
		// SBB Internal has priority over SBB public: set as preferred.
		trip.Stops[i].PrefferedDataSource = traindata.DataSourceSBBInternal
		enrichedStops++
	}

	log.Println("SBB Internal enrichment for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"), "- enriched", enrichedStops, "of", swissStops, "Swiss stops")

	return enrichedStops, sbbTrip.Stops, nil
}
