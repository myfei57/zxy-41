package pump

import (
	"context"
	"fmt"
)

// LockUnit puts one unit into maintenance and prevents it from starting.
func (g *Group) LockUnit(unitID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	unit, err := findUnit(g, unitID)
	if err != nil {
		return err
	}
	unit.Locked = true
	unit.Status = StatusMaintenance
	if g.ActiveUnit == unit.ID {
		g.ActiveUnit = ""
	}
	return nil
}

// IsLocked reports whether a unit is locked out for maintenance.
func (g *Group) IsLocked(unitID string) (bool, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	unit, err := findUnit(g, unitID)
	if err != nil {
		return false, err
	}
	return unit.Locked, nil
}

// ClearLockout releases a unit after maintenance: the lock is removed and the
// unit returns to standby so it can rejoin the healthy pool.
func ClearLockout(ctx context.Context, group *Group, unitID string) error {
	group.mu.Lock()
	defer group.mu.Unlock()
	unit, err := findUnit(group, unitID)
	if err != nil {
		return err
	}
	unit.Locked = false
	if unit.Status == StatusMaintenance {
		unit.Status = StatusStandby
	}
	if group.fs != nil {
		return group.fs.WriteJSON("pumps/"+group.ID+".json", groupRecord(group))
	}
	return fmt.Errorf("pump: group %s has no store", group.ID)
}
