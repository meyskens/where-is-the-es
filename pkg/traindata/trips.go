package traindata

import "time"

type Trip struct {
	TrainNumber string
	Stops       []Stop
	Composition Composition

	IsRunning bool

	Date time.Time
}

type Stop struct {
	StationName   string
	StationUIC    int
	ArrivalTime   time.Time
	DepartureTime time.Time
	Platform      string

	DataSources         []DataSource
	PrefferedDataSource DataSource

	RealPlatform      string
	RealArrivalTime   time.Time
	RealDepartureTime time.Time
	IsRealTime        bool

	NextDay   bool
	Cancelled bool
}

type TripBrowser struct {
	TrainNumber string        `json:"TrainNumber"`
	Stops       []StopBrowser `json:"Stops"`
	Composition CompositionBrowser
	IsRunning   bool   `json:"IsRunning"`
	Date        string `json:"Date"`
}

type StopBrowser struct {
	StationName         string   `json:"StationName"`
	StationUIC          int      `json:"StationUIC"`
	ArrivalTime         string   `json:"ArrivalTime"`
	DepartureTime       string   `json:"DepartureTime"`
	Platform            string   `json:"Platform"`
	DataSources         []string `json:"DataSources"`
	PrefferedDataSource string   `json:"PrefferedDataSource"`
	RealPlatform        string   `json:"RealPlatform"`
	RealArrivalTime     string   `json:"RealArrivalTime"`
	RealDepartureTime   string   `json:"RealDepartureTime"`
	IsRealTime          bool     `json:"IsRealTime"`
	NextDay             bool     `json:"NextDay"`
	Cancelled           bool     `json:"Cancelled"`
}

func (t *Trip) ToBrowser() TripBrowser {
	stops := make([]StopBrowser, len(t.Stops))
	for i, stop := range t.Stops {
		stops[i] = StopBrowser{
			StationName:         stop.StationName,
			StationUIC:          stop.StationUIC,
			ArrivalTime:         stop.ArrivalTime.Format(time.RFC3339),
			DepartureTime:       stop.DepartureTime.Format(time.RFC3339),
			Platform:            stop.Platform,
			DataSources:         DataSourcesToStrings(stop.DataSources),
			PrefferedDataSource: stop.PrefferedDataSource.String(),
			RealPlatform:        stop.RealPlatform,
			RealArrivalTime:     stop.RealArrivalTime.Format(time.RFC3339),
			RealDepartureTime:   stop.RealDepartureTime.Format(time.RFC3339),
			IsRealTime:          stop.IsRealTime,
			NextDay:             stop.NextDay,
			Cancelled:           stop.Cancelled,
		}
	}

	return TripBrowser{
		TrainNumber: t.TrainNumber,
		Stops:       stops,
		Composition: t.Composition.ToBrowser(),
		IsRunning:   t.IsRunning,
		Date:        t.Date.Format("2006-01-02"),
	}
}
