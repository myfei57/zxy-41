package store

import (
	"encoding/json"
	"fmt"
)

const executionLog = "executions.jsonl"

// ExecutionLog persists pump and valve execution receipts.
type ExecutionLog struct {
	fs *FileStore
}

// NewExecutionLog creates a receipt log backed by the given store.
func NewExecutionLog(fs *FileStore) *ExecutionLog {
	return &ExecutionLog{fs: fs}
}

// Append writes one receipt.
func (l *ExecutionLog) Append(record ExecutionRecord) error {
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("execution: marshal: %w", err)
	}
	return l.fs.AppendLine(executionLog, data)
}

// Recent returns the newest receipts, at most limit entries.
func (l *ExecutionLog) Recent(limit int) ([]ExecutionRecord, error) {
	lines, err := l.fs.ReadLines(executionLog)
	if err != nil {
		return nil, err
	}
	records := make([]ExecutionRecord, 0, len(lines))
	for _, line := range lines {
		var record ExecutionRecord
		if err := json.Unmarshal(line, &record); err != nil {
			return nil, fmt.Errorf("execution: decode row: %w", err)
		}
		records = append(records, record)
	}
	if limit > 0 && len(records) > limit {
		records = records[len(records)-limit:]
	}
	return records, nil
}

// CountByTarget counts receipts for one target id.
func (l *ExecutionLog) CountByTarget(target string) (int, error) {
	records, err := l.Recent(0)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, record := range records {
		if record.Target == target {
			count++
		}
	}
	return count, nil
}
