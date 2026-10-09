package europeansleeper

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"

	"github.com/meyskens/where-is-the-es/pkg/sbbinternal"
	"github.com/meyskens/where-is-the-es/pkg/traindata"
)

const (
	sbbInternalStationNameSimilarityThreshold = 0.95
	sbbInternalStationMatchRateThreshold      = 0.95
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

	// Verify station names before using any of the SBB data as train number 400 is re-used
	matchedStops := make(map[int]traindata.Stop, swissStops)
	for i := range trip.Stops {
		if !isSwissUIC(trip.Stops[i].StationUIC) {
			continue
		}

		key := normalizeStationName(trip.Stops[i].StationName)
		bestScore := 0.0
		bestStop := traindata.Stop{}
		for _, candidate := range sbbTrip.Stops {
			score := stringSimilarity(key, normalizeStationName(candidate.StationName))
			if score > bestScore {
				bestScore = score
				bestStop = candidate
			}
		}
		if bestScore < sbbInternalStationNameSimilarityThreshold {
			log.Println("No SBB Internal name match for trip stop", trip.Stops[i].StationName, "(UIC", trip.Stops[i].StationUIC, ") on train", trip.TrainNumber, "best candidate:", bestStop.StationName, "score:", bestScore)
			continue
		}
		matchedStops[i] = bestStop
	}

	minimumMatches := int(math.Ceil(float64(swissStops) * sbbInternalStationMatchRateThreshold))
	if len(matchedStops) < minimumMatches {
		return 0, sbbTrip.Stops, fmt.Errorf("SBB Internal station-name match below 95%%: matched %d of %d Swiss stops", len(matchedStops), swissStops)
	}

	enrichedStops := 0
	for i := range trip.Stops {
		s, ok := matchedStops[i]
		if !ok {
			continue
		}

		// SBB Internal has priority over SBB public. Overwrite the
		if s.IsRealTime && !s.RealArrivalTime.IsZero() {
			trip.Stops[i].RealArrivalTime = s.RealArrivalTime
		} else if !trip.Stops[i].ArrivalTime.IsZero() {
			trip.Stops[i].RealArrivalTime = trip.Stops[i].ArrivalTime
		}
		if s.IsRealTime && !s.RealDepartureTime.IsZero() {
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
		trip.Stops[i].DataSources = appendDataSource(trip.Stops[i].DataSources, traindata.DataSourceSBBInternal)
		// SBB Internal has priority over SBB public: set as preferred.
		trip.Stops[i].PrefferedDataSource = traindata.DataSourceSBBInternal
		enrichedStops++
	}

	log.Println("SBB Internal enrichment for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"), "- enriched", enrichedStops, "of", swissStops, "Swiss stops")

	return enrichedStops, sbbTrip.Stops, nil
}
