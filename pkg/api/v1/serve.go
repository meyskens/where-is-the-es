package v1

import (
	"log"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/meyskens/where-is-the-es/pkg/europeansleeper"
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
}

func New() *APIV1 {
	return &APIV1{
		compositionCache: make(map[string]traindata.Composition),
		timetableCache:   make(map[Service]*traindata.Trip),
		refreshTimer:     time.NewTicker(1 * time.Minute),
	}
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
		date := c.Param("date")
		number := c.Param("number")

		service := Service{
			TrainNumber: number,
			Date:        date,
		}

		tt, ok := a.timetableCache[service]
		if !ok {
			return c.String(404, "Timetable not found")
		}
		return c.JSON(200, tt)
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

func (a *APIV1) refreshCache() {
	for _, train := range europeansleeper.Trains {
		composition, err := europeansleeper.GetComposition(train)
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
				trip, err := europeansleeper.FetchTimetable(train, date)
				if err != nil {
					log.Println("Failed to get trip for train", train, "on date", date, ":", err)
					continue
				}
				a.timetableCache[service] = trip
			}
		}
	}
}
