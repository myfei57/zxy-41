package pressure

import (
	"fmt"
	"time"

	"waternet/internal/store"
)

// Window is a half-open time range [Start, End) used for pressure aggregation.
type Window struct {
	Zone  string
	Start time.Time
	End   time.Time
}

// NewWindow creates a half-open window.
func NewWindow(zone string, start, end time.Time) Window {
	return Window{Zone: zone, Start: start, End: end}
}

// Contains reports whether a timestamp falls inside the window.
func (w Window) Contains(ts time.Time) bool {
	return !ts.Before(w.Start) && ts.Before(w.End)
}

// Duration returns the window length.
func (w Window) Duration() time.Duration {
	return w.End.Sub(w.Start)
}

// Slot returns the minute slot used to bucket readings.
func (w Window) Slot() string {
	return w.Start.UTC().Format("2006-01-02T15:04")
}

// WindowStore reads persisted readings for a window.
type WindowStore struct {
	agg *store.WindowAggregator
}

// NewWindowStore creates a window store over an aggregator.
func NewWindowStore(agg *store.WindowAggregator) *WindowStore {
	return &WindowStore{agg: agg}
}

// Readings returns the persisted samples inside a window.
func (s *WindowStore) Readings(zone string, window Window) ([]store.ReadingRecord, error) {
	records, err := s.agg.Load()
	if err != nil {
		return nil, err
	}
	var out []store.ReadingRecord
	for _, record := range records {
		at, err := time.Parse(time.RFC3339Nano, record.At)
		if err != nil {
			continue
		}
		if record.Zone == zone && window.Contains(at) {
			out = append(out, record)
		}
	}
	return out, nil
}

// Peak returns the highest pressure inside a window.
func (s *WindowStore) Peak(zone string, window Window) (float64, error) {
	records, err := s.Readings(zone, window)
	if err != nil {
		return 0, err
	}
	if len(records) == 0 {
		return 0, fmt.Errorf("pressure: no readings in window")
	}
	peak := records[0].MPA
	for _, record := range records[1:] {
		if record.MPA > peak {
			peak = record.MPA
		}
	}
	return peak, nil
}

// Record persists one reading inside a window.
func (s *WindowStore) Record(record store.ReadingRecord) error {
	return s.agg.Append(record)
}
