package audit

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"waternet/internal/store"
)

const logName = "audit.jsonl"

// Event is one immutable audit record.
type Event struct {
	ID     string `json:"id"`
	Action string `json:"action"`
	Target string `json:"target"`
	Detail string `json:"detail"`
	At     string `json:"at"`
}

// Recorder appends operational events to the append-only audit log.
type Recorder struct {
	mu sync.Mutex
	fs *store.FileStore
}

// NewRecorder creates an audit recorder over the given store.
func NewRecorder(fs *store.FileStore) *Recorder {
	return &Recorder{fs: fs}
}

// Record writes one event and returns it.
func (r *Recorder) Record(action, target, detail string) (Event, error) {
	event := Event{
		ID:     uuid.NewString(),
		Action: action,
		Target: target,
		Detail: detail,
		At:     time.Now().UTC().Format(time.RFC3339Nano),
	}
	data, err := EncodeEvent(event)
	if err != nil {
		return event, fmt.Errorf("audit: marshal: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.fs.AppendLine(logName, data); err != nil {
		return event, err
	}
	return event, nil
}

// Recent returns the newest events, at most limit entries.
func (r *Recorder) Recent(limit int) ([]Event, error) {
	lines, err := r.fs.ReadLines(logName)
	if err != nil {
		return nil, err
	}
	events := make([]Event, 0, len(lines))
	for _, line := range lines {
		var event Event
		if err := json.Unmarshal(line, &event); err != nil {
			return nil, fmt.Errorf("audit: decode row: %w", err)
		}
		events = append(events, event)
	}
	if limit > 0 && len(events) > limit {
		events = events[len(events)-limit:]
	}
	return events, nil
}

// Count returns how many events match an action prefix.
func (r *Recorder) Count(action string) (int, error) {
	events, err := r.Recent(0)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, event := range events {
		if event.Action == action {
			count++
		}
	}
	return count, nil
}
