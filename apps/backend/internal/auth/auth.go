// Package auth implements the minimal Agent trust boundary for Phase A6.
//
// Design decisions:
//   - Tokens are opaque random strings (crypto/rand, 32 bytes hex-encoded).
//   - Storage is in-memory only for Phase A — no JWT, no DB, no expiry logic.
//   - One token per serverId. Re-registering invalidates the old token.
//   - Validation is O(1) via a reverse map (token → serverID).
//   - The registration endpoint itself is NOT token-protected so agents can
//     bootstrap. In production (A7+) this would require a one-time install key.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
)

const tokenHeader = "X-Agent-Token"

// TokenHeader is the HTTP header name agents must include.
const TokenHeader = tokenHeader

// Store issues and validates Agent tokens.
type Store struct {
	mu           sync.RWMutex
	byServer     map[string]string // serverID → token
	byToken      map[string]string // token    → serverID
}

// NewStore creates an empty token store.
func NewStore() *Store {
	return &Store{
		byServer: make(map[string]string),
		byToken:  make(map[string]string),
	}
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
func (s *Store) Revoke(serverID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tok, ok := s.byServer[serverID]; ok {
		delete(s.byToken, tok)
		delete(s.byServer, serverID)
	}
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
