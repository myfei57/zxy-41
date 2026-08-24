package station

import (
	"fmt"
	"sync"
)

// Tank is one clean water tank at a station.
type Tank struct {
	mu       sync.RWMutex
	ID       string
	Name     string
	Capacity float64
	Level    float64
}

// NewTank creates a tank with the given capacity.
func NewTank(id, name string, capacity float64) *Tank {
	return &Tank{ID: id, Name: name, Capacity: capacity}
}

// LevelNow returns the current water level.
func (t *Tank) LevelNow() float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Level
}

// SetLevel updates the water level.
func (t *Tank) SetLevel(level float64) error {
	if level < 0 {
		return fmt.Errorf("station: negative tank level %.3f", level)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Level = level
	return nil
}

// Fill adds water and reports the new level, clamped to capacity.
func (t *Tank) Fill(delta float64) error {
	if delta < 0 {
		return fmt.Errorf("station: negative fill %.3f", delta)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Level += delta
	if t.Level > t.Capacity {
		t.Level = t.Capacity
	}
	return nil
}

// Overflowing reports whether the tank reached its capacity.
func (t *Tank) Overflowing() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Level >= t.Capacity
}

// Remaining reports the free capacity.
func (t *Tank) Remaining() float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	remaining := t.Capacity - t.Level
	if remaining < 0 {
		return 0
	}
	return remaining
}
