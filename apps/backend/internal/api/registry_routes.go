// registry_routes.go wires the Content Registry API (RFC-0059).
package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"pzlauncher/apps/backend/internal/content"
	"pzlauncher/apps/backend/internal/metrics"
	"pzlauncher/apps/backend/internal/obs"
	"pzlauncher/apps/backend/internal/storage"
)

// registerRegistryRoutes mounts all /api/v1/registry/* handlers onto mux.
func registerRegistryRoutes(mux *http.ServeMux, store storage.Store, reg *content.Registry) {
	if reg == nil {
		return
	}

	// GET /api/v1/registry/catalog?game=pz
	// Returns the full version catalog for a game, used by Launcher on startup.
	mux.HandleFunc("GET /api/v1/registry/catalog", func(w http.ResponseWriter, r *http.Request) {
		gameID := r.URL.Query().Get("game")
		records := reg.List(gameID)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"game":     gameID,
			"versions": records,
		})
	})

	// GET /api/v1/registry/versions?game=pz&platform=windows-x64&trust=verified
	// Lists available versions with optional filters.
	mux.HandleFunc("GET /api/v1/registry/versions", func(w http.ResponseWriter, r *http.Request) {
		gameID := r.URL.Query().Get("game")
		platform := r.URL.Query().Get("platform")
		trustFilter := r.URL.Query().Get("trust")

		all := reg.List(gameID)
		filtered := all[:0]
		for _, rec := range all {
			if platform != "" && rec.Platform != platform {
				continue
			}
			if trustFilter != "" && string(rec.TrustLevel) != trustFilter {
				continue
			}
			filtered = append(filtered, rec)
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"versions": filtered,
		})
	})

	// GET /api/v1/registry/versions/{game}/{version}/{platform}
	// Downloads the game client binary for the requested version.
	mux.HandleFunc("GET /api/v1/registry/versions/{game}/{version}/{platform}", func(w http.ResponseWriter, r *http.Request) {
		gameID := r.PathValue("game")
		version := r.PathValue("version")
		platform := r.PathValue("platform")

		rec := reg.Get(gameID, version, platform)
		if rec == nil {
			writeError(w, http.StatusNotFound, "REGISTRY_VERSION_NOT_FOUND",
				fmt.Sprintf("no binary registered for %s/%s/%s", gameID, version, platform))
			return
		}
		if rec.TrustLevel == content.TrustRejected {
			writeError(w, http.StatusConflict, "REGISTRY_VERSION_REJECTED",
				"this version has a hash conflict and cannot be served")
			return
		}
		if store == nil {
			writeError(w, http.StatusServiceUnavailable, "STORE_NOT_CONFIGURED", "blob store unavailable")
			return
		}
		rc, size, err := store.Get(rec.SHA256)
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "REGISTRY_BLOB_NOT_FOUND",
				fmt.Sprintf("binary registered but blob missing for %s/%s/%s", gameID, version, platform))
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "REGISTRY_DOWNLOAD_ERROR", err.Error())
			return
		}
		defer rc.Close()

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("X-Content-SHA256", rec.SHA256)
		w.Header().Set("X-Trust-Level", string(rec.TrustLevel))
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		if size > 0 {
			w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, rc)

		metrics.RegistryDownloadTotal.WithLabelValues(gameID, version, platform).Inc()
		obs.Log(r.Context(), "registry.download",
			"game", gameID, "version", version, "platform", platform,
			"trust", string(rec.TrustLevel),
		)
	})

	// POST /api/v1/registry/versions/{game}/{version}/{platform}/submit
	// Community upload step 1: submit a SHA256 hash without uploading the file.
	// If the hash is new and no blob exists → upload_required.
	// If the hash matches an existing record → counted (increments trust).
	// If the hash conflicts with an existing different hash → conflict.
	mux.HandleFunc("POST /api/v1/registry/versions/{game}/{version}/{platform}/submit", func(w http.ResponseWriter, r *http.Request) {
		gameID := r.PathValue("game")
		version := r.PathValue("version")
		platform := r.PathValue("platform")

		var body struct {
			SHA256    string `json:"sha256"`
			SizeBytes int64  `json:"sizeBytes"`
		}
		if err := decodeJSON(r, &body); err != nil || body.SHA256 == "" {
			writeError(w, http.StatusBadRequest, "REGISTRY_SUBMIT_INVALID",
				"sha256 is required")
			return
		}

		result := reg.Submit(gameID, version, platform, body.SHA256, body.SizeBytes, r.RemoteAddr)
		if result.Conflict {
			writeError(w, http.StatusConflict, "REGISTRY_HASH_CONFLICT",
				"a different sha256 already exists for this version — conflict flagged for admin review")
			return
		}

		resp := map[string]interface{}{
			"status":      result.Status,
			"trustLevel":  result.TrustLevel,
			"uploadCount": result.UploadCount,
		}

		if result.Status == "upload_required" {
			resp["uploadUrl"] = fmt.Sprintf("/api/v1/registry/versions/%s/%s/%s", gameID, version, platform)
		}

		metrics.RegistrySubmitTotal.WithLabelValues(gameID, version, result.Status).Inc()
		obs.Log(r.Context(), "registry.submit",
			"game", gameID, "version", version, "platform", platform,
			"status", result.Status, "trust", string(result.TrustLevel),
		)
		writeJSON(w, http.StatusOK, resp)
	})

	// PUT /api/v1/registry/versions/{game}/{version}/{platform}
	// Community upload step 2: upload the actual binary.
	// Requires X-Content-SHA256 header matching a prior submit call.
	mux.HandleFunc("PUT /api/v1/registry/versions/{game}/{version}/{platform}", func(w http.ResponseWriter, r *http.Request) {
		gameID := r.PathValue("game")
		version := r.PathValue("version")
		platform := r.PathValue("platform")

		sha256hex := r.Header.Get("X-Content-SHA256")
		if sha256hex == "" {
			writeError(w, http.StatusBadRequest, "REGISTRY_UPLOAD_MISSING_HASH",
				"X-Content-SHA256 header required")
			return
		}

		rec := reg.Get(gameID, version, platform)
		if rec == nil {
			writeError(w, http.StatusBadRequest, "REGISTRY_UPLOAD_NOT_SUBMITTED",
				"submit the hash first via POST .../submit before uploading")
			return
		}
		if rec.SHA256 != sha256hex {
			writeError(w, http.StatusConflict, "REGISTRY_UPLOAD_HASH_MISMATCH",
				"X-Content-SHA256 does not match the registered hash for this version")
			return
		}
		if store == nil {
			writeError(w, http.StatusServiceUnavailable, "STORE_NOT_CONFIGURED", "blob store unavailable")
			return
		}
		if store.Has(sha256hex) {
			// Blob already present — just record the submitter.
			reg.MarkUploaded(sha256hex, r.RemoteAddr)
			w.WriteHeader(http.StatusOK)
			return
		}

		if err := store.Put(sha256hex, r.Body); err != nil {
			writeError(w, http.StatusBadRequest, "REGISTRY_UPLOAD_ERROR", err.Error())
			return
		}
		reg.MarkUploaded(sha256hex, r.RemoteAddr)

		if cl, err := strconv.ParseInt(r.Header.Get("Content-Length"), 10, 64); err == nil && cl > 0 {
			metrics.RegistryUploadBytes.Add(float64(cl))
		}
		metrics.RegistryUploadTotal.WithLabelValues(gameID, version, platform).Inc()
		obs.Log(r.Context(), "registry.upload",
			"game", gameID, "version", version, "platform", platform,
			"sha256", sha256hex[:12],
		)
		w.WriteHeader(http.StatusCreated)
	})
}
