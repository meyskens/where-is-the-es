package v1

import (
	"testing"
	"time"

	"github.com/meyskens/where-is-the-es/pkg/traindata"
)

func TestPreserveRealtimeStops_UsesPreviousWhenCurrentHasNoRealtime(t *testing.T) {
	scheduledArrival := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	scheduledDeparture := time.Date(2026, 4, 25, 10, 5, 0, 0, time.UTC)
	oldRealArrival := time.Date(2026, 4, 25, 10, 12, 0, 0, time.UTC)
	oldRealDeparture := time.Date(2026, 4, 25, 10, 17, 0, 0, time.UTC)

	current := &traindata.Trip{Stops: []traindata.Stop{
		{
			StationUIC:        8000105,
			StationName:       "Berlin Hbf",
			ArrivalTime:       scheduledArrival,
			DepartureTime:     scheduledDeparture,
			Platform:          "4",
			RealArrivalTime:   scheduledArrival,
			RealDepartureTime: scheduledDeparture,
			RealPlatform:      "4",
			IsRealTime:        false,
		},
	}}

	previous := &traindata.Trip{Stops: []traindata.Stop{
		{
			StationUIC:        8000105,
			StationName:       "Berlin Hbf",
			ArrivalTime:       scheduledArrival,
			DepartureTime:     scheduledDeparture,
			Platform:          "4",
			RealArrivalTime:   oldRealArrival,
			RealDepartureTime: oldRealDeparture,
			RealPlatform:      "6",
			IsRealTime:        true,
		},
	}}

	preserveRealtimeStops(current, previous)

	stop := current.Stops[0]
	if !stop.RealArrivalTime.Equal(oldRealArrival) {
		t.Fatalf("expected RealArrivalTime %v, got %v", oldRealArrival, stop.RealArrivalTime)
	}
	if !stop.RealDepartureTime.Equal(oldRealDeparture) {
		t.Fatalf("expected RealDepartureTime %v, got %v", oldRealDeparture, stop.RealDepartureTime)
	}
	if stop.RealPlatform != "6" {
		t.Fatalf("expected RealPlatform 6, got %q", stop.RealPlatform)
	}
	if !stop.IsRealTime {
		t.Fatalf("expected IsRealTime true after merge")
	}
}

func TestPreserveRealtimeStops_KeepsCurrentWhenCurrentHasRealtime(t *testing.T) {
	scheduled := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	currentReal := time.Date(2026, 4, 25, 10, 8, 0, 0, time.UTC)
	previousReal := time.Date(2026, 4, 25, 10, 12, 0, 0, time.UTC)

	current := &traindata.Trip{Stops: []traindata.Stop{
		{
			StationName:     "Leipzig Hbf",
			ArrivalTime:     scheduled,
			RealArrivalTime: currentReal,
			Platform:        "7",
			RealPlatform:    "7a",
			IsRealTime:      true,
		},
	}}

	previous := &traindata.Trip{Stops: []traindata.Stop{
		{
			StationName:     "Leipzig Hbf",
			ArrivalTime:     scheduled,
			RealArrivalTime: previousReal,
			Platform:        "7",
			RealPlatform:    "9",
			IsRealTime:      true,
		},
	}}

	preserveRealtimeStops(current, previous)

	stop := current.Stops[0]
	if !stop.RealArrivalTime.Equal(currentReal) {
		t.Fatalf("expected current realtime to be kept, got %v", stop.RealArrivalTime)
	}
	if stop.RealPlatform != "7a" {
		t.Fatalf("expected current realtime platform to be kept, got %q", stop.RealPlatform)
	}
}
