package europeansleeper

import (
	"errors"
	"log"
	"time"

	"github.com/meyskens/where-is-the-es/pkg/nmbs"
	"github.com/meyskens/where-is-the-es/pkg/traindata"
)

// EnhanceWithNMBS fills realtime arrival/departure information on trip stops
// whose UIC code is in the Belgian number range (UIC country prefix 88) using
// the SNCB/NMBS delay-certificate scrape. It is a no-op when the trip has no
// Belgian stops or no NMBSFetcher is configured.
func EnhanceWithNMBS(fetcher *nmbs.NMBSFetcher, trip *traindata.Trip) (int, error) {
	if fetcher == nil || trip == nil {
		return 0, nil
	}

	hasBE := false
	belgianStops := 0
	for _, s := range trip.Stops {
		if isBelgianUIC(s.StationUIC) {
			hasBE = true
			belgianStops++
		}
	}
	if !hasBE {
		return 0, nil
	}

	log.Println("Enhancing trip with NMBS for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"), "- Belgian stops:", belgianStops)

	stops, err := fetcher.FetchTimetable(trip.TrainNumber, trip.Date)
	if err != nil {
		// Overnight ES trains may be indexed by NMBS under the calendar
		// day they arrive in Belgium (return leg) rather than the day
		// they depart. On a "no train" response retry with date+1.
		if errors.Is(err, nmbs.ErrNoTrain) {
			retryDate := trip.Date.AddDate(0, 0, 1)
			log.Println("NMBS: no train for", trip.TrainNumber, "on", trip.Date.Format("2006-01-02"), "- retrying with", retryDate.Format("2006-01-02"))
			stops, err = fetcher.FetchTimetable(trip.TrainNumber, retryDate)
		}
		if err != nil {
			log.Println("Failed to fetch NMBS timetable for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"), ":", err)
			return 0, err
		}
	}

	log.Println("Fetched", len(stops), "NMBS stops for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"))

	type nmbsEntry struct {
		normalized string
		stop       traindata.Stop
	}
	nmbsByName := make(map[string]traindata.Stop, len(stops))
	nmbsEntries := make([]nmbsEntry, 0, len(stops))
	for _, s := range stops {
		if s.StationName == "" {
			continue
		}
		norm := normalizeStationName(s.StationName)
		if norm == "" {
			continue
		}
		if _, ok := nmbsByName[norm]; !ok {
			nmbsByName[norm] = s
		}
		nmbsEntries = append(nmbsEntries, nmbsEntry{normalized: norm, stop: s})
	}

	enrichedStops := 0
	for i := range trip.Stops {
		if !isBelgianUIC(trip.Stops[i].StationUIC) {
			continue
		}

		key := normalizeStationName(trip.Stops[i].StationName)
		s, ok := nmbsByName[key]
		if !ok {
			bestScore := 0.0
			bestName := ""
			for _, e := range nmbsEntries {
				score := stringSimilarity(key, e.normalized)
				if score > bestScore {
					bestScore = score
					bestName = e.stop.StationName
					s = e.stop
				}
			}
			if bestScore < 0.6 {
				log.Println("No NMBS name match for trip stop", trip.Stops[i].StationName, "(UIC", trip.Stops[i].StationUIC, ") on train", trip.TrainNumber, "best candidate:", bestName, "score:", bestScore)
				continue
			}
		}

		// NMBS times are parsed as time-of-day only (year 0001-01-01 UTC).
		// Anchor them to the date of the trip stop's planned arrival /
		// departure time so we end up with a real instant.
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
		trip.Stops[i].DataSources = appendDataSource(trip.Stops[i].DataSources, traindata.DataSourceNMBS)
		trip.Stops[i].PrefferedDataSource = traindata.DataSourceNMBS
		enrichedStops++
	}

	log.Println("NMBS enrichment for train", trip.TrainNumber, "on date", trip.Date.Format("2006-01-02"), "- enriched", enrichedStops, "of", belgianStops, "Belgian stops")

	return enrichedStops, nil
}

// isBelgianUIC reports whether a UIC station code belongs to Belgium
// (country code prefix 88, i.e. 88xxxxx).
func isBelgianUIC(uic int) bool {
	return uic >= 8800000 && uic < 8900000
}

// combineDateTime takes the date components from anchor and the time
// components (and location, if anchor has none) from t. If anchor is zero
// it returns t unchanged.
func combineDateTime(anchor, t time.Time) time.Time {
	if anchor.IsZero() {
		return t
	}
	loc := anchor.Location()
	return time.Date(
		anchor.Year(), anchor.Month(), anchor.Day(),
		t.Hour(), t.Minute(), t.Second(), 0,
		loc,
	)
}
