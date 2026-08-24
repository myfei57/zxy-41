package ns

import (
	"fmt"
	"sort"
	"sync"

	"waternet/internal/store"
)

// Registry owns the namespaces of the whole water network and persists their
// layout into the shared file store.
type Registry struct {
	mu         sync.RWMutex
	namespaces map[string]*Namespace
	fs         *store.FileStore
}

// NewRegistry creates an empty registry over the given store.
func NewRegistry(fs *store.FileStore) *Registry {
	return &Registry{namespaces: map[string]*Namespace{}, fs: fs}
}

// Register adds a namespace; duplicate ids are rejected.
func (r *Registry) Register(n *Namespace) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.namespaces[n.ID]; exists {
		return fmt.Errorf("ns: namespace %s already registered", n.ID)
	}
	r.namespaces[n.ID] = n
	return r.saveLocked()
}

// Get returns one namespace by id.
func (r *Registry) Get(id string) (*Namespace, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n, ok := r.namespaces[id]
	if !ok {
		return nil, fmt.Errorf("ns: namespace %s not found", id)
	}
	return n, nil
}

// All returns every registered namespace in stable order.
func (r *Registry) All() []*Namespace {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.namespaces))
	for id := range r.namespaces {
		names = append(names, id)
	}
	sort.Strings(names)
	out := make([]*Namespace, 0, len(names))
	for _, id := range names {
		out = append(out, r.namespaces[id])
	}
	return out
}

// Save persists the namespace layout.
func (r *Registry) Save() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.saveLocked()
}

func (r *Registry) saveLocked() error {
	ids := make([]string, 0, len(r.namespaces))
	for id := range r.namespaces {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	records := make([]store.NamespaceRecord, 0, len(ids))
	for _, id := range ids {
		n := r.namespaces[id]
		records = append(records, store.NamespaceRecord{
			ID:    n.ID,
			Name:  n.Name,
			Zones: n.ZoneIDs(),
		})
	}
	return r.fs.WriteJSON("namespaces.json", records)
}

// Load restores the registry from disk, replacing current contents.
func (r *Registry) Load() error {
	var records []store.NamespaceRecord
	if err := r.fs.ReadJSON("namespaces.json", &records); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.namespaces = map[string]*Namespace{}
	for _, record := range records {
		n := NewNamespace(record.ID, record.Name)
		for _, zoneID := range record.Zones {
			_ = n.AddZone(zoneID, zoneID)
		}
		r.namespaces[n.ID] = n
	}
	return nil
}
