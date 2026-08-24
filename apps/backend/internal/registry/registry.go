// Package registry manages the server list (in-memory, loaded from registry.json).
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"pzlauncher/apps/backend/internal/manifest"
)

// ServerRecord is a single server entry in the registry.
type ServerRecord struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Region       string   `json:"region,omitempty"`
	GameVersion  string   `json:"gameVersion,omitempty"`
	PlayerCount  int      `json:"playerCount"`
	MaxPlayers   int      `json:"maxPlayers"`
	Status       string   `json:"status"` // "online" | "offline"
	Tags         []string `json:"tags,omitempty"`
	ManifestPath string   `json:"manifestPath,omitempty"`
}

// AgentStatus represents the liveness state of a registered Agent.
type AgentStatus string

const (
	AgentOnline   AgentStatus = "online"
	AgentDegraded AgentStatus = "degraded" // no heartbeat for >2×interval
	AgentOffline  AgentStatus = "offline"  // no heartbeat for >5 min
)

// AgentState holds the last-known state for a registered Agent.
type AgentState struct {
	ServerID string      `json:"serverId"`
	Status   AgentStatus `json:"status"`
	LastSeen time.Time   `json:"lastSeen"`
	ModCount int         `json:"modCount"`
	Version  string      `json:"version,omitempty"`
}

// Registry holds a mutable in-memory list of servers, versioned manifests, and agent liveness.
type Registry struct {
	mu        sync.RWMutex
	servers   map[string]*ServerRecord
	manifests *manifest.Store        // versioned manifest store (B4)
	agents    map[string]*AgentState // serverID → agent liveness
	filePath  string                 // "" = no write-back (in-memory only)
}

type registryFile struct {
	Servers []*ServerRecord `json:"servers"`
}

// NewMemoryRegistry returns an empty registry.
func NewMemoryRegistry() *Registry {
	return &Registry{
		servers:   make(map[string]*ServerRecord),
		manifests: manifest.NewStore(),
		agents:    make(map[string]*AgentState),
	}
}

// LoadFromFile reads a JSON file and returns a populated Registry.
// The Registry remembers path and writes back to it (best-effort) whenever
// Upsert adds or replaces a server — e.g. servers an Agent auto-registers —
// so they survive a Backend restart instead of only existing in memory.
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
	reg.filePath = path
	for _, s := range f.Servers { //nolint:gocritic
		if s.Status == "" {
			s.Status = "online"
		}
		reg.servers[s.ID] = s
	}
	return reg, nil
}

// SetManifestStore replaces the Registry's manifest store — used to swap in
// a disk-backed manifest.Store (see manifest.NewDiskStore) after construction.
func (r *Registry) SetManifestStore(m *manifest.Store) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.manifests = m
}

// UpsertManifest stores a new versioned manifest for serverID (pushed by Agent).
// Returns the assigned version number.
func (r *Registry) UpsertManifest(serverID string, data []byte) (int, error) {
	return r.manifests.Put(serverID, data)
}

// GetManifest returns the raw JSON of the latest manifest for serverID, or nil.
func (r *Registry) GetManifest(serverID string) []byte {
	return r.manifests.LatestRaw(serverID)
}

// ManifestStore exposes the underlying versioned manifest.Store for
// endpoints that need history or diff access.
func (r *Registry) ManifestStore() *manifest.Store {
	return r.manifests
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

// Upsert adds or replaces a server record and writes the registry back to
// disk (if loaded via LoadFromFile) so the change survives a restart.
// Returns any write-back error; the in-memory update always succeeds.
func (r *Registry) Upsert(s *ServerRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.servers[s.ID] = s
	return r.saveLocked()
}

// saveLocked writes the current server list to r.filePath. Caller must hold
// r.mu. No-op when the Registry wasn't loaded from a file.
func (r *Registry) saveLocked() error {
	if r.filePath == "" {
		return nil
	}
	servers := make([]*ServerRecord, 0, len(r.servers))
	for _, s := range r.servers {
		servers = append(servers, s)
	}
	data, err := json.MarshalIndent(registryFile{Servers: servers}, "", "  ")
	if err != nil {
		return fmt.Errorf("registry: marshal: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(r.filePath), 0o755); err != nil {
		return fmt.Errorf("registry: mkdir: %w", err)
	}
	return os.WriteFile(r.filePath, data, 0o644)
}

// RecordHeartbeat updates the agent state for serverID.
func (r *Registry) RecordHeartbeat(serverID string, modCount int, version string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	state, ok := r.agents[serverID]
	if !ok {
		state = &AgentState{ServerID: serverID}
		r.agents[serverID] = state
	}
	state.LastSeen = time.Now().UTC()
	state.ModCount = modCount
	state.Status = AgentOnline
	if version != "" {
		state.Version = version
	}
}

// AgentStateFor returns the current AgentState for serverID, computing
// derived status (degraded/offline) based on staleness.
func (r *Registry) AgentStateFor(serverID string) *AgentState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	state, ok := r.agents[serverID]
	if !ok {
		return nil
	}
	// Return a copy with computed status.
	copy := *state
	copy.Status = computeStatus(state.LastSeen)
	return &copy
}

// ListAgents returns all known AgentStates with computed status.
func (r *Registry) ListAgents() []*AgentState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*AgentState, 0, len(r.agents))
	for _, a := range r.agents {
		copy := *a
		copy.Status = computeStatus(a.LastSeen)
		out = append(out, &copy)
	}
	return out
}

func computeStatus(lastSeen time.Time) AgentStatus {
	age := time.Since(lastSeen)
	switch {
	case age < 90*time.Second:
		return AgentOnline
	case age < 5*time.Minute:
		return AgentDegraded
	default:
		return AgentOffline
	}
}
