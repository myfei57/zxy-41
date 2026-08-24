package verifycase

import (
	"context"
	"testing"

	"waternet/internal/audit"
	"waternet/internal/dispatch"
	"waternet/internal/pump"
	"waternet/internal/station"
	"waternet/internal/store"
)

func TestWnDispatchAppliesInIssueOrder(t *testing.T) {
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
	queue := dispatch.NewOrderQueue()
	lower := pump.NewCommand("g-1", "u-1", pump.ActionSetTarget)
	lower.TargetMPA = 0.35
	raise := pump.NewCommand("g-1", "u-1", pump.ActionSetTarget)
	raise.TargetMPA = 0.55
	queue.Enqueue(dispatch.NewOrder(2, raise))
	queue.Enqueue(dispatch.NewOrder(1, lower))
	count, err := station.ApplyDispatch(ctx, st, queue)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("applied %d commands, want 2", count)
	}
	if group.TargetMPA != 0.55 {
		t.Fatalf("final target %v, want 0.55 (issue order: seq 1 then seq 2)", group.TargetMPA)
	}
}
