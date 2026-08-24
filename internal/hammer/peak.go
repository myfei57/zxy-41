package hammer

import (
	"context"
	"time"

	"waternet/internal/pressure"
	"waternet/internal/store"
)

// ClosurePeakWindow returns the pressure window that must cover the closure
// instant so the highest spike is never lost from history.
func ClosurePeakWindow(plan Plan, closureAt time.Time) pressure.Window {
	start, end := plan.PeakWindow(closureAt)
	return pressure.NewWindow(plan.Zone, start, end)
}

// RecordClosurePeak stores the pressure observed at the closure instant inside
// the peak window of the plan.
func RecordClosurePeak(ctx context.Context, windows *pressure.WindowStore, plan Plan, closureAt time.Time, mpa float64) error {
	window := ClosurePeakWindow(plan, closureAt)
	return windows.Record(store.ReadingRecord{
		Zone:  plan.Zone,
		Slot:  window.Slot(),
		Valve: "closure-peak",
		MPA:   mpa,
		At:    closureAt.UTC().Format(time.RFC3339Nano),
	})
}

// PeakInWindow reports the highest pressure stored inside the closure window.
func PeakInWindow(ctx context.Context, windows *pressure.WindowStore, plan Plan, closureAt time.Time) (float64, error) {
	return windows.Peak(plan.Zone, ClosurePeakWindow(plan, closureAt))
}
