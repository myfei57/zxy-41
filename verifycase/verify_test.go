package verifycase

import (
	"context"
	"testing"
	"time"

	"waternet/internal/hammer"
	"waternet/internal/pressure"
	"waternet/internal/store"
)

func TestWnPeakWindowCoversClosure(t *testing.T) {
	ctx := context.Background()
	closureAt := time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)
	plan := hammer.NewPlan("north", "v-up", "v-down", 0.25, 5, 5)
	window := hammer.ClosurePeakWindow(plan, closureAt)
	if !window.Contains(closureAt) {
		t.Fatalf("peak window [%s, %s) does not cover the closure instant %s", window.Start, window.End, closureAt)
	}
	fs, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	windows := pressure.NewWindowStore(store.NewWindowAggregator(fs))
	if err := hammer.RecordClosurePeak(ctx, windows, plan, closureAt, 0.85); err != nil {
		t.Fatal(err)
	}
	peak, err := hammer.PeakInWindow(ctx, windows, plan, closureAt)
	if err != nil {
		t.Fatal(err)
	}
	if peak != 0.85 {
		t.Fatalf("closure peak %v not recorded in history, want 0.85", peak)
	}
}
