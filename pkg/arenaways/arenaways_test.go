package arenaways

import (
	_ "embed"
	"testing"
	"time"

	"github.com/meyskens/where-is-the-es/pkg/traindata"
	"github.com/stretchr/testify/assert"
)

//go:embed testdata_train400.html
var testTrain400HTML string

func mustParseTime(t *testing.T, timeStr string) time.Time {
	t.Helper()
	parsedTime, err := time.Parse("15:04", timeStr)
	if err != nil {
		t.Fatalf("failed to parse time %q: %v", timeStr, err)
	}
	return parsedTime
}

func TestParseTimetable_Train400(t *testing.T) {
	f := &ArenaWaysFetcher{}

	stops, err := f.ParseTimetable([]byte(testTrain400HTML))
	if err != nil {
		t.Fatalf("ParseTimetable() unexpected error: %v", err)
	}

	expected := []traindata.Stop{
		{
			StationName:       "MILANO PORTA GARIBALDI",
			ArrivalTime:       time.Time{},
			RealArrivalTime:   time.Time{},
			DepartureTime:     mustParseTime(t, "15:45"),
			RealDepartureTime: mustParseTime(t, "15:48"),
			IsRealTime:        true,
		},
		{
			StationName:       "COMO S.GIOVANNI",
			ArrivalTime:       mustParseTime(t, "16:55"),
			RealArrivalTime:   mustParseTime(t, "16:56"),
			DepartureTime:     mustParseTime(t, "16:57"),
			RealDepartureTime: mustParseTime(t, "17:01"),
			IsRealTime:        true,
		},
		{
			StationName:       "CHIASSO",
			ArrivalTime:       mustParseTime(t, "17:04"),
			RealArrivalTime:   mustParseTime(t, "17:06"),
			DepartureTime:     mustParseTime(t, "18:16"),
			RealDepartureTime: mustParseTime(t, "18:18"),
			IsRealTime:        true,
		},
		{
			StationName:       "LUGANO",
			ArrivalTime:       mustParseTime(t, "18:42"),
			RealArrivalTime:   mustParseTime(t, "18:42"),
			DepartureTime:     mustParseTime(t, "18:44"),
			RealDepartureTime: mustParseTime(t, "18:44"),
			IsRealTime:        true,
		},
		{
			StationName:       "BELLINZONA",
			ArrivalTime:       mustParseTime(t, "19:21"),
			RealArrivalTime:   mustParseTime(t, "19:21"),
			DepartureTime:     mustParseTime(t, "19:23"),
			RealDepartureTime: mustParseTime(t, "19:23"),
			IsRealTime:        true,
		},
		{
			StationName:       "GOSCHENEN",
			ArrivalTime:       mustParseTime(t, "20:30"),
			RealArrivalTime:   mustParseTime(t, "20:30"),
			DepartureTime:     mustParseTime(t, "20:52"),
			RealDepartureTime: mustParseTime(t, "20:52"),
			IsRealTime:        true,
		},
		{
			StationName:       "ARTH GOLDAU",
			ArrivalTime:       mustParseTime(t, "21:49"),
			RealArrivalTime:   mustParseTime(t, "21:49"),
			DepartureTime:     mustParseTime(t, "22:00"),
			RealDepartureTime: mustParseTime(t, "22:00"),
			IsRealTime:        true,
		},
		{
			StationName:       "AARAU",
			ArrivalTime:       mustParseTime(t, "22:55"),
			RealArrivalTime:   mustParseTime(t, "22:55"),
			DepartureTime:     mustParseTime(t, "22:58"),
			RealDepartureTime: mustParseTime(t, "22:58"),
			IsRealTime:        true,
		},
		{
			StationName:       "AACHEN HBF",
			ArrivalTime:       mustParseTime(t, "08:48"),
			RealArrivalTime:   mustParseTime(t, "08:48"),
			DepartureTime:     mustParseTime(t, "08:55"),
			RealDepartureTime: mustParseTime(t, "08:55"),
			IsRealTime:        false,
		},
		{
			StationName:       "VERVIES",
			ArrivalTime:       mustParseTime(t, "09:18"),
			RealArrivalTime:   mustParseTime(t, "09:18"),
			DepartureTime:     mustParseTime(t, "09:24"),
			RealDepartureTime: mustParseTime(t, "09:24"),
			IsRealTime:        false,
		},
		{
			StationName:       "BRUXELLES-MIDI",
			ArrivalTime:       mustParseTime(t, "11:42"),
			RealArrivalTime:   mustParseTime(t, "11:42"),
			DepartureTime:     time.Time{},
			RealDepartureTime: time.Time{},
			IsRealTime:        false,
		},
	}

	assert.Equal(t, len(expected), len(stops), "expected %d stops, got %d", len(expected), len(stops))
	if len(stops) != len(expected) {
		t.Fatalf("stopping comparison — stop count mismatch")
	}

	for i, want := range expected {
		got := stops[i]
		assert.Equal(t, want.StationName, got.StationName, "stop %d (%s) StationName", i, want.StationName)
		assert.True(t, got.ArrivalTime.Equal(want.ArrivalTime), "stop %d (%s) ArrivalTime: want %v, got %v", i, want.StationName, want.ArrivalTime, got.ArrivalTime)
		assert.True(t, got.RealArrivalTime.Equal(want.RealArrivalTime), "stop %d (%s) RealArrivalTime: want %v, got %v", i, want.StationName, want.RealArrivalTime, got.RealArrivalTime)
		assert.True(t, got.DepartureTime.Equal(want.DepartureTime), "stop %d (%s) DepartureTime: want %v, got %v", i, want.StationName, want.DepartureTime, got.DepartureTime)
		assert.True(t, got.RealDepartureTime.Equal(want.RealDepartureTime), "stop %d (%s) RealDepartureTime: want %v, got %v", i, want.StationName, want.RealDepartureTime, got.RealDepartureTime)
		assert.Equal(t, want.IsRealTime, got.IsRealTime, "stop %d (%s) IsRealTime", i, want.StationName)
	}
}

func TestParseTimetable_EmptyHTML(t *testing.T) {
	f := &ArenaWaysFetcher{}

	_, err := f.ParseTimetable([]byte("<div></div>"))
	if err == nil {
		t.Fatal("expected error on empty HTML, got nil")
	}
	assert.ErrorIs(t, err, ErrNoTrain)
}

func TestParseTimetable_FirstAndLastStopTimes(t *testing.T) {
	// A minimal HTML snippet with two stops: the first should have its
	// arrival cleared, the last should have its departure cleared.
	html := `<div class="relative flex items-start">
		<div class="mb-4 ml-4 flex-1 border-b pb-4">
			<p class="pb-6 font-semibold">FIRST</p>
			<div class="mb-2 flex justify-between gap-4 md:grid md:grid-cols-3">
				<p class="text-sm">Arrivo: 10:00</p>
				<p class="text-sm">Partenza: 10:05</p>
			</div>
		</div>
	</div>
	<div class="relative flex items-start">
		<div class="mb-4 ml-4 flex-1 border-b pb-4">
			<p class="pb-6 font-semibold">LAST</p>
			<div class="mb-2 flex justify-between gap-4 md:grid md:grid-cols-3">
				<p class="text-sm">Arrivo: 11:00</p>
				<p class="text-sm">Partenza: 11:05</p>
			</div>
		</div>
	</div>`

	f := &ArenaWaysFetcher{}
	stops, err := f.ParseTimetable([]byte(html))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stops) != 2 {
		t.Fatalf("expected 2 stops, got %d", len(stops))
	}

	// First stop: arrival should be zero, departure should be set.
	if !stops[0].ArrivalTime.IsZero() {
		t.Errorf("first stop arrival should be zero, got %v", stops[0].ArrivalTime)
	}
	if stops[0].DepartureTime.IsZero() {
		t.Errorf("first stop departure should not be zero")
	}

	// Last stop: arrival should be set, departure should be zero.
	if stops[1].ArrivalTime.IsZero() {
		t.Errorf("last stop arrival should not be zero")
	}
	if !stops[1].DepartureTime.IsZero() {
		t.Errorf("last stop departure should be zero, got %v", stops[1].DepartureTime)
	}
}
