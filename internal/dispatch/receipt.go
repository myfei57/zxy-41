package dispatch

import (
	"sort"

	"waternet/internal/store"
)

// Receipts returns the newest execution receipts.
func (e *Executor) Receipts(limit int) ([]store.ExecutionRecord, error) {
	return e.receipts.Recent(limit)
}

// ReceiptSummary counts receipts by action.
func ReceiptSummary(records []store.ExecutionRecord) map[string]int {
	summary := map[string]int{}
	for _, record := range records {
		summary[record.Action]++
	}
	return summary
}

// SortedActions returns the summary keys in stable order.
func SortedActions(summary map[string]int) []string {
	actions := make([]string, 0, len(summary))
	for action := range summary {
		actions = append(actions, action)
	}
	sort.Strings(actions)
	return actions
}
