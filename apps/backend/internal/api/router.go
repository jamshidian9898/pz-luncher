// Package api wires HTTP routes for the Backend.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"pzlauncher/apps/backend/internal/join"
	"pzlauncher/apps/backend/internal/registry"
)

// NewRouter builds and returns the Backend HTTP mux.
// baseURL is the public base URL of this Backend instance (e.g. "http://localhost:8080").
func NewRouter(reg *registry.Registry, baseURL string) http.Handler {
	mux := http.NewServeMux()

	resolver := join.NewResolver(reg, baseURL)

	// Health
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": "2.0.0"})
	})

	// Server registry
	mux.HandleFunc("GET /api/v1/servers", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"servers": reg.List(),
		})
	})

	mux.HandleFunc("GET /api/v1/servers/{serverId}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("serverId")
		srv, ok := reg.Get(id)
		if !ok {
			writeError(w, http.StatusNotFound, "JOIN_SERVER_NOT_FOUND", fmt.Sprintf("server %q not found", id))
			return
		}
		writeJSON(w, http.StatusOK, srv)
	})

	// Join
	mux.HandleFunc("POST /api/v1/join/{serverId}", func(w http.ResponseWriter, r *http.Request) {
		serverID := r.PathValue("serverId")
		sessionID := fmt.Sprintf("sess-%d", time.Now().UnixNano())
		issuedAt := time.Now().UTC().Format(time.RFC3339)

		resp, err := resolver.Resolve(serverID, sessionID, issuedAt)
		if err != nil {
			code := http.StatusInternalServerError
			errCode := "JOIN_INTERNAL"
			switch {
			case isNotFound(err):
				code = http.StatusNotFound
				errCode = "JOIN_SERVER_NOT_FOUND"
			case isOffline(err):
				code = http.StatusConflict
				errCode = "JOIN_SERVER_OFFLINE"
			case isManifestUnavailable(err):
				code = http.StatusServiceUnavailable
				errCode = "JOIN_MANIFEST_UNAVAILABLE"
			}
			writeError(w, code, errCode, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})

	// Download (Phase A: 404 stub — real blobs served in A4)
	mux.HandleFunc("GET /api/v1/download/{sha256}", func(w http.ResponseWriter, r *http.Request) {
		sha256 := r.PathValue("sha256")
		writeError(w, http.StatusNotFound, "DOWNLOAD_NOT_CACHED",
			fmt.Sprintf("content %q not in backend store (Phase A4 not yet implemented)", sha256))
	})

	return cors(mux)
}

func isNotFound(err error) bool {
	return err != nil && contains(err.Error(), "not found")
}

func isOffline(err error) bool {
	return err != nil && contains(err.Error(), "offline")
}

func isManifestUnavailable(err error) bool {
	return err != nil && contains(err.Error(), "manifest unavailable")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

// errorEnvelope is the standard error response shape (RFC-0052).
type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorEnvelope{Error: errorBody{Code: code, Message: message}})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
