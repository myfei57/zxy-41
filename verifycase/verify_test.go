package verifycase

import (
	"context"
	"testing"

	"waternet/internal/hammer"
	"waternet/internal/store"
	"waternet/internal/valve"
)

func TestWnValveStrokeFollowsProtectionOrder(t *testing.T) {
	ctx := context.Background()
	fs, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	receipts := store.NewExecutionLog(fs)
	executor := valve.NewExecutor(receipts, fs)
	up := valve.NewValve("v-up", "st-1", "上游隔离阀", fs)
	down := valve.NewValve("v-down", "st-1", "下游控制阀", fs)
	plan := hammer.PlanForStrokes("north", up.ID, down.ID)
	valves := map[string]*valve.Valve{up.ID: up, down.ID: down}
	count, err := hammer.Protect(ctx, executor, plan, valves)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected two stroke steps, got %d", count)
	}
	records, err := receipts.Recent(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("expected two receipts, got %d", len(records))
	}
	if records[0].Target != up.ID {
		t.Fatalf("stroke order mismatch: first step closed %s, want upstream %s", records[0].Target, up.ID)
	}
	if records[1].Target != down.ID {
		t.Fatalf("stroke order mismatch: second step closed %s, want downstream %s", records[1].Target, down.ID)
	}
}
