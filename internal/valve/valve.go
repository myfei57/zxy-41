package valve

import (
	"fmt"
	"sync"

	"waternet/internal/store"
)

// Action is the direction of one valve movement.
type Action string

const (
	ActionOpen  Action = "open"
	ActionClose Action = "close"
)

// Valve is one isolation or control valve on the network.
type Valve struct {
	mu         sync.RWMutex
	ID         string
	StationID  string
	Name       string
	Position   float64
	LastAction Action
	Zone       string
	fs         *store.FileStore
}

// NewValve creates a fully open valve.
func NewValve(id, stationID, name string, fs *store.FileStore) *Valve {
	return &Valve{ID: id, StationID: stationID, Name: name, Position: 1.0, fs: fs}
}

// PositionNow returns the current opening in 0..1.
func (v *Valve) PositionNow() float64 {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.Position
}

// SetPosition moves the valve and records the last action.
func (v *Valve) SetPosition(action Action, position float64) error {
	if position < 0 || position > 1 {
		return fmt.Errorf("valve: position %.3f out of range", position)
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.Position = position
	v.LastAction = action
	return nil
}

// SetZone changes the zone the valve currently isolates.
func (v *Valve) SetZone(zoneID string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.Zone = zoneID
}

// ZoneNow returns the zone the valve currently belongs to.
func (v *Valve) ZoneNow() string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.Zone
}

// Record returns the serializable form of the valve.
func (v *Valve) Record() store.ValveRecord {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return store.ValveRecord{
		ID:        v.ID,
		Station:   v.StationID,
		Name:      v.Name,
		Position:  v.Position,
		Direction: string(v.LastAction),
		Zone:      v.Zone,
	}
}

// Restore rebuilds a valve from a persisted record.
func (v *Valve) Restore(record store.ValveRecord) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.ID = record.ID
	v.StationID = record.Station
	v.Name = record.Name
	v.Position = record.Position
	v.LastAction = Action(record.Direction)
	v.Zone = record.Zone
	return nil
}

// Save persists the valve state.
func (v *Valve) Save() error {
	if v.fs == nil {
		return fmt.Errorf("valve: %s has no store", v.ID)
	}
	return v.fs.WriteJSON("valves/"+v.ID+".json", v.Record())
}
