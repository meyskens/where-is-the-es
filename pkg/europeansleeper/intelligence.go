package europeansleeper

import "time"

var Trains = []string{
	"453",
	"452",
	"475",
	"474",
}

var TrainDays = map[string][]time.Weekday{
	"453": {time.Monday, time.Wednesday, time.Friday},
	"452": {time.Tuesday, time.Thursday, time.Sunday},
	"475": {time.Tuesday, time.Thursday, time.Sunday},
	"474": {time.Monday, time.Wednesday, time.Friday},
}
