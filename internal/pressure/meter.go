package pressure

import (
	"context"
	"fmt"

	"waternet/internal/store"
)

// Meter is one bidirectional flow meter attached to a zone.
type Meter struct {
	ID   string
	Zone string
}

// NewMeter creates a flow meter descriptor.
func NewMeter(id, zone string) Meter {
	return Meter{ID: id, Zone: zone}
}

// AddFlow accumulates one flow delta into the ledger.
func (m Meter) AddFlow(ctx context.Context, ledger *store.CumulativeLedger, delta float64) error {
	if delta < 0 {
		return fmt.Errorf("pressure: negative flow delta %.3f", delta)
	}
	ledger.Add(m.ID, delta)
	return nil
}

// SegmentTotal returns the flow accumulated since the last direction switch.
func (m Meter) SegmentTotal(ledger *store.CumulativeLedger) (float64, error) {
	return ledger.SegmentTotal(m.ID)
}

// LifetimeTotal returns the total flow of the meter since installation.
func (m Meter) LifetimeTotal(ledger *store.CumulativeLedger) (float64, error) {
	return ledger.Total(m.ID)
}
