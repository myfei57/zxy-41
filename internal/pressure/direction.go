package pressure

import (
	"context"
	"fmt"

	"waternet/internal/store"
)

// Supported flow directions of a bidirectional meter.
const (
	DirectionForward = "forward"
	DirectionReverse = "reverse"
)

// SwitchDirection changes the flow direction of a meter and resets the
// cumulative base so the new direction starts accounting from zero.
func SwitchDirection(ctx context.Context, ledger *store.CumulativeLedger, meterID, direction string) error {
	if direction != DirectionForward && direction != DirectionReverse {
		return fmt.Errorf("pressure: unsupported direction %q", direction)
	}
	ledger.SetDirection(meterID, direction)
	return nil
}

// DirectionOf returns the current direction of a meter.
func DirectionOf(ledger *store.CumulativeLedger, meterID string) string {
	return ledger.Direction(meterID)
}
