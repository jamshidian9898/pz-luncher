package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"pzlauncher/libs/pipeline"
	"pzlauncher/libs/settings"
	"pzlauncher/libs/sharedtypes"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// UIService handles UI business logic
type UIService struct {
	ctx context.Context
	mu  sync.Mutex

	workspaceRoot string
	pipeline      *pipeline.Service

	lastJoin         *pipeline.JoinResult
	lastServer       string
	lastJoinResponse *pipeline.BackendJoinResponse
	sessions         map[string]*SessionStatus
}

// NewUIService creates UI service
func NewUIService() *UIService {
	root := pipeline.WorkspaceRoot()
	st, _ := settings.Load(root)
	return &UIService{
		workspaceRoot: root,
		pipeline:      pipeline.NewService(settings.ToPipelineConfig(root, st)),
		sessions:      make(map[string]*SessionStatus),
	}
}

func (s *UIService) getWorkspaceRoot() string {
	if s.workspaceRoot != "" {
		return s.workspaceRoot
	}
	return pipeline.WorkspaceRoot()
}

// ReloadConfig rebuilds pipeline using the current PZ_LAUNCHER_ROOT env (set in startup).
func (s *UIService) ReloadConfig() {
	root := pipeline.WorkspaceRoot()
	st, _ := settings.Load(root)
	s.mu.Lock()
	s.workspaceRoot = root
	s.pipeline = pipeline.NewService(settings.ToPipelineConfig(root, st))
	s.mu.Unlock()
}

// SetContext sets Wails context for events
func (s *UIService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

func (s *UIService) emitEvent(event UIEvent) {
	if s.ctx != nil {
		runtime.EventsEmit(s.ctx, "launcher:event", event)
	}
	s.updateSessionFromEvent(event)
}

func (s *UIService) pipelineEmit() pipeline.Emitter {
	return func(ev pipeline.Event) {
		ui := UIEvent{
			Type:      mapPipelineEventType(ev.Type),
			Timestamp: time.Now().Unix(),
			SessionID: ev.SessionID,
			PackageID: ev.PackageID,
			Error:     ev.Error,
			Metadata:  ev.Metadata,
		}
		if ev.Progress != nil {
			ui.Progress = &Progress{
				Current: ev.Progress.Current,
				Total:   ev.Progress.Total,
				Percent: ev.Progress.Percent,
				Speed:   ev.Progress.Speed,
				ETA:     ev.Progress.ETA,
			}
		}
		s.emitEvent(ui)
	}
}

func mapPipelineEventType(t string) UIEventType {
	switch t {
	case "session.start":
		return EventSessionStart
	case "session.complete":
		return EventSessionComplete
	case "mod.resolve.start":
		return EventModResolveStart
	case "mod.resolve.complete":
		return EventModResolveComplete
	case "download.start":
		return EventDownloadStart
	case "download.progress":
		return EventDownloadProgress
	case "download.complete":
		return EventDownloadComplete
	case "install.complete":
		return EventInstallComplete
	case "error":
		return EventError
	default:
		return EventTraceUpdated
	}
}

func (s *UIService) updateSessionFromEvent(ev UIEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.sessions[ev.SessionID]
	if !ok {
		st = &SessionStatus{SessionID: ev.SessionID, State: "resolving", Errors: []string{}}
		s.sessions[ev.SessionID] = st
	}
	switch ev.Type {
	case EventModResolveStart:
		st.State = "resolving"
		st.Progress = 5
	case EventModResolveComplete:
		st.State = "resolving"
		st.Progress = 15
	case EventDownloadStart:
		st.State = "downloading"
		st.CurrentMod = ev.PackageID
	case EventDownloadProgress:
		st.State = "downloading"
		if ev.Progress != nil {
			st.Progress = float64(ev.Progress.Percent)
			st.DownloadSpeed = ev.Progress.Speed
			st.ETA = ev.Progress.ETA
		}
	case EventDownloadComplete:
		st.Progress = 80
	case EventInstallComplete, EventSessionComplete:
		st.State = "complete"
		st.Progress = 100
		st.CurrentMod = ""
	case EventError:
		st.State = "error"
		if ev.Error != "" {
			st.Errors = append(st.Errors, ev.Error)
		}
	}
}

// JoinServer calls Backend POST /api/v1/join and runs the v2 download pipeline.
func (s *UIService) JoinServer(serverID string) error {
	go func() {
		ctx := context.Background()
		emit := s.pipelineEmit()

		// Call Backend join API (A3)
		var jr pipeline.BackendJoinResponse
		url := fmt.Sprintf("%s/api/v1/join/%s", s.backendURL(), serverID)
		if err := s.backendPost(url, &jr); err != nil {
			emit(pipeline.Event{Type: "error", Error: fmt.Sprintf("BACKEND_JOIN: %v", err)})
			return
		}

		result, err := s.pipeline.RunJoinFromBackend(ctx, jr, emit)
		s.mu.Lock()
		if err != nil {
			s.mu.Unlock()
			return
		}
		s.lastJoin = result
		s.lastServer = serverID
		s.lastJoinResponse = &jr
		s.mu.Unlock()
	}()
	return nil
}

// LaunchServer launches the game for the last successful join.
func (s *UIService) LaunchServer(serverID string) error {
	s.mu.Lock()
	join := s.lastJoin
	jr := s.lastJoinResponse
	s.mu.Unlock()
	if join == nil || !join.Ready {
		return fmt.Errorf("LAUNCH_PROFILE_NOT_READY: join server first")
	}
	// v2 path: we have a BackendJoinResponse with launch args
	if jr != nil {
		if jr.Manifest.ServerID != serverID && serverID != "" {
			return fmt.Errorf("LAUNCH_PROFILE_NOT_READY: no join session for server %s", serverID)
		}
		go func() {
			_ = s.pipeline.LaunchFromBackend(context.Background(), jr.Manifest.ServerID, join.ProfilePath, *jr, s.pipelineEmit())
		}()
		return nil
	}
	// v1 fallback: manifest embedded in JoinResult
	if join.Manifest != nil && join.Manifest.ServerID != serverID && serverID != "" {
		return fmt.Errorf("LAUNCH_PROFILE_NOT_READY: no join session for server %s", serverID)
	}
	go func() {
		_ = s.pipeline.Launch(context.Background(), serverID, join.ProfilePath, s.pipelineEmit())
	}()
	return nil
}

// GetServerList fetches the server list from the Backend registry API (A2).
func (s *UIService) GetServerList() []ServerInfo {
	type backendServer struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Description string   `json:"description"`
		PlayerCount int      `json:"playerCount"`
		MaxPlayers  int      `json:"maxPlayers"`
		Status      string   `json:"status"`
		Tags        []string `json:"tags"`
	}
	type response struct {
		Servers []backendServer `json:"servers"`
	}

	baseURL := s.backendURL()
	var resp response
	if err := s.backendGet(baseURL+"/api/v1/servers", &resp); err != nil {
		return s.fallbackServerList()
	}
	out := make([]ServerInfo, 0, len(resp.Servers))
	for _, d := range resp.Servers {
		out = append(out, ServerInfo{
			ID:          d.ID,
			Name:        d.Name,
			Description: d.Description,
			PlayerCount: d.PlayerCount,
			MaxPlayers:  d.MaxPlayers,
		})
	}
	return out
}

// GetServerDetails fetches server details via the Backend join API (A2).
// For Phase A2 we use GET /api/v1/servers/{id}; mod details added in A3.
func (s *UIService) GetServerDetails(serverID string) (*ServerDetails, error) {
	type backendServer struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		PlayerCount int    `json:"playerCount"`
		MaxPlayers  int    `json:"maxPlayers"`
	}

	baseURL := s.backendURL()
	var srv backendServer
	if err := s.backendGet(fmt.Sprintf("%s/api/v1/servers/%s", baseURL, serverID), &srv); err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}
	return &ServerDetails{
		ServerInfo: ServerInfo{
			ID:          srv.ID,
			Name:        srv.Name,
			Description: srv.Description,
			PlayerCount: srv.PlayerCount,
			MaxPlayers:  srv.MaxPlayers,
		},
	}, nil
}

func (s *UIService) backendURL() string {
	root := s.getWorkspaceRoot()
	st, _ := settings.Load(root)
	if st != nil && st.BackendURL != "" {
		return st.BackendURL
	}
	return "http://localhost:8080"
}

func (s *UIService) backendGet(url string, out interface{}) error {
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("backend %s: %s", resp.Status, body)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (s *UIService) backendPost(url string, out interface{}) error {
	resp, err := http.Post(url, "application/json", nil) //nolint:noctx
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("backend %s: %s", resp.Status, body)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (s *UIService) fallbackServerList() []ServerInfo {
	return []ServerInfo{{
		ID: "demo-survival", Name: "Demo Survival", Description: "Backend unreachable — offline demo",
		PlayerCount: 0, MaxPlayers: 32,
	}}
}

// GetSessionStatus returns tracked session progress
func (s *UIService) GetSessionStatus(sessionID string) (*SessionStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if st, ok := s.sessions[sessionID]; ok {
		copy := *st
		return &copy, nil
	}
	return &SessionStatus{SessionID: sessionID, State: "idle", Progress: 0}, nil
}

// RepairCache mocks cache repair
func (s *UIService) RepairCache() error {
	s.emitEvent(UIEvent{Type: EventCacheRepairStart, Timestamp: time.Now().Unix()})
	time.Sleep(500 * time.Millisecond)
	s.emitEvent(UIEvent{Type: EventCacheRepairComplete, Timestamp: time.Now().Unix()})
	return nil
}

// GetSettings returns launcher settings (RFC-0036)
func (s *UIService) GetSettings() (*Settings, error) {
	st, err := settings.Load(s.getWorkspaceRoot())
	if err != nil {
		return nil, err
	}
	return sharedToUI(st), nil
}

// SaveSettings saves settings
func (s *UIService) SaveSettings(ui Settings) error {
	st := uiToShared(ui)
	root := s.getWorkspaceRoot()
	if err := settings.Save(root, st); err != nil {
		return err
	}
	settings.ApplyGamePathEnv(st)
	s.pipeline = pipeline.NewService(settings.ToPipelineConfig(root, st))
	return nil
}

func sharedToUI(st *sharedtypes.LauncherSettings) *Settings {
	return &Settings{
		GamePath:         st.GamePath,
		BackendURL:       st.BackendURL,
		CacheLocation:    st.CachePath,
		ProfilesLocation: st.ProfilesPath,
		MaxConcurrent:    st.ConcurrentDownloads,
		BandwidthLimit:   st.BandwidthLimitMbps,
		VerifyChecksum:   st.VerifyChecksum,
	}
}

func uiToShared(ui Settings) *sharedtypes.LauncherSettings {
	return &sharedtypes.LauncherSettings{
		GamePath:            ui.GamePath,
		BackendURL:          ui.BackendURL,
		CachePath:           ui.CacheLocation,
		ProfilesPath:        ui.ProfilesLocation,
		ConcurrentDownloads: ui.MaxConcurrent,
		BandwidthLimitMbps:  ui.BandwidthLimit,
		VerifyChecksum:      ui.VerifyChecksum,
	}
}

// UI Event Types (RFC 0022)
type UIEventType string

const (
	EventSessionStart        UIEventType = "session.start"
	EventModResolveStart     UIEventType = "mod.resolve.start"
	EventModResolveComplete  UIEventType = "mod.resolve.complete"
	EventDownloadStart       UIEventType = "download.start"
	EventDownloadProgress    UIEventType = "download.progress"
	EventDownloadComplete    UIEventType = "download.complete"
	EventInstallStart        UIEventType = "install.start"
	EventInstallComplete     UIEventType = "install.complete"
	EventSessionComplete     UIEventType = "session.complete"
	EventError               UIEventType = "error"
	EventTraceUpdated        UIEventType = "trace.updated"
	EventCacheRepairStart    UIEventType = "cache.repair.start"
	EventCacheRepairComplete UIEventType = "cache.repair.complete"
)

// UIEvent matches RFC 0022
type UIEvent struct {
	Type      UIEventType            `json:"type"`
	Timestamp int64                  `json:"timestamp"`
	SessionID string                 `json:"sessionId"`
	PackageID string                 `json:"packageId,omitempty"`
	Progress  *Progress              `json:"progress,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Progress matches RFC 0022
type Progress struct {
	Current int64 `json:"current"`
	Total   int64 `json:"total"`
	Percent int   `json:"percent"`
	Speed   int64 `json:"speed,omitempty"`
	ETA     int   `json:"eta,omitempty"`
}
