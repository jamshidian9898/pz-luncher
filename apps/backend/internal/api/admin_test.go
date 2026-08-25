package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"pzlauncher/apps/backend/internal/auth"
	"pzlauncher/apps/backend/internal/registry"
	"pzlauncher/apps/backend/internal/storage"
)

const adminSecret = "op-secret"
const adminHeader = "X-Admin-Token"

func newAdminTestRouter(t *testing.T) (http.Handler, *auth.Store, *registry.Registry) {
	t.Helper()
	reg := registry.NewMemoryRegistry()
	store, err := storage.NewDiskStore(filepath.Join(t.TempDir(), "store"))
	if err != nil {
		t.Fatalf("new disk store: %v", err)
	}
	tokens := auth.NewStore()
	mux := NewRouter(reg, "http://localhost:8080", store, tokens, adminSecret, nil)
	return mux, tokens, reg
}

func doAuth(t *testing.T, mux http.Handler, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestAdminRoutes_DisabledWithoutSecret(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	store, err := storage.NewDiskStore(t.TempDir())
	if err != nil {
		t.Fatalf("new disk store: %v", err)
	}
	mux := NewRouter(reg, "http://localhost:8080", store, auth.NewStore(), "", nil)

	rec := doJSON(t, mux, "GET", "/api/v1/admin/tokens", nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when admin token unset, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "ADMIN_NOT_CONFIGURED") {
		t.Fatalf("expected ADMIN_NOT_CONFIGURED code, got %s", rec.Body.String())
	}
}

func TestAdminRoutes_RejectBadSecret(t *testing.T) {
	mux, _, _ := newAdminTestRouter(t)

	if got := doJSON(t, mux, "GET", "/api/v1/admin/tokens", nil).Code; got != http.StatusUnauthorized {
		t.Fatalf("expected 401 without header, got %d", got)
	}
	hdrs := map[string]string{adminHeader: "wrong"}
	if got := doAuth(t, mux, "GET", "/api/v1/admin/tokens", hdrs).Code; got != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong secret, got %d", got)
	}
}

func TestAdminToken_IssueListRevoke(t *testing.T) {
	mux, tokens, _ := newAdminTestRouter(t)
	hdr := map[string]string{adminHeader: adminSecret}

	rec := doAuth(t, mux, "POST", "/api/v1/admin/tokens/pz-test-x", hdr)
	if rec.Code != http.StatusCreated {
		t.Fatalf("issue: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var issued struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &issued); err != nil || issued.Token == "" {
		t.Fatalf("issue body %q (err=%v)", rec.Body.String(), err)
	}
	if _, ok := tokens.Validate(issued.Token); !ok {
		t.Fatal("issued token does not validate")
	}

	rec = doAuth(t, mux, "GET", "/api/v1/admin/tokens", hdr)
	var listed struct {
		Tokens []struct {
			ServerID string `json:"serverId"`
			HasToken bool   `json:"hasToken"`
		} `json:"tokens"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("list decode: %v", err)
	}
	if len(listed.Tokens) != 1 || listed.Tokens[0].ServerID != "pz-test-x" || !listed.Tokens[0].HasToken {
		t.Fatalf("list mismatch: %+v", listed.Tokens)
	}

	rec = doAuth(t, mux, "DELETE", "/api/v1/admin/tokens/pz-test-x", hdr)
	if rec.Code != http.StatusOK {
		t.Fatalf("revoke: expected 200, got %d", rec.Code)
	}
	if _, ok := tokens.Validate(issued.Token); ok {
		t.Fatal("token still valid after revoke")
	}
}

func TestBlobs_ListAfterUploadRecordsSource(t *testing.T) {
	mux, _, _ := newAdminTestRouter(t)

	rec := doJSON(t, mux, "GET", "/api/v1/blobs", nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"count": 0`) {
		t.Fatalf("empty store: expected 200/count 0, got %d %s", rec.Code, rec.Body.String())
	}

	regRec := doJSON(t, mux, "POST", "/api/v1/agents/register", map[string]string{"serverId": "pz-test-1"})
	if regRec.Code != http.StatusOK {
		t.Fatalf("register: got %d", regRec.Code)
	}
	var reg struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(regRec.Body.Bytes(), &reg); err != nil || reg.Token == "" {
		t.Fatalf("register body: %q", regRec.Body.String())
	}

	content := "some mod blob payload"
	sum := sha256.Sum256([]byte(content))
	sha := hex.EncodeToString(sum[:])
	req := httptest.NewRequest("PUT", "/api/v1/blobs/"+sha, strings.NewReader(content))
	req.Header.Set(auth.TokenHeader, reg.Token)
	upRec := httptest.NewRecorder()
	mux.ServeHTTP(upRec, req)
	if upRec.Code != http.StatusCreated {
		t.Fatalf("upload: expected 201, got %d: %s", upRec.Code, upRec.Body.String())
	}

	rec = doJSON(t, mux, "GET", "/api/v1/blobs", nil)
	var list struct {
		Blobs []struct {
			SHA256       string `json:"sha256"`
			SizeBytes    int64  `json:"sizeBytes"`
			SourceServer string `json:"sourceServer"`
		} `json:"blobs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("blobs decode: %v", err)
	}
	if len(list.Blobs) != 1 {
		t.Fatalf("expected 1 blob, got %+v", list.Blobs)
	}
	b := list.Blobs[0]
	if b.SHA256 != sha || b.SizeBytes != int64(len(content)) || b.SourceServer != "pz-test-1" {
		t.Fatalf("blob mismatch: %+v", b)
	}
}

func TestDownload_IncrementsBlobDownloadCount(t *testing.T) {
	mux, tokensStore, _ := newAdminTestRouter(t)

	regRec := doJSON(t, mux, "POST", "/api/v1/agents/register", map[string]string{"serverId": "pz-dl-1"})
	var reg struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(regRec.Body.Bytes(), &reg); err != nil || reg.Token == "" {
		t.Fatalf("register: %q", regRec.Body.String())
	}

	content := "download-stats-payload"
	sum := sha256.Sum256([]byte(content))
	sha := hex.EncodeToString(sum[:])
	req := httptest.NewRequest("PUT", "/api/v1/blobs/"+sha, strings.NewReader(content))
	req.Header.Set(auth.TokenHeader, reg.Token)
	upRec := httptest.NewRecorder()
	mux.ServeHTTP(upRec, req)
	if upRec.Code != http.StatusCreated {
		t.Fatalf("upload: got %d", upRec.Code)
	}

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/download/"+sha, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("download %d: got %d", i, rec.Code)
		}
		if got := rec.Body.String(); got != content {
			t.Fatalf("download %d body mismatch: %q", i, got)
		}
	}

	list := doJSON(t, mux, "GET", "/api/v1/blobs", nil)
	var decoded struct {
		Blobs []struct {
			SHA256    string `json:"sha256"`
			Downloads int64  `json:"downloads"`
		} `json:"blobs"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("blobs decode: %v", err)
	}
	if len(decoded.Blobs) != 1 || decoded.Blobs[0].Downloads != 3 {
		t.Fatalf("expected downloads=3, got %+v", decoded.Blobs)
	}
	_ = tokensStore
}
