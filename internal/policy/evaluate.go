package policy

import (
	"context"
	"fmt"
	"time"

	"waternet/internal/pressure"
	"waternet/internal/store"
)

// Result is one evaluation of a zone against its published pressure target.
type Result struct {
	Zone    string  `json:"zone"`
	Current float64 `json:"current"`
	Target  float64 `json:"target"`
	Status  string  `json:"status"`
	Action  string  `json:"action"`
}

// Evaluator compares persisted window readings with the published targets.
type Evaluator struct {
	windows *pressure.WindowStore
	config  *Config
}

// NewEvaluator wires an evaluator to its reading source and policy config.
func NewEvaluator(windows *pressure.WindowStore, config *Config) *Evaluator {
	return &Evaluator{windows: windows, config: config}
}

// Evaluate computes the zone status from a batch of readings.
func (e *Evaluator) Evaluate(zone string, readings []store.ReadingRecord) (Result, error) {
	target, err := e.config.TargetFor(zone)
	if err != nil {
		return Result{}, err
	}
	if len(readings) == 0 {
		return Result{}, fmt.Errorf("policy: no readings for zone %s", zone)
	}
	current := readings[len(readings)-1].MPA
	status := "ok"
	action := ""
	switch {
	case current < target.MinMPA:
		status, action = "low", "raise"
	case current > target.MaxMPA:
		status, action = "high", "lower"
	case current != target.TargetMPA:
		action = "tune"
	}
	return Result{Zone: zone, Current: current, Target: target.TargetMPA, Status: status, Action: action}, nil
}

// EvaluateZone evaluates the current window of a zone.
func (e *Evaluator) EvaluateZone(ctx context.Context, zone string) (Result, error) {
	window := pressure.NewWindow(zone, time.Now().UTC().Add(-time.Minute), time.Now().UTC())
	readings, err := e.windows.Readings(zone, window)
	if err != nil {
		return Result{}, err
	}
	return e.Evaluate(zone, readings)
}
