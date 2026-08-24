package ns

import (
	"fmt"
	"sort"
	"sync"
)

// Zone is one pressure zone inside a water network namespace.
type Zone struct {
	ID   string
	Name string
}

// Namespace groups stations, zones and pressure targets of one district.
type Namespace struct {
	mu    sync.RWMutex
	ID    string
	Name  string
	zones map[string]Zone
}

// NewNamespace creates an empty namespace.
func NewNamespace(id, name string) *Namespace {
	return &Namespace{ID: id, Name: name, zones: map[string]Zone{}}
}

// AddZone registers a zone; duplicate ids are rejected.
func (n *Namespace) AddZone(id, name string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, exists := n.zones[id]; exists {
		return fmt.Errorf("ns: zone %s already registered", id)
	}
	n.zones[id] = Zone{ID: id, Name: name}
	return nil
}

// Zone returns one zone by id.
func (n *Namespace) Zone(id string) (Zone, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	zone, ok := n.zones[id]
	if !ok {
		return Zone{}, fmt.Errorf("ns: zone %s not found", id)
	}
	return zone, nil
}

// ZoneIDs returns the registered zone ids in stable order.
func (n *Namespace) ZoneIDs() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	ids := make([]string, 0, len(n.zones))
	for id := range n.zones {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// ZoneCount returns how many zones are registered.
func (n *Namespace) ZoneCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.zones)
}
