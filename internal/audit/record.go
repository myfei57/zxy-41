package audit

import (
	"encoding/json"
	"fmt"
)

// Summary groups audit events by action for the console overview page.
type Summary struct {
	Total  int            `json:"total"`
	ByKind map[string]int `json:"by_kind"`
	Latest []Event        `json:"latest"`
}

// Summarize builds a summary from a flat event list.
func Summarize(events []Event, latest int) Summary {
	summary := Summary{ByKind: map[string]int{}}
	for _, event := range events {
		summary.Total++
		summary.ByKind[event.Action]++
	}
	if latest > 0 && len(events) > latest {
		events = events[len(events)-latest:]
	}
	summary.Latest = events
	return summary
}

// DecodeEvent parses one log line into an event.
func DecodeEvent(line []byte) (Event, error) {
	var event Event
	if err := json.Unmarshal(line, &event); err != nil {
		return event, fmt.Errorf("audit: decode: %w", err)
	}
	return event, nil
}

// EncodeEvent serializes one event to a log line.
func EncodeEvent(event Event) ([]byte, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("audit: encode: %w", err)
	}
	return data, nil
}
