package quota

import "time"

// Status is a snapshot of one zone quota for the console.
type Status struct {
	Zone   string `json:"zone"`
	Used   int    `json:"used"`
	Limit  int    `json:"limit"`
	Window string `json:"window"`
}

// Status returns a copy of the current quota state of a zone.
func (l *Ledger) Status(zone string) Status {
	l.mu.Lock()
	defer l.mu.Unlock()
	return Status{Zone: zone, Used: l.used[zone], Limit: l.limit, Window: l.window}
}

// Rollover opens a new quota window, resetting every zone usage.
func (l *Ledger) Rollover() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.window = time.Now().UTC().Format("20060102")
	l.used = map[string]int{}
}

// Remaining returns how many samples a zone can still consume.
func (l *Ledger) Remaining(zone string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	remaining := l.limit - l.used[zone]
	if remaining < 0 {
		return 0
	}
	return remaining
}
