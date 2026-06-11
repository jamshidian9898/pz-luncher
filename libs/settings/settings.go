package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"pzlauncher/libs/pipeline"
	"pzlauncher/libs/sharedtypes"
)

const fileName = "launcher-settings.json"
const envFileName = "launcher.env"

// Load reads settings from workspace config/ or returns defaults.
// Environment variables and launcher.env always override JSON values:
//
//	PZ_BACKEND_URL  — backend URL (e.g. http://192.168.1.242:8080)
//	PZ_GAME_PATH    — game installation path
func Load(workspaceRoot string) (*sharedtypes.LauncherSettings, error) {
	// Load .env file first (soft errors — file is optional)
	loadEnvFile(filepath.Join(workspaceRoot, "config", envFileName))

	path := filepath.Join(workspaceRoot, "config", fileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			s := Default(workspaceRoot)
			applyEnvOverrides(s)
			return s, nil
		}
		return nil, err
	}
	var s sharedtypes.LauncherSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	applyDefaults(&s, workspaceRoot)
	applyEnvOverrides(&s)
	return &s, nil
}

// loadEnvFile reads KEY=VALUE pairs from a .env file into os environment.
func loadEnvFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range splitLines(string(data)) {
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		for i, c := range line {
			if c == '=' {
				key := strings.TrimSpace(line[:i])
				val := strings.TrimSpace(line[i+1:])
				if key != "" && os.Getenv(key) == "" {
					_ = os.Setenv(key, val)
				}
				break
			}
		}
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, strings.TrimRight(s[start:i], "\r"))
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, strings.TrimRight(s[start:], "\r"))
	}
	return lines
}

// applyEnvOverrides lets env vars take priority over JSON settings.
func applyEnvOverrides(s *sharedtypes.LauncherSettings) {
	if v := os.Getenv("PZ_BACKEND_URL"); v != "" {
		s.BackendURL = v
	}
	if v := os.Getenv("PZ_GAME_PATH"); v != "" {
		s.GamePath = v
	}
}

// Save persists settings.
func Save(workspaceRoot string, s *sharedtypes.LauncherSettings) error {
	applyDefaults(s, workspaceRoot)
	dir := filepath.Join(workspaceRoot, "config")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, fileName), data, 0o644)
}

func Default(workspaceRoot string) *sharedtypes.LauncherSettings {
	s := &sharedtypes.LauncherSettings{
		ConcurrentDownloads: 3,
		VerifyChecksum:      true,
	}
	applyDefaults(s, workspaceRoot)
	return s
}

func applyDefaults(s *sharedtypes.LauncherSettings, root string) {
	if s.BackendURL == "" {
		s.BackendURL = "http://localhost:8080"
	}
	if s.CachePath == "" {
		s.CachePath = filepath.Join(root, "cache")
	}
	if s.ProfilesPath == "" {
		s.ProfilesPath = filepath.Join(root, "profiles")
	}
	if s.ConcurrentDownloads <= 0 {
		s.ConcurrentDownloads = 3
	}
}

// ToPipelineConfig maps settings into pipeline configuration.
func ToPipelineConfig(root string, s *sharedtypes.LauncherSettings) pipeline.Config {
	cfg := pipeline.DefaultConfig(root)
	if s == nil {
		return cfg
	}
	if s.CachePath != "" {
		cfg.CacheDir = filepath.Join(s.CachePath, "sha256")
	}
	if s.ProfilesPath != "" {
		cfg.ProfilesDir = s.ProfilesPath
		cfg.SessionsDir = filepath.Join(s.ProfilesPath, ".sessions")
	}
	return cfg
}

// ApplyGamePathEnv sets PZ_PATH when gamePath configured.
func ApplyGamePathEnv(s *sharedtypes.LauncherSettings) {
	if s != nil && s.GamePath != "" {
		_ = os.Setenv("PZ_PATH", s.GamePath)
	}
}
