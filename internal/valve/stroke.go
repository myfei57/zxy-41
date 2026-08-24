package valve

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"waternet/internal/store"
)

// Step is one incremental movement of one valve.
type Step struct {
	Valve  *Valve
	Action Action
	Delta  float64
}

// Executor applies stroke steps and persists every execution receipt.
type Executor struct {
	mu  sync.Mutex
	log *store.ExecutionLog
	fs  *store.FileStore
}

// NewExecutor creates a valve executor over the given receipt log.
func NewExecutor(log *store.ExecutionLog, fs *store.FileStore) *Executor {
	return &Executor{log: log, fs: fs}
}

// Apply executes the steps in order and records one receipt per step.
func (e *Executor) Apply(ctx context.Context, steps []Step) ([]store.ExecutionRecord, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	records := make([]store.ExecutionRecord, 0, len(steps))
	for _, step := range steps {
		start := step.Valve.PositionNow()
		var next float64
		switch step.Action {
		case ActionOpen:
			next = start + step.Delta
			if next > 1 {
				next = 1
			}
		case ActionClose:
			next = start - step.Delta
			if next < 0 {
				next = 0
			}
		default:
			return nil, fmt.Errorf("valve: unsupported stroke action %q", step.Action)
		}
		if err := step.Valve.SetPosition(step.Action, next); err != nil {
			return nil, err
		}
		record := store.ExecutionRecord{
			ID:        uuid.NewString(),
			CommandID: "stroke",
			Target:    step.Valve.ID,
			Action:    string(step.Action),
			Result:    fmt.Sprintf("pos=%.3f", next),
			At:        time.Now().UTC().Format(time.RFC3339Nano),
		}
		if err := e.log.Append(record); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if e.fs != nil {
		for _, step := range steps {
			_ = step.Valve.Save()
		}
	}
	return records, nil
}

// CloseFully strokes a valve to the fully closed position in one call.
func (e *Executor) CloseFully(ctx context.Context, v *Valve) error {
	_, err := e.Apply(ctx, []Step{{Valve: v, Action: ActionClose, Delta: 1.0}})
	return err
}

// OpenFully strokes a valve to the fully open position in one call.
func (e *Executor) OpenFully(ctx context.Context, v *Valve) error {
	_, err := e.Apply(ctx, []Step{{Valve: v, Action: ActionOpen, Delta: 1.0}})
	return err
}
