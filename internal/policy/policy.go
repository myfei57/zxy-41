package policy

import (
	"fmt"
	"sort"
	"sync"

	"waternet/internal/store"
)

const policyFile = "policy.json"

// Target is the published pressure envelope of one zone.
type Target struct {
	Zone      string  `json:"zone"`
	TargetMPA float64 `json:"target_mpa"`
	MinMPA    float64 `json:"min_mpa"`
	MaxMPA    float64 `json:"max_mpa"`
	Strategy  string  `json:"strategy"`
}

// Config holds the published pressure targets of every zone.
type Config struct {
	mu      sync.RWMutex
	targets map[string]Target
	fs      *store.FileStore
}

// NewConfig creates an empty policy config.
func NewConfig(fs *store.FileStore) *Config {
	return &Config{targets: map[string]Target{}, fs: fs}
}

// SetTarget publishes a new pressure target for a zone.
func (c *Config) SetTarget(target Target) error {
	if target.Zone == "" {
		return fmt.Errorf("policy: target zone is empty")
	}
	if target.MinMPA < 0 || target.MaxMPA < target.MinMPA {
		return fmt.Errorf("policy: invalid envelope for zone %s", target.Zone)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.targets[target.Zone] = target
	return c.saveLocked()
}

// TargetFor returns the published target of a zone.
func (c *Config) TargetFor(zone string) (Target, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	target, ok := c.targets[zone]
	if !ok {
		return Target{}, fmt.Errorf("policy: no target for zone %s", zone)
	}
	return target, nil
}

// All returns every published target in stable zone order.
func (c *Config) All() []Target {
	c.mu.RLock()
	defer c.mu.RUnlock()
	zones := make([]string, 0, len(c.targets))
	for zone := range c.targets {
		zones = append(zones, zone)
	}
	sort.Strings(zones)
	out := make([]Target, 0, len(zones))
	for _, zone := range zones {
		out = append(out, c.targets[zone])
	}
	return out
}

func (c *Config) saveLocked() error {
	if c.fs == nil {
		return nil
	}
	zones := make([]string, 0, len(c.targets))
	for zone := range c.targets {
		zones = append(zones, zone)
	}
	sort.Strings(zones)
	targets := make([]Target, 0, len(zones))
	for _, zone := range zones {
		targets = append(targets, c.targets[zone])
	}
	return c.fs.WriteJSON(policyFile, targets)
}

// Save persists the published targets.
func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.saveLocked()
}

// Load restores the published targets from disk.
func (c *Config) Load() error {
	var targets []Target
	if err := c.fs.ReadJSON(policyFile, &targets); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.targets = map[string]Target{}
	for _, target := range targets {
		c.targets[target.Zone] = target
	}
	return nil
}
