package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ErrNotFound is returned when a requested record does not exist on disk.
var ErrNotFound = errors.New("store: record not found")

// FileStore persists small JSON records and append-only logs under one
// directory. All writes go through a temporary file plus rename so a crashed
// process never leaves a half-written record behind.
type FileStore struct {
	Dir string
}

// New opens the data directory, creating it when missing.
func New(dir string) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("store: create dir: %w", err)
	}
	return &FileStore{Dir: dir}, nil
}

// Path resolves a record name below the store directory.
func (s *FileStore) Path(name string) string {
	return filepath.Join(s.Dir, name)
}

// WriteJSON marshals the value and stores it atomically.
func (s *FileStore) WriteJSON(name string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("store: marshal %s: %w", name, err)
	}
	return s.WriteBytes(name, data)
}

// ReadJSON loads and decodes a stored record.
func (s *FileStore) ReadJSON(name string, target any) error {
	data, err := os.ReadFile(s.Path(name))
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s: %w", name, ErrNotFound)
		}
		return fmt.Errorf("store: read %s: %w", name, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("store: decode %s: %w", name, err)
	}
	return nil
}

// WriteBytes persists raw bytes atomically.
func (s *FileStore) WriteBytes(name string, data []byte) error {
	path := s.Path(name)
	if dir := filepath.Dir(path); dir != s.Dir {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("store: create dir for %s: %w", name, err)
		}
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("store: write %s: %w", name, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("store: commit %s: %w", name, err)
	}
	return nil
}

// ReadBytes returns the raw content of a stored record.
func (s *FileStore) ReadBytes(name string) ([]byte, error) {
	return os.ReadFile(s.Path(name))
}

// AppendLine appends one line to an append-only log, creating it on first use.
func (s *FileStore) AppendLine(name string, line []byte) error {
	path := s.Path(name)
	if dir := filepath.Dir(path); dir != s.Dir {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("store: create dir for %s: %w", name, err)
		}
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("store: append %s: %w", name, err)
	}
	defer file.Close()
	if len(line) == 0 || line[len(line)-1] != '\n' {
		line = append(line, '\n')
	}
	if _, err := file.Write(line); err != nil {
		return fmt.Errorf("store: append %s: %w", name, err)
	}
	return nil
}

// ReadLines returns every non-empty line of an append-only log.
func (s *FileStore) ReadLines(name string) ([][]byte, error) {
	data, err := os.ReadFile(s.Path(name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("store: read log %s: %w", name, err)
	}
	var out [][]byte
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) != "" {
			out = append(out, []byte(line))
		}
	}
	return out, nil
}

// Exists reports whether a record is present.
func (s *FileStore) Exists(name string) bool {
	_, err := os.Stat(s.Path(name))
	return err == nil
}

// ListNames returns the sorted names of regular files below a sub-directory.
func (s *FileStore) ListNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(s.Path(dir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("store: list %s: %w", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// Remove deletes a record, treating a missing file as success.
func (s *FileStore) Remove(name string) error {
	err := os.Remove(s.Path(name))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
