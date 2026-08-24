package pressure

import (
	"context"
	"fmt"
	"time"

	"waternet/internal/audit"
	"waternet/internal/quota"
	"waternet/internal/store"
)

// Reading is one pressure and flow sample attributed to a zone.
type Reading struct {
	Zone  string
	Valve string
	MPA   float64
	Flow  float64
	At    time.Time
}

// Sampler ingests pressure samples: it attributes each sample to the current
// zone of its valve, enforces the sampling quota and persists the row.
type Sampler struct {
	mapper   *ZoneMap
	agg      *store.WindowAggregator
	ledger   *quota.Ledger
	recorder *audit.Recorder
	fs       *store.FileStore
}

// NewSampler wires a sampler to its persistence and quota dependencies.
func NewSampler(mapper *ZoneMap, agg *store.WindowAggregator, ledger *quota.Ledger, recorder *audit.Recorder, fs *store.FileStore) *Sampler {
	return &Sampler{mapper: mapper, agg: agg, ledger: ledger, recorder: recorder, fs: fs}
}

// Sample persists one reading and returns its zone attribution.
func (s *Sampler) Sample(ctx context.Context, valveID string, mpa, flow float64, at time.Time) (Reading, error) {
	zone := s.mapper.ZoneFor(valveID)
	if zone == "" {
		return Reading{}, fmt.Errorf("pressure: valve %s has no zone attribution", valveID)
	}
	if err := s.ledger.TryConsume(zone, 1); err != nil {
		return Reading{}, err
	}
	reading := Reading{Zone: zone, Valve: valveID, MPA: mpa, Flow: flow, At: at}
	row := store.ReadingRecord{
		Zone:  zone,
		Slot:  at.UTC().Format("2006-01-02T15:04"),
		Valve: valveID,
		MPA:   mpa,
		Flow:  flow,
		At:    at.UTC().Format(time.RFC3339Nano),
	}
	if err := s.agg.Append(row); err != nil {
		s.ledger.Release(zone, 1)
		return Reading{}, err
	}
	if s.recorder != nil {
		_, _ = s.recorder.Record("sample", valveID, fmt.Sprintf("zone=%s mpa=%.3f", zone, mpa))
	}
	return reading, nil
}

// SampleBatch ingests several readings from one valve.
func (s *Sampler) SampleBatch(ctx context.Context, valveID string, readings []Reading) (int, error) {
	for _, reading := range readings {
		if _, err := s.Sample(ctx, valveID, reading.MPA, reading.Flow, reading.At); err != nil {
			return 0, err
		}
	}
	return len(readings), nil
}
