package quota

import (
	"errors"
	"fmt"
	"sync"

	"waternet/internal/store"
)

// ErrQuotaExceeded is returned when a zone has consumed its sampling quota.
var ErrQuotaExceeded = errors.New("quota: zone sampling quota exceeded")

const quotaFile = "quota.json"

// Ledger tracks per-zone sampling usage against a hard limit.
type Ledger struct {
	mu     sync.Mutex
	used   map[string]int
	limit  int
	window string
	fs     *store.FileStore
}

// NewLedger creates a quota ledger with the given per-zone limit.
func NewLedger(fs *store.FileStore, limit int) *Ledger {
	return &Ledger{
		used:   map[string]int{},
		limit:  limit,
		window: "current",
		fs:     fs,
	}
}

// TryConsume reserves n samples for a zone when the limit allows it.
func (l *Ledger) TryConsume(zone string, n int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.used[zone]+n > l.limit {
		return fmt.Errorf("%w: zone %s used %d limit %d", ErrQuotaExceeded, zone, l.used[zone], l.limit)
	}
	l.used[zone] += n
	return nil
}

// Release returns n samples to a zone, never going below zero.
func (l *Ledger) Release(zone string, n int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.used[zone] >= n {
		l.used[zone] -= n
	} else {
		l.used[zone] = 0
	}
}

// Used returns how many samples a zone has consumed.
func (l *Ledger) Used(zone string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.used[zone]
}

// Limit returns the configured per-zone sampling limit.
func (l *Ledger) Limit() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.limit
}

// Save persists the ledger.
func (l *Ledger) Save() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	type zoneRow struct {
		Zone string `json:"zone"`
		Used int    `json:"used"`
	}
	rows := make([]zoneRow, 0, len(l.used))
	for zone, used := range l.used {
		rows = append(rows, zoneRow{Zone: zone, Used: used})
	}
	payload := struct {
		Limit  int       `json:"limit"`
		Window string    `json:"window"`
		Zones  []zoneRow `json:"zones"`
	}{Limit: l.limit, Window: l.window, Zones: rows}
	return l.fs.WriteJSON(quotaFile, payload)
}

// Load restores the ledger from disk.
func (l *Ledger) Load() error {
	var payload struct {
		Limit  int    `json:"limit"`
		Window string `json:"window"`
		Zones  []struct {
			Zone string `json:"zone"`
			Used int    `json:"used"`
		} `json:"zones"`
	}
	if err := l.fs.ReadJSON(quotaFile, &payload); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.limit = payload.Limit
	l.window = payload.Window
	l.used = map[string]int{}
	for _, row := range payload.Zones {
		l.used[row.Zone] = row.Used
	}
	return nil
}
