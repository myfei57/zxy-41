package store

import (
	"encoding/json"
	"fmt"
)

// ReadingRecord is one persisted pressure sample.
type ReadingRecord struct {
	Zone  string  `json:"zone"`
	Slot  string  `json:"slot"`
	Valve string  `json:"valve"`
	MPA   float64 `json:"mpa"`
	Flow  float64 `json:"flow"`
	At    string  `json:"at"`
}

// WindowAggregator persists pressure samples as an append-only log and
// recomputes window aggregates from the raw rows.
type WindowAggregator struct {
	fs *FileStore
}

// NewWindowAggregator creates an aggregator backed by the given store.
func NewWindowAggregator(fs *FileStore) *WindowAggregator {
	return &WindowAggregator{fs: fs}
}

const readingsLog = "readings.jsonl"

// Append persists one pressure sample.
func (a *WindowAggregator) Append(record ReadingRecord) error {
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("aggregate: marshal: %w", err)
	}
	return a.fs.AppendLine(readingsLog, data)
}

// Load returns every persisted sample.
func (a *WindowAggregator) Load() ([]ReadingRecord, error) {
	lines, err := a.fs.ReadLines(readingsLog)
	if err != nil {
		return nil, err
	}
	records := make([]ReadingRecord, 0, len(lines))
	for _, line := range lines {
		var record ReadingRecord
		if err := json.Unmarshal(line, &record); err != nil {
			return nil, fmt.Errorf("aggregate: decode row: %w", err)
		}
		records = append(records, record)
	}
	return records, nil
}

// Peak returns the highest pressure seen for a zone inside one window slot.
func (a *WindowAggregator) Peak(zone, slot string) (float64, error) {
	records, err := a.Load()
	if err != nil {
		return 0, err
	}
	var peak float64
	found := false
	for _, record := range records {
		if record.Zone != zone || record.Slot != slot {
			continue
		}
		if !found || record.MPA > peak {
			peak = record.MPA
			found = true
		}
	}
	if !found {
		return 0, fmt.Errorf("%w: no reading in %s/%s", ErrNotFound, zone, slot)
	}
	return peak, nil
}

// Count returns how many samples were stored for a zone in a slot.
func (a *WindowAggregator) Count(zone, slot string) (int, error) {
	records, err := a.Load()
	if err != nil {
		return 0, err
	}
	count := 0
	for _, record := range records {
		if record.Zone == zone && record.Slot == slot {
			count++
		}
	}
	return count, nil
}

// ZoneSlots returns the distinct slots seen for a zone, newest first.
func (a *WindowAggregator) ZoneSlots(zone string) ([]string, error) {
	records, err := a.Load()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var slots []string
	for _, record := range records {
		if record.Zone != zone || seen[record.Slot] {
			continue
		}
		seen[record.Slot] = true
		slots = append(slots, record.Slot)
	}
	return slots, nil
}
