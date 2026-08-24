package store

// StateRecord is the serializable form of the whole water network snapshot.
type StateRecord struct {
	Version    int                `json:"version"`
	Namespaces []NamespaceRecord  `json:"namespaces"`
	Stations   []StationRecord    `json:"stations"`
	Pumps      []PumpRecord       `json:"pumps"`
	Valves     []ValveRecord      `json:"valves"`
	ZoneMap    map[string]string  `json:"zone_map"`
	Quota      QuotaRecord        `json:"quota"`
	Policy     PolicyRecord       `json:"policy"`
	Cumulative []CumulativeRecord `json:"cumulative"`
	SnapshotAt string             `json:"snapshot_at"`
}

// NamespaceRecord describes one water district namespace and its zones.
type NamespaceRecord struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Zones []string `json:"zones"`
}

// StationRecord describes one pumping station and its attached equipment.
type StationRecord struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Zone        string            `json:"zone"`
	TankIDs     []string          `json:"tank_ids"`
	PumpIDs     []string          `json:"pump_ids"`
	ValveIDs    []string          `json:"valve_ids"`
	CachedPumps map[string]string `json:"cached_pumps"`
}

// PumpRecord describes one pump group and its units.
type PumpRecord struct {
	ID           string       `json:"id"`
	Station      string       `json:"station"`
	Name         string       `json:"name"`
	Units        []UnitRecord `json:"units"`
	ActiveUnit   string       `json:"active_unit"`
	DischargeMPA float64      `json:"discharge_mpa"`
	TargetMPA    float64      `json:"target_mpa"`
}

// UnitRecord describes one pump unit inside a group.
type UnitRecord struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	Locked    bool    `json:"locked"`
	Vibration float64 `json:"vibration"`
}

// ValveRecord describes one valve and its current position.
type ValveRecord struct {
	ID        string  `json:"id"`
	Station   string  `json:"station"`
	Name      string  `json:"name"`
	Position  float64 `json:"position"`
	Direction string  `json:"direction"`
	Zone      string  `json:"zone"`
}

// QuotaRecord describes the sampling quota of one zone.
type QuotaRecord struct {
	Zone   string `json:"zone"`
	Used   int    `json:"used"`
	Limit  int    `json:"limit"`
	Window string `json:"window"`
}

// PolicyRecord describes the pressure target of one zone.
type PolicyRecord struct {
	Zone      string  `json:"zone"`
	TargetMPA float64 `json:"target_mpa"`
	MinMPA    float64 `json:"min_mpa"`
	MaxMPA    float64 `json:"max_mpa"`
	Strategy  string  `json:"strategy"`
}

// CumulativeRecord describes the cumulative flow state of one meter.
type CumulativeRecord struct {
	MeterID   string  `json:"meter_id"`
	Base      float64 `json:"base"`
	Total     float64 `json:"total"`
	Direction string  `json:"direction"`
}

// ExecutionRecord is one persisted pump or valve execution receipt.
type ExecutionRecord struct {
	ID        string `json:"id"`
	CommandID string `json:"command_id"`
	Target    string `json:"target"`
	Action    string `json:"action"`
	Result    string `json:"result"`
	At        string `json:"at"`
}
