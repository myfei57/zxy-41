package hammer

import (
	"time"

	"waternet/internal/valve"
)

// Plan describes one protected closure: which valves are stroked, how far each
// step moves, and how the pressure peak window is anchored to the closure.
type Plan struct {
	Zone         string
	UpstreamID   string
	DownstreamID string
	Step         float64
	PreSeconds   int
	PostSeconds  int
}

// NewPlan builds a closure plan with the default protection envelope.
func NewPlan(zone, upstreamID, downstreamID string, step float64, pre, post int) Plan {
	return Plan{
		Zone:         zone,
		UpstreamID:   upstreamID,
		DownstreamID: downstreamID,
		Step:         step,
		PreSeconds:   pre,
		PostSeconds:  post,
	}
}

// OrderedValveIDs returns the stroke order required by water-hammer
// protection: the upstream valve closes first, then the downstream valve.
func (p Plan) OrderedValveIDs() []string {
	return []string{p.UpstreamID, p.DownstreamID}
}

// Steps converts the protection order into executable valve steps.
func (p Plan) Steps(valves map[string]*valve.Valve) ([]valve.Step, error) {
	ids := p.OrderedValveIDs()
	steps := make([]valve.Step, 0, len(ids))
	for _, id := range ids {
		v, ok := valves[id]
		if !ok {
			return nil, errUnknownValve(id)
		}
		steps = append(steps, valve.Step{Valve: v, Action: valve.ActionClose, Delta: p.Step})
	}
	return steps, nil
}

// PeakWindow computes the time range that must cover the closure instant.
func (p Plan) PeakWindow(closureAt time.Time) (time.Time, time.Time) {
	start := closureAt.Add(-time.Duration(p.PreSeconds) * time.Second)
	end := closureAt.Add(time.Duration(p.PostSeconds) * time.Second)
	return start, end
}
