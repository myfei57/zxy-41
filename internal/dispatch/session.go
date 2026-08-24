package dispatch

import (
	"context"
	"time"

	"github.com/google/uuid"

	"waternet/internal/audit"
	"waternet/internal/pump"
	"waternet/internal/store"
)

// Executor applies dispatch commands to pump groups. Commands targeting the
// same group are serialized so concurrent sessions cannot interleave.
type Executor struct {
	receipts *store.ExecutionLog
	recorder *audit.Recorder
}

// NewExecutor creates a dispatch executor over the receipt log.
func NewExecutor(receipts *store.ExecutionLog, recorder *audit.Recorder) *Executor {
	return &Executor{
		receipts: receipts,
		recorder: recorder,
	}
}

// Apply runs one command and persists the execution receipt.
func (e *Executor) Apply(ctx context.Context, command pump.Command, group *pump.Group) error {
	if err := pump.Apply(ctx, group, command); err != nil {
		return err
	}
	return e.record(ctx, command, "applied")
}

func (e *Executor) record(ctx context.Context, command pump.Command, result string) error {
	record := store.ExecutionRecord{
		ID:        uuid.NewString(),
		CommandID: command.ID,
		Target:    command.GroupID,
		Action:    command.Action,
		Result:    result,
		At:        time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := e.receipts.Append(record); err != nil {
		return err
	}
	if e.recorder != nil {
		_, _ = e.recorder.Record("dispatch_apply", command.GroupID, command.Action)
	}
	return nil
}

// Session is one operator dispatch session.
type Session struct {
	id   string
	exec *Executor
}

// NewSession creates a dispatch session over an executor.
func NewSession(exec *Executor) *Session {
	return &Session{id: uuid.NewString(), exec: exec}
}

// ID returns the session identifier.
func (s *Session) ID() string {
	return s.id
}

// Send issues one command through the executor.
func (s *Session) Send(ctx context.Context, command pump.Command, group *pump.Group) error {
	return s.exec.Apply(ctx, command, group)
}
