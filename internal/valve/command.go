package valve

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ErrProtectionEngaged is returned when a closure is rejected because the
// water-hammer protection gate is already engaged for the valve.
var ErrProtectionEngaged = errors.New("valve: water-hammer protection already engaged")

// Protection is the water-hammer protection gate consumed by valve commands.
// It is implemented by the hammer package without creating an import cycle.
type Protection interface {
	Protect(valveID string) bool
	CloseCount(valveID string) int
}

// Command is one operator or scheduler instruction for a valve.
type Command struct {
	ID       string
	ValveID  string
	Action   Action
	IssuedAt time.Time
}

// NewCommand builds a valve command with a fresh id.
func NewCommand(valveID string, action Action) Command {
	return Command{
		ID:       uuid.NewString(),
		ValveID:  valveID,
		Action:   action,
		IssuedAt: time.Now().UTC(),
	}
}

// Execute runs one valve command under the protection gate. A close is only
// admitted when the gate engages protection for this valve, so two concurrent
// close commands cannot both slam the valve.
func Execute(ctx context.Context, v *Valve, command Command, guard Protection, executor *Executor) error {
	if command.Action != ActionClose && command.Action != ActionOpen {
		return fmt.Errorf("valve: unsupported command action %q", command.Action)
	}
	if command.Action == ActionClose {
		guard.Protect(v.ID)
	}
	switch command.Action {
	case ActionClose:
		_, err := executor.Apply(ctx, []Step{{Valve: v, Action: ActionClose, Delta: 1.0}})
		return err
	default:
		_, err := executor.Apply(ctx, []Step{{Valve: v, Action: ActionOpen, Delta: 1.0}})
		return err
	}
}
