// Package pzdetect auto-discovers a running Project Zomboid dedicated server
// on the local machine: process detection, server name, and mods directory.
package pzdetect

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// Result holds the auto-detected PZ server information.
type Result struct {
	ServerName  string // e.g. "servertest" from ServerName.ini
	ModsDir     string // path to Workshop mods directory
	ZomboidDir  string // base Zomboid user directory
	PID         string // process ID of the running server
	GameVersion string // e.g. "42.8.0", "unknown" if undetectable
	MaxPlayers  int    // from server .ini MaxPlayers=, 0 if undetectable
}

// Detect attempts to find a running PZ dedicated server and its configuration.
// Returns nil if no server is found.
func Detect() *Result {
	pid, zomboidDir := findServerProcess()
	if pid == "" {
		// Fallback: try common Zomboid directories even without a running process.
		zomboidDir = findZomboidDir()
		if zomboidDir == "" {
			return nil
		}
		log.Printf("pzdetect: no running PZ server process found, using Zomboid dir: %s", zomboidDir)
	} else {
		log.Printf("pzdetect: found PZ server process (PID %s)", pid)
	}

	serverName := detectServerName(zomboidDir)
	modsDir := detectModsDir(zomboidDir)

	if modsDir == "" {
		return nil
	}

	return &Result{
		ServerName:  serverName,
		ModsDir:     modsDir,
		ZomboidDir:  zomboidDir,
		PID:         pid,
		GameVersion: detectGameVersion(pid),
		MaxPlayers:  detectMaxPlayers(filepath.Join(zomboidDir, "Server"), serverName),
	}
}

// findServerProcess looks for a running PZ server process and returns its PID
// and the Zomboid user directory (parsed from -Duser.home or inferred).
func findServerProcess() (pid, zomboidDir string) {
	if runtime.GOOS == "windows" {
		return findServerProcessWindows()
	}
	return findServerProcessUnix()
}

func findServerProcessUnix() (string, string) {
	// Look for the PZ server process via ps.
	out, err := exec.Command("ps", "aux").Output()
	if err != nil {
		return "", ""
	}

	for _, line := range strings.Split(string(out), "\n") {
		if !isPZServerLine(line) {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid := fields[1]

		// Try to extract -Duser.home from the java command line.
		zomboidDir := extractUserHome(line)
		if zomboidDir == "" {
			zomboidDir = findZomboidDir()
		}
		return pid, zomboidDir
	}
	return "", ""
}

func findServerProcessWindows() (string, string) {
	// On Windows, use tasklist to find the PZ server.
	out, err := exec.Command("tasklist", "/FO", "CSV", "/V").Output()
	if err != nil {
		return "", ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "projectzomboid") {
			fields := strings.Split(line, ",")
			if len(fields) >= 2 {
				pid := strings.Trim(fields[1], "\" ")
				zomboidDir := findZomboidDir()
				return pid, zomboidDir
			}
		}
	}
	return "", ""
}

func isPZServerLine(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "projectzomboid64") ||
		strings.Contains(lower, "projectzomboid32") ||
		(strings.Contains(lower, "java") && strings.Contains(lower, "zombie.network.gameserver"))
}

func extractUserHome(cmdline string) string {
	for _, part := range strings.Fields(cmdline) {
		if strings.HasPrefix(part, "-Duser.home=") {
			home := strings.TrimPrefix(part, "-Duser.home=")
			zdir := filepath.Join(home, "Zomboid")
			if dirExists(zdir) {
				return zdir
			}
		}
	}
	return ""
}

// findZomboidDir searches common locations for the Zomboid directory.
func findZomboidDir() string {
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, "Zomboid"),
		"/root/Zomboid",
		"/home/pzuser/Zomboid",
	}

	// Also check /home/*/Zomboid for any user.
	if homes, err := filepath.Glob("/home/*/Zomboid"); err == nil {
		candidates = append(candidates, homes...)
	}

	for _, c := range candidates {
		if dirExists(c) {
			return c
		}
	}
	return ""
}

// detectServerName reads the server name from .ini files in the Server directory.
func detectServerName(zomboidDir string) string {
	serverDir := filepath.Join(zomboidDir, "Server")
	if !dirExists(serverDir) {
		return ""
	}

	entries, err := os.ReadDir(serverDir)
	if err != nil {
		return ""
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".ini") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".ini")
		// Skip _SandboxVars and _spawnpoints companion files.
		if strings.Contains(name, "_SandboxVars") || strings.Contains(name, "_spawnpoints") || strings.Contains(name, "_spawnregions") {
			continue
		}
		log.Printf("pzdetect: found server config: %s", e.Name())
		return name
	}
	return ""
}

// detectModsDir finds the Workshop mods directory.
// PZ Workshop mods are typically in:
//   - <ZomboidDir>/mods/                           (manually installed)
//   - <Steam>/steamapps/workshop/content/108600/    (Workshop subscribed)
//
// We check the server INI for WorkshopItems and Mods settings,
// and also scan for the workshop content directory.
func detectModsDir(zomboidDir string) string {
	// 1. Check if there's a mods/ dir directly in Zomboid dir.
	localMods := filepath.Join(zomboidDir, "mods")
	if dirExists(localMods) && !isDirEmpty(localMods) {
		log.Printf("pzdetect: found local mods dir: %s", localMods)
		return localMods
	}

	// 2. Find Steam Workshop content directory for PZ (appid 108600).
	workshopDirs := findWorkshopDirs()
	for _, d := range workshopDirs {
		if !isDirEmpty(d) {
			log.Printf("pzdetect: found Workshop mods dir: %s", d)
			return d
		}
	}

	// 3. Parse server INI for Mods= line and resolve from workshop.
	serverDir := filepath.Join(zomboidDir, "Server")
	modsFromINI := parseModsFromINI(serverDir)
	if modsFromINI != "" {
		return modsFromINI
	}

	// 4. Fallback: create and use localMods path.
	if !dirExists(localMods) {
		_ = os.MkdirAll(localMods, 0755)
	}
	return localMods
}

// findWorkshopDirs searches for Steam Workshop content for PZ (appid 108600).
func findWorkshopDirs() []string {
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".steam", "steam", "steamapps", "workshop", "content", "108600"),
		filepath.Join(home, ".local", "share", "Steam", "steamapps", "workshop", "content", "108600"),
		"/home/steam/Steam/steamapps/workshop/content/108600",
		"/home/pzuser/Steam/steamapps/workshop/content/108600",
	}

	// Windows paths.
	for _, drive := range []string{"C", "D", "E"} {
		candidates = append(candidates,
			fmt.Sprintf("%s:\\Program Files (x86)\\Steam\\steamapps\\workshop\\content\\108600", drive),
			fmt.Sprintf("%s:\\Program Files\\Steam\\steamapps\\workshop\\content\\108600", drive),
		)
	}

	// Glob for any user.
	if globs, err := filepath.Glob("/home/*/.steam/steam/steamapps/workshop/content/108600"); err == nil {
		candidates = append(candidates, globs...)
	}
	if globs, err := filepath.Glob("/home/*/.local/share/Steam/steamapps/workshop/content/108600"); err == nil {
		candidates = append(candidates, globs...)
	}

	var found []string
	for _, c := range candidates {
		if dirExists(c) {
			found = append(found, c)
		}
	}
	return found
}

// parseModsFromINI reads server .ini files looking for Mods= and WorkshopItems= lines.
func parseModsFromINI(serverDir string) string {
	if !dirExists(serverDir) {
		return ""
	}

	entries, err := os.ReadDir(serverDir)
	if err != nil {
		return ""
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".ini") {
			continue
		}
		if strings.Contains(e.Name(), "_SandboxVars") || strings.Contains(e.Name(), "_spawnpoints") {
			continue
		}

		iniPath := filepath.Join(serverDir, e.Name())
		f, err := os.Open(iniPath)
		if err != nil {
			continue
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "WorkshopItems=") {
				items := strings.TrimPrefix(line, "WorkshopItems=")
				if items != "" {
					log.Printf("pzdetect: server INI has WorkshopItems: %s", items)
					// Workshop items found — mods are managed by Steam Workshop.
					// Return the first available workshop dir.
					dirs := findWorkshopDirs()
					if len(dirs) > 0 {
						return dirs[0]
					}
				}
			}
		}
	}
	return ""
}

// detectMaxPlayers reads MaxPlayers= from the server's .ini file.
// Returns 0 if the setting is absent or unparsable.
func detectMaxPlayers(serverDir, serverName string) int {
	if !dirExists(serverDir) {
		return 0
	}
	iniPath := filepath.Join(serverDir, serverName+".ini")
	f, err := os.Open(iniPath)
	if err != nil {
		return 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "MaxPlayers=") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "MaxPlayers="))
			n, err := strconv.Atoi(v)
			if err == nil && n > 0 {
				return n
			}
		}
	}
	return 0
}

// versionFileNames are checked, in order, inside a candidate install directory.
var versionFileNames = []string{".pz-version", "version.txt"}

// dirVersionRe extracts a version token from a path component such as
// "ProjectZomboid-42.16.3" or "pzserver-42.8".
var dirVersionRe = regexp.MustCompile(`(?i)(?:ProjectZomboid|PZ)[-_\s]?v?(\d+(?:\.\d+)*\w*)`)

// fallbackVersionRe matches any bare version-looking token as a last resort.
var fallbackVersionRe = regexp.MustCompile(`(\d+(?:\.\d+){1,3})`)

// detectGameVersion tries to determine the installed PZ dedicated server
// version by locating the server's install directory (via the running
// process) and checking it for known version markers. Returns "unknown"
// if no marker is found — never a guessed/hardcoded default.
func detectGameVersion(pid string) string {
	gamePath := detectGamePath(pid)
	if gamePath == "" {
		return "unknown"
	}
	return detectGameVersionFromPath(gamePath)
}

// detectGameVersionFromPath applies the same heuristics used elsewhere in
// the launcher (see apps/launcher-ui/contribute.go detectGameVersion) to a
// dedicated-server install directory.
func detectGameVersionFromPath(gamePath string) string {
	for _, name := range versionFileNames {
		if data, err := os.ReadFile(filepath.Join(gamePath, name)); err == nil {
			if v := strings.TrimSpace(string(data)); v != "" {
				return v
			}
		}
	}

	parts := strings.Split(filepath.ToSlash(gamePath), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if m := dirVersionRe.FindStringSubmatch(parts[i]); len(m) > 1 {
			return m[1]
		}
	}
	for i := len(parts) - 1; i >= 0; i-- {
		if m := fallbackVersionRe.FindStringSubmatch(parts[i]); len(m) > 1 {
			return m[1]
		}
	}
	return "unknown"
}

// detectGamePath resolves the working/install directory of the running PZ
// server process identified by pid. Returns "" if it cannot be determined
// (e.g. no process running, or platform tooling unavailable).
func detectGamePath(pid string) string {
	if pid == "" {
		return ""
	}
	if runtime.GOOS == "windows" {
		return detectGamePathWindows(pid)
	}
	return detectGamePathUnix(pid)
}

func detectGamePathUnix(pid string) string {
	// /proc/<pid>/cwd is the most reliable source on Linux.
	if link, err := os.Readlink(filepath.Join("/proc", pid, "cwd")); err == nil && link != "" {
		return link
	}
	return ""
}

func detectGamePathWindows(pid string) string {
	out, err := exec.Command("wmic", "process", "where", "ProcessId="+pid, "get", "ExecutablePath", "/value").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ExecutablePath=") {
			exe := strings.TrimSpace(strings.TrimPrefix(line, "ExecutablePath="))
			if exe != "" {
				return filepath.Dir(exe)
			}
		}
	}
	return ""
}

func dirExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

func isDirEmpty(path string) bool {
	entries, err := os.ReadDir(path)
	return err != nil || len(entries) == 0
}
