package station

import (
	"context"

	"waternet/internal/dispatch"
	"waternet/internal/pump"
)

// ApplyDispatch drains the order queue and applies every command to the
// matching pump group in issue order.
func ApplyDispatch(ctx context.Context, s *Station, queue *dispatch.OrderQueue) (int, error) {
	count := 0
	for {
		order, ok := queue.Next()
		if !ok {
			break
		}
		group, err := s.Group(order.Command.GroupID)
		if err != nil {
			return count, err
		}
		if err := pump.Apply(ctx, group, order.Command); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// DispatchCommand enqueues one command into the station order queue.
func DispatchCommand(queue *dispatch.OrderQueue, seq int64, command pump.Command) error {
	queue.Enqueue(dispatch.NewOrder(seq, command))
	return nil
}
