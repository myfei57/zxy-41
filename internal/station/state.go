package station

import (
	"context"
	"time"

	"waternet/internal/pump"
)

// State is a snapshot of the pump and tank conditions of a station.
type State struct {
	PumpStatuses map[string]pump.Status
	TankLevels   map[string]float64
	ZoneID       string
	UpdatedAt    time.Time
}

// CurrentState reads the station state live from every attached pump group,
// so callers always see the latest unit statuses.
func (s *Station) CurrentState(ctx context.Context) (State, error) {
	state := State{
		PumpStatuses: map[string]pump.Status{},
		TankLevels:   map[string]float64{},
		ZoneID:       s.ZoneID,
		UpdatedAt:    time.Now().UTC(),
	}
	s.mu.RLock()
	for _, group := range s.Groups {
		for unitID, status := range group.Statuses() {
			state.PumpStatuses[unitID] = status
		}
	}
	for _, tank := range s.Tanks {
		state.TankLevels[tank.ID] = tank.LevelNow()
	}
	s.mu.RUnlock()
	return state, nil
}

// CachedState returns the snapshot captured by the last refresh cycle.
func (s *Station) CachedState(ctx context.Context) (State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cachedState, nil
}

// Refresh captures a fresh state snapshot and persists it.
func (s *Station) Refresh(ctx context.Context) error {
	state, err := s.CurrentState(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.cachedState = state
	s.mu.Unlock()
	return s.Save()
}

// PumpStatuses exposes the live statuses for the failover view.
func (s *Station) PumpStatuses(ctx context.Context) (map[string]pump.Status, error) {
	state, err := s.CurrentState(ctx)
	if err != nil {
		return nil, err
	}
	return state.PumpStatuses, nil
}

// CachedPumpStatuses exposes the snapshot statuses for the failover view.
func (s *Station) CachedPumpStatuses(ctx context.Context) (map[string]pump.Status, error) {
	state, err := s.CachedState(ctx)
	if err != nil {
		return nil, err
	}
	return state.PumpStatuses, nil
}
