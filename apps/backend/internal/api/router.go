// Package api wires HTTP routes for the Backend.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"pzlauncher/apps/backend/internal/auth"
	"pzlauncher/apps/backend/internal/join"
	"pzlauncher/apps/backend/internal/metrics"
	"pzlauncher/apps/backend/internal/obs"
	"pzlauncher/apps/backend/internal/registry"
	"pzlauncher/apps/backend/internal/storage"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewRouter builds and returns the Backend HTTP mux.
// baseURL is the public base URL of this Backend instance (e.g. "http://localhost:8080").
// store is the content-addressable blob store; may be nil (download will 503).
// tokens is the agent auth store; may be nil (auth disabled — dev only).
func NewRouter(reg *registry.Registry, baseURL string, store storage.Store, tokens *auth.Store) http.Handler {
	mux := http.NewServeMux()

	agentAuth := requireAgentToken(tokens)

	resolver := join.NewResolver(reg, baseURL, store)

	// Prometheus metrics — GET /metrics
	mux.Handle("GET /metrics", promhttp.Handler())

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

	// Agent list — GET /api/v1/agents (B1)
	mux.HandleFunc("GET /api/v1/agents", func(w http.ResponseWriter, _ *http.Request) {
		agents := reg.ListAgents()
		var online, degraded, offline int
		for _, a := range agents {
			switch a.Status {
			case registry.AgentOnline:
				online++
			case registry.AgentDegraded:
				degraded++
			case registry.AgentOffline:
				offline++
			}
		}
		metrics.UpdateAgentGauges(online, degraded, offline)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"agents": agents,
		})
	})

	// Join
	mux.HandleFunc("POST /api/v1/join/{serverId}", func(w http.ResponseWriter, r *http.Request) {
		serverID := r.PathValue("serverId")
		sessionID := fmt.Sprintf("sess-%d", time.Now().UnixNano())
		issuedAt := time.Now().UTC().Format(time.RFC3339)
		t0 := time.Now()

		traceID := obs.NewTraceID()
		ctx := obs.WithTrace(r.Context(), traceID)

		resp, err := resolver.Resolve(ctx, serverID, sessionID, issuedAt)
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
			metrics.JoinTotal.WithLabelValues(serverID, "error").Inc()
			obs.LogError(ctx, "join.http_error", "server_id", serverID, "code", errCode, "error", err)
			writeError(w, code, errCode, err.Error())
			return
		}
		metrics.JoinTotal.WithLabelValues(serverID, "ok").Inc()
		metrics.JoinDuration.WithLabelValues(serverID).Observe(time.Since(t0).Seconds())
		writeJSON(w, http.StatusOK, resp)
	})

	// Download — content-addressable blob serving (A4)
	mux.HandleFunc("GET /api/v1/download/{sha256}", func(w http.ResponseWriter, r *http.Request) {
		sha256hex := r.PathValue("sha256")
		if store == nil {
			writeError(w, http.StatusServiceUnavailable, "STORE_NOT_CONFIGURED", "content store not available")
			return
		}
		rc, size, err := store.Get(sha256hex)
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "DOWNLOAD_NOT_FOUND",
				fmt.Sprintf("blob %q not in store", sha256hex))
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DOWNLOAD_ERROR", err.Error())
			return
		}
		defer rc.Close()
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("X-Content-SHA256", sha256hex)
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		if size > 0 {
			w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, rc)
		metrics.BlobDownloadTotal.Inc()
	})

	// Agent registration — POST /api/v1/agents/register (A6)
	// Not token-protected: this is the bootstrap endpoint.
	// Auto-creates a server record in the registry if it doesn't exist yet.
	mux.HandleFunc("POST /api/v1/agents/register", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ServerID    string `json:"serverId"`
			ServerName  string `json:"serverName,omitempty"`
			GameVersion string `json:"gameVersion,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ServerID == "" {
			writeError(w, http.StatusBadRequest, "REGISTER_INVALID", "serverId is required")
			return
		}

		// Auto-create server in registry if not already present.
		if _, exists := reg.Get(req.ServerID); !exists {
			name := req.ServerName
			if name == "" {
				name = req.ServerID
			}
			gv := req.GameVersion
			if gv == "" {
				gv = "42.8"
			}
			reg.Upsert(&registry.ServerRecord{
				ID:          req.ServerID,
				Name:        name,
				Description: "Auto-registered by agent",
				GameVersion: gv,
				MaxPlayers:  64,
				Status:      "online",
				Tags:        []string{"auto"},
			})
			obs.Log(r.Context(), "agent.server_auto_created", "server_id", req.ServerID)
		}

		if tokens == nil {
			// no-auth dev mode: return a placeholder token so the agent doesn't retry.
			reg.RecordHeartbeat(req.ServerID, 0, "")
			writeJSON(w, http.StatusOK, map[string]string{
				"token":    "dev-no-auth",
				"serverId": req.ServerID,
			})
			return
		}
		token, err := tokens.Register(req.ServerID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "REGISTER_ERROR", err.Error())
			return
		}
		// Seed initial agent state so health check shows 'online' immediately.
		reg.RecordHeartbeat(req.ServerID, 0, "")
		obs.Log(r.Context(), "agent.registered", "server_id", req.ServerID)
		writeJSON(w, http.StatusOK, map[string]string{
			"token":    token,
			"serverId": req.ServerID,
		})
	})

	// Agent ingestion — PUT /api/v1/blobs/{sha256} (A5, auth A6)
	mux.HandleFunc("HEAD /api/v1/blobs/{sha256}", agentAuth(func(w http.ResponseWriter, r *http.Request) {
		sha256hex := r.PathValue("sha256")
		if store != nil && store.Has(sha256hex) {
			w.Header().Set("X-Content-SHA256", sha256hex)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))

	mux.HandleFunc("PUT /api/v1/blobs/{sha256}", agentAuth(func(w http.ResponseWriter, r *http.Request) {
		sha256hex := r.PathValue("sha256")
		if store == nil {
			writeError(w, http.StatusServiceUnavailable, "STORE_NOT_CONFIGURED", "content store not available")
			return
		}
		if store.Has(sha256hex) {
			w.WriteHeader(http.StatusOK)
			return
		}
		if err := store.Put(sha256hex, r.Body); err != nil {
			if errors.Is(err, storage.ErrNotFound) || contains(err.Error(), "CHECKSUM_MISMATCH") {
				writeError(w, http.StatusBadRequest, "BLOB_CHECKSUM_MISMATCH", err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "BLOB_STORE_ERROR", err.Error())
			return
		}
		metrics.BlobUploadTotal.Inc()
		if cl, err2 := strconv.ParseInt(r.Header.Get("Content-Length"), 10, 64); err2 == nil && cl > 0 {
			metrics.BlobUploadBytes.Add(float64(cl))
		}
		obs.Log(r.Context(), "agent.blob_stored",
			"sha256", sha256hex[:12],
			"size", r.Header.Get("Content-Length"),
		)
		w.WriteHeader(http.StatusCreated)
	}))

	// Agent manifest ingestion — PUT /api/v1/manifests/{serverId} (A5, auth A6, versioned B4)
	mux.HandleFunc("PUT /api/v1/manifests/{serverId}", agentAuth(func(w http.ResponseWriter, r *http.Request) {
		serverID := r.PathValue("serverId")
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			writeError(w, http.StatusBadRequest, "MANIFEST_READ_ERROR", err.Error())
			return
		}
		version, err := reg.UpsertManifest(serverID, body)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "MANIFEST_STORE_ERROR", err.Error())
			return
		}
		metrics.ManifestPublishTotal.WithLabelValues(serverID).Inc()
		metrics.ManifestVersionsTotal.WithLabelValues(serverID).Set(float64(version))
		obs.Log(r.Context(), "agent.manifest_updated", "server_id", serverID, "version", version)
		writeJSON(w, http.StatusOK, map[string]interface{}{"version": version, "serverId": serverID})
	}))

	// Manifest history — GET /api/v1/manifests/{serverId}/history (B4)
	mux.HandleFunc("GET /api/v1/manifests/{serverId}/history", func(w http.ResponseWriter, r *http.Request) {
		serverID := r.PathValue("serverId")
		history := reg.ManifestStore().History(serverID)
		if history == nil {
			writeError(w, http.StatusNotFound, "MANIFEST_NOT_FOUND",
				fmt.Sprintf("no manifest history for server %q", serverID))
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"serverId": serverID,
			"versions": history,
		})
	})

	// Manifest diff — GET /api/v1/manifests/{serverId}/diff?from=N&to=M (B4)
	// from=0 means "compare against empty" (all mods are Added).
	mux.HandleFunc("GET /api/v1/manifests/{serverId}/diff", func(w http.ResponseWriter, r *http.Request) {
		serverID := r.PathValue("serverId")
		var fromVer, toVer int
		fmt.Sscanf(r.URL.Query().Get("from"), "%d", &fromVer)
		fmt.Sscanf(r.URL.Query().Get("to"), "%d", &toVer)
		if toVer == 0 {
			// Default: diff from previous to latest.
			latest := reg.ManifestStore().Latest(serverID)
			if latest == nil {
				writeError(w, http.StatusNotFound, "MANIFEST_NOT_FOUND",
					fmt.Sprintf("no manifests for server %q", serverID))
				return
			}
			toVer = latest.Version
			if fromVer == 0 && toVer > 1 {
				fromVer = toVer - 1
			}
		}
		diff, err := reg.ManifestStore().Diff(serverID, fromVer, toVer)
		if err != nil {
			writeError(w, http.StatusNotFound, "MANIFEST_DIFF_ERROR", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, diff)
	})

	// Agent heartbeat — POST /api/v1/agents/heartbeat (A5, auth A6)
	mux.HandleFunc("POST /api/v1/agents/heartbeat", agentAuth(func(w http.ResponseWriter, r *http.Request) {
		var hbBody struct {
			ServerID  string `json:"serverId"`
			ModCount  int    `json:"modCount"`
			Timestamp string `json:"timestamp"`
			Version   string `json:"version,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&hbBody); err != nil {
			writeError(w, http.StatusBadRequest, "HEARTBEAT_PARSE_ERROR", err.Error())
			return
		}
		reg.RecordHeartbeat(hbBody.ServerID, hbBody.ModCount, hbBody.Version)
		metrics.HeartbeatTotal.WithLabelValues(hbBody.ServerID).Inc()
		// Update agent status gauges after every heartbeat.
		{
			all := reg.ListAgents()
			var on, deg, off int
			for _, a := range all {
				switch a.Status {
				case registry.AgentOnline:
					on++
				case registry.AgentDegraded:
					deg++
				case registry.AgentOffline:
					off++
				}
			}
			metrics.UpdateAgentGauges(on, deg, off)
		}
		obs.Log(r.Context(), "agent.heartbeat",
			"server_id", hbBody.ServerID,
			"mod_count", hbBody.ModCount,
			"at", hbBody.Timestamp,
		)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "serverTime": time.Now().UTC().Format(time.RFC3339)})
	}))

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

// requireAgentToken returns a middleware that validates X-Agent-Token.
// If tokens is nil, auth is disabled (dev mode) and all requests pass through.
func requireAgentToken(tokens *auth.Store) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if tokens == nil {
				next(w, r)
				return
			}
			token := r.Header.Get(auth.TokenHeader)
			if _, ok := tokens.Validate(token); !ok {
				writeError(w, http.StatusUnauthorized, "AGENT_UNAUTHORIZED",
					"missing or invalid "+auth.TokenHeader)
				return
			}
			next(w, r)
		}
	}
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, HEAD, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, "+auth.TokenHeader)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
