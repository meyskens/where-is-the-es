package sbb

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/meyskens/where-is-the-es/pkg/traindata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tripsResponseJSON is a trimmed-down copy of the real GraphQL response
// captured from graphql.www.sbb.ch for the Lugano -> Bruxelles-Midi trip
// search. It contains two trips: a multi-leg connection and the direct EN
// 400 single-leg journey (the one the enhancer should match).
const tripsResponseJSON = `{
  "data": {
    "trips": {
      "trips": [
        {
          "id": "3HA.multi-leg",
          "legs": [
            {
              "duration": 69,
              "id": "0",
              "__typename": "PTRideLeg",
              "start": {"__typename": "StopPlace", "id": "8505300", "name": "Lugano"},
              "end": {"__typename": "StopPlace", "id": "8505004", "name": "Arth-Goldau"},
              "arrival": {"time": "2026-09-27T18:11:00+02:00", "delay": 0, "delayText": null, "quayFormatted": "4", "quayChanged": false, "quayChangedText": null},
              "departure": {"time": "2026-09-27T17:02:00+02:00", "delay": 0, "delayText": null, "quayFormatted": "3", "quayChanged": false, "quayChangedText": null},
              "serviceJourney": {
                "id": "ic2-886",
                "stopPoints": [
                  {"place": {"id": "8505300", "name": "Lugano"}, "occupancy": {"firstClass": "LOW", "secondClass": "MEDIUM"}, "stopStatus": "PLANNED", "stopStatusFormatted": null, "delayUndefined": false},
                  {"place": {"id": "8505213", "name": "Bellinzona"}, "occupancy": {"firstClass": "LOW", "secondClass": "MEDIUM"}, "stopStatus": "PLANNED", "stopStatusFormatted": null, "delayUndefined": false},
                  {"place": {"id": "8505004", "name": "Arth-Goldau"}, "occupancy": {"firstClass": "UNKNOWN", "secondClass": "UNKNOWN"}, "stopStatus": "PLANNED", "stopStatusFormatted": null, "delayUndefined": false}
                ],
                "serviceProducts": [{"name": "IC 2 886", "line": "2", "number": "886", "vehicleMode": "TRAIN", "vehicleSubModeShortName": "IC", "corporateIdentityIcon": "ic-2", "corporateIdentityPictogram": "train-right", "routeIndexFrom": 0, "routeIndexTo": 4}],
                "direction": "Zürich HB",
                "serviceAlteration": {"cancelled": false, "partiallyCancelled": false, "redirected": false, "redirectedText": null, "reachable": true, "reachableText": null, "delayText": null, "unplannedStopPointsText": null, "quayChangedText": null, "cancelledExpected": false}
              }
            }
          ]
        },
        {
          "id": "3HA.en400-direct",
          "legs": [
            {
              "duration": 1078,
              "id": "0",
              "__typename": "PTRideLeg",
              "start": {"__typename": "StopPlace", "id": "8505300", "name": "Lugano"},
              "end": {"__typename": "StopPlace", "id": "8814001", "name": "Bruxelles-Midi"},
              "arrival": {"time": "2026-09-28T11:42:00+02:00", "delay": 0, "delayText": null, "quayFormatted": null, "quayChanged": false, "quayChangedText": null},
              "departure": {"time": "2026-09-27T17:44:00+02:00", "delay": 0, "delayText": null, "quayFormatted": null, "quayChanged": false, "quayChangedText": null},
              "serviceJourney": {
                "id": "eNodyTEOgCAQRNG7TC3JsGEN0Nt6AWO4gRotWe7uavfmT8eNimhKSlKjZTVRFqHMmHD6-TGwBFEPF-rW_6wx59Y8HT4S6Xpcy4qxjxeHABR5",
                "stopPoints": [
                  {"place": {"id": "8505300", "name": "Lugano"}, "occupancy": {"firstClass": "UNKNOWN", "secondClass": "UNKNOWN"}, "stopStatus": "PLANNED", "stopStatusFormatted": null, "delayUndefined": false},
                  {"place": {"id": "8505213", "name": "Bellinzona"}, "occupancy": {"firstClass": "UNKNOWN", "secondClass": "UNKNOWN"}, "stopStatus": "PLANNED", "stopStatusFormatted": null, "delayUndefined": false},
                  {"place": {"id": "8505119", "name": "Göschenen"}, "occupancy": {"firstClass": "UNKNOWN", "secondClass": "UNKNOWN"}, "stopStatus": "PLANNED", "stopStatusFormatted": null, "delayUndefined": false},
                  {"place": {"id": "8505004", "name": "Arth-Goldau"}, "occupancy": {"firstClass": "UNKNOWN", "secondClass": "UNKNOWN"}, "stopStatus": "PLANNED", "stopStatusFormatted": null, "delayUndefined": false},
                  {"place": {"id": "8503505", "name": "Wettingen"}, "occupancy": {"firstClass": "UNKNOWN", "secondClass": "UNKNOWN"}, "stopStatus": "PLANNED", "stopStatusFormatted": null, "delayUndefined": false},
                  {"place": {"id": "8015458", "name": "Köln Hbf"}, "occupancy": {"firstClass": "UNKNOWN", "secondClass": "UNKNOWN"}, "stopStatus": "PLANNED", "stopStatusFormatted": null, "delayUndefined": false},
                  {"place": {"id": "8015345", "name": "Aachen Hbf"}, "occupancy": {"firstClass": "UNKNOWN", "secondClass": "UNKNOWN"}, "stopStatus": "PLANNED", "stopStatusFormatted": null, "delayUndefined": false},
                  {"place": {"id": "8844008", "name": "Verviers-Central"}, "occupancy": {"firstClass": "UNKNOWN", "secondClass": "UNKNOWN"}, "stopStatus": "PLANNED", "stopStatusFormatted": null, "delayUndefined": false},
                  {"place": {"id": "8814001", "name": "Bruxelles-Midi"}, "occupancy": {"firstClass": "UNKNOWN", "secondClass": "UNKNOWN"}, "stopStatus": "PLANNED", "stopStatusFormatted": null, "delayUndefined": false}
                ],
                "serviceProducts": [{"name": "EN 400", "line": null, "number": "400", "vehicleMode": "TRAIN", "vehicleSubModeShortName": "EN", "corporateIdentityIcon": "en", "corporateIdentityPictogram": "train-right", "routeIndexFrom": 2, "routeIndexTo": 10}],
                "direction": "Bruxelles-Midi",
                "serviceAlteration": {"cancelled": false, "partiallyCancelled": false, "redirected": false, "redirectedText": null, "reachable": true, "reachableText": null, "delayText": null, "unplannedStopPointsText": null, "quayChangedText": null, "cancelledExpected": false}
              }
            }
          ]
        }
      ],
      "paginationCursor": {"previous": null, "next": null}
    }
  }
}`

// serviceJourneyResponseJSON is a trimmed-down copy of the real
// serviceJourneyById response for EN 400. It carries full per-stop
// arrival/departure events including the overnight gap between Aarau and
// Köln Hbf.
const serviceJourneyResponseJSON = `{
  "data": {
    "serviceJourneyById": {
      "id": "eNodirENgDAMBHdxTaSPFSMnJYiWBRDKBoCgjLM7hu7udI1uKhRNAE5iMBVjQWbwaHOZaKDTh08DcmDxcFHZ2p8lqtbq6XBJgNPjtKzU9_4C72AVtA",
      "stopPoints": [
        {
          "stopStatus": "PLANNED",
          "stopStatusFormatted": null,
          "requestStop": false,
          "delayUndefined": false,
          "arrival": null,
          "departure": {"time": "2026-09-25T15:45:00+02:00", "delay": 0, "delayText": null, "quayFormatted": null, "quayChanged": false, "quayChangedText": null},
          "occupancy": {"firstClass": "UNKNOWN", "secondClass": "UNKNOWN"},
          "place": {"id": "8301645", "name": "Milano Porta Garibaldi"},
          "forBoarding": true,
          "forAlighting": true
        },
        {
          "stopStatus": "PLANNED",
          "stopStatusFormatted": null,
          "requestStop": false,
          "delayUndefined": false,
          "arrival": null,
          "departure": {"time": "2026-09-25T16:57:00+02:00", "delay": 0, "delayText": null, "quayFormatted": null, "quayChanged": false, "quayChangedText": null},
          "occupancy": {"firstClass": "UNKNOWN", "secondClass": "UNKNOWN"},
          "place": {"id": "8301307", "name": "Como S. Giovanni"},
          "forBoarding": true,
          "forAlighting": false
        },
        {
          "stopStatus": "PLANNED",
          "stopStatusFormatted": null,
          "requestStop": false,
          "delayUndefined": false,
          "arrival": {"time": "2026-09-25T18:42:00+02:00", "delay": 0, "delayText": null, "quayFormatted": null, "quayChanged": false, "quayChangedText": null},
          "departure": {"time": "2026-09-25T18:44:00+02:00", "delay": 0, "delayText": null, "quayFormatted": null, "quayChanged": false, "quayChangedText": null},
          "occupancy": {"firstClass": "UNKNOWN", "secondClass": "UNKNOWN"},
          "place": {"id": "8505300", "name": "Lugano"},
          "forBoarding": true,
          "forAlighting": true
        },
        {
          "stopStatus": "PLANNED",
          "stopStatusFormatted": null,
          "requestStop": false,
          "delayUndefined": false,
          "arrival": {"time": "2026-09-25T19:21:00+02:00", "delay": 0, "delayText": null, "quayFormatted": null, "quayChanged": false, "quayChangedText": null},
          "departure": {"time": "2026-09-25T19:23:00+02:00", "delay": 0, "delayText": null, "quayFormatted": null, "quayChanged": false, "quayChangedText": null},
          "occupancy": {"firstClass": "UNKNOWN", "secondClass": "UNKNOWN"},
          "place": {"id": "8505213", "name": "Bellinzona"},
          "forBoarding": true,
          "forAlighting": true
        },
        {
          "stopStatus": "PLANNED",
          "stopStatusFormatted": null,
          "requestStop": false,
          "delayUndefined": false,
          "arrival": {"time": "2026-09-25T20:30:00+02:00", "delay": 0, "delayText": null, "quayFormatted": null, "quayChanged": false, "quayChangedText": null},
          "departure": {"time": "2026-09-25T20:52:00+02:00", "delay": 0, "delayText": null, "quayFormatted": null, "quayChanged": false, "quayChangedText": null},
          "occupancy": {"firstClass": "UNKNOWN", "secondClass": "UNKNOWN"},
          "place": {"id": "8505119", "name": "Göschenen"},
          "forBoarding": true,
          "forAlighting": true
        },
        {
          "stopStatus": "PLANNED",
          "stopStatusFormatted": null,
          "requestStop": false,
          "delayUndefined": false,
          "arrival": {"time": "2026-09-25T21:49:00+02:00", "delay": 0, "delayText": null, "quayFormatted": null, "quayChanged": false, "quayChangedText": null},
          "departure": {"time": "2026-09-25T22:00:00+02:00", "delay": 0, "delayText": null, "quayFormatted": null, "quayChanged": false, "quayChangedText": null},
          "occupancy": {"firstClass": "UNKNOWN", "secondClass": "UNKNOWN"},
          "place": {"id": "8505004", "name": "Arth-Goldau"},
          "forBoarding": true,
          "forAlighting": true
        },
        {
          "stopStatus": "PLANNED",
          "stopStatusFormatted": null,
          "requestStop": false,
          "delayUndefined": false,
          "arrival": {"time": "2026-09-25T22:55:00+02:00", "delay": 0, "delayText": null, "quayFormatted": null, "quayChanged": false, "quayChangedText": null},
          "departure": {"time": "2026-09-25T22:58:00+02:00", "delay": 0, "delayText": null, "quayFormatted": null, "quayChanged": false, "quayChangedText": null},
          "occupancy": {"firstClass": "UNKNOWN", "secondClass": "UNKNOWN"},
          "place": {"id": "8502113", "name": "Aarau"},
          "forBoarding": true,
          "forAlighting": true
        },
        {
          "stopStatus": "PLANNED",
          "stopStatusFormatted": null,
          "requestStop": false,
          "delayUndefined": false,
          "arrival": {"time": "2026-09-26T08:00:00+02:00", "delay": 0, "delayText": null, "quayFormatted": null, "quayChanged": false, "quayChangedText": null},
          "departure": {"time": "2026-09-26T08:03:00+02:00", "delay": 0, "delayText": null, "quayFormatted": null, "quayChanged": false, "quayChangedText": null},
          "occupancy": {"firstClass": "UNKNOWN", "secondClass": "UNKNOWN"},
          "place": {"id": "8015458", "name": "Köln Hbf"},
          "forBoarding": true,
          "forAlighting": true
        },
        {
          "stopStatus": "PLANNED",
          "stopStatusFormatted": null,
          "requestStop": false,
          "delayUndefined": false,
          "arrival": {"time": "2026-09-26T08:48:00+02:00", "delay": 0, "delayText": null, "quayFormatted": null, "quayChanged": false, "quayChangedText": null},
          "departure": {"time": "2026-09-26T08:55:00+02:00", "delay": 0, "delayText": null, "quayFormatted": null, "quayChanged": false, "quayChangedText": null},
          "occupancy": {"firstClass": "UNKNOWN", "secondClass": "UNKNOWN"},
          "place": {"id": "8015345", "name": "Aachen Hbf"},
          "forBoarding": true,
          "forAlighting": true
        },
        {
          "stopStatus": "PLANNED",
          "stopStatusFormatted": null,
          "requestStop": false,
          "delayUndefined": false,
          "arrival": {"time": "2026-09-26T09:18:00+02:00", "delay": 0, "delayText": null, "quayFormatted": null, "quayChanged": false, "quayChangedText": null},
          "departure": null,
          "occupancy": {"firstClass": "UNKNOWN", "secondClass": "UNKNOWN"},
          "place": {"id": "8844008", "name": "Verviers-Central"},
          "forBoarding": false,
          "forAlighting": true
        },
        {
          "stopStatus": "PLANNED",
          "stopStatusFormatted": null,
          "requestStop": false,
          "delayUndefined": false,
          "arrival": {"time": "2026-09-26T11:42:00+02:00", "delay": 0, "delayText": null, "quayFormatted": null, "quayChanged": false, "quayChangedText": null},
          "departure": null,
          "occupancy": {"firstClass": "UNKNOWN", "secondClass": "UNKNOWN"},
          "place": {"id": "8814001", "name": "Bruxelles-Midi"},
          "forBoarding": true,
          "forAlighting": true
        }
      ]
    }
  }
}`

// newTestServer spins up an httptest server that dispatches GraphQL
// responses based on the operation name embedded in the request body.
// It also asserts the SBB-specific Apollo headers are present.
func newTestServer(t *testing.T, tripsBody, journeyBody string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, apolloClientName, r.Header.Get("apollographql-client-name"))
		assert.Equal(t, apolloClientVersion, r.Header.Get("apollographql-client-version"))
		assert.Equal(t, apolloClientOrigin, r.Header.Get("apollographql-client-origin"))
		assert.Equal(t, userAgent, r.Header.Get("User-Agent"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(string(body), "query Trips"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(tripsBody))
		case strings.Contains(string(body), "getServiceJourneyById"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(journeyBody))
		default:
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"errors":[{"message":"unknown operation"}]}`))
		}
	})
	return httptest.NewServer(mux)
}

func TestNewClient(t *testing.T) {
	client := NewClient()
	require.NotNil(t, client)
	assert.Equal(t, defaultEndpoint, client.endpoint)
	assert.NotNil(t, client.httpClient)
	assert.NotNil(t, client.limiter)
}

func TestStripPrefix(t *testing.T) {
	assert.Equal(t, "400", stripPrefix("EN400"))
	assert.Equal(t, "400", stripPrefix("400"))
	assert.Equal(t, "", stripPrefix("EN"))
}

func TestParseSBBTime(t *testing.T) {
	zone := time.FixedZone("+0200", 2*60*60)

	got := parseSBBTime("2026-09-27T17:44:00+02:00")
	assert.True(t, got.Equal(time.Date(2026, 9, 27, 17, 44, 0, 0, zone)))

	assert.True(t, parseSBBTime("").IsZero())
	assert.True(t, parseSBBTime("not-a-time").IsZero())
}

func TestGetServiceJourney(t *testing.T) {
	server := newTestServer(t, tripsResponseJSON, serviceJourneyResponseJSON)
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))

	detail, err := client.GetServiceJourney(context.Background(), "eNodyTEOgCAQRNG7TC3JsGEN0Nt6AWO4gRotWe7uavfmT8eNimhKSlKjZTVRFqHMmHD6-TGwBFEPF-rW_6wx59Y8HT4S6Xpcy4qxjxeHABR5")
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Equal(t, "eNodirENgDAMBHdxTaSPFSMnJYiWBRDKBoCgjLM7hu7udI1uKhRNAE5iMBVjQWbwaHOZaKDTh08DcmDxcFHZ2p8lqtbq6XBJgNPjtKzU9_4C72AVtA", detail.ID)
	assert.Len(t, detail.StopPoints, 11)

	// Lugano: arrival + departure both present.
	assert.Equal(t, "Lugano", detail.StopPoints[2].Place.Name)
	require.NotNil(t, detail.StopPoints[2].Arrival)
	assert.Equal(t, "2026-09-25T18:42:00+02:00", detail.StopPoints[2].Arrival.Time)
	require.NotNil(t, detail.StopPoints[2].Departure)
	assert.Equal(t, "2026-09-25T18:44:00+02:00", detail.StopPoints[2].Departure.Time)

	// First stop (Milano): arrival is null, departure present.
	assert.Nil(t, detail.StopPoints[0].Arrival)
	require.NotNil(t, detail.StopPoints[0].Departure)

	// Last stop (Bruxelles-Midi): departure is null, arrival present.
	assert.Nil(t, detail.StopPoints[10].Departure)
	require.NotNil(t, detail.StopPoints[10].Arrival)
}

func TestGetServiceJourney_Errors(t *testing.T) {
	t.Run("empty id", func(t *testing.T) {
		client := NewClient(WithEndpoint("http://example.invalid"))
		_, err := client.GetServiceJourney(context.Background(), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty service journey id")
	})
}

func TestGetTimetable_FullPerStopData(t *testing.T) {
	server := newTestServer(t, tripsResponseJSON, serviceJourneyResponseJSON)
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))

	// Match the EN 400 leg. The planned departure is 17:44 from Lugano.
	departure := time.Date(2026, 9, 27, 17, 44, 0, 0, time.FixedZone("+0200", 2*60*60))
	trip, err := client.GetTimetable(context.Background(), "EN400", "8505300", "8814001", departure)
	require.NoError(t, err)
	require.NotNil(t, trip)

	assert.Equal(t, "EN400", trip.TrainNumber)
	// The detail query has 11 stops; the trip should use that full list.
	require.Len(t, trip.Stops, 11)

	// The fixture uses Central European Summer Time (+02:00).
	zone := time.FixedZone("+0200", 2*60*60)

	// Every stop is tagged with the SBB data source.
	for i, s := range trip.Stops {
		assert.Equal(t, traindata.DataSourceSBB, s.PrefferedDataSource, "stop %d (%s)", i, s.StationName)
		assert.Contains(t, s.DataSources, traindata.DataSourceSBB)
	}

	// Lugano is the 3rd stop in the journey (after Milano and Como).
	lugano := trip.Stops[2]
	assert.Equal(t, "Lugano", lugano.StationName)
	assert.Equal(t, 8505300, lugano.StationUIC)
	assert.True(t, lugano.ArrivalTime.Equal(time.Date(2026, 9, 25, 18, 42, 0, 0, zone)))
	assert.True(t, lugano.DepartureTime.Equal(time.Date(2026, 9, 25, 18, 44, 0, 0, zone)))
	// No delay reported, so realtime == scheduled.
	assert.True(t, lugano.RealArrivalTime.Equal(lugano.ArrivalTime))
	assert.True(t, lugano.RealDepartureTime.Equal(lugano.DepartureTime))
	assert.False(t, lugano.IsRealTime)

	// Origin (Milano) has no arrival, only a departure.
	milano := trip.Stops[0]
	assert.Equal(t, "Milano Porta Garibaldi", milano.StationName)
	assert.Equal(t, 8301645, milano.StationUIC)
	assert.True(t, milano.ArrivalTime.IsZero())
	assert.True(t, milano.DepartureTime.Equal(time.Date(2026, 9, 25, 15, 45, 0, 0, zone)))

	// Destination (Bruxelles-Midi) has no departure, only an arrival.
	brussels := trip.Stops[10]
	assert.Equal(t, "Bruxelles-Midi", brussels.StationName)
	assert.Equal(t, 8814001, brussels.StationUIC)
	assert.True(t, brussels.DepartureTime.IsZero())
	assert.True(t, brussels.ArrivalTime.Equal(time.Date(2026, 9, 26, 11, 42, 0, 0, zone)))

	// The overnight gap is preserved: Aarau (22:58) -> Köln Hbf (08:00 next day).
	aarau := trip.Stops[6]
	assert.Equal(t, "Aarau", aarau.StationName)
	assert.True(t, aarau.DepartureTime.Equal(time.Date(2026, 9, 25, 22, 58, 0, 0, zone)))
	koln := trip.Stops[7]
	assert.Equal(t, "Köln Hbf", koln.StationName)
	assert.True(t, koln.ArrivalTime.Equal(time.Date(2026, 9, 26, 8, 0, 0, 0, zone)))
	assert.True(t, koln.ArrivalTime.After(aarau.DepartureTime))
}

func TestGetTimetable_NoMatchingTrain(t *testing.T) {
	server := newTestServer(t, tripsResponseJSON, serviceJourneyResponseJSON)
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))

	departure := time.Date(2026, 9, 27, 17, 44, 0, 0, time.FixedZone("+0200", 2*60*60))
	_, err := client.GetTimetable(context.Background(), "EN999", "8505300", "8814001", departure)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetTimetable_EmptyNumber(t *testing.T) {
	client := NewClient(WithEndpoint("http://example.invalid"))
	_, err := client.GetTimetable(context.Background(), "EN", "8505300", "8814001", time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty train number")
}

func TestGetTimetable_MissingUIC(t *testing.T) {
	client := NewClient(WithEndpoint("http://example.invalid"))
	_, err := client.GetTimetable(context.Background(), "EN400", "", "8814001", time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "origin and destination UIC are required")
}

func TestFindLegByTrainNumber(t *testing.T) {
	var resp tripsResponse
	require.NoError(t, parseJSON(tripsResponseJSON, &resp))

	// The direct EN 400 leg should be matched by number "400".
	leg, err := findLegByTrainNumber(resp.Trips.Trips, "400")
	require.NoError(t, err)
	assert.Equal(t, "PTRideLeg", leg.Typename)
	assert.Equal(t, "Lugano", leg.Start.Name)
	assert.Equal(t, "Bruxelles-Midi", leg.End.Name)
	require.Len(t, leg.ServiceJourney.ServiceProducts, 1)
	assert.Equal(t, "400", leg.ServiceJourney.ServiceProducts[0].Number)

	// IC 2 886 from the multi-leg trip should match "886".
	leg886, err := findLegByTrainNumber(resp.Trips.Trips, "886")
	require.NoError(t, err)
	assert.Equal(t, "Lugano", leg886.Start.Name)
	assert.Equal(t, "Arth-Goldau", leg886.End.Name)

	// A number that does not appear in any leg.
	_, err = findLegByTrainNumber(resp.Trips.Trips, "12345")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestLegToTrip_WithDetail(t *testing.T) {
	var tripResp tripsResponse
	require.NoError(t, parseJSON(tripsResponseJSON, &tripResp))
	leg, err := findLegByTrainNumber(tripResp.Trips.Trips, "400")
	require.NoError(t, err)

	var detailResp serviceJourneyResponse
	require.NoError(t, parseJSON(serviceJourneyResponseJSON, &detailResp))
	require.NotNil(t, detailResp.ServiceJourneyByID)

	date := time.Date(2026, 9, 25, 15, 45, 0, 0, time.FixedZone("+0200", 2*60*60))
	trip := legToTrip(leg, detailResp.ServiceJourneyByID, "EN400", date)

	require.Len(t, trip.Stops, 11)
	// Stops come from the detail payload, not the shorter leg stop list.
	assert.Equal(t, "Milano Porta Garibaldi", trip.Stops[0].StationName)
	assert.Equal(t, "Bruxelles-Midi", trip.Stops[10].StationName)
	// Swiss stop keeps its UIC.
	assert.Equal(t, 8505300, trip.Stops[2].StationUIC)
	// Not cancelled.
	assert.False(t, trip.Stops[0].Cancelled)
}

func TestLegToTrip_FallbackWithoutDetail(t *testing.T) {
	var tripResp tripsResponse
	require.NoError(t, parseJSON(tripsResponseJSON, &tripResp))
	leg, err := findLegByTrainNumber(tripResp.Trips.Trips, "400")
	require.NoError(t, err)

	date := time.Date(2026, 9, 27, 17, 44, 0, 0, time.FixedZone("+0200", 2*60*60))
	// detail == nil: falls back to the leg stop list + boundary times.
	trip := legToTrip(leg, nil, "EN400", date)

	// The leg itself only lists 9 stop points (no Milano/Como/Aarau).
	require.Len(t, trip.Stops, 9)

	// Only the leg start (Lugano, 8505300) gets a departure time.
	var lugano *traindata.Stop
	for i := range trip.Stops {
		if trip.Stops[i].StationUIC == 8505300 {
			lugano = &trip.Stops[i]
		}
	}
	require.NotNil(t, lugano)
	assert.True(t, lugano.DepartureTime.Equal(time.Date(2026, 9, 27, 17, 44, 0, 0, time.FixedZone("+0200", 2*60*60))))
	// Intermediate stops (e.g. Göschenen) have no times in fallback mode.
	var goschenen *traindata.Stop
	for i := range trip.Stops {
		if trip.Stops[i].StationUIC == 8505119 {
			goschenen = &trip.Stops[i]
		}
	}
	require.NotNil(t, goschenen)
	assert.True(t, goschenen.ArrivalTime.IsZero())
	assert.True(t, goschenen.DepartureTime.IsZero())
}

// parseJSON mirrors the machinebox/graphql client: it decodes the contents
// of the GraphQL `data` field from the captured response fixture into the
// typed struct (the outer `{"data": ...}` wrapper is stripped).
func parseJSON(src string, dst any) error {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(src), &envelope); err != nil {
		return err
	}
	return json.Unmarshal(envelope.Data, dst)
}
