package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"pzlauncher/apps/pz-agent/internal/discover"
)

func TestRegister_Success(t *testing.T) {
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/agents/register" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "issued-token"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "my-server").WithServerName("my-server")
	token, err := client.Register(context.Background(), "42.10.1")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if token != "issued-token" {
		t.Fatalf("expected issued-token, got %q", token)
	}
	if gotBody["serverId"] != "my-server" {
		t.Fatalf("expected serverId in body, got %v", gotBody)
	}
	if gotBody["gameVersion"] != "42.10.1" {
		t.Fatalf("expected gameVersion 42.10.1 in body, got %v", gotBody)
	}
}

func TestRegister_OmitsUnknownGameVersion(t *testing.T) {
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "t"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "srv")
	if _, err := client.Register(context.Background(), "unknown"); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, ok := gotBody["gameVersion"]; ok {
		t.Fatalf("expected gameVersion omitted for 'unknown', got %v", gotBody)
	}
}

func TestRegister_ClientErrorIsPermanent(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "srv")
	if _, err := client.Register(context.Background(), ""); err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Fatalf("expected exactly 1 attempt for a 4xx (permanent), got %d", attempts)
	}
}

func TestPushBlob_SkipsUploadWhenAlreadyPresent(t *testing.T) {
	putCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.WriteHeader(http.StatusOK)
		case http.MethodPut:
			putCalls++
			w.WriteHeader(http.StatusCreated)
		}
	}))
	defer srv.Close()

	mod := writeTempMod(t, "hello world")
	client := NewClient(srv.URL, "srv").WithToken("tok")
	if err := client.PushBlob(context.Background(), mod); err != nil {
		t.Fatalf("push blob: %v", err)
	}
	if putCalls != 0 {
		t.Fatalf("expected no PUT when HEAD reports blob present, got %d", putCalls)
	}
}

func TestPushBlob_UploadsWhenMissing(t *testing.T) {
	var gotAuth string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.WriteHeader(http.StatusNotFound)
		case http.MethodPut:
			gotAuth = r.Header.Get("X-Agent-Token")
			gotBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
		}
	}))
	defer srv.Close()

	mod := writeTempMod(t, "some mod content")
	client := NewClient(srv.URL, "srv").WithToken("tok-123")
	if err := client.PushBlob(context.Background(), mod); err != nil {
		t.Fatalf("push blob: %v", err)
	}
	if gotAuth != "tok-123" {
		t.Fatalf("expected auth header tok-123, got %q", gotAuth)
	}
	if string(gotBody) != "some mod content" {
		t.Fatalf("expected uploaded content to match, got %q", gotBody)
	}
}

func TestPushBlob_401ReturnsErrUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.WriteHeader(http.StatusNotFound)
		case http.MethodPut:
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer srv.Close()

	mod := writeTempMod(t, "content")
	client := NewClient(srv.URL, "srv").WithToken("stale-token")
	err := client.PushBlob(context.Background(), mod)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestPublishManifest_SendsWorkshopIDWhenPresent(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "srv").WithToken("tok")
	mods := []discover.Mod{{ID: "ModA", Name: "Mod A", SHA256: "abc", WorkshopID: "123456"}}
	if err := client.PublishManifest(context.Background(), mods, "42.8", "v1"); err != nil {
		t.Fatalf("publish manifest: %v", err)
	}
	modEntries, _ := gotBody["mods"].([]any)
	if len(modEntries) != 1 {
		t.Fatalf("expected 1 mod entry, got %v", gotBody)
	}
	entry := modEntries[0].(map[string]any)
	if entry["workshopId"] != "123456" {
		t.Fatalf("expected workshopId 123456, got %v", entry["workshopId"])
	}
}

func TestPublishManifest_401ReturnsErrUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "srv").WithToken("stale")
	err := client.PublishManifest(context.Background(), []discover.Mod{{ID: "A", SHA256: "x"}}, "42.8", "v1")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestHeartbeat_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "srv").WithToken("tok")
	if err := client.Heartbeat(context.Background(), 3); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
}

func TestHeartbeat_401ReturnsErrUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "srv").WithToken("stale")
	err := client.Heartbeat(context.Background(), 1)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

// writeTempMod creates a temp file with the given content and returns a
// discover.Mod describing it (matching what discover.Scan would produce).
func writeTempMod(t *testing.T, content string) discover.Mod {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "TestMod.zip")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp mod: %v", err)
	}
	sum := sha256.Sum256([]byte(content))
	return discover.Mod{
		ID:        "TestMod",
		Name:      "TestMod",
		Path:      path,
		SHA256:    hex.EncodeToString(sum[:]),
		SizeBytes: int64(len(content)),
		Version:   "unknown",
	}
}
