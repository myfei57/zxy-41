package valve

// Status is the console-facing summary of one valve.
type Status struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Position float64 `json:"position"`
	Action   string  `json:"action"`
	Zone     string  `json:"zone"`
}

// StatusOf renders the current status of a valve.
func StatusOf(v *Valve) Status {
	record := v.Record()
	return Status{
		ID:       record.ID,
		Name:     record.Name,
		Position: record.Position,
		Action:   record.Direction,
		Zone:     record.Zone,
	}
}

// FullyOpen reports whether the valve is completely open.
func FullyOpen(v *Valve) bool {
	return v.PositionNow() >= 0.999
}

// FullyClosed reports whether the valve is completely closed.
func FullyClosed(v *Valve) bool {
	return v.PositionNow() <= 0.001
}
