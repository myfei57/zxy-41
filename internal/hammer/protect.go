package hammer

import (
	"context"
	"fmt"

	"waternet/internal/valve"
)

// Protect executes a closure plan under the water-hammer protection stroke
// order, returning the receipts of every executed step.
func Protect(ctx context.Context, executor *valve.Executor, plan Plan, valves map[string]*valve.Valve) (int, error) {
	steps, err := plan.Steps(valves)
	if err != nil {
		return 0, err
	}
	records, err := executor.Apply(ctx, steps)
	if err != nil {
		return 0, err
	}
	return len(records), nil
}

// PlanForStrokes builds the protection plan used for a two-valve isolation.
func PlanForStrokes(zone, upstreamID, downstreamID string) Plan {
	return NewPlan(zone, upstreamID, downstreamID, 0.25, 5, 5)
}

func errUnknownValve(id string) error {
	return fmt.Errorf("hammer: valve %s not found", id)
}
