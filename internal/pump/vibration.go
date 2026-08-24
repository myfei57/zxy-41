package pump

import (
	"context"
	"errors"
	"fmt"
)

// VibrationLimit is the running vibration threshold in mm/s.
const VibrationLimit = 4.5

// ErrVibrationOverLimit is returned when a running unit vibrates too hard.
var ErrVibrationOverLimit = errors.New("pump: vibration over limit")

// RecordVibration stores the latest vibration reading of a unit.
func (g *Group) RecordVibration(unitID string, value float64) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, unit := range g.Units {
		if unit.ID == unitID {
			unit.Vibration = value
			return nil
		}
	}
	return fmt.Errorf("pump: unit %s not found in group %s", unitID, g.ID)
}

// VibrationOverLimit reports whether any running unit exceeds the limit.
func (g *Group) VibrationOverLimit() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, unit := range g.Units {
		if unit.Status == StatusRunning && unit.Vibration > VibrationLimit {
			return true
		}
	}
	return false
}

// FirstRunningUnit returns the id of the first unit in running state.
func (g *Group) FirstRunningUnit() (string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, unit := range g.Units {
		if unit.Status == StatusRunning {
			return unit.ID, nil
		}
	}
	return "", fmt.Errorf("pump: group %s has no running unit", g.ID)
}

// Preflight rejects an operation when vibration protection is required.
func Preflight(ctx context.Context, group *Group) error {
	if group.VibrationOverLimit() {
		return fmt.Errorf("%w: running unit exceeds %.1f mm/s", ErrVibrationOverLimit, VibrationLimit)
	}
	return nil
}

// Trip stops the vibrating unit immediately and clears the active unit.
func Trip(ctx context.Context, group *Group, unitID string) error {
	group.mu.Lock()
	defer group.mu.Unlock()
	unit, err := findUnit(group, unitID)
	if err != nil {
		return err
	}
	unit.Status = StatusStopped
	if group.ActiveUnit == unit.ID {
		group.ActiveUnit = ""
	}
	return nil
}
