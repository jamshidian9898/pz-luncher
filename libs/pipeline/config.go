package pipeline

import (
	"os"
	"path/filepath"
	"runtime"
)

type Config struct {
	Root         string
	CacheDir     string
	ProfilesDir  string
	SessionsDir  string
	RegistryPath string
	DemoSeedMods []string // mod ids to seed into cache (offline demo)
	AutoLaunch   bool
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
	// Try UserConfigDir first, then explicit env vars as fallback
	configRoot := ""
	if configDir, err := os.UserConfigDir(); err == nil && configDir != "" {
		configRoot = configDir
	} else if appdata := os.Getenv("APPDATA"); appdata != "" {
		// Windows fallback
		configRoot = appdata
	} else if home, err := os.UserHomeDir(); err == nil {
		// Unix fallback
		configRoot = filepath.Join(home, ".config")
	}
	if configRoot != "" {
		root := filepath.Join(configRoot, "PZLauncher")
		if err := os.MkdirAll(root, 0o755); err == nil {
			return root
		}
	}

	// 4. Last resort: exe directory (NOT cwd — avoids Downloads/Desktop confusion)
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		// On macOS .app bundles the real exe is deep inside Contents/MacOS
		if runtime.GOOS == "darwin" {
			for i := 0; i < 5; i++ {
				if filepath.Ext(dir) == ".app" {
					dir = filepath.Dir(dir)
					break
				}
				dir = filepath.Dir(dir)
			}
		}
		_ = os.MkdirAll(filepath.Join(dir, "config"), 0o755)
		return dir
	}

	if cwd != "" {
		return cwd
	}
	return "."
}
