package pressure

import (
	"fmt"
	"sort"
	"sync"
)

// ZoneMap keeps the current zone attribution of every pressure point. It is
// updated when an isolation valve switches zones so readings never cross the
// isolation boundary.
type ZoneMap struct {
	mu          sync.RWMutex
	attribution map[string]string
}

// NewZoneMap creates an empty zone map.
func NewZoneMap() *ZoneMap {
	return &ZoneMap{attribution: map[string]string{}}
}

// AttachZone attributes a valve to a zone immediately.
func (m *ZoneMap) AttachZone(valveID, zoneID string) error {
	if zoneID == "" {
		return fmt.Errorf("pressure: cannot attach %s to empty zone", valveID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attribution[valveID] = zoneID
	return nil
}

// DetachZone removes the attribution of a valve.
func (m *ZoneMap) DetachZone(valveID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.attribution, valveID)
}

// ZoneFor returns the current zone of a valve.
func (m *ZoneMap) ZoneFor(valveID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.attribution[valveID]
}

// Snapshot returns a copy of the whole attribution table.
func (m *ZoneMap) Snapshot() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]string, len(m.attribution))
	for valveID, zoneID := range m.attribution {
		out[valveID] = zoneID
	}
	return out
}

// Restore replaces the attribution table, used on recovery.
func (m *ZoneMap) Restore(attribution map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attribution = make(map[string]string, len(attribution))
	for valveID, zoneID := range attribution {
		m.attribution[valveID] = zoneID
	}
}

// ValveIDs returns the attributed valve ids in stable order.
func (m *ZoneMap) ValveIDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.attribution))
	for valveID := range m.attribution {
		ids = append(ids, valveID)
	}
	sort.Strings(ids)
	return ids
}
