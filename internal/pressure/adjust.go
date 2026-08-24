package pressure

import (
	"context"
	"fmt"

	"waternet/internal/pump"
)

// Adjust applies a new discharge pressure target to a pump group. Vibration
// protection preempts the adjustment: when a running unit vibrates over the
// limit the unit is tripped and the target is left untouched.
func Adjust(ctx context.Context, group *pump.Group, target float64) error {
	if group.VibrationOverLimit() {
		if running, err := group.FirstRunningUnit(); err == nil {
			if err := pump.Trip(ctx, group, running); err != nil {
				return err
			}
		}
		return fmt.Errorf("%w", pump.ErrVibrationOverLimit)
	}
	return group.SetTarget(target)
}

// AdjustBatch applies targets to several groups sequentially.
func AdjustBatch(ctx context.Context, groups []*pump.Group, targets []float64) error {
	if len(groups) != len(targets) {
		return fmt.Errorf("pressure: group/target count mismatch")
	}
	for index, group := range groups {
		if err := Adjust(ctx, group, targets[index]); err != nil {
			return err
		}
	}
	return nil
}
