package pump

import (
	"fmt"
	"sync"

	"waternet/internal/store"
)

// Status is the operational state of one pump unit.
type Status string

const (
	StatusRunning     Status = "running"
	StatusStandby     Status = "standby"
	StatusMaintenance Status = "maintenance"
	StatusStopped     Status = "stopped"
)

// Unit is one physical pump inside a group.
type Unit struct {
	ID        string
	Name      string
	Status    Status
	Locked    bool
	Vibration float64
}

// Group is a set of pump units that serve one discharge header. Exactly one
// unit is expected to run per group under normal scheduling.
type Group struct {
	mu           sync.RWMutex
	ID           string
	StationID    string
	Name         string
	Units        []*Unit
	ActiveUnit   string
	DischargeMPA float64
	TargetMPA    float64
	fs           *store.FileStore
}

// NewGroup creates a pump group without units.
func NewGroup(id, stationID, name string, fs *store.FileStore) *Group {
	return &Group{ID: id, StationID: stationID, Name: name, fs: fs}
}

// AddUnit registers a unit in standby state and returns it.
func (g *Group) AddUnit(id, name string) *Unit {
	g.mu.Lock()
	defer g.mu.Unlock()
	unit := &Unit{ID: id, Name: name, Status: StatusStandby}
	g.Units = append(g.Units, unit)
	return unit
}

// Unit returns one unit by id.
func (g *Group) Unit(id string) (*Unit, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, unit := range g.Units {
		if unit.ID == id {
			return unit, nil
		}
	}
	return nil, fmt.Errorf("pump: unit %s not found in group %s", id, g.ID)
}

// SetStatus changes the status of one unit.
func (g *Group) SetStatus(id string, status Status) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, unit := range g.Units {
		if unit.ID == id {
			unit.Status = status
			return nil
		}
	}
	return fmt.Errorf("pump: unit %s not found in group %s", id, g.ID)
}

// Statuses returns a snapshot of every unit status keyed by unit id.
func (g *Group) Statuses() map[string]Status {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make(map[string]Status, len(g.Units))
	for _, unit := range g.Units {
		out[unit.ID] = unit.Status
	}
	return out
}

// Save persists the group state.
func (g *Group) Save() error {
	g.mu.RLock()
	record := groupRecord(g)
	g.mu.RUnlock()
	return g.fs.WriteJSON("pumps/"+g.ID+".json", record)
}

func groupRecord(g *Group) store.PumpRecord {
	record := store.PumpRecord{
		ID:           g.ID,
		Station:      g.StationID,
		Name:         g.Name,
		ActiveUnit:   g.ActiveUnit,
		DischargeMPA: g.DischargeMPA,
		TargetMPA:    g.TargetMPA,
	}
	for _, unit := range g.Units {
		record.Units = append(record.Units, store.UnitRecord{
			ID:        unit.ID,
			Name:      unit.Name,
			Status:    string(unit.Status),
			Locked:    unit.Locked,
			Vibration: unit.Vibration,
		})
	}
	return record
}

// Restore rebuilds a group from a persisted record.
func (g *Group) Restore(record store.PumpRecord) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.ID = record.ID
	g.StationID = record.Station
	g.Name = record.Name
	g.ActiveUnit = record.ActiveUnit
	g.DischargeMPA = record.DischargeMPA
	g.TargetMPA = record.TargetMPA
	g.Units = nil
	for _, unitRecord := range record.Units {
		g.Units = append(g.Units, &Unit{
			ID:        unitRecord.ID,
			Name:      unitRecord.Name,
			Status:    Status(unitRecord.Status),
			Locked:    unitRecord.Locked,
			Vibration: unitRecord.Vibration,
		})
	}
	return nil
}

// Record returns the serializable form of the group.
func (g *Group) Record() store.PumpRecord {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return groupRecord(g)
}
