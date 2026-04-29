package ns

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/meyskens/where-is-the-es/pkg/traindata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTimetable(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/reisinformatie-api/api/v2/journey", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-key", r.Header.Get("Ocp-Apim-Subscription-Key"))
		assert.Equal(t, "452", r.URL.Query().Get("train"))
		assert.Equal(t, "false", r.URL.Query().Get("omitCrowdForecast"))
		assert.Empty(t, r.URL.Query().Get("dateTime"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"payload": {
				"stops": [
					{
						"id": "HANN_0",
						"stop": {
							"name": "Hannover Hbf",
							"uicCode": "8013552",
							"countryCode": "D"
						},
						"status": "ORIGIN",
						"arrivals": [],
						"departures": [
							{
								"plannedTime": "2026-04-29T06:06:00+0200",
								"actualTime": "2026-04-29T06:28:15+0200",
								"plannedTrack": "4",
								"actualTrack": "4",
								"cancelled": false,
								"crowdForecast": "UNKNOWN"
							}
						]
					},
					{
						"id": "DV_0",
						"stop": {
							"name": "Deventer",
							"uicCode": "8400173",
							"countryCode": "NL"
						},
						"status": "STOP",
						"arrivals": [
							{
								"plannedTime": "2026-04-29T06:06:00+0200",
								"actualTime": "2026-04-29T06:28:15+0200",
								"plannedTrack": "4",
								"actualTrack": "4",
								"cancelled": false,
								"crowdForecast": "UNKNOWN"
							}
						],
						"departures": [
							{
								"plannedTime": "2026-04-29T06:08:00+0200",
								"actualTime": "2026-04-29T06:30:25+0200",
								"plannedTrack": "4",
								"actualTrack": "4",
								"cancelled": false,
								"crowdForecast": "UNKNOWN"
							}
						]
					},
					{
						"id": "BRUSZ_0",
						"stop": {
							"name": "Brussel-Zuid",
							"uicCode": "8814001",
							"countryCode": "B"
						},
						"status": "DESTINATION",
						"arrivals": [
							{
								"plannedTime": "2026-04-29T09:11:00+0200",
								"actualTime": "2026-04-29T09:55:34+0200",
								"plannedTrack": "4",
								"actualTrack": "4",
								"cancelled": false,
								"crowdForecast": "UNKNOWN"
							}
						],
						"departures": []
					}
				]
			}
		}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL+"/reisinformatie-api/api/v2"))

	trip, err := client.GetTimetable(context.Background(), "ES452", time.Time{})
	require.NoError(t, err)
	require.Len(t, trip.Stops, 3)

	// Origin stop
	assert.Equal(t, "Hannover Hbf", trip.Stops[0].StationName)
	assert.Equal(t, 8013552, trip.Stops[0].StationUIC)
	assert.True(t, trip.Stops[0].DepartureTime.Equal(time.Date(2026, 4, 29, 6, 6, 0, 0, time.FixedZone("+0200", 2*60*60))))
	assert.True(t, trip.Stops[0].RealDepartureTime.Equal(time.Date(2026, 4, 29, 6, 28, 15, 0, time.FixedZone("+0200", 2*60*60))))
	assert.Equal(t, "4", trip.Stops[0].Platform)
	assert.Equal(t, "4", trip.Stops[0].RealPlatform)
	assert.True(t, trip.Stops[0].IsRealTime)
	assert.False(t, trip.Stops[0].Cancelled)

	// Intermediate stop
	assert.Equal(t, "Deventer", trip.Stops[1].StationName)
	assert.Equal(t, 8400173, trip.Stops[1].StationUIC)
	assert.True(t, trip.Stops[1].ArrivalTime.Equal(time.Date(2026, 4, 29, 6, 6, 0, 0, time.FixedZone("+0200", 2*60*60))))
	assert.True(t, trip.Stops[1].RealArrivalTime.Equal(time.Date(2026, 4, 29, 6, 28, 15, 0, time.FixedZone("+0200", 2*60*60))))
	assert.True(t, trip.Stops[1].DepartureTime.Equal(time.Date(2026, 4, 29, 6, 8, 0, 0, time.FixedZone("+0200", 2*60*60))))
	assert.True(t, trip.Stops[1].RealDepartureTime.Equal(time.Date(2026, 4, 29, 6, 30, 25, 0, time.FixedZone("+0200", 2*60*60))))
	assert.True(t, trip.Stops[1].IsRealTime)

	// Destination stop
	assert.Equal(t, "Brussel-Zuid", trip.Stops[2].StationName)
	assert.Equal(t, 8814001, trip.Stops[2].StationUIC)
	assert.True(t, trip.Stops[2].ArrivalTime.Equal(time.Date(2026, 4, 29, 9, 11, 0, 0, time.FixedZone("+0200", 2*60*60))))
	assert.True(t, trip.Stops[2].RealArrivalTime.Equal(time.Date(2026, 4, 29, 9, 55, 34, 0, time.FixedZone("+0200", 2*60*60))))
	assert.True(t, trip.Stops[2].IsRealTime)

	// Data sources
	for _, stop := range trip.Stops {
		assert.Equal(t, []traindata.DataSource{traindata.DataSourceNS}, stop.DataSources)
		assert.Equal(t, traindata.DataSourceNS, stop.PrefferedDataSource)
	}
}

func TestGetTimetable_EmptyTrainNumber(t *testing.T) {
	client := NewClient("test-key")
	_, err := client.GetTimetable(context.Background(), "ES", time.Time{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty train number")
}

func TestGetTimetable_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Access denied"}`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.GetTimetable(context.Background(), "452", time.Time{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status 401")
}

func TestStripPrefix(t *testing.T) {
	assert.Equal(t, "302", stripPrefix("ES302"))
	assert.Equal(t, "452", stripPrefix("452"))
	assert.Equal(t, "", stripPrefix("ES"))
	assert.Equal(t, "", stripPrefix(""))
}

func TestParseNSTime(t *testing.T) {
	expected := time.Date(2026, 4, 29, 6, 6, 0, 0, time.FixedZone("+0200", 2*60*60))
	assert.True(t, parseNSTime("2026-04-29T06:06:00+0200").Equal(expected))

	assert.True(t, parseNSTime("").IsZero())
	assert.True(t, parseNSTime("invalid").IsZero())
}

func TestGetTimetable_WithDate(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/reisinformatie-api/api/v2/journey", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-key", r.Header.Get("Ocp-Apim-Subscription-Key"))
		assert.Equal(t, "452", r.URL.Query().Get("train"))
		assert.Equal(t, "2026-04-29T00:00:00+02:00", r.URL.Query().Get("dateTime"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"payload": {
				"stops": [
					{
						"id": "HANN_0",
						"stop": {
							"name": "Hannover Hbf",
							"uicCode": "8013552",
							"countryCode": "D"
						},
						"status": "ORIGIN",
						"arrivals": [],
						"departures": [
							{
								"plannedTime": "2026-04-29T06:06:00+0200",
								"actualTime": "2026-04-29T06:28:15+0200",
								"plannedTrack": "4",
								"actualTrack": "4",
								"cancelled": false,
								"crowdForecast": "UNKNOWN"
							}
						]
					}
				]
			}
		}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL+"/reisinformatie-api/api/v2"))

	date := time.Date(2026, 4, 29, 0, 0, 0, 0, time.FixedZone("+0200", 2*60*60))
	trip, err := client.GetTimetable(context.Background(), "ES452", date)
	require.NoError(t, err)
	require.Len(t, trip.Stops, 1)
	assert.Equal(t, "Hannover Hbf", trip.Stops[0].StationName)
}
