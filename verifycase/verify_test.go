package verifycase

import (
	"context"
	"testing"

	"waternet/internal/pressure"
	"waternet/internal/pump"
	"waternet/internal/store"
)

func TestWnVibrationTripPreemptsAdjust(t *testing.T) {
	ctx := context.Background()
	fs, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	group := pump.NewGroup("g-1", "st-1", "主泵组", fs)
	group.AddUnit("u-1", "1号主泵")
	if err := group.StartUnit("u-1"); err != nil {
		t.Fatal(err)
	}
	if err := group.RecordVibration("u-1", 5.2); err != nil {
		t.Fatal(err)
	}
	before := group.TargetMPA
	if err := pressure.Adjust(ctx, group, 0.45); err == nil {
		t.Fatalf("adjust must be rejected by the vibration trip")
	}
	if group.TargetMPA != before {
		t.Fatalf("adjustment applied while vibration is over the limit: target %v want %v", group.TargetMPA, before)
	}
	unit, err := group.Unit("u-1")
	if err != nil {
		t.Fatal(err)
	}
	if unit.Status != pump.StatusStopped {
		t.Fatalf("vibrating unit must be tripped, status=%s", unit.Status)
	}
}
