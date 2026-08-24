package verifycase

import (
	"context"
	"sync"
	"testing"

	"waternet/internal/audit"
	"waternet/internal/dispatch"
	"waternet/internal/pump"
	"waternet/internal/store"
)

func TestWnConcurrentDispatchSerialized(t *testing.T) {
	ctx := context.Background()
	fs, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	group := pump.NewGroup("g-1", "st-1", "主泵组", fs)
	group.AddUnit("u-1", "1号主泵")
	group.AddUnit("u-2", "1号备泵")
	receipts := store.NewExecutionLog(fs)
	recorder := audit.NewRecorder(fs)
	exec := dispatch.NewExecutor(receipts, recorder)
	sessionA := dispatch.NewSession(exec)
	sessionB := dispatch.NewSession(exec)

	for round := 0; round < 20; round++ {
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_ = sessionA.Send(ctx, pump.NewCommand("g-1", "u-1", pump.ActionStart), group)
		}()
		go func() {
			defer wg.Done()
			<-start
			_ = sessionB.Send(ctx, pump.NewCommand("g-1", "u-2", pump.ActionStart), group)
		}()
		close(start)
		wg.Wait()
		running := group.RunningUnits()
		if len(running) != 1 {
			t.Fatalf("round %d: expected exactly one running unit, got %v", round, running)
		}
	}
}
