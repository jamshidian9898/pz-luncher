package pipeline

import (
	"os"
	"path/filepath"
)

type Config struct {
	Root          string
	CacheDir      string
	ProfilesDir   string
	SessionsDir   string
	RegistryPath  string
	DemoSeedMods  []string // mod ids to seed into cache (offline demo)
	AutoLaunch    bool
}

func DefaultConfig(root string) Config {
	if root == "" {
		root = WorkspaceRoot()
	}
	return Config{
		Root:         root,
		CacheDir:     filepath.Join(root, "cache", "sha256"),
		ProfilesDir:  filepath.Join(root, "profiles"),
		SessionsDir:  filepath.Join(root, "profiles", ".sessions"),
		RegistryPath: filepath.Join(root, "examples", "servers.json"),
		DemoSeedMods: []string{"mod-a", "base-mod"},
	}
}

func WorkspaceRoot() string {
	if r := os.Getenv("PZ_LAUNCHER_ROOT"); r != "" {
		return r
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	dir := cwd
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return cwd
}
