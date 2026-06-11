package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

// App struct
type App struct {
	ctx context.Context
	ui  *UIService
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		ui: NewUIService(),
	}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// In production, set workspace root to exe directory so pipeline can find config/
	if exeDir := getExeDir(); exeDir != "" {
		_ = os.Setenv("PZ_LAUNCHER_ROOT", exeDir)
	}
	a.ui.SetContext(ctx)
	a.ui.ReloadConfig()
}

func getExeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(exe)
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

// CheckBackend runs health checks from Go side and returns status
func (a *App) CheckBackend() HealthStatus {
	return a.ui.CheckBackend()
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
	Mods          []ModInfo `json:"mods"`
	TotalSize     int64     `json:"totalSize"`
	InstalledSize int64     `json:"installedSize"`
	MissingSize   int64     `json:"missingSize"`
}

// ModInfo represents a mod
type ModInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	WorkshopID string `json:"workshopId"`
	Size       int64  `json:"size"`
	Installed  bool   `json:"installed"`
	UpToDate   bool   `json:"upToDate"`
	Required   bool   `json:"required"`
}

// SessionStatus shows current operation progress
type SessionStatus struct {
	SessionID     string   `json:"sessionId"`
	State         string   `json:"state"` // resolving, downloading, installing, complete
	Progress      float64  `json:"progress"`
	CurrentMod    string   `json:"currentMod,omitempty"`
	DownloadSpeed int64    `json:"downloadSpeed,omitempty"`
	ETA           int      `json:"eta,omitempty"`
	Errors        []string `json:"errors,omitempty"`
}

// Settings for the launcher (RFC-0036 v2.0.0 — mirrors shared/contracts/settings.schema.json)
type Settings struct {
	GamePath         string `json:"gamePath"`
	BackendURL       string `json:"backendUrl"`
	CacheLocation    string `json:"cacheLocation"`
	ProfilesLocation string `json:"profilesLocation"`
	MaxConcurrent    int    `json:"maxConcurrent"`
	BandwidthLimit   int    `json:"bandwidthLimit"`
	VerifyChecksum   bool   `json:"verifyChecksum"`
	LaunchOptions    string `json:"launchOptions,omitempty"`
}

func main() {
	app := NewApp()

	dist, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		log.Fatal("failed to get frontend/dist sub-fs:", err)
	}

	err = wails.Run(&options.App{
		Title:     "PZ Launcher",
		Width:     1200,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: dist,
		},
		BackgroundColour: &options.RGBA{R: 15, G: 23, B: 42, A: 255},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
			About: &mac.AboutInfo{
				Title:   "PZ Launcher",
				Message: "Beta 1",
			},
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
