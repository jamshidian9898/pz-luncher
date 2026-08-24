// Package auth implements the minimal Agent trust boundary for Phase A6.
//
// Design decisions:
//   - Tokens are opaque random strings (crypto/rand, 32 bytes hex-encoded).
//   - One token per serverId. Re-registering invalidates the old token.
//   - Validation is O(1) via a reverse map (token → serverID).
//   - The registration endpoint itself is NOT token-protected so agents can
//     bootstrap. In production (A7+) this would require a one-time install key.
//   - Optionally persisted to a JSON file (dataPath) so a Backend restart
//     doesn't force every Agent to re-register. Persistence is best-effort:
//     an empty dataPath keeps the store in-memory only (e.g. tests).
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const tokenHeader = "X-Agent-Token"

// TokenHeader is the HTTP header name agents must include.
const TokenHeader = tokenHeader

// Store issues and validates Agent tokens.
type Store struct {
	mu       sync.RWMutex
	byServer map[string]string // serverID → token
	byToken  map[string]string // token    → serverID
	dataPath string
}

// NewStore creates an empty, in-memory-only token store (no persistence).
func NewStore() *Store {
	return &Store{
		byServer: make(map[string]string),
		byToken:  make(map[string]string),
	}
}

// NewPersistentStore creates a token store backed by dataPath. If dataPath
// exists it is loaded; otherwise an empty store is created. Every Register
// and Revoke call persists the updated state (best-effort — a write failure
// is returned but the in-memory state still reflects the change).
func NewPersistentStore(dataPath string) (*Store, error) {
	s := NewStore()
	s.dataPath = dataPath
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// Register issues a new token for serverID.
// If serverID was already registered the old token is revoked.
func (s *Store) Register(serverID string) (string, error) {
	if serverID == "" {
		return "", fmt.Errorf("auth: serverID must not be empty")
	}
	token, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("auth: generate token: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Revoke old token if present
	if old, ok := s.byServer[serverID]; ok {
		delete(s.byToken, old)
	}

	s.byServer[serverID] = token
	s.byToken[token] = serverID
	if err := s.persist(); err != nil {
		return token, fmt.Errorf("auth: persist: %w", err)
	}
	return token, nil
}

// Validate checks whether token is valid and returns the associated serverID.
// Returns ("", false) for unknown or revoked tokens.
func (s *Store) Validate(token string) (serverID string, ok bool) {
	if token == "" {
		return "", false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, exists := s.byToken[token]
	return id, exists
}

// Revoke removes a token by serverID (e.g. on server decommission).
func (s *Store) Revoke(serverID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tok, ok := s.byServer[serverID]; ok {
		delete(s.byToken, tok)
		delete(s.byServer, serverID)
	}
	return s.persist()
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// persistedState is the on-disk representation of the token store.
type persistedState struct {
	ByServer map[string]string `json:"byServer"`
}

// load reads dataPath into the store. A missing file is not an error (first
// run). Caller must hold no lock — load is only called from NewPersistentStore
// before the store is shared.
func (s *Store) load() error {
	if s.dataPath == "" {
		return nil
	}
	data, err := os.ReadFile(s.dataPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("auth: load %q: %w", s.dataPath, err)
	}
	var state persistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("auth: parse %q: %w", s.dataPath, err)
	}
	for serverID, token := range state.ByServer {
		s.byServer[serverID] = token
		s.byToken[token] = serverID
	}
	return nil
}

// persist writes the current state to dataPath. Caller must hold s.mu.
// A no-op (nil error) when dataPath is empty.
func (s *Store) persist() error {
	if s.dataPath == "" {
		return nil
	}
	state := persistedState{ByServer: s.byServer}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.dataPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.dataPath, data, 0o600)
}
