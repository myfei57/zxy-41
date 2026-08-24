package dispatch

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"waternet/internal/pump"
)

// Order wraps one pump command with its issue sequence number.
type Order struct {
	ID        string
	IssueSeq  int64
	Command   pump.Command
	ArrivedAt time.Time
}

// NewOrder builds an order with a fresh id and arrival time.
func NewOrder(issueSeq int64, command pump.Command) Order {
	return Order{
		ID:        uuid.NewString(),
		IssueSeq:  issueSeq,
		Command:   command,
		ArrivedAt: time.Now().UTC(),
	}
}

// OrderQueue holds dispatch orders until a station applies them. Orders are
// served in issue order, never in arrival order.
type OrderQueue struct {
	mu     sync.Mutex
	orders []Order
	cursor int
}

// NewOrderQueue creates an empty order queue.
func NewOrderQueue() *OrderQueue {
	return &OrderQueue{}
}

// Enqueue appends one order.
func (q *OrderQueue) Enqueue(order Order) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.orders = append(q.orders, order)
}

// Len returns the number of orders still pending.
func (q *OrderQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.orders) - q.cursor
}

// Next returns the pending order with the smallest issue sequence.
func (q *OrderQueue) Next() (Order, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.cursor >= len(q.orders) {
		return Order{}, false
	}
	best := q.cursor
	for index := q.cursor + 1; index < len(q.orders); index++ {
		if q.orders[index].IssueSeq < q.orders[best].IssueSeq {
			best = index
		}
	}
	q.orders[q.cursor], q.orders[best] = q.orders[best], q.orders[q.cursor]
	order := q.orders[q.cursor]
	q.cursor++
	return order, true
}

// Reset restores the queue to empty.
func (q *OrderQueue) Reset() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.orders = nil
	q.cursor = 0
}
