package europeansleeper

import (
	"bytes"
	"testing"
	"time"

	"github.com/meyskens/where-is-the-es/pkg/traindata"
	"github.com/stretchr/testify/assert"
)

func mustParseTime(t *testing.T, timeStr string) time.Time {
	t.Helper()
	parsedTime, err := time.Parse("15:04", timeStr)
	if err != nil {
		t.Fatalf("failed to parse time: %v", err)
	}
	return parsedTime
}

func Test_parseTimetable(t *testing.T) {
	tests := []struct {
		name        string
		html        string
		trainNumber string
		want        *traindata.Trip
		wantErr     bool
	}{
		{
			name:        "parses valid timetable",
			html:        timetableData,
			trainNumber: "453",
			want: &traindata.Trip{
				TrainNumber: "453",
				Stops: []traindata.Stop{
					{
						StationName:   "Bruxelles-Midi",
						DepartureTime: mustParseTime(t, "19:22"),
					},
					{
						StationName:   "Antwerpen-Centraal",
						ArrivalTime:   mustParseTime(t, "19:58"),
						DepartureTime: mustParseTime(t, "20:02"),
					},
					{
						StationName:   "Roosendaal",
						ArrivalTime:   mustParseTime(t, "20:41"),
						DepartureTime: mustParseTime(t, "20:44"),
					},
					{
						StationName:   "Rotterdam Centraal",
						ArrivalTime:   mustParseTime(t, "21:19"),
						DepartureTime: mustParseTime(t, "21:22"),
					},
					{
						StationName:   "Den Haag HS",
						ArrivalTime:   mustParseTime(t, "21:40"),
						DepartureTime: mustParseTime(t, "21:42"),
					},
					{
						StationName:   "Amsterdam Centraal",
						ArrivalTime:   mustParseTime(t, "22:28"),
						DepartureTime: mustParseTime(t, "22:34"),
					},
					{
						StationName:   "Amersfoort Centraal",
						ArrivalTime:   mustParseTime(t, "23:08"),
						DepartureTime: mustParseTime(t, "23:13"),
					},
					{
						StationName:   "Deventer",
						ArrivalTime:   mustParseTime(t, "23:48"),
						DepartureTime: mustParseTime(t, "23:52"),
					},
					{
						StationName:   "Berlin Hauptbahnhof",
						ArrivalTime:   mustParseTime(t, "06:16"),
						DepartureTime: mustParseTime(t, "06:20"),
						NextDay:       true,
					},
					{
						StationName:   "Berlin Ostbahnhof",
						ArrivalTime:   mustParseTime(t, "06:27"),
						DepartureTime: mustParseTime(t, "06:29"),
						NextDay:       true,
					},
					{
						StationName:   "Dresden Hbf",
						ArrivalTime:   mustParseTime(t, "08:50"),
						DepartureTime: mustParseTime(t, "08:54"),
						NextDay:       true,
					},
					{
						StationName:   "Bad Schandau",
						ArrivalTime:   mustParseTime(t, "09:21"),
						DepartureTime: mustParseTime(t, "09:23"),
						NextDay:       true,
					},
					{
						StationName:   "Decin hl.n.",
						ArrivalTime:   mustParseTime(t, "09:41"),
						DepartureTime: mustParseTime(t, "09:46"),
						NextDay:       true,
					},
					{
						StationName:   "Prague hl.n. (main station)",
						DepartureTime: mustParseTime(t, "11:24"),
						NextDay:       true,
					},
				},
			},
			wantErr: false,
		},
		{
			name:        "returns error for invalid HTML",
			html:        "invalid HTML",
			trainNumber: "123",
			wantErr:     true,
		},
		{
			name:        "returns error when train not found",
			html:        "<h3>Train ES 456</h3>",
			trainNumber: "123",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTimetable(tt.trainNumber, bytes.NewReader([]byte(tt.html)))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
