package main

import (
	"context"
	"fmt"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// UIService handles UI business logic
type UIService struct {
	ctx context.Context
}

// NewUIService creates UI service
func NewUIService() *UIService {
	return &UIService{}
}

// SetContext sets Wails context for events
func (s *UIService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// emitEvent sends event to frontend
func (s *UIService) emitEvent(event UIEvent) {
	if s.ctx != nil {
		application.EventsEmit(s.ctx, "launcher:event", event)
	}
}

// JoinServer starts server join flow
func (s *UIService) JoinServer(serverID string) error {
	sessionID := fmt.Sprintf("session-%d", time.Now().Unix())

	// Emit start event
	s.emitEvent(UIEvent{
		Type:      EventSessionStart,
		Timestamp: time.Now().Unix(),
		SessionID: sessionID,
		Metadata: map[string]interface{}{
			"serverId": serverID,
		},
	})

	// TODO: Integrate with launcher-core session manager
	// For now, simulate workflow

	go s.simulateSession(sessionID, serverID)

	return nil
}

// simulateSession mocks the join flow for UI testing
func (s *UIService) simulateSession(sessionID, serverID string) {
	// Phase 1: Resolve mods
	s.emitEvent(UIEvent{
		Type:      EventModResolveStart,
		Timestamp: time.Now().Unix(),
		SessionID: sessionID,
	})

	time.Sleep(500 * time.Millisecond)

	s.emitEvent(UIEvent{
		Type:      EventModResolveComplete,
		Timestamp: time.Now().Unix(),
		SessionID: sessionID,
		Metadata: map[string]interface{}{
			"modCount": 3,
		},
	})

	// Phase 2: Download mods
	mods := []string{"Brita Weapons", "Common Sense", "True Music"}
	for i, mod := range mods {
		s.emitEvent(UIEvent{
			Type:      EventDownloadStart,
			Timestamp: time.Now().Unix(),
			SessionID: sessionID,
			PackageID: mod,
		})

		// Simulate progress
		for progress := 0; progress <= 100; progress += 20 {
			s.emitEvent(UIEvent{
				Type:      EventDownloadProgress,
				Timestamp: time.Now().Unix(),
				SessionID: sessionID,
				PackageID: mod,
				Progress: &Progress{
					Current: int64(progress),
					Total:   100,
					Percent: progress,
					Speed:   1024 * 1024, // 1 MB/s
					ETA:     (100 - progress) / 20,
				},
			})
			time.Sleep(100 * time.Millisecond)
		}

		s.emitEvent(UIEvent{
			Type:      EventDownloadComplete,
			Timestamp: time.Now().Unix(),
			SessionID: sessionID,
			PackageID: mod,
			Metadata: map[string]interface{}{
				"modIndex": i + 1,
				"totalMods": len(mods),
			},
		})
	}

	// Phase 3: Complete
	s.emitEvent(UIEvent{
		Type:      EventSessionComplete,
		Timestamp: time.Now().Unix(),
		SessionID: sessionID,
		Metadata: map[string]interface{}{
			"ready": true,
		},
	})
}

// GetServerList returns mock servers
func (s *UIService) GetServerList() []ServerInfo {
	return []ServerInfo{
		{
			ID:          "server-1",
			Name:        "One Life",
			Description: "Hardcore survival server",
			PlayerCount: 42,
			MaxPlayers:  64,
			Ping:        45,
			ModCount:    15,
			Installed:   false,
			UpToDate:    false,
		},
		{
			ID:          "server-2",
			Name:        "Casual RP",
			Description: "Relaxed roleplay server",
			PlayerCount: 28,
			MaxPlayers:  128,
			Ping:        30,
			ModCount:    8,
			Installed:   true,
			UpToDate:    true,
		},
		{
			ID:          "server-3",
			Name:        "PvP Arena",
			Description: "Player vs player combat",
			PlayerCount: 15,
			MaxPlayers:  32,
			Ping:        60,
			ModCount:    5,
			Installed:   true,
			UpToDate:    false,
		},
	}
}

// GetServerDetails returns detailed info
func (s *UIService) GetServerDetails(serverID string) (*ServerDetails, error) {
	servers := s.GetServerList()
	for _, s := range servers {
		if s.ID == serverID {
			return &ServerDetails{
				ServerInfo:    s,
				Mods:          s.getMockMods(),
				TotalSize:     1024 * 1024 * 1024, // 1 GB
				InstalledSize: 512 * 1024 * 1024,  // 512 MB
				MissingSize:   512 * 1024 * 1024,  // 512 MB
			}, nil
		}
	}
	return nil, fmt.Errorf("server not found: %s", serverID)
}

// getMockMods returns mock mod list
func (s ServerInfo) getMockMods() []ModInfo {
	return []ModInfo{
		{
			ID:         "mod-1",
			Name:       "Brita Weapons",
			WorkshopID: "2200148440",
			Size:       100 * 1024 * 1024,
			Installed:  s.Installed,
			UpToDate:   s.UpToDate,
			Required:   true,
		},
		{
			ID:         "mod-2",
			Name:       "Common Sense",
			WorkshopID: "2875848298",
			Size:       50 * 1024 * 1024,
			Installed:  s.Installed,
			UpToDate:   s.UpToDate,
			Required:   true,
		},
		{
			ID:         "mod-3",
			Name:       "True Music",
			WorkshopID: "2529746725",
			Size:       20 * 1024 * 1024,
			Installed:  s.Installed,
			UpToDate:   s.UpToDate,
			Required:   false,
		},
	}
}

// GetSessionStatus returns mock status
func (s *UIService) GetSessionStatus(sessionID string) (*SessionStatus, error) {
	return &SessionStatus{
		SessionID:     sessionID,
		State:         "complete",
		Progress:      100,
		DownloadSpeed: 0,
		ETA:           0,
	}, nil
}

// RepairCache mocks cache repair
func (s *UIService) RepairCache() error {
	s.emitEvent(UIEvent{
		Type:      EventCacheRepairStart,
		Timestamp: time.Now().Unix(),
	})

	// Simulate repair
	time.Sleep(2 * time.Second)

	s.emitEvent(UIEvent{
		Type:      EventCacheRepairComplete,
		Timestamp: time.Now().Unix(),
	})

	return nil
}

// GetSettings returns default settings
func (s *UIService) GetSettings() (*Settings, error) {
	return &Settings{
		SteamCMDPath:     "/usr/bin/steamcmd",
		CacheLocation:    "~/PZLauncher/cache",
		ProfilesLocation: "~/PZLauncher/profiles",
		MaxConcurrent:    3,
		BandwidthLimit:   0,
	}, nil
}

// SaveSettings saves settings
func (s *UIService) SaveSettings(settings Settings) error {
	// TODO: Persist to config file
	return nil
}

// UI Event Types (RFC 0022)
type UIEventType string

const (
	EventSessionStart         UIEventType = "session.start"
	EventModResolveStart      UIEventType = "mod.resolve.start"
	EventModResolveComplete   UIEventType = "mod.resolve.complete"
	EventDownloadStart        UIEventType = "download.start"
	EventDownloadProgress     UIEventType = "download.progress"
	EventDownloadComplete     UIEventType = "download.complete"
	EventInstallStart         UIEventType = "install.start"
	EventInstallComplete      UIEventType = "install.complete"
	EventSessionComplete      UIEventType = "session.complete"
	EventError                UIEventType = "error"
	EventCacheRepairStart     UIEventType = "cache.repair.start"
	EventCacheRepairComplete  UIEventType = "cache.repair.complete"
)

// UIEvent matches RFC 0022
type UIEvent struct {
	Type      UIEventType          `json:"type"`
	Timestamp int64                `json:"timestamp"`
	SessionID string               `json:"sessionId"`
	PackageID string               `json:"packageId,omitempty"`
	Progress  *Progress            `json:"progress,omitempty"`
	Error     string               `json:"error,omitempty"`
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
