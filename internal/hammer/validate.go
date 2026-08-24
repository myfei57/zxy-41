package hammer

import "sync"

// Guard is the water-hammer protection gate shared by every valve command.
// Engage is atomic so two concurrent close commands cannot both bypass the
// protection and slam the valve.
type Guard struct {
	mu      sync.Mutex
	engaged map[string]bool
	closes  map[string]int
}

// NewGuard creates an empty protection gate.
func NewGuard() *Guard {
	return &Guard{
		engaged: map[string]bool{},
		closes:  map[string]int{},
	}
}

// Protect engages protection for one valve. It returns true when this call is
// the one admitted closure; a second concurrent call sees the gate engaged and
// is rejected.
func (g *Guard) Protect(valveID string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.engaged[valveID] {
		return false
	}
	g.engaged[valveID] = true
	g.closes[valveID]++
	return true
}

// CloseCount returns how many closures were admitted for a valve.
func (g *Guard) CloseCount(valveID string) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.closes[valveID]
}

// IsEngaged reports whether protection is currently engaged for a valve.
func (g *Guard) IsEngaged(valveID string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.engaged[valveID]
}

// Release disengages protection after the stroke completes.
func (g *Guard) Release(valveID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.engaged, valveID)
}
