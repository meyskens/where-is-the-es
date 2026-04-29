package v1

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/meyskens/where-is-the-es/pkg/bahn"
	"github.com/meyskens/where-is-the-es/pkg/europeansleeper"
	"github.com/meyskens/where-is-the-es/pkg/nmbs"
	"github.com/meyskens/where-is-the-es/pkg/ns"
	"github.com/meyskens/where-is-the-es/pkg/traindata"
)

type Service struct {
	TrainNumber string `json:"train_number"`
	Date        string `json:"date"`
}

type APIV1 struct {
	compositionCache map[string]traindata.Composition
	timetableCache   map[Service]*traindata.Trip

	refreshTimer *time.Ticker

	initDone bool

	tcURL       string
	bahnClient  *bahn.Client
	nsClient    *ns.Client
	nmbsFetcher *nmbs.NMBSFetcher
}

func New(tcURL, dbAPIKey, dbClientID, nsSubscriptionKey, flareSolverrURL string) *APIV1 {
	a := &APIV1{
		compositionCache: make(map[string]traindata.Composition),
		timetableCache:   make(map[Service]*traindata.Trip),
		refreshTimer:     time.NewTicker(1 * time.Minute),
		tcURL:            tcURL,
	}
	if dbAPIKey != "" && dbClientID != "" {
		a.bahnClient = bahn.NewClient(dbAPIKey, dbClientID)
	}
	if nsSubscriptionKey != "" {
		a.nsClient = ns.NewClient(nsSubscriptionKey)
	}
	if flareSolverrURL != "" {
		fetcher, err := nmbs.NewNMBSFetcher(flareSolverrURL)
		if err != nil {
			log.Println("Failed to initialise NMBS fetcher:", err)
		} else {
			a.nmbsFetcher = fetcher
		}
	}
	return a
}

func (a *APIV1) init() {
	if a.initDone {
		return
	}
	a.initDone = true

	// start the refresher
	go func() {
		for range a.refreshTimer.C {
			a.refresher()
		}
	}()
}

func (a *APIV1) Register(e *echo.Echo) {
	a.init()
	a.refreshCache()

	e.GET("/api/v1/composition/:number", func(c echo.Context) error {
		tc, ok := a.compositionCache[c.Param("number")]
		if !ok {
			return c.String(404, "Composition not found")
		}
		return c.JSON(200, tc.ToBrowser())
	})

	e.GET("/api/v1/timetable/:date/:number", func(c echo.Context) error {
		dateParam := c.Param("date")
		number := c.Param("number")

		var service Service

		if dateParam == "next" {
			// Find the next available departure for this train
			nextDate, found := a.findNextDeparture(number)
			if !found {
				return c.String(404, "No upcoming departures found for this train")
			}
			service = Service{
				TrainNumber: number,
				Date:        nextDate,
			}
		} else {
			service = Service{
				TrainNumber: number,
				Date:        dateParam,
			}
		}

		tt, ok := a.timetableCache[service]
		if !ok {
			return c.String(404, "Timetable not found")
		}
		return c.JSON(200, tt.ToBrowser())
	})
}

func (a *APIV1) refresher() {
	a.refreshCache()
}

func contains(s []time.Weekday, w time.Weekday) bool {
	for _, v := range s {
		if v == w {
			return true
		}
	}
	return false
}

func (a *APIV1) findNextDeparture(trainNumber string) (string, bool) {
	// Check if this train number exists in the European Sleeper system
	trainDays, exists := europeansleeper.TrainDays[trainNumber]
	if !exists {
		return "", false
	}

	// Start searching from yesterday to cover any potential edge cases
	// where "next" might actually be yesterday or today depending on current time
	for i := -1; i <= 14; i++ { // Search up to 2 weeks ahead
		checkDate := time.Now().AddDate(0, 0, i)

		// Check if this train runs on this day of the week
		if contains(trainDays, checkDate.Weekday()) {
			dateStr := checkDate.Format("2006-01-02")
			service := Service{
				TrainNumber: trainNumber,
				Date:        dateStr,
			}

			// Check if we have this service in our cache
			if _, exists := a.timetableCache[service]; exists {
				return dateStr, true
			}
		}
	}

	return "", false
}

func (a *APIV1) refreshCache() {
	for _, train := range europeansleeper.Trains {
		composition, err := europeansleeper.GetComposition(train, a.tcURL)
		if err != nil {
			log.Println("Failed to get composition for train", train, ":", err)
			continue
		}
		a.compositionCache[train] = composition

		// fetch timetable for past and coming week
		for i := -7; i <= 7; i++ {
			date := time.Now().AddDate(0, 0, i)
			service := Service{
				TrainNumber: train,
				Date:        date.Format("2006-01-02"),
			}

			if europeansleeper.TrainDays[train] != nil && contains(europeansleeper.TrainDays[train], date.Weekday()) {
				trip, err := europeansleeper.FetchTimetable(train, date, a.tcURL)
				if err != nil {
					log.Println("Failed to get trip for train", train, "on date", date, ":", err)
					continue
				}
				if a.bahnClient != nil {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					_, err := europeansleeper.EnhanceWithDB(ctx, a.bahnClient, trip)
					if err != nil {
						log.Println("Failed to enhance trip with DB for train", train, "on date", date, ":", err)
					}
					cancel()
				}
				if a.nmbsFetcher != nil {
					_, err := europeansleeper.EnhanceWithNMBS(a.nmbsFetcher, trip)
					if err != nil {
						log.Println("Failed to enhance trip with NMBS for train", train, "on date", date, ":", err)
					}
				}
				if a.nsClient != nil {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					_, err := europeansleeper.EnhanceWithNS(ctx, a.nsClient, trip)
					if err != nil {
						log.Println("Failed to enhance trip with NS for train", train, "on date", date, ":", err)
					}
					cancel()
				}
				if oldTrip, exists := a.timetableCache[service]; exists {
					preserveRealtimeStops(trip, oldTrip)
				}
				a.timetableCache[service] = trip
			}
		}
	}
}

func preserveRealtimeStops(current, previous *traindata.Trip) {
	if current == nil || previous == nil {
		return
	}

	oldStopsByKey := make(map[string]traindata.Stop, len(previous.Stops))
	for _, stop := range previous.Stops {
		oldStopsByKey[stopCacheKey(stop)] = stop
	}

	for i := range current.Stops {
		key := stopCacheKey(current.Stops[i])
		oldStop, ok := oldStopsByKey[key]
		if !ok {
			continue
		}

		if hasRealtimeSignal(current.Stops[i]) || !hasRealtimeSignal(oldStop) {
			continue
		}

		current.Stops[i].RealArrivalTime = oldStop.RealArrivalTime
		current.Stops[i].RealDepartureTime = oldStop.RealDepartureTime
		current.Stops[i].RealPlatform = oldStop.RealPlatform
		current.Stops[i].IsRealTime = oldStop.IsRealTime
	}
}

func stopCacheKey(stop traindata.Stop) string {
	if stop.StationUIC != 0 {
		return fmt.Sprintf("uic:%d", stop.StationUIC)
	}

	name := strings.TrimSpace(strings.ToLower(stop.StationName))
	return "name:" + name
}

func hasRealtimeSignal(stop traindata.Stop) bool {
	if stop.Cancelled {
		return true
	}

	if !stop.RealArrivalTime.IsZero() {
		if stop.ArrivalTime.IsZero() || !stop.RealArrivalTime.Equal(stop.ArrivalTime) {
			return true
		}
	}

	if !stop.RealDepartureTime.IsZero() {
		if stop.DepartureTime.IsZero() || !stop.RealDepartureTime.Equal(stop.DepartureTime) {
			return true
		}
	}

	if stop.RealPlatform != "" {
		if stop.Platform == "" || stop.RealPlatform != stop.Platform {
			return true
		}
	}

	return false
}
