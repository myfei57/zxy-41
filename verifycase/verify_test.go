package verifycase

import (
	"context"
	"testing"
	"time"

	"waternet/internal/audit"
	"waternet/internal/pressure"
	"waternet/internal/quota"
	"waternet/internal/station"
	"waternet/internal/store"
	"waternet/internal/valve"
)

func TestWnZoneMapFollowsValveSwitch(t *testing.T) {
	ctx := context.Background()
	fs, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mapper := pressure.NewZoneMap()
	if err := mapper.AttachZone("v-iso", "north"); err != nil {
		t.Fatal(err)
	}
	v := valve.NewValve("v-iso", "st-1", "南北隔离阀", fs)
	v.SetZone("north")
	recorder := audit.NewRecorder(fs)
	st := station.New("st-1", "主泵站", "downtown", "north", fs, recorder)
	if err := st.AttachValve(v); err != nil {
		t.Fatal(err)
	}
	ledger := quota.NewLedger(fs, 1000)
	sampler := pressure.NewSampler(mapper, store.NewWindowAggregator(fs), ledger, recorder, fs)
	first, err := sampler.Sample(ctx, v.ID, 0.40, 10, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if first.Zone != "north" {
		t.Fatalf("initial attribution %s want north", first.Zone)
	}
	if err := valve.SwitchZone(ctx, v, "south", mapper); err != nil {
		t.Fatal(err)
	}
	second, err := sampler.Sample(ctx, v.ID, 0.42, 12, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if second.Zone != "south" {
		t.Fatalf("zone attribution did not follow the isolation valve switch: got %s want south", second.Zone)
	}
}
