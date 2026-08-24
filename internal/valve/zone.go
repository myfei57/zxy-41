package valve

import (
	"context"
	"fmt"

	"waternet/internal/pressure"
)

// SwitchZone moves a valve to another zone and immediately re-attributes its
// pressure readings by updating the zone map. Readings sampled after the switch
// are recorded under the new zone instead of leaking back into the old one.
func SwitchZone(ctx context.Context, v *Valve, zoneID string, mapper *pressure.ZoneMap) error {
	if zoneID == "" {
		return fmt.Errorf("valve: empty zone id")
	}
	if mapper != nil {
		if err := mapper.AttachZone(v.ID, zoneID); err != nil {
			return err
		}
	}
	v.SetZone(zoneID)
	return v.Save()
}

// IsolationZone returns the zone a valve isolates now.
func IsolationZone(v *Valve) string {
	return v.ZoneNow()
}
