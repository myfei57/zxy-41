package store

import (
	"fmt"
	"time"
)

// SnapshotService rotates the main state file into timestamped archives so a
// recovery can always find the latest durable snapshot plus its history.
type SnapshotService struct {
	fs *FileStore
}

// NewSnapshotService creates a snapshot rotator for the given store.
func NewSnapshotService(fs *FileStore) *SnapshotService {
	return &SnapshotService{fs: fs}
}

// Write persists a snapshot and returns its content digest.
func (s *SnapshotService) Write(record StateRecord) (string, error) {
	if err := s.fs.SaveState(record); err != nil {
		return "", err
	}
	return s.fs.StateFingerprint()
}

// Rotate moves the current state file into the snapshots directory.
func (s *SnapshotService) Rotate() (string, error) {
	name := fmt.Sprintf("state-%s.json", time.Now().UTC().Format("20060102T150405"))
	if !s.fs.Exists(StateFile) {
		return name, nil
	}
	data, err := s.fs.ReadBytes(StateFile)
	if err != nil {
		return "", err
	}
	if err := s.fs.WriteBytes("snapshots/"+name, data); err != nil {
		return "", err
	}
	if err := s.fs.Remove(StateFile); err != nil {
		return "", err
	}
	return name, nil
}

// Latest loads the newest archived snapshot, falling back to the live file.
func (s *SnapshotService) Latest() (StateRecord, bool, error) {
	var record StateRecord
	names, err := s.fs.ListNames("snapshots")
	if err != nil {
		return record, false, err
	}
	for i := len(names) - 1; i >= 0; i-- {
		if err := s.fs.ReadJSON("snapshots/"+names[i], &record); err == nil {
			return record, true, nil
		}
	}
	if s.fs.HasState() {
		if err := s.fs.ReadJSON(StateFile, &record); err != nil {
			return record, false, err
		}
		return record, true, nil
	}
	return record, false, nil
}
