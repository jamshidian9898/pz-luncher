package pipeline

import (
	"os"
	"path/filepath"
	"runtime"
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
	// 1. Explicit override (dev / CI)
	if r := os.Getenv("PZ_LAUNCHER_ROOT"); r != "" {
		return r
	}

	// 2. Dev mode: walk up to find go.mod (monorepo root)
	cwd, err := os.Getwd()
	if err == nil {
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
	}

	// 3. Production binary: use OS user config directory
	if configDir, err := os.UserConfigDir(); err == nil {
		root := filepath.Join(configDir, "PZLauncher")
		_ = os.MkdirAll(root, 0o755)
		return root
	}

	// 4. Last resort: exe directory
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		// On macOS .app bundles the real exe is deep inside Contents/MacOS
		if runtime.GOOS == "darwin" {
			// Walk up to .app bundle root if inside one
			for i := 0; i < 5; i++ {
				if filepath.Ext(dir) == ".app" {
					dir = filepath.Dir(dir)
					break
				}
				dir = filepath.Dir(dir)
			}
		}
		return dir
	}

	if cwd != "" {
		return cwd
	}
	return "."
}
