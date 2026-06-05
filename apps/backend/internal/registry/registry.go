// Package registry manages the server list (in-memory, loaded from registry.json).
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// ServerRecord is a single server entry in the registry.
type ServerRecord struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Region      string   `json:"region,omitempty"`
	GameVersion string   `json:"gameVersion,omitempty"`
	PlayerCount int      `json:"playerCount"`
	MaxPlayers  int      `json:"maxPlayers"`
	Status      string   `json:"status"` // "online" | "offline"
	Tags        []string `json:"tags,omitempty"`
	ManifestPath string  `json:"manifestPath,omitempty"`
}

// Registry holds a mutable in-memory list of servers.
type Registry struct {
	mu      sync.RWMutex
	servers map[string]*ServerRecord
}

type registryFile struct {
	Servers []*ServerRecord `json:"servers"`
}

// NewMemoryRegistry returns an empty registry.
func NewMemoryRegistry() *Registry {
	return &Registry{servers: make(map[string]*ServerRecord)}
}

// LoadFromFile reads a JSON file and returns a populated Registry.
func LoadFromFile(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("registry: read %q: %w", path, err)
	}
	var f registryFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("registry: parse %q: %w", path, err)
	}
	reg := NewMemoryRegistry()
	for _, s := range f.Servers {
		if s.Status == "" {
			s.Status = "online"
		}
		reg.servers[s.ID] = s
	}
	return reg, nil
}

// List returns all servers as a slice.
func (r *Registry) List() []*ServerRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ServerRecord, 0, len(r.servers))
	for _, s := range r.servers {
		out = append(out, s)
	}
	return out
}

// Get returns a single server by ID.
func (r *Registry) Get(id string) (*ServerRecord, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.servers[id]
	return s, ok
}

// Upsert adds or replaces a server record.
func (r *Registry) Upsert(s *ServerRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.servers[s.ID] = s
}
