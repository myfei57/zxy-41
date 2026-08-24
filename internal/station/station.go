package station

import (
	"fmt"
	"sort"
	"sync"

	"waternet/internal/audit"
	"waternet/internal/pump"
	"waternet/internal/store"
	"waternet/internal/valve"
)

// Station is one pumping station: it owns tanks, pump groups and valves and
// keeps the state snapshot used by failover and the console.
type Station struct {
	mu          sync.RWMutex
	ID          string
	Name        string
	NamespaceID string
	ZoneID      string
	Tanks       []*Tank
	Groups      []*pump.Group
	Valves      []*valve.Valve
	cachedState State
	fs          *store.FileStore
	recorder    *audit.Recorder
}

// New creates an empty station.
func New(id, name, namespaceID, zoneID string, fs *store.FileStore, recorder *audit.Recorder) *Station {
	return &Station{
		ID:          id,
		Name:        name,
		NamespaceID: namespaceID,
		ZoneID:      zoneID,
		fs:          fs,
		recorder:    recorder,
	}
}

// AttachGroup registers a pump group under the station.
func (s *Station) AttachGroup(group *pump.Group) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if group.StationID != s.ID {
		return fmt.Errorf("station: group %s belongs to station %s", group.ID, group.StationID)
	}
	s.Groups = append(s.Groups, group)
	return nil
}

// AttachValve registers a valve under the station.
func (s *Station) AttachValve(v *valve.Valve) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Valves = append(s.Valves, v)
	return nil
}

// AddTank registers a clean water tank under the station.
func (s *Station) AddTank(tank *Tank) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Tanks = append(s.Tanks, tank)
	return nil
}

// Group returns one pump group by id.
func (s *Station) Group(id string) (*pump.Group, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, group := range s.Groups {
		if group.ID == id {
			return group, nil
		}
	}
	return nil, fmt.Errorf("station: group %s not found", id)
}

// Valve returns one valve by id.
func (s *Station) Valve(id string) (*valve.Valve, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range s.Valves {
		if v.ID == id {
			return v, nil
		}
	}
	return nil, fmt.Errorf("station: valve %s not found", id)
}

// Tank returns one tank by id.
func (s *Station) Tank(id string) (*Tank, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, tank := range s.Tanks {
		if tank.ID == id {
			return tank, nil
		}
	}
	return nil, fmt.Errorf("station: tank %s not found", id)
}

// PumpIDs returns the registered pump group ids.
func (s *Station) PumpIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.Groups))
	for _, group := range s.Groups {
		ids = append(ids, group.ID)
	}
	sort.Strings(ids)
	return ids
}

// ValveIDs returns the registered valve ids.
func (s *Station) ValveIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.Valves))
	for _, v := range s.Valves {
		ids = append(ids, v.ID)
	}
	sort.Strings(ids)
	return ids
}

// Save persists the station snapshot record.
func (s *Station) Save() error {
	if s.fs == nil {
		return nil
	}
	record := s.Record()
	return s.fs.WriteJSON("stations/"+s.ID+".json", record)
}

// Record returns the serializable station record.
func (s *Station) Record() store.StationRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record := store.StationRecord{
		ID:          s.ID,
		Name:        s.Name,
		Namespace:   s.NamespaceID,
		Zone:        s.ZoneID,
		TankIDs:     make([]string, 0, len(s.Tanks)),
		PumpIDs:     make([]string, 0, len(s.Groups)),
		ValveIDs:    make([]string, 0, len(s.Valves)),
		CachedPumps: map[string]string{},
	}
	for _, tank := range s.Tanks {
		record.TankIDs = append(record.TankIDs, tank.ID)
	}
	for _, group := range s.Groups {
		record.PumpIDs = append(record.PumpIDs, group.ID)
	}
	for _, v := range s.Valves {
		record.ValveIDs = append(record.ValveIDs, v.ID)
	}
	for unitID, status := range s.cachedState.PumpStatuses {
		record.CachedPumps[unitID] = string(status)
	}
	return record
}
