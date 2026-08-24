package verifycase

import (
	"context"
	"sync"
	"testing"

	"waternet/internal/hammer"
	"waternet/internal/store"
	"waternet/internal/valve"
)

func TestWnConcurrentValveCloseProtectionHolds(t *testing.T) {
	ctx := context.Background()
	fs, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	v := valve.NewValve("v-1", "st-1", "下游控制阀", fs)
	guard := hammer.NewGuard()
	receipts := store.NewExecutionLog(fs)
	executor := valve.NewExecutor(receipts, fs)

	countFor := func() int {
		records, err := receipts.Recent(0)
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, record := range records {
			if record.Target == v.ID && record.Action == "close" {
				count++
			}
		}
		return count
	}

	for round := 0; round < 10; round++ {
		before := countFor()
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_ = valve.Execute(ctx, v, valve.NewCommand(v.ID, valve.ActionClose), guard, executor)
		}()
		go func() {
			defer wg.Done()
			<-start
			_ = valve.Execute(ctx, v, valve.NewCommand(v.ID, valve.ActionClose), guard, executor)
		}()
		close(start)
		wg.Wait()
		after := countFor()
		if after-before != 1 {
			t.Fatalf("round %d: water-hammer protection admitted %d closures, want 1", round, after-before)
		}
		guard.Release(v.ID)
	}
}
