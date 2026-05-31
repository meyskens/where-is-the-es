package europeansleeper

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/meyskens/where-is-the-es/pkg/sncfgc"
	"github.com/meyskens/where-is-the-es/pkg/traindata"
)

// EnhanceWithSNCFGC fills realtime arrival/departure information on trip stops
// whose UIC code is in the French number range (UIC country prefix 87) using
// the SNCF Gares & Connexions API. It is a no-op when the trip has no French stops
// or no SNCF GC client is configured.
func EnhanceWithSNCFGC(ctx context.Context, client *sncfgc.Client, trip *traindata.Trip) (int, error) {
	if client == nil || trip == nil {
		return 0, nil
	}

	hasFR := false
	frenchStops := 0
	var firstFrenchStopUIC string
	var firstFrenchStopName string
	for _, s := range trip.Stops {
		if isFrenchUIC(s.StationUIC) {
			hasFR = true
			frenchStops++
			if firstFrenchStopUIC == "" {
				firstFrenchStopUIC = formatUIC(s.StationUIC)
				firstFrenchStopName = s.StationName
				log.Printf("DEBUG: Found French stop: name=%s, raw UIC=%d, formatted UIC=%s", s.StationName, s.StationUIC, firstFrenchStopUIC)
			}
		}
	}
	if !hasFR {
		return 0, nil
	}

	log.Println("Enhancing trip with SNCF GC for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"), "- French stops:", frenchStops)

	// Determine if this is a departure or arrival based on the first French stop
	// If it's the first stop overall, it's a departure
	isDeparture := false
	if len(trip.Stops) > 0 && isFrenchUIC(trip.Stops[0].StationUIC) {
		isDeparture = true
	}

	log.Printf("DEBUG: About to call SNCF GC API: train=%s, uic=%s (stop=%s), date=%s, isDeparture=%v",
		trip.TrainNumber, firstFrenchStopUIC, firstFrenchStopName, trip.Date.Format("2006-01-02"), isDeparture)

	sncfgcTrip, err := client.GetTimetable(ctx, trip.TrainNumber, firstFrenchStopUIC, trip.Date, isDeparture)
	if err != nil {
		if errors.Is(err, sncfgc.ErrNotFound) {
			log.Println("SNCF GC: no matching train for", trip.TrainNumber, "at UIC", firstFrenchStopUIC)
			return 0, nil
		}
		// Log the full error for debugging
		log.Printf("Failed to fetch SNCF GC timetable for train %s at UIC %s on date %s: %v",
			trip.TrainNumber, firstFrenchStopUIC, trip.Date.Format("2006-01-02"), err)
		return 0, err
	}

	log.Println("Fetched", len(sncfgcTrip.Stops), "SNCF GC stops for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"))

	sncfgcByUIC := make(map[int]traindata.Stop, len(sncfgcTrip.Stops))
	sncfgcByName := make(map[string]traindata.Stop, len(sncfgcTrip.Stops))
	sncfgcEntries := make([]struct {
		normalized string
		stop       traindata.Stop
	}, 0, len(sncfgcTrip.Stops))
	for _, s := range sncfgcTrip.Stops {
		if s.StationUIC != 0 {
			sncfgcByUIC[s.StationUIC] = s
		}
		if s.StationName != "" {
			norm := normalizeStationName(s.StationName)
			if norm != "" {
				if _, ok := sncfgcByName[norm]; !ok {
					sncfgcByName[norm] = s
				}
				sncfgcEntries = append(sncfgcEntries, struct {
					normalized string
					stop       traindata.Stop
				}{normalized: norm, stop: s})
			}
		}
	}

	enrichedStops := 0
	for i := range trip.Stops {
		if !isFrenchUIC(trip.Stops[i].StationUIC) {
			continue
		}

		var s traindata.Stop
		var ok bool

		// Prefer matching by UIC code.
		if trip.Stops[i].StationUIC != 0 {
			s, ok = sncfgcByUIC[trip.Stops[i].StationUIC]
		}
		if !ok {
			key := normalizeStationName(trip.Stops[i].StationName)
			s, ok = sncfgcByName[key]
			if !ok {
				bestScore := 0.0
				bestName := ""
				for _, e := range sncfgcEntries {
					score := stringSimilarity(key, e.normalized)
					if score > bestScore {
						bestScore = score
						bestName = e.stop.StationName
						s = e.stop
					}
				}
				if bestScore < 0.6 {
					log.Println("No SNCF GC name match for trip stop", trip.Stops[i].StationName, "(UIC", trip.Stops[i].StationUIC, ") on train", trip.TrainNumber, "best candidate:", bestName, "score:", bestScore)
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
		trip.Stops[i].DataSources = appendDataSource(trip.Stops[i].DataSources, traindata.DataSourceSNCFGC)
		trip.Stops[i].PrefferedDataSource = traindata.DataSourceSNCFGC
		enrichedStops++
	}

	log.Println("SNCF GC enrichment for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"), "- enriched", enrichedStops, "of", frenchStops, "French stops")

	return enrichedStops, nil
}

// isFrenchUIC reports whether a UIC station code belongs to France
// (country code prefix 87, i.e. 87xxxxx).
func isFrenchUIC(uic int) bool {
	return uic >= 8700000 && uic < 8800000
}

// formatUIC formats a UIC code as a 10-digit string with leading zeros
// as expected by the SNCF API (e.g., 87271007 -> "0087271007").
func formatUIC(uic int) string {
	// UIC codes for France should be 8 digits starting with 87
	uicStr := fmt.Sprintf("%d", uic)

	// If UIC is 7 digits and starts with 87, it might be missing the last digit SNCF somehow added
	// Paris Gare du Nord: 8727100 should be 87271007
	if len(uicStr) == 7 && strings.HasPrefix(uicStr, "87") {
		// Try to fix common UIC codes SNCF broke
		switch uicStr {
		case "8727100":
			uicStr = "87271007" // Paris Gare du Nord
		case "8729560": // Aulnoye-Aymeries
			uicStr = "87295600" // Example for a generic French station
		default:
			// For other 7-digit codes, append a 0 as a guess
			uicStr = uicStr + "0"
		}
	}

	// Pad to 10 digits with leading zeros
	return fmt.Sprintf("%010s", uicStr)
}
