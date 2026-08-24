package valve

import (
	"context"
	"fmt"

	"waternet/internal/pressure"
)

// SwitchZone moves a valve to another zone and immediately re-attributes its
// pressure readings by updating the zone map.
func SwitchZone(ctx context.Context, v *Valve, zoneID string, mapper *pressure.ZoneMap) error {
	if zoneID == "" {
		return fmt.Errorf("valve: empty zone id")
	}
	v.SetZone(zoneID)
	if err := mapper.AttachZone(v.ID, zoneID); err != nil {
		return err
	}
	return v.Save()
}

// IsolationZone returns the zone a valve isolates now.
func IsolationZone(v *Valve) string {
	return v.ZoneNow()
}
