package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"pzlauncher/apps/backend/internal/auth"
	"pzlauncher/apps/backend/internal/registry"
	"pzlauncher/apps/backend/internal/storage"
)

func newTestRouter(t *testing.T) (http.Handler, *auth.Store, *registry.Registry) {
	t.Helper()
	reg := registry.NewMemoryRegistry()
	dir := t.TempDir()
	store, err := storage.NewDiskStore(filepath.Join(dir, "store"))
	if err != nil {
		t.Fatalf("new disk store: %v", err)
	}
	tokens := auth.NewStore()
	mux := NewRouter(reg, "http://localhost:8080", store, tokens, "", nil)
	return mux, tokens, reg
}

func doJSON(t *testing.T, mux http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(data)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestHealth(t *testing.T) {
	mux, _, _ := newTestRouter(t)
	rec := doJSON(t, mux, http.MethodGet, "/api/v1/health", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestServers_GetUnknownReturns404(t *testing.T) {
	mux, _, _ := newTestRouter(t)
	rec := doJSON(t, mux, http.MethodGet, "/api/v1/servers/nope", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestAgentRegister_AutoCreatesServer(t *testing.T) {
	mux, tokens, reg := newTestRouter(t)
	rec := doJSON(t, mux, http.MethodPost, "/api/v1/agents/register", map[string]string{
		"serverId":    "srv-1",
		"gameVersion": "42.10.1",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Token == "" {
		t.Fatal("expected a token in the response")
	}
	if _, ok := tokens.Validate(resp.Token); !ok {
		t.Fatal("expected issued token to validate")
	}
	srv, ok := reg.Get("srv-1")
	if !ok || srv.GameVersion != "42.10.1" {
		t.Fatalf("expected auto-created server with gameVersion 42.10.1, got %+v ok=%v", srv, ok)
	}
}

func TestBlobPut_RequiresToken(t *testing.T) {
	mux, _, _ := newTestRouter(t)
	body := bytes.NewReader([]byte("x"))
	req := httptest.NewRequest(http.MethodPut, "/api/v1/blobs/somehash", body)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}
}

func TestBlobPut_SucceedsWithValidToken(t *testing.T) {
	mux, tokens, _ := newTestRouter(t)
	token, err := tokens.Register("srv-1")
	if err != nil {
		t.Fatalf("register token: %v", err)
	}

	content := "hello"
	// sha256("hello")
	hash := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	req := httptest.NewRequest(http.MethodPut, "/api/v1/blobs/"+hash, bytes.NewReader([]byte(content)))
	req.Header.Set("X-Agent-Token", token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHeartbeat_UpdatesAgentList(t *testing.T) {
	mux, tokens, _ := newTestRouter(t)
	token, _ := tokens.Register("srv-1")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/heartbeat", bytes.NewReader(mustJSON(t, map[string]any{
		"serverId": "srv-1",
		"modCount": 3,
	})))
	req.Header.Set("X-Agent-Token", token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec2 := doJSON(t, mux, http.MethodGet, "/api/v1/agents", nil)
	var resp struct {
		Agents []struct {
			ServerID string `json:"serverId"`
			ModCount int    `json:"modCount"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Agents) != 1 || resp.Agents[0].ServerID != "srv-1" || resp.Agents[0].ModCount != 3 {
		t.Fatalf("unexpected agents list: %+v", resp.Agents)
	}
}

func TestAgentRevoke_SelfServiceOnly(t *testing.T) {
	mux, tokens, _ := newTestRouter(t)
	token, _ := tokens.Register("srv-1")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/revoke", nil)
	req.Header.Set("X-Agent-Token", token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if _, ok := tokens.Validate(token); ok {
		t.Fatal("expected token to be revoked")
	}
}

func TestAgentRevoke_RequiresValidToken(t *testing.T) {
	mux, _, _ := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/revoke", nil)
	req.Header.Set("X-Agent-Token", "not-a-real-token")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return data
}
