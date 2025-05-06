package v1

import (
	"log"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/meyskens/where-is-the-es/pkg/europeansleeper"
	"github.com/meyskens/where-is-the-es/pkg/traindata"
)

type APIV1 struct {
	compositionCache map[string]traindata.Composition

	refreshTimer *time.Ticker

	initDone bool
}

func New() *APIV1 {
	return &APIV1{
		compositionCache: make(map[string]traindata.Composition),
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
	a.refreshCompositionCache()

	e.GET("/api/v1/composition/:number", func(c echo.Context) error {
		tc, ok := a.compositionCache[c.Param("number")]
		if !ok {
			return c.String(404, "Composition not found")
		}
		return c.JSON(200, tc.ToBrowser())
	})
}

func (a *APIV1) refresher() {
	a.refreshCompositionCache()
}

func (a *APIV1) refreshCompositionCache() {
	for _, train := range europeansleeper.Trains {
		composition, err := europeansleeper.GetComposition(train)
		if err != nil {
			log.Println("Failed to get composition for train", train, ":", err)
			continue
		}
		a.compositionCache[train] = composition
	}
}
