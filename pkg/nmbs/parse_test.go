package nmbs

import (
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

func TestParseTimetable(t *testing.T) {
	tests := []struct {
		name    string
		html    string
		want    []traindata.Stop
		wantErr bool
	}{
		{
			name: "successful parse",
			html: testResponse,
			want: []traindata.Stop{
				{
					StationName:       "BRUSSEL-ZUID",
					ArrivalTime:       time.Time{},
					RealArrivalTime:   time.Time{},
					DepartureTime:     mustParseTime(t, "19:22"),
					RealDepartureTime: mustParseTime(t, "19:23"),
				},
				{
					StationName:       "ANTWERPEN-CENTRAAL",
					ArrivalTime:       mustParseTime(t, "19:58"),
					RealArrivalTime:   mustParseTime(t, "20:02"),
					DepartureTime:     mustParseTime(t, "20:02"),
					RealDepartureTime: mustParseTime(t, "20:07"),
				},
				{
					StationName:       "ROOSENDAAL",
					ArrivalTime:       mustParseTime(t, "20:41"),
					RealArrivalTime:   mustParseTime(t, "20:41"),
					DepartureTime:     mustParseTime(t, "20:44"),
					RealDepartureTime: mustParseTime(t, "20:44"),
				},
				{
					StationName:       "ROTTERDAM CENTRAAL",
					ArrivalTime:       mustParseTime(t, "21:19"),
					RealArrivalTime:   mustParseTime(t, "21:19"),
					DepartureTime:     mustParseTime(t, "21:22"),
					RealDepartureTime: mustParseTime(t, "21:22"),
				},
				{
					StationName:       "DEN HAAG HOLLANDS SPOOR",
					ArrivalTime:       mustParseTime(t, "21:40"),
					RealArrivalTime:   mustParseTime(t, "21:40"),
					DepartureTime:     mustParseTime(t, "21:42"),
					RealDepartureTime: mustParseTime(t, "21:42"),
				},
				{
					StationName:       "AMSTERDAM CENTRAAL",
					ArrivalTime:       mustParseTime(t, "22:28"),
					RealArrivalTime:   mustParseTime(t, "22:28"),
					DepartureTime:     mustParseTime(t, "22:34"),
					RealDepartureTime: mustParseTime(t, "22:34"),
				},
				{
					StationName:       "AMERSFOORT CENTRAAL",
					ArrivalTime:       mustParseTime(t, "23:08"),
					RealArrivalTime:   mustParseTime(t, "23:08"),
					DepartureTime:     mustParseTime(t, "23:13"),
					RealDepartureTime: mustParseTime(t, "23:13"),
				},
				{
					StationName:       "DEVENTER",
					ArrivalTime:       mustParseTime(t, "23:48"),
					RealArrivalTime:   mustParseTime(t, "23:48"),
					DepartureTime:     mustParseTime(t, "23:52"),
					RealDepartureTime: mustParseTime(t, "23:52"),
				},
				{
					StationName:       "BERLIN HBF",
					ArrivalTime:       mustParseTime(t, "06:16"),
					RealArrivalTime:   mustParseTime(t, "06:16"),
					DepartureTime:     mustParseTime(t, "06:20"),
					RealDepartureTime: mustParseTime(t, "06:20"),
				},
				{
					StationName:       "BERLIN OSTBAHNHOF",
					ArrivalTime:       mustParseTime(t, "06:27"),
					RealArrivalTime:   mustParseTime(t, "06:27"),
					DepartureTime:     mustParseTime(t, "06:29"),
					RealDepartureTime: mustParseTime(t, "06:29"),
				},
				{
					StationName:       "DRESDEN HBF",
					ArrivalTime:       mustParseTime(t, "08:50"),
					RealArrivalTime:   mustParseTime(t, "08:50"),
					DepartureTime:     mustParseTime(t, "08:54"),
					RealDepartureTime: mustParseTime(t, "08:54"),
				},
				{
					StationName:       "BAD SCHANDAU",
					ArrivalTime:       mustParseTime(t, "09:21"),
					RealArrivalTime:   mustParseTime(t, "09:21"),
					DepartureTime:     mustParseTime(t, "09:23"),
					RealDepartureTime: mustParseTime(t, "09:23"),
				},
				{
					StationName:       "DECIN HL.N.",
					ArrivalTime:       mustParseTime(t, "09:41"),
					RealArrivalTime:   mustParseTime(t, "09:41"),
					DepartureTime:     mustParseTime(t, "09:46"),
					RealDepartureTime: mustParseTime(t, "09:46"),
				},
				{
					StationName:       "PRAHA HL.N.",
					ArrivalTime:       mustParseTime(t, "11:24"),
					RealArrivalTime:   mustParseTime(t, "11:24"),
					DepartureTime:     time.Time{},
					RealDepartureTime: time.Time{},
				},
			},
			wantErr: false,
		},
		{
			name:    "error on empty response",
			html:    "<div></div>",
			want:    []traindata.Stop(nil),
			wantErr: true,
		},
		{
			name:    "error on search error",
			html:    `<input type="hidden" id="trainSearchErrorResult" value="true" />`,
			want:    []traindata.Stop(nil),
			wantErr: true,
		},
	}

	f := &NMBSFetcher{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := f.ParseTimetable([]byte(tt.html))

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTimetable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.Equal(t, got, tt.want, "ParseTimetable() got = %v, want %v", got, tt.want)
		})
	}
}
