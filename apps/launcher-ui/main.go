package main

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// App struct
type App struct {
	ctx    context.Context
	ui     *UIService
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		ui: NewUIService(),
	}
}

// OnStartup is called when the app starts
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx
	a.ui.SetContext(ctx)
}

// JoinServer starts joining a server with mod resolution
func (a *App) JoinServer(serverID string) error {
	return a.ui.JoinServer(serverID)
}

// LaunchServer starts the game after a successful join
func (a *App) LaunchServer(serverID string) error {
	return a.ui.LaunchServer(serverID)
}

// GetServerList returns list of available servers
func (a *App) GetServerList() []ServerInfo {
	return a.ui.GetServerList()
}

// GetServerDetails returns detailed info for a server
func (a *App) GetServerDetails(serverID string) (*ServerDetails, error) {
	return a.ui.GetServerDetails(serverID)
}

// GetSessionStatus returns current session progress
func (a *App) GetSessionStatus(sessionID string) (*SessionStatus, error) {
	return a.ui.GetSessionStatus(sessionID)
}

// RepairCache validates and repairs local cache
func (a *App) RepairCache() error {
	return a.ui.RepairCache()
}

// GetSettings returns launcher settings
func (a *App) GetSettings() (*Settings, error) {
	return a.ui.GetSettings()
}

// SaveSettings saves launcher settings
func (a *App) SaveSettings(settings Settings) error {
	return a.ui.SaveSettings(settings)
}

// ServerInfo represents a server in the list
type ServerInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PlayerCount int    `json:"playerCount"`
	MaxPlayers  int    `json:"maxPlayers"`
	Ping        int    `json:"ping"`
	ModCount    int    `json:"modCount"`
	Installed   bool   `json:"installed"`
	UpToDate    bool   `json:"upToDate"`
}

// ServerDetails contains full server information
type ServerDetails struct {
	ServerInfo
	Mods         []ModInfo `json:"mods"`
	TotalSize    int64     `json:"totalSize"`
	InstalledSize int64    `json:"installedSize"`
	MissingSize  int64     `json:"missingSize"`
}

// ModInfo represents a mod
type ModInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	WorkshopID  string `json:"workshopId"`
	Size        int64  `json:"size"`
	Installed   bool   `json:"installed"`
	UpToDate    bool   `json:"upToDate"`
	Required    bool   `json:"required"`
}

// SessionStatus shows current operation progress
type SessionStatus struct {
	SessionID     string         `json:"sessionId"`
	State         string         `json:"state"` // resolving, downloading, installing, complete
	Progress      float64        `json:"progress"`
	CurrentMod    string         `json:"currentMod,omitempty"`
	DownloadSpeed int64          `json:"downloadSpeed,omitempty"`
	ETA           int            `json:"eta,omitempty"`
	Errors        []string       `json:"errors,omitempty"`
}

// Settings for the launcher (RFC-0036 — mirrors shared/contracts/settings.schema.json)
type Settings struct {
	GamePath         string `json:"gamePath"`
	SteamCMDPath     string `json:"steamcmdPath"`
	CacheLocation    string `json:"cacheLocation"`
	ProfilesLocation string `json:"profilesLocation"`
	MaxConcurrent    int    `json:"maxConcurrent"`
	BandwidthLimit   int    `json:"bandwidthLimit"`
	VerifyChecksum   bool   `json:"verifyChecksum"`
}

func main() {
	app := NewApp()

	wailsApp := application.New(application.Options{
		Title:  "PZ Launcher",
		Width:  1200,
		Height: 800,
		Assets: application.AlphaAssets,
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	wailsApp.Bind(app)

	// Create window
	window := wailsApp.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title:     "PZ Launcher",
		Width:     1200,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
	})

	// Load frontend
	window.LoadURL("http://localhost:5173")

	// Run app
	if err := wailsApp.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
