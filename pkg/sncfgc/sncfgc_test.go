package sncfgc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-key")
	assert.NotNil(t, client)
	assert.Equal(t, "test-key", client.subscriptionKey)
	assert.NotNil(t, client.httpClient)
	assert.NotNil(t, client.limiter)
}

func TestParseSNCFCGTime(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedUnix int64
		isZero       bool
	}{
		{
			name:         "valid RFC3339 time",
			input:        "2026-05-31T09:09:00+00:00",
			expectedUnix: time.Date(2026, 5, 31, 9, 9, 0, 0, time.UTC).Unix(),
			isZero:       false,
		},
		{
			name:   "empty string",
			input:  "",
			isZero: true,
		},
		{
			name:   "invalid format",
			input:  "invalid-time",
			isZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSNCFCGTime(tt.input)
			if tt.isZero {
				assert.True(t, result.IsZero())
			} else {
				assert.Equal(t, tt.expectedUnix, result.Unix())
			}
		})
	}
}

func TestGetTimetable(t *testing.T) {
	// Sample response from the SNCF Gares & Connexions API
	sampleResponse := `[
  {
    "shortTermInformations": [],
    "composition": null,
    "listeReservations": null,
    "listServicesABord": [],
    "brand": null,
    "stationName": "Paris Gare du Nord",
    "stopoverNumber": "0",
    "isLeft": false,
    "trainLength": null,
    "JourneyStop": [
      {
        "scheduledTime": "2026-05-31T09:09:00+00:00",
        "actualTime": "2026-05-31T09:59:00+00:00",
        "informationStatus": {
          "trainStatus": "RETARD",
          "eventLevel": "Information",
          "delay": 50
        },
        "platform": {
          "track": "2",
          "isTrackactive": true,
          "trackGroupTitle": "Hall",
          "trackGroupValue": "2",
          "backgroundColor": null,
          "trackPosition": null
        },
        "statusModification": null,
        "shortTermInformations": [],
        "stationName": "Paris Gare du Nord",
        "downtime": null,
        "theoreticalOccupancy": null,
        "realOccupancy": null,
        "uic": "0087271007",
        "isNewOrigin": false,
        "isNewDestination": false
      },
      {
        "scheduledTime": "2026-05-31T11:30:00+00:00",
        "actualTime": "2026-05-31T12:20:00+00:00",
        "informationStatus": {
          "trainStatus": "Ontime",
          "eventLevel": "Information",
          "delay": null
        },
        "platform": {
          "track": "",
          "isTrackactive": false,
          "trackGroupTitle": null,
          "trackGroupValue": null,
          "backgroundColor": null,
          "trackPosition": null
        },
        "statusModification": null,
        "shortTermInformations": [],
        "stationName": "London St-Pancras",
        "downtime": null,
        "theoreticalOccupancy": null,
        "realOccupancy": null,
        "uic": "0070154005",
        "isNewOrigin": false,
        "isNewDestination": false
      }
    ],
    "direction": "Departure",
    "trainNumber": "9023",
    "scheduledTime": "2026-05-31T09:09:00+00:00",
    "actualTime": "2026-05-31T09:59:00+00:00",
    "trainType": "Eurostar",
    "trainMode": "TRAIN",
    "platform": {
      "track": "",
      "isTrackactive": true,
      "trackGroupTitle": "Hall",
      "trackGroupValue": "2",
      "backgroundColor": null,
      "trackPosition": null
    },
    "informationStatus": {
      "trainStatus": "RETARD",
      "eventLevel": "Information",
      "delay": 50
    },
    "traffic": {
      "origin": "Paris Gare du Nord",
      "destination": "London St-Pancras",
      "oldOrigin": "",
      "oldDestination": "",
      "eventStatus": "Ontime",
      "eventLevel": "Information"
    },
    "statusModification": null,
    "TrafficDetailsUrl": "https://www.sncf-voyageurs.com/...",
    "uic": "0087271007",
    "missionCode": null,
    "trainLine": null,
    "isGL": false,
    "presentation": {
      "colorCode": "#1d1e27",
      "textColorCode": "#FFFFFF"
    },
    "stops": [
      "Paris Gare du Nord",
      "London St-Pancras"
    ],
    "alternativeMeans": null
  }
]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		assert.Equal(t, "test-subscription-key", r.Header.Get("Ocp-Apim-Subscription-Key"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		// Verify query parameters
		query := r.URL.Query()
		assert.Equal(t, "9023", query.Get("trainNumber"))
		assert.Equal(t, "0087271007", query.Get("uic"))
		assert.Equal(t, "true", query.Get("isDeparture"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sampleResponse))
	}))
	defer server.Close()

	client := NewClient("test-subscription-key", WithBaseURL(server.URL))

	date := time.Date(2026, 5, 31, 9, 9, 0, 0, time.UTC)
	trip, err := client.GetTimetable(context.Background(), "9023", "0087271007", date, true)

	require.NoError(t, err)
	require.NotNil(t, trip)
	assert.Equal(t, "9023", trip.TrainNumber)
	assert.Len(t, trip.Stops, 2)

	// Check first stop (Paris Gare du Nord)
	stop1 := trip.Stops[0]
	assert.Equal(t, "Paris Gare du Nord", stop1.StationName)
	assert.Equal(t, 87271007, stop1.StationUIC)
	assert.Equal(t, "2", stop1.Platform)
	assert.Equal(t, "2", stop1.RealPlatform)
	assert.True(t, stop1.IsRealTime)

	// Check second stop (London St-Pancras)
	stop2 := trip.Stops[1]
	assert.Equal(t, "London St-Pancras", stop2.StationName)
	assert.Equal(t, 70154005, stop2.StationUIC)
}

func TestGetTimetable_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	date := time.Date(2026, 5, 31, 9, 9, 0, 0, time.UTC)

	_, err := client.GetTimetable(context.Background(), "9999", "0087271007", date, true)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetTimetable_EmptyTrainNumber(t *testing.T) {
	client := NewClient("test-key")
	date := time.Date(2026, 5, 31, 9, 9, 0, 0, time.UTC)

	_, err := client.GetTimetable(context.Background(), "", "0087271007", date, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty train number")
}

func TestGetTimetable_EmptyUIC(t *testing.T) {
	client := NewClient("test-key")
	date := time.Date(2026, 5, 31, 9, 9, 0, 0, time.UTC)

	_, err := client.GetTimetable(context.Background(), "9023", "", date, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty UIC station code")
}

func TestGetTimetable_Cancelled(t *testing.T) {
	// Sample response from the SNCF Gares & Connexions API for a partially
	// cancelled European Sleeper train (SUPPRESSION_PARTIELLE).
	sampleResponse := `[
  {
    "shortTermInformations": [],
    "composition": null,
    "listeReservations": null,
    "listServicesABord": [],
    "brand": null,
    "stationName": "Paris Gare du Nord",
    "stopoverNumber": "4",
    "isLeft": false,
    "trainLength": null,
    "JourneyStop": [
      {
        "scheduledTime": "2026-06-28T16:00:00+00:00",
        "actualTime": null,
        "informationStatus": {
          "trainStatus": "SUPPRESSION_PARTIELLE",
          "eventLevel": "Warning",
          "delay": null
        },
        "platform": {
          "track": "",
          "isTrackactive": false,
          "trackGroupTitle": null,
          "trackGroupValue": null,
          "backgroundColor": null,
          "trackPosition": null
        },
        "statusModification": null,
        "shortTermInformations": [],
        "stationName": "Paris Gare du Nord",
        "downtime": null,
        "theoreticalOccupancy": null,
        "realOccupancy": null,
        "uic": "0087271007",
        "isNewOrigin": false,
        "isNewDestination": false
      },
      {
        "scheduledTime": "2026-06-28T18:18:00+00:00",
        "actualTime": "2026-06-28T20:18:00+00:00",
        "informationStatus": {
          "trainStatus": "SUPPRESSION_PARTIELLE",
          "eventLevel": "Warning",
          "delay": null
        },
        "platform": {
          "track": "",
          "isTrackactive": false,
          "trackGroupTitle": null,
          "trackGroupValue": null,
          "backgroundColor": null,
          "trackPosition": null
        },
        "statusModification": null,
        "shortTermInformations": [],
        "stationName": "Aulnoye-Aymeries",
        "downtime": 900,
        "theoreticalOccupancy": null,
        "realOccupancy": null,
        "uic": "0087295600",
        "isNewOrigin": false,
        "isNewDestination": false
      },
      {
        "scheduledTime": "2026-06-28T07:00:00+00:00",
        "actualTime": "2026-06-28T09:00:00+00:00",
        "informationStatus": {
          "trainStatus": "SUPPRESSION_PARTIELLE",
          "eventLevel": "Warning",
          "delay": null
        },
        "platform": {
          "track": "",
          "isTrackactive": false,
          "trackGroupTitle": null,
          "trackGroupValue": null,
          "backgroundColor": null,
          "trackPosition": null
        },
        "statusModification": null,
        "shortTermInformations": [],
        "stationName": "Berlin-Gesundbrunnen",
        "downtime": null,
        "theoreticalOccupancy": null,
        "realOccupancy": null,
        "uic": "0080077990",
        "isNewOrigin": false,
        "isNewDestination": true
      }
    ],
    "direction": "Departure",
    "trainNumber": "475",
    "scheduledTime": "2026-06-28T16:00:00+00:00",
    "actualTime": "2026-06-28T18:00:00+00:00",
    "trainType": "European Sleeper",
    "trainMode": "TRAIN",
    "platform": {
      "track": "",
      "isTrackactive": false,
      "trackGroupTitle": null,
      "trackGroupValue": null,
      "backgroundColor": null,
      "trackPosition": null
    },
    "informationStatus": {
      "trainStatus": "SUPPRESSION_PARTIELLE",
      "eventLevel": "Warning",
      "delay": null
    },
    "traffic": {
      "origin": "Paris Gare du Nord",
      "destination": "Berlin-Gesundbrunnen",
      "oldOrigin": "",
      "oldDestination": "",
      "eventStatus": "SUPPRESSION",
      "eventLevel": "Warning"
    },
    "statusModification": null,
    "TrafficDetailsUrl": "https://www.sncf-voyageurs.com/...",
    "uic": "0087271007",
    "missionCode": null,
    "trainLine": null,
    "isGL": false,
    "presentation": {
      "colorCode": "#1d1e27",
      "textColorCode": "#FFFFFF"
    },
    "stops": [
      "Paris Gare du Nord",
      "Aulnoye-Aymeries",
      "Berlin-Gesundbrunnen"
    ],
    "alternativeMeans": null
  }
]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sampleResponse))
	}))
	defer server.Close()

	client := NewClient("test-subscription-key", WithBaseURL(server.URL))

	date := time.Date(2026, 6, 28, 16, 0, 0, 0, time.UTC)
	trip, err := client.GetTimetable(context.Background(), "475", "0087271007", date, true)

	require.NoError(t, err)
	require.NotNil(t, trip)
	assert.Equal(t, "475", trip.TrainNumber)
	assert.Len(t, trip.Stops, 3)

	// Every stop should be marked as cancelled because the trainStatus is
	// SUPPRESSION_PARTIELLE on each JourneyStop.
	for i, stop := range trip.Stops {
		assert.Truef(t, stop.Cancelled, "stop %d (%s) should be cancelled", i, stop.StationName)
		assert.Truef(t, stop.IsRealTime, "stop %d (%s) should be real-time", i, stop.StationName)
	}
}

func TestIsSuppressionStatus(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{"SUPPRESSION", true},
		{"SUPPRESSION_PARTIELLE", true},
		{"suppression_partielle", true},
		{"  Suppression ", true},
		{"RETARD", false},
		{"Ontime", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			assert.Equal(t, tt.expected, isSuppressionStatus(tt.status))
		})
	}
}

func TestTrainDetailsResponseParsing(t *testing.T) {
	jsonData := `{
		"trainNumber": "475",
		"trainType": "European Sleeper",
		"stationName": "Paris Gare du Nord",
		"uic": "0087271007",
		"JourneyStop": [
			{
				"stationName": "Paris Gare du Nord",
				"uic": "0087271007",
				"scheduledTime": "2026-05-31T16:13:00+00:00",
				"actualTime": "2026-05-31T16:13:00+00:00",
				"informationStatus": {
					"trainStatus": "Ontime",
					"delay": null
				},
				"platform": {
					"track": "",
					"isTrackactive": false
				}
			}
		]
	}`

	var response TrainDetailsResponse
	err := json.Unmarshal([]byte(jsonData), &response)
	require.NoError(t, err)

	assert.Equal(t, "475", response.TrainNumber)
	assert.Equal(t, "European Sleeper", response.TrainType)
	assert.Len(t, response.JourneyStop, 1)
	assert.Equal(t, "Paris Gare du Nord", response.JourneyStop[0].StationName)
}
