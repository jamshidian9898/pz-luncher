package settings

import (
	"encoding/json"
	"os"
	"path/filepath"

	"pzlauncher/libs/pipeline"
	"pzlauncher/libs/sharedtypes"
)

const fileName = "launcher-settings.json"

// Load reads settings from workspace config/ or returns defaults.
func Load(workspaceRoot string) (*sharedtypes.LauncherSettings, error) {
	path := filepath.Join(workspaceRoot, "config", fileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(workspaceRoot), nil
		}
		return nil, err
	}
	var s sharedtypes.LauncherSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	applyDefaults(&s, workspaceRoot)
	return &s, nil
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
