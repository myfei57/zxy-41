package policy

import (
	"errors"
	"fmt"

	"waternet/internal/pump"
)

// ErrNoAction is returned when the zone is already within the envelope.
var ErrNoAction = errors.New("policy: no action required")

// Decider translates evaluation results into dispatch commands.
type Decider struct {
	config *Config
}

// NewDecider creates a decider over the published policy config.
func NewDecider(config *Config) *Decider {
	return &Decider{config: config}
}

// Decide maps one evaluation result to a pump command. A low zone starts the
// preferred unit, a high zone stops it, and an off-target zone tunes the
// discharge target.
func (d *Decider) Decide(zone string, result Result, groupID, unitID string) (pump.Command, error) {
	if result.Action == "" {
		return pump.Command{}, fmt.Errorf("%w for zone %s", ErrNoAction, zone)
	}
	switch result.Action {
	case "raise":
		return pump.NewCommand(groupID, unitID, pump.ActionStart), nil
	case "lower":
		return pump.NewCommand(groupID, unitID, pump.ActionStop), nil
	case "tune":
		target, err := d.config.TargetFor(zone)
		if err != nil {
			return pump.Command{}, err
		}
		command := pump.NewCommand(groupID, unitID, pump.ActionSetTarget)
		command.TargetMPA = target.TargetMPA
		return command, nil
	default:
		return pump.Command{}, fmt.Errorf("policy: unknown action %q", result.Action)
	}
}

// DecideBatch maps several results to commands, skipping zones without action.
func (d *Decider) DecideBatch(zone string, results []Result, groupID, unitID string) ([]pump.Command, error) {
	var commands []pump.Command
	for _, result := range results {
		command, err := d.Decide(zone, result, groupID, unitID)
		if errors.Is(err, ErrNoAction) {
			continue
		}
		if err != nil {
			return nil, err
		}
		commands = append(commands, command)
	}
	return commands, nil
}
