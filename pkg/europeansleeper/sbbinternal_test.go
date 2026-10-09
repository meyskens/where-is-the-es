package europeansleeper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/meyskens/where-is-the-es/pkg/sbbinternal"
	"github.com/meyskens/where-is-the-es/pkg/traindata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnhanceWithSBBInternalKeepsESTimetableTimes(t *testing.T) {
	server := newSBBInternalTestServer(t, `{
		"fahrten": [{
			"fahrtInfo": {"zugnummer": 401},
			"zuglaeufe": [{"haltestellen": [
				{
					"bezeichnung": "Zürich HB", "bpUic": 8503000,
					"ankunftszeiten": {"kb": "2026-10-09T18:00:00+02:00", "ist": "2026-10-09T18:10:00+02:00"},
					"abfahrtszeiten": {"kb": "2026-10-09T18:05:00+02:00", "ist": "2026-10-09T18:15:00+02:00"}
				},
				{
					"bezeichnung": "Basel SBB", "bpUic": 8500010,
					"ankunftszeiten": {"kb": "2026-10-09T19:00:00+02:00"},
					"abfahrtszeiten": {"kb": "2026-10-09T19:05:00+02:00"}
				}
			]}]
		}]
	}`)
	defer server.Close()

	client := sbbinternal.NewClient(
		sbbinternal.WithEndpoint(server.URL),
		sbbinternal.WithHTTPClient(server.Client()),
	)
	plannedArrival := mustParseRFC3339(t, "2026-10-09T17:50:00+02:00")
	plannedDeparture := mustParseRFC3339(t, "2026-10-09T17:55:00+02:00")
	plannedArrival2 := mustParseRFC3339(t, "2026-10-09T18:50:00+02:00")
	plannedDeparture2 := mustParseRFC3339(t, "2026-10-09T18:55:00+02:00")
	trip := &traindata.Trip{
		TrainNumber: "401",
		Date:        time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC),
		Stops: []traindata.Stop{
			{StationName: "Zürich HB", StationUIC: 8503000, ArrivalTime: plannedArrival, DepartureTime: plannedDeparture},
			{StationName: "Basel SBB", StationUIC: 8500010, ArrivalTime: plannedArrival2, DepartureTime: plannedDeparture2},
		},
	}

	enriched, _, err := EnhanceWithSBBInternal(context.Background(), client, trip)

	require.NoError(t, err)
	assert.Equal(t, 2, enriched)
	assert.Equal(t, plannedArrival, trip.Stops[0].ArrivalTime)
	assert.Equal(t, plannedDeparture, trip.Stops[0].DepartureTime)
	assert.Equal(t, mustParseRFC3339(t, "2026-10-09T18:10:00+02:00"), trip.Stops[0].RealArrivalTime)
	assert.Equal(t, mustParseRFC3339(t, "2026-10-09T18:15:00+02:00"), trip.Stops[0].RealDepartureTime)
	// SBB's KB (scheduled) values must not be presented as realtime; when
	// no actual/expected time exists, realtime falls back to the ES schedule.
	assert.Equal(t, plannedArrival2, trip.Stops[1].ArrivalTime)
	assert.Equal(t, plannedDeparture2, trip.Stops[1].DepartureTime)
	assert.Equal(t, plannedArrival2, trip.Stops[1].RealArrivalTime)
	assert.Equal(t, plannedDeparture2, trip.Stops[1].RealDepartureTime)
}

func TestEnhanceWithSBBInternalRejectsInsufficientStationNameMatches(t *testing.T) {
	server := newSBBInternalTestServer(t, `{
		"fahrten": [{
			"fahrtInfo": {"zugnummer": 401},
			"zuglaeufe": [{"haltestellen": [
				{"bezeichnung": "Bern", "bpUic": 8507000, "ankunftszeiten": {"ist": "2026-10-09T18:10:00+02:00"}},
				{"bezeichnung": "Completely Different Station", "bpUic": 8500010, "ankunftszeiten": {"ist": "2026-10-09T19:10:00+02:00"}}
			]}]
		}]
	}`)
	defer server.Close()

	client := sbbinternal.NewClient(
		sbbinternal.WithEndpoint(server.URL),
		sbbinternal.WithHTTPClient(server.Client()),
	)
	planned := mustParseRFC3339(t, "2026-10-09T17:50:00+02:00")
	trip := &traindata.Trip{
		TrainNumber: "401",
		Date:        time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC),
		Stops: []traindata.Stop{
			{StationName: "Bern", StationUIC: 8507000, ArrivalTime: planned},
			// Same UIC is deliberately not sufficient when the name disagrees.
			{StationName: "Basel SBB", StationUIC: 8500010, ArrivalTime: planned},
		},
	}

	enriched, raw, err := EnhanceWithSBBInternal(context.Background(), client, trip)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "station-name match below 95%")
	assert.Equal(t, 0, enriched)
	assert.Len(t, raw, 2)
	assert.Equal(t, planned, trip.Stops[0].ArrivalTime)
	assert.Equal(t, planned, trip.Stops[1].ArrivalTime)
	assert.False(t, trip.Stops[0].IsRealTime)
	assert.False(t, trip.Stops[1].IsRealTime)
}

func newSBBInternalTestServer(t *testing.T, response string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/ws/fahrt/401") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	return server
}

func mustParseRFC3339(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)
	return parsed
}
