package pump

import (
	"fmt"
	"sort"
)

// IsPrimary reports whether a unit is the designated primary of the group.
func (g *Group) IsPrimary(unitID string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	for index, unit := range g.Units {
		if unit.ID == unitID {
			return index == 0
		}
	}
	return false
}

// RunningUnits returns the ids of units currently in running state.
func (g *Group) RunningUnits() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var running []string
	for _, unit := range g.Units {
		if unit.Status == StatusRunning {
			running = append(running, unit.ID)
		}
	}
	sort.Strings(running)
	return running
}

// HealthyUnits returns units that may start: standby or running and unlocked.
func (g *Group) HealthyUnits() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var healthy []string
	for _, unit := range g.Units {
		if unit.Locked {
			continue
		}
		if unit.Status == StatusStandby || unit.Status == StatusRunning {
			healthy = append(healthy, unit.ID)
		}
	}
	sort.Strings(healthy)
	return healthy
}

// StartUnit marks one unit as running and stops the previously active unit.
func (g *Group) StartUnit(unitID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	unit, err := findUnit(g, unitID)
	if err != nil {
		return err
	}
	if unit.Locked {
		return fmt.Errorf("pump: unit %s is locked out", unitID)
	}
	for _, candidate := range g.Units {
		if candidate.ID == unitID {
			candidate.Status = StatusRunning
		} else if candidate.Status == StatusRunning {
			candidate.Status = StatusStandby
		}
	}
	g.ActiveUnit = unit.ID
	return nil
}

// StopUnit returns one unit to standby and clears the active unit when needed.
func (g *Group) StopUnit(unitID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	unit, err := findUnit(g, unitID)
	if err != nil {
		return err
	}
	unit.Status = StatusStandby
	if g.ActiveUnit == unit.ID {
		g.ActiveUnit = ""
	}
	return nil
}

// SetDischarge updates the discharge header pressure of the group.
func (g *Group) SetDischarge(mpa float64) error {
	if mpa < 0 || mpa > 1.5 {
		return fmt.Errorf("pump: discharge %.3f out of range", mpa)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.DischargeMPA = mpa
	return nil
}

// SetTarget records the scheduler target for the group.
func (g *Group) SetTarget(mpa float64) error {
	if mpa < 0 || mpa > 1.5 {
		return fmt.Errorf("pump: target %.3f out of range", mpa)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.TargetMPA = mpa
	return nil
}

func findUnit(g *Group, unitID string) (*Unit, error) {
	for _, unit := range g.Units {
		if unit.ID == unitID {
			return unit, nil
		}
	}
	return nil, fmt.Errorf("pump: unit %s not found in group %s", unitID, g.ID)
}
