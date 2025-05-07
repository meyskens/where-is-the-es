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
	ArrivalTime   time.Time
	DepartureTime time.Time
	Platform      string

	DataSources         []DataSource
	PrefferedDataSource DataSource

	RealPlatform      string
	RealArrivalTime   time.Time
	RealDepartureTime time.Time

	NextDay   bool
	Cancelled bool
}
