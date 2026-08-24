package store

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

const cumulativeFile = "cumulative.json"

// CumulativeLedger tracks per-meter cumulative flow and the base value at the
// last direction switch. The reported segment total is total minus base, so a
// direction switch restarts the accounting without losing the lifetime volume.
type CumulativeLedger struct {
	mu         sync.Mutex
	totals     map[string]float64
	bases      map[string]float64
	directions map[string]string
	fs         *FileStore
}

// NewCumulativeLedger creates an in-memory ledger backed by the given store.
func NewCumulativeLedger(fs *FileStore) *CumulativeLedger {
	return &CumulativeLedger{
		totals:     map[string]float64{},
		bases:      map[string]float64{},
		directions: map[string]string{},
		fs:         fs,
	}
}

// Add accumulates one flow delta for a meter.
func (l *CumulativeLedger) Add(meterID string, delta float64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.totals[meterID] += delta
	if l.directions[meterID] == "" {
		l.directions[meterID] = "forward"
	}
}

// SwitchBase records the current total as the new segment base, which resets
// the reported segment total to zero for the new direction.
func (l *CumulativeLedger) SwitchBase(meterID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	base, ok := l.totals[meterID]
	if !ok {
		return fmt.Errorf("%s: %w", meterID, ErrNotFound)
	}
	l.bases[meterID] = base
	return nil
}

// SetDirection records the current flow direction of a meter.
func (l *CumulativeLedger) SetDirection(meterID, direction string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.directions[meterID] = direction
}

// Direction returns the current direction of a meter.
func (l *CumulativeLedger) Direction(meterID string) string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.directions[meterID]
}

// SegmentTotal reports the accumulated flow since the last direction switch.
func (l *CumulativeLedger) SegmentTotal(meterID string) (float64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	total, ok := l.totals[meterID]
	if !ok {
		return 0, fmt.Errorf("%s: %w", meterID, ErrNotFound)
	}
	return total - l.bases[meterID], nil
}

// Total reports the lifetime cumulative flow of a meter.
func (l *CumulativeLedger) Total(meterID string) (float64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	total, ok := l.totals[meterID]
	if !ok {
		return 0, fmt.Errorf("%s: %w", meterID, ErrNotFound)
	}
	return total, nil
}

// Save persists the ledger state.
func (l *CumulativeLedger) Save() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	meterIDs := make([]string, 0, len(l.totals))
	for meterID := range l.totals {
		meterIDs = append(meterIDs, meterID)
	}
	sort.Strings(meterIDs)
	records := make([]CumulativeRecord, 0, len(meterIDs))
	for _, meterID := range meterIDs {
		records = append(records, CumulativeRecord{
			MeterID:   meterID,
			Base:      l.bases[meterID],
			Total:     l.totals[meterID],
			Direction: l.directions[meterID],
		})
	}
	return l.fs.WriteJSON(cumulativeFile, records)
}

// Load restores the ledger from disk.
func (l *CumulativeLedger) Load() error {
	var records []CumulativeRecord
	if err := l.fs.ReadJSON(cumulativeFile, &records); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, record := range records {
		l.totals[record.MeterID] = record.Total
		l.bases[record.MeterID] = record.Base
		l.directions[record.MeterID] = record.Direction
	}
	return nil
}
