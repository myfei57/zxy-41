package station

import (
	"context"

	"waternet/internal/pump"
)

// ReleasePump finishes maintenance for one unit: the maintenance lockout is
// cleared so the healthy pump can rejoin the running pool.
func ReleasePump(ctx context.Context, s *Station, group *pump.Group, unitID string) error {
	if err := pump.ClearLockout(ctx, group, unitID); err != nil {
		return err
	}
	if s.recorder != nil {
		_, _ = s.recorder.Record("pump_release", unitID, group.ID)
	}
	return s.Save()
}

// LockPumpForMaintenance puts one unit into maintenance and records the event.
func LockPumpForMaintenance(ctx context.Context, s *Station, group *pump.Group, unitID string) error {
	if err := group.LockUnit(unitID); err != nil {
		return err
	}
	if s.recorder != nil {
		_, _ = s.recorder.Record("pump_lockout", unitID, group.ID)
	}
	return s.Save()
}
