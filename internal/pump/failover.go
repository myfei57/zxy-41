package pump

import (
	"context"
	"fmt"
	"sort"

	"waternet/internal/store"
)

// StationView exposes the pump statuses a station currently sees. Failover
// reads the live view so the active unit always follows the latest state — a
// recovered primary must be visible to the switch decision, which a stale
// refresh-loop snapshot could never provide.
type StationView interface {
	// PumpStatuses returns the current statuses read live from every group.
	PumpStatuses(ctx context.Context) (map[string]Status, error)
}

// Failover decides which unit must run and applies the switch. The decision
// is based on the live station view, read fresh on every call: a recovered
// primary takes over from the backup; otherwise the first healthy standby
// unit is promoted. Reading live (never the refresh-loop snapshot) is what
// makes a newly recovered primary visible to the switch decision.
func Failover(ctx context.Context, group *Group, view StationView) (string, error) {
	statuses, err := view.PumpStatuses(ctx)
	if err != nil {
		return "", fmt.Errorf("pump: failover status: %w", err)
	}
	group.mu.Lock()
	defer group.mu.Unlock()
	var primary string
	healthy := make([]string, 0, len(group.Units))
	for index, unit := range group.Units {
		if index == 0 {
			primary = unit.ID
		}
		if statuses[unit.ID] == StatusRunning || statuses[unit.ID] == StatusStandby {
			if !unit.Locked {
				healthy = append(healthy, unit.ID)
			}
		}
	}
	sort.Strings(healthy)
	chosen := ""
	if primary != "" && statuses[primary] == StatusRunning {
		chosen = primary
	} else if len(healthy) > 0 {
		chosen = healthy[0]
	}
	if chosen == "" {
		return "", fmt.Errorf("pump: group %s has no healthy unit to run", group.ID)
	}
	for _, unit := range group.Units {
		if unit.ID == chosen {
			unit.Status = StatusRunning
		} else if unit.Status == StatusRunning {
			unit.Status = StatusStandby
		}
	}
	group.ActiveUnit = chosen
	if group.fs != nil {
		_ = group.fs.WriteJSON("pumps/"+group.ID+".json", groupRecord(group))
	}
	return chosen, nil
}

// SaveSnapshot persists the group state without changing it, used by the
// station refresh loop to keep recovery consistent.
func SaveSnapshot(group *Group) error {
	if group.fs == nil {
		return nil
	}
	record := group.Record()
	return group.fs.WriteJSON("pumps/"+group.ID+".json", record)
}

// LoadGroup restores a group from the store.
func LoadGroup(fs *store.FileStore, id string) (*Group, error) {
	var record store.PumpRecord
	if err := fs.ReadJSON("pumps/"+id+".json", &record); err != nil {
		return nil, err
	}
	group := NewGroup(id, record.Station, record.Name, fs)
	if err := group.Restore(record); err != nil {
		return nil, err
	}
	return group, nil
}
