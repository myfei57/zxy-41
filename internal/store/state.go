package store

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
)

// StateFile is the well-known snapshot file inside the store directory.
const StateFile = "state.json"

// SaveState persists the current network snapshot.
func (s *FileStore) SaveState(record StateRecord) error {
	return s.WriteJSON(StateFile, record)
}

// LoadState reads the latest persisted snapshot.
func (s *FileStore) LoadState() (StateRecord, error) {
	var record StateRecord
	if err := s.ReadJSON(StateFile, &record); err != nil {
		return record, err
	}
	return record, nil
}

// HasState reports whether a snapshot has been persisted before.
func (s *FileStore) HasState() bool {
	return s.Exists(StateFile)
}

// StateFingerprint returns a hex digest over the persisted snapshot bytes.
func (s *FileStore) StateFingerprint() (string, error) {
	data, err := os.ReadFile(s.Path(StateFile))
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
