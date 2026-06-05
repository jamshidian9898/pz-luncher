// dev-api serves registry JSON and product pipeline for browser dev (no Wails).
// Run: go run ./apps/dev-api
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"pzlauncher/libs/pipeline"
	"pzlauncher/libs/settings"
	"pzlauncher/libs/sharedtypes"
)

// sseBroker broadcasts pipeline events to SSE clients keyed by sessionId.
type sseBroker struct {
	mu      sync.RWMutex
	clients map[string][]chan pipeline.Event
}

func newSSEBroker() *sseBroker {
	return &sseBroker{clients: make(map[string][]chan pipeline.Event)}
}

func (b *sseBroker) subscribe(sessionID string) chan pipeline.Event {
	ch := make(chan pipeline.Event, 64)
	b.mu.Lock()
	b.clients[sessionID] = append(b.clients[sessionID], ch)
	b.mu.Unlock()
	return ch
}

func (b *sseBroker) unsubscribe(sessionID string, ch chan pipeline.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	list := b.clients[sessionID]
	for i, c := range list {
		if c == ch {
			b.clients[sessionID] = append(list[:i], list[i+1:]...)
			break
		}
	}
	if len(b.clients[sessionID]) == 0 {
		delete(b.clients, sessionID)
	}
}

func (b *sseBroker) publish(ev pipeline.Event) {
	b.mu.RLock()
	list := b.clients[ev.SessionID]
	copied := make([]chan pipeline.Event, len(list))
	copy(copied, list)
	b.mu.RUnlock()
	for _, ch := range copied {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (b *sseBroker) emitterFor(sessionID string) pipeline.Emitter {
	return func(ev pipeline.Event) {
		ev.SessionID = sessionID
		log.Printf("[event] %s sid=%s pkg=%s", ev.Type, ev.SessionID, ev.PackageID)
		b.publish(ev)
	}
}

const addr = ":8765"

func main() {
	root := pipeline.WorkspaceRoot()
	st, _ := settings.Load(root)
	settings.ApplyGamePathEnv(st)
	svc := pipeline.NewService(settings.ToPipelineConfig(root, st))
	broker := newSSEBroker()

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
		sessionID := fmt.Sprintf("session-%d", nowUnix())
		emit := broker.emitterFor(sessionID)
		go func() {
			result, err := svc.RunJoin(context.Background(), serverID, emit)
			if err != nil {
				emit(pipeline.Event{Type: "error", Error: err.Error()})
				return
			}
			mu.Lock()
			lastJoin[serverID] = result
			mu.Unlock()
		}()
		writeJSON(w, map[string]interface{}{
			"sessionId": sessionID,
			"serverId":  serverID,
		})
	})

	mux.HandleFunc("GET /api/events/{sessionId}", func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.PathValue("sessionId")
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}
		ch := broker.subscribe(sessionID)
		defer broker.unsubscribe(sessionID, ch)
		for {
			select {
			case ev, ok := <-ch:
				if !ok {
					return
				}
				data, err := json.Marshal(ev)
				if err != nil {
					continue
				}
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
				if ev.Type == "session.complete" || ev.Type == "error" {
					return
				}
			case <-r.Context().Done():
				return
			}
		}
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
		launchSessionID := fmt.Sprintf("launch-%d", nowUnix())
		emit := broker.emitterFor(launchSessionID)
		if err := svc.Launch(r.Context(), serverID, result.ProfilePath, emit); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, map[string]string{"status": "ok", "sessionId": launchSessionID})
	})

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]string{"status": "ok", "root": root})
	})

	log.Printf("dev-api http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, cors(mux)))
}

type uiSettings struct {
	GamePath         string `json:"gamePath"`
	BackendURL       string `json:"backendUrl"`
	CacheLocation    string `json:"cacheLocation"`
	ProfilesLocation string `json:"profilesLocation"`
	MaxConcurrent    int    `json:"maxConcurrent"`
	BandwidthLimit   int    `json:"bandwidthLimit"`
	VerifyChecksum   bool   `json:"verifyChecksum"`
}

func (u uiSettings) toShared() sharedtypes.LauncherSettings {
	return sharedtypes.LauncherSettings{
		GamePath:            u.GamePath,
		BackendURL:          u.BackendURL,
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

func nowUnix() int64 {
	return time.Now().UnixNano()
}
