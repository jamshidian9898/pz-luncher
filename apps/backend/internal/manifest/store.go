// Package manifest manages versioned server manifests.
//
// Design constraints (B4):
//   - Versions are sequential integers (1, 2, 3 …). "Latest" is always max.
//   - Storage is in-memory for Phase B. Disk persistence is B2/Phase-C scope.
//   - Diff is computed deterministically from SHA256 identity: if SHA256
//     matches, the mod is considered identical regardless of name or version.
//   - History is bounded to MaxVersions per server to avoid unbounded growth.
//   - All operations are safe for concurrent use.
package manifest

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// MaxVersions is the maximum number of versions kept per server.
const MaxVersions = 20

// ModEntry is a single mod in a versioned manifest.
// SHA256 is the canonical identity key for diff computation.
type ModEntry struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	SHA256       string   `json:"sha256"`
	SizeBytes    int64    `json:"sizeBytes,omitempty"`
	Dependencies []string `json:"dependencies"`
}

// Manifest is the immutable payload of a single version.
type Manifest struct {
	ServerID    string      `json:"serverId"`
	Version     int         `json:"version"`
	GameVersion string      `json:"gameVersion"`
	Mods        []ModEntry  `json:"mods"`
	LaunchArgs  []string    `json:"launchArgs"`
	Profile     interface{} `json:"profile"`
	PublishedAt time.Time   `json:"publishedAt"`
}

// VersionRecord is a stored manifest version with its raw JSON (for serving).
type VersionRecord struct {
	Version     int
	Manifest    *Manifest
	Raw         []byte // original JSON as received from Agent
	PublishedAt time.Time
}

// Diff describes what changed between two manifest versions.
type Diff struct {
	ServerID    string     `json:"serverId"`
	FromVersion int        `json:"fromVersion"`
	ToVersion   int        `json:"toVersion"`
	Added       []ModEntry `json:"added"`
	Removed     []ModEntry `json:"removed"`
	Updated     []ModEntry `json:"updated"`
	Unchanged   int        `json:"unchanged"`
}

// VersionSummary is used by the history endpoint.
type VersionSummary struct {
	Version     int       `json:"version"`
	ModCount    int       `json:"modCount"`
	PublishedAt time.Time `json:"publishedAt"`
}

// Store manages versioned manifests for all servers.
type Store struct {
	mu      sync.RWMutex
	servers map[string]*serverHistory
}

type serverHistory struct {
	versions []*VersionRecord // ordered oldest→newest, capped at MaxVersions
}

// NewStore creates an empty versioned manifest store.
func NewStore() *Store {
	return &Store{servers: make(map[string]*serverHistory)}
}

// agentManifest is a flexible parse target for Agent-submitted manifests.
// The "version" field from the Agent is a timestamp string (e.g. "20260605T…")
// which we discard — the backend assigns a sequential integer version.
type agentManifest struct {
	ServerID    string          `json:"serverId"`
	GameVersion string          `json:"gameVersion"`
	Mods        []ModEntry      `json:"mods"`
	LaunchArgs  []string        `json:"launchArgs"`
	Profile     json.RawMessage `json:"profile"`
	// Version intentionally omitted: backend overwrites it.
}

// Put stores a new manifest version for serverID.
// It parses rawJSON, assigns the next sequential version number, and trims
// history to MaxVersions. Returns the assigned version number.
func (s *Store) Put(serverID string, rawJSON []byte) (int, error) {
	if serverID == "" {
		return 0, fmt.Errorf("manifest: empty serverID")
	}

	// Parse incoming manifest using the flexible agent struct.
	var a agentManifest
	if err := json.Unmarshal(rawJSON, &a); err != nil {
		return 0, fmt.Errorf("manifest: parse: %w", err)
	}

	// Build a clean Manifest from the agent payload.
	m := Manifest{
		ServerID:    serverID,
		GameVersion: a.GameVersion,
		Mods:        a.Mods,
		LaunchArgs:  a.LaunchArgs,
		Profile:     a.Profile,
		PublishedAt: time.Now().UTC(),
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	h, ok := s.servers[serverID]
	if !ok {
		h = &serverHistory{}
		s.servers[serverID] = h
	}

	// Assign next version.
	nextVer := 1
	if len(h.versions) > 0 {
		nextVer = h.versions[len(h.versions)-1].Version + 1
	}
	m.Version = nextVer

	// Re-marshal with version field included.
	versioned, err := json.Marshal(m)
	if err != nil {
		return 0, fmt.Errorf("manifest: re-marshal: %w", err)
	}

	rec := &VersionRecord{
		Version:     nextVer,
		Manifest:    &m,
		Raw:         versioned,
		PublishedAt: m.PublishedAt,
	}
	h.versions = append(h.versions, rec)

	// Trim to MaxVersions.
	if len(h.versions) > MaxVersions {
		h.versions = h.versions[len(h.versions)-MaxVersions:]
	}

	return nextVer, nil
}

// Latest returns the most recent VersionRecord for serverID, or nil if none.
func (s *Store) Latest(serverID string) *VersionRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h, ok := s.servers[serverID]
	if !ok || len(h.versions) == 0 {
		return nil
	}
	return h.versions[len(h.versions)-1]
}

// LatestRaw returns the raw JSON of the latest version, or nil.
func (s *Store) LatestRaw(serverID string) []byte {
	r := s.Latest(serverID)
	if r == nil {
		return nil
	}
	return r.Raw
}

// Get returns the VersionRecord for a specific version number, or nil.
func (s *Store) Get(serverID string, version int) *VersionRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h, ok := s.servers[serverID]
	if !ok {
		return nil
	}
	for _, r := range h.versions {
		if r.Version == version {
			return r
		}
	}
	return nil
}

// History returns VersionSummary list for serverID, newest first.
func (s *Store) History(serverID string) []VersionSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h, ok := s.servers[serverID]
	if !ok {
		return nil
	}
	out := make([]VersionSummary, 0, len(h.versions))
	for i := len(h.versions) - 1; i >= 0; i-- {
		r := h.versions[i]
		out = append(out, VersionSummary{
			Version:     r.Version,
			ModCount:    len(r.Manifest.Mods),
			PublishedAt: r.PublishedAt,
		})
	}
	return out
}

// Diff computes the difference between two versions.
// If fromVersion == 0, all mods in toVersion are treated as Added.
func (s *Store) Diff(serverID string, fromVersion, toVersion int) (*Diff, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	h, ok := s.servers[serverID]
	if !ok {
		return nil, fmt.Errorf("manifest: server %q not found", serverID)
	}

	// Resolve toVersion.
	var toRec *VersionRecord
	for _, r := range h.versions {
		if r.Version == toVersion {
			toRec = r
			break
		}
	}
	if toRec == nil {
		return nil, fmt.Errorf("manifest: version %d not found for server %q", toVersion, serverID)
	}

	// Resolve fromVersion (may be absent for initial diff).
	var fromMods []ModEntry
	if fromVersion > 0 {
		for _, r := range h.versions {
			if r.Version == fromVersion {
				fromMods = r.Manifest.Mods
				break
			}
		}
		if fromMods == nil {
			return nil, fmt.Errorf("manifest: from-version %d not found for server %q", fromVersion, serverID)
		}
	}

	d := computeDiff(serverID, fromVersion, toVersion, fromMods, toRec.Manifest.Mods)
	return d, nil
}

// computeDiff is the pure diff function (no locking needed, called with lock held).
// Identity key: SHA256.
func computeDiff(serverID string, fromVer, toVer int, fromMods, toMods []ModEntry) *Diff {
	fromBySHA := make(map[string]ModEntry, len(fromMods))
	for _, m := range fromMods {
		fromBySHA[m.SHA256] = m
	}
	toBySHA := make(map[string]ModEntry, len(toMods))
	for _, m := range toMods {
		toBySHA[m.SHA256] = m
	}

	d := &Diff{
		ServerID:    serverID,
		FromVersion: fromVer,
		ToVersion:   toVer,
	}

	// Added or Updated: in "to" but not in "from" (by SHA256).
	for sha, toMod := range toBySHA {
		if _, exists := fromBySHA[sha]; !exists {
			// Check if same modID exists with different SHA256 → Updated.
			foundOld := false
			for _, fromMod := range fromMods {
				if fromMod.ID == toMod.ID {
					d.Updated = append(d.Updated, toMod)
					foundOld = true
					break
				}
			}
			if !foundOld {
				d.Added = append(d.Added, toMod)
			}
		} else {
			d.Unchanged++
		}
	}

	// Removed: in "from" but not in "to" (by SHA256), and modID also absent.
	for sha, fromMod := range fromBySHA {
		if _, exists := toBySHA[sha]; !exists {
			// Only removed if the modID is also gone from "to".
			if _, idExists := toBySHA[modIDKey(toMods, fromMod.ID)]; !idExists {
				d.Removed = append(d.Removed, fromMod)
			}
		}
	}

	return d
}

// modIDKey returns the SHA256 of the mod with the given ID in mods, or "".
func modIDKey(mods []ModEntry, id string) string {
	for _, m := range mods {
		if m.ID == id {
			return m.SHA256
		}
	}
	return ""
}
