package verifycase

import (
	"context"
	"testing"

	"waternet/internal/audit"
	"waternet/internal/pump"
	"waternet/internal/station"
	"waternet/internal/store"
)

func TestWnLockoutClearedOnRelease(t *testing.T) {
	ctx := context.Background()
	fs, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	group := pump.NewGroup("g-1", "st-1", "主泵组", fs)
	group.AddUnit("u-1", "1号主泵")
	recorder := audit.NewRecorder(fs)
	st := station.New("st-1", "主泵站", "downtown", "north", fs, recorder)
	if err := st.AttachGroup(group); err != nil {
		t.Fatal(err)
	}
	if err := station.LockPumpForMaintenance(ctx, st, group, "u-1"); err != nil {
		t.Fatal(err)
	}
	locked, err := group.IsLocked("u-1")
	if err != nil {
		t.Fatal(err)
	}
	if !locked {
		t.Fatalf("precondition: unit must be locked after maintenance starts")
	}
	if err := station.ReleasePump(ctx, st, group, "u-1"); err != nil {
		t.Fatal(err)
	}
	locked, err = group.IsLocked("u-1")
	if err != nil {
		t.Fatal(err)
	}
	if locked {
		t.Fatalf("maintenance lockout was not cleared on release")
	}
	unit, err := group.Unit("u-1")
	if err != nil {
		t.Fatal(err)
	}
	if unit.Status != pump.StatusStandby {
		t.Fatalf("released unit must return to standby, got %s", unit.Status)
	}
}
