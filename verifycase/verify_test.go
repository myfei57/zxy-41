package verifycase

import (
	"context"
	"testing"

	"waternet/internal/pressure"
	"waternet/internal/store"
)

func TestWnFlowBaseResetOnDirectionSwitch(t *testing.T) {
	ctx := context.Background()
	fs, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ledger := store.NewCumulativeLedger(fs)
	meter := pressure.NewMeter("m-1", "north")
	if err := meter.AddFlow(ctx, ledger, 100); err != nil {
		t.Fatal(err)
	}
	if err := pressure.SwitchDirection(ctx, ledger, meter.ID, pressure.DirectionReverse); err != nil {
		t.Fatal(err)
	}
	if err := meter.AddFlow(ctx, ledger, 50); err != nil {
		t.Fatal(err)
	}
	if pressure.DirectionOf(ledger, meter.ID) != pressure.DirectionReverse {
		t.Fatalf("direction did not switch to reverse")
	}
	segment, err := meter.SegmentTotal(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if segment != 50 {
		t.Fatalf("segment total after direction switch is %v, want 50 (base must reset)", segment)
	}
}
