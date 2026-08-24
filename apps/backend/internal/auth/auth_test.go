package auth

import (
	"path/filepath"
	"testing"
)

func TestRegister_RoundTrip(t *testing.T) {
	s := NewStore()
	token, err := s.Register("server-a")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if token == "" {
		t.Fatal("expected a non-empty token")
	}
	serverID, ok := s.Validate(token)
	if !ok || serverID != "server-a" {
		t.Fatalf("expected valid token for server-a, got (%q, %v)", serverID, ok)
	}
}

func TestRegister_EmptyServerIDFails(t *testing.T) {
	s := NewStore()
	if _, err := s.Register(""); err == nil {
		t.Fatal("expected error for empty serverID")
	}
}

func TestRegister_ReRegisterRevokesOldToken(t *testing.T) {
	s := NewStore()
	old, _ := s.Register("server-a")
	fresh, _ := s.Register("server-a")

	if old == fresh {
		t.Fatal("expected a new token on re-registration")
	}
	if _, ok := s.Validate(old); ok {
		t.Fatal("expected old token to be invalidated after re-registration")
	}
	if _, ok := s.Validate(fresh); !ok {
		t.Fatal("expected new token to be valid")
	}
}

func TestValidate_UnknownTokenIsInvalid(t *testing.T) {
	s := NewStore()
	if _, ok := s.Validate("not-a-real-token"); ok {
		t.Fatal("expected unknown token to be invalid")
	}
	if _, ok := s.Validate(""); ok {
		t.Fatal("expected empty token to be invalid")
	}
}

func TestRevoke_InvalidatesToken(t *testing.T) {
	s := NewStore()
	token, _ := s.Register("server-a")
	if err := s.Revoke("server-a"); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, ok := s.Validate(token); ok {
		t.Fatal("expected revoked token to be invalid")
	}
}

func TestRevoke_UnknownServerIsNoop(t *testing.T) {
	s := NewStore()
	if err := s.Revoke("never-registered"); err != nil {
		t.Fatalf("expected no error revoking an unregistered server, got %v", err)
	}
}

func TestNewPersistentStore_SurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")

	s1, err := NewPersistentStore(path)
	if err != nil {
		t.Fatalf("new persistent store: %v", err)
	}
	token, err := s1.Register("server-a")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// Simulate a Backend restart: a fresh Store loaded from the same file.
	s2, err := NewPersistentStore(path)
	if err != nil {
		t.Fatalf("reload persistent store: %v", err)
	}
	serverID, ok := s2.Validate(token)
	if !ok || serverID != "server-a" {
		t.Fatalf("expected token to survive restart, got (%q, %v)", serverID, ok)
	}
}

func TestNewPersistentStore_MissingFileStartsEmpty(t *testing.T) {
	dir := t.TempDir()
	s, err := NewPersistentStore(filepath.Join(dir, "does-not-exist.json"))
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if _, ok := s.Validate("anything"); ok {
		t.Fatal("expected empty store")
	}
}

func TestRevoke_PersistsAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")

	s1, _ := NewPersistentStore(path)
	token, _ := s1.Register("server-a")
	if err := s1.Revoke("server-a"); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	s2, err := NewPersistentStore(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if _, ok := s2.Validate(token); ok {
		t.Fatal("expected revoked token to stay revoked after reload")
	}
}

func TestNewStore_NoPersistenceByDefault(t *testing.T) {
	// NewStore (not NewPersistentStore) must never touch disk.
	s := NewStore()
	if _, err := s.Register("server-a"); err != nil {
		t.Fatalf("register: %v", err)
	}
	if s.dataPath != "" {
		t.Fatalf("expected empty dataPath for NewStore, got %q", s.dataPath)
	}
}
