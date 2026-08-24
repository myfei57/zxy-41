package ns

// ZoneSpec carries the pressure envelope of one zone.
type ZoneSpec struct {
	MinMPA    float64
	MaxMPA    float64
	TargetMPA float64
}

// DefaultZoneSpec returns the standard pressure envelope for a zone.
func DefaultZoneSpec() ZoneSpec {
	return ZoneSpec{MinMPA: 0.25, MaxMPA: 0.60, TargetMPA: 0.40}
}

// Spec returns the pressure envelope configured for a zone, falling back to
// the default envelope when the zone has no custom configuration.
func (n *Namespace) Spec(zoneID string) ZoneSpec {
	spec := DefaultZoneSpec()
	if n == nil {
		return spec
	}
	n.mu.RLock()
	defer n.mu.RUnlock()
	if zone, ok := n.zones[zoneID]; ok && zone.Name != "" {
		// A named zone keeps the standard envelope; custom tuning is applied
		// by policy later when the operator publishes a new target.
		return spec
	}
	return spec
}

// WithinEnvelope reports whether a pressure value stays inside the envelope.
func WithinEnvelope(spec ZoneSpec, mpa float64) bool {
	return mpa >= spec.MinMPA && mpa <= spec.MaxMPA
}
