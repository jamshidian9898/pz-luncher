// dev-api serves registry JSON and product pipeline for browser dev (no Wails).
// Run: go run ./apps/dev-api
package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"sync"

	"pzlauncher/libs/pipeline"
	"pzlauncher/libs/settings"
	"pzlauncher/libs/sharedtypes"
)

const addr = ":8765"

func main() {
	root := pipeline.WorkspaceRoot()
	st, _ := settings.Load(root)
	settings.ApplyGamePathEnv(st)
	svc := pipeline.NewService(settings.ToPipelineConfig(root, st))

	var mu sync.Mutex
	lastJoin := make(map[string]*pipeline.JoinResult)

	regDir := filepath.Join(root, "apps", "launcher-ui", "frontend", "public", "registry")
	mux := http.NewServeMux()
	mux.Handle("/registry/", http.StripPrefix("/registry/", http.FileServer(http.Dir(regDir))))

	mux.HandleFunc("GET /api/settings", func(w http.ResponseWriter, r *http.Request) {
		st, err := settings.Load(root)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, st)
	})

	mux.HandleFunc("PUT /api/settings", func(w http.ResponseWriter, r *http.Request) {
		raw, err := readBody(r)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		var body sharedtypes.LauncherSettings
		if err := json.Unmarshal(raw, &body); err != nil {
			var ui uiSettings
			if err2 := json.Unmarshal(raw, &ui); err2 != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			body = ui.toShared()
		}
		if err := settings.Save(root, &body); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		settings.ApplyGamePathEnv(&body)
		svc = pipeline.NewService(settings.ToPipelineConfig(root, &body))
		writeJSON(w, &body)
	})

	mux.HandleFunc("POST /api/join/{serverId}", func(w http.ResponseWriter, r *http.Request) {
		serverID := r.PathValue("serverId")
		result, err := svc.RunJoin(r.Context(), serverID, func(ev pipeline.Event) {
			log.Printf("[event] %s", ev.Type)
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		mu.Lock()
		lastJoin[serverID] = result
		mu.Unlock()
		writeJSON(w, map[string]interface{}{
			"sessionId":   result.SessionID,
			"profilePath": result.ProfilePath,
			"ready":       result.Ready,
		})
	})

	mux.HandleFunc("POST /api/launch/{serverId}", func(w http.ResponseWriter, r *http.Request) {
		serverID := r.PathValue("serverId")
		mu.Lock()
		result := lastJoin[serverID]
		mu.Unlock()
		if result == nil || !result.Ready {
			http.Error(w, "join required first", 400)
			return
		}
		st, _ := settings.Load(root)
		settings.ApplyGamePathEnv(st)
		if err := svc.Launch(r.Context(), serverID, result.ProfilePath, func(ev pipeline.Event) {
			log.Printf("[launch] %s", ev.Type)
		}); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]string{"status": "ok", "root": root})
	})

	log.Printf("dev-api http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, cors(mux)))
}

type uiSettings struct {
	GamePath         string `json:"gamePath"`
	SteamCMDPath     string `json:"steamcmdPath"`
	CacheLocation    string `json:"cacheLocation"`
	ProfilesLocation string `json:"profilesLocation"`
	MaxConcurrent    int    `json:"maxConcurrent"`
	BandwidthLimit   int    `json:"bandwidthLimit"`
	VerifyChecksum   bool   `json:"verifyChecksum"`
}

func (u uiSettings) toShared() sharedtypes.LauncherSettings {
	return sharedtypes.LauncherSettings{
		GamePath:            u.GamePath,
		SteamCMDPath:        u.SteamCMDPath,
		CachePath:           u.CacheLocation,
		ProfilesPath:        u.ProfilesLocation,
		ConcurrentDownloads: u.MaxConcurrent,
		BandwidthLimitMbps:  u.BandwidthLimit,
		VerifyChecksum:      u.VerifyChecksum,
	}
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func readBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	return io.ReadAll(r.Body)
}
