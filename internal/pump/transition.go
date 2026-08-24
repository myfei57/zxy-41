package pump

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Command actions understood by the pump transition logic.
const (
	ActionStart     = "start"
	ActionStop      = "stop"
	ActionSetTarget = "set_target"
)

// ErrUnknownAction is returned for an unsupported command action.
var ErrUnknownAction = errors.New("pump: unknown command action")

// Command is one dispatch instruction targeting a pump group.
type Command struct {
	ID        string
	GroupID   string
	UnitID    string
	Action    string
	TargetMPA float64
	IssueSeq  int64
	IssuedAt  time.Time
}

// NewCommand builds a command with a fresh id.
func NewCommand(groupID, unitID, action string) Command {
	return Command{
		ID:       uuid.NewString(),
		GroupID:  groupID,
		UnitID:   unitID,
		Action:   action,
		IssuedAt: time.Now().UTC(),
	}
}

// Apply executes one command on the group. The whole transition is serialized
// on the group mutex so concurrent dispatch sessions cannot interleave.
func Apply(ctx context.Context, group *Group, command Command) error {
	group.mu.Lock()
	defer group.mu.Unlock()
	return applyLocked(group, command)
}

func applyLocked(group *Group, command Command) error {
	switch command.Action {
	case ActionStart:
		unit, err := findUnit(group, command.UnitID)
		if err != nil {
			return err
		}
		if unit.Locked {
			return fmt.Errorf("pump: unit %s is locked out", command.UnitID)
		}
		for _, candidate := range group.Units {
			if candidate.ID == command.UnitID {
				candidate.Status = StatusRunning
			} else if candidate.Status == StatusRunning {
				candidate.Status = StatusStandby
			}
		}
		group.ActiveUnit = command.UnitID
	case ActionStop:
		unit, err := findUnit(group, command.UnitID)
		if err != nil {
			return err
		}
		unit.Status = StatusStandby
		if group.ActiveUnit == unit.ID {
			group.ActiveUnit = ""
		}
	case ActionSetTarget:
		if command.TargetMPA < 0 || command.TargetMPA > 1.5 {
			return fmt.Errorf("pump: target %.3f out of range", command.TargetMPA)
		}
		group.TargetMPA = command.TargetMPA
	default:
		return fmt.Errorf("%w: %s", ErrUnknownAction, command.Action)
	}
	return nil
}

// ApplyCommandList applies a batch of commands in order and returns the count.
func ApplyCommandList(ctx context.Context, group *Group, commands []Command) (int, error) {
	for _, command := range commands {
		if err := Apply(ctx, group, command); err != nil {
			return 0, err
		}
	}
	return len(commands), nil
}
