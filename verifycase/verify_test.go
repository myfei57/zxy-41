package verifycase

import (
	"context"
	"testing"

	"waternet/internal/pump"
	"waternet/internal/station"
	"waternet/internal/store"
)

func TestWnFailoverUsesCurrentStationState(t *testing.T) {
	ctx := context.Background()
	fs, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	group := pump.NewGroup("g-1", "st-1", "主泵组", fs)
	group.AddUnit("u-1", "1号主泵")
	group.AddUnit("u-2", "1号备泵")
	st := station.New("st-1", "主泵站", "downtown", "north", fs, nil)
	if err := st.AttachGroup(group); err != nil {
		t.Fatal(err)
	}
	if err := group.SetStatus("u-1", pump.StatusMaintenance); err != nil {
		t.Fatal(err)
	}
	if err := group.SetStatus("u-2", pump.StatusRunning); err != nil {
		t.Fatal(err)
	}
	if err := st.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	active1, err := pump.Failover(ctx, group, st)
	if err != nil {
		t.Fatal(err)
	}
	if active1 != "u-2" {
		t.Fatalf("backup should run while primary is down, got %s", active1)
	}
	if err := group.SetStatus("u-1", pump.StatusRunning); err != nil {
		t.Fatal(err)
	}
	active2, err := pump.Failover(ctx, group, st)
	if err != nil {
		t.Fatal(err)
	}
	if active2 != "u-1" {
		t.Fatalf("failover must read the current station state: got %s want u-1", active2)
	}
}
