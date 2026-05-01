package europeansleeper

import (
	"context"
	"errors"
	"log"

	"github.com/meyskens/where-is-the-es/pkg/grapper"
	"github.com/meyskens/where-is-the-es/pkg/traindata"
)

// EnhanceWithGrapper fills realtime arrival/departure information on trip stops
// whose UIC code is in the Czech number range (UIC country prefix 54) using
// a private GRAPP to JSON instance. It is a no-op when the trip has no Czech
// stops or no grapper client is configured.
func EnhanceWithGrapper(ctx context.Context, client *grapper.Client, trip *traindata.Trip) (int, error) {
	if client == nil || trip == nil {
		return 0, nil
	}

	hasCZ := false
	czechStops := 0
	for _, s := range trip.Stops {
		if isCzechUIC(s.StationUIC) {
			hasCZ = true
			czechStops++
		}
	}
	if !hasCZ {
		return 0, nil
	}

	log.Println("Enhancing trip with Grapper for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"), "- Czech stops:", czechStops)

	grapperTrip, err := client.GetTimetable(ctx, trip.TrainNumber, trip.Date)
	if err != nil {
		if errors.Is(err, grapper.ErrNotFound) || errors.Is(err, grapper.ErrTitleMismatch) {
			log.Println("Grapper: no matching train for", trip.TrainNumber, ":", err)
			return 0, nil
		}
		log.Println("Failed to fetch Grapper timetable for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"), ":", err)
		return 0, err
	}

	log.Println("Fetched", len(grapperTrip.Stops), "Grapper stops for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"))

	grapperByName := make(map[string]traindata.Stop, len(grapperTrip.Stops))
	grapperEntries := make([]struct {
		normalized string
		stop       traindata.Stop
	}, 0, len(grapperTrip.Stops))
	for _, s := range grapperTrip.Stops {
		if s.StationName == "" {
			continue
		}
		norm := normalizeStationName(s.StationName)
		if norm == "" {
			continue
		}
		if _, ok := grapperByName[norm]; !ok {
			grapperByName[norm] = s
		}
		grapperEntries = append(grapperEntries, struct {
			normalized string
			stop       traindata.Stop
		}{normalized: norm, stop: s})
	}

	enrichedStops := 0
	for i := range trip.Stops {
		if !isCzechUIC(trip.Stops[i].StationUIC) {
			continue
		}

		key := normalizeStationName(trip.Stops[i].StationName)
		s, ok := grapperByName[key]
		if !ok {
			bestScore := 0.0
			bestName := ""
			for _, e := range grapperEntries {
				score := stringSimilarity(key, e.normalized)
				if score > bestScore {
					bestScore = score
					bestName = e.stop.StationName
					s = e.stop
				}
			}
			if bestScore < 0.6 {
				log.Println("No Grapper name match for trip stop", trip.Stops[i].StationName, "(UIC", trip.Stops[i].StationUIC, ") on train", trip.TrainNumber, "best candidate:", bestName, "score:", bestScore)
				continue
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
		if s.ArrivalTime.IsZero() && s.DepartureTime.IsZero() {
			// Grapper has no planned times, skip planned time updates
		} else {
			if !s.ArrivalTime.IsZero() {
				trip.Stops[i].ArrivalTime = s.ArrivalTime
			}
			if !s.DepartureTime.IsZero() {
				trip.Stops[i].DepartureTime = s.DepartureTime
			}
		}

		trip.Stops[i].IsRealTime = true
		trip.Stops[i].DataSources = appendDataSource(trip.Stops[i].DataSources, traindata.DataSourceSZ)
		trip.Stops[i].PrefferedDataSource = traindata.DataSourceSZ
		enrichedStops++
	}

	log.Println("Grapper enrichment for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"), "- enriched", enrichedStops, "of", czechStops, "Czech stops")

	return enrichedStops, nil
}

// isCzechUIC reports whether a UIC station code belongs to the Czech Republic
// (country code prefix 54, i.e. 54xxxxx).
func isCzechUIC(uic int) bool {
	return uic >= 5400000 && uic < 5500000
}
