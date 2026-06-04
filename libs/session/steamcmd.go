package session

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SteamCMDClient wraps steamcmd for downloading Workshop items
// Used as fallback when Steam Web API doesn't provide direct URLs
type SteamCMDClient struct {
	ExecutablePath string
	SteamUsername  string
	SteamPassword  string // Optional: for private items
	Anonymous      bool   // Use anonymous login (default true)
}

// NewSteamCMDClient creates a new steamcmd client
// executablePath should point to steamcmd.sh (Linux/Mac) or steamcmd.exe (Windows)
func NewSteamCMDClient(executablePath string) *SteamCMDClient {
	return &SteamCMDClient{
		ExecutablePath: executablePath,
		Anonymous:      true,
	}
}

// DownloadWorkshopItem downloads a Workshop item using steamcmd
// appID is the game ID (e.g., 108600 for Project Zomboid)
// workshopID is the published file ID
// Returns the path to the downloaded file
func (c *SteamCMDClient) DownloadWorkshopItem(ctx context.Context, appID int, workshopID string, outputDir string, progress ProgressCallback) (string, error) {
	if err := c.validateExecutable(); err != nil {
		return "", err
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}

	// Build steamcmd commands
	commands := c.buildDownloadCommands(appID, workshopID, outputDir)

	// Create command
	cmd := exec.CommandContext(ctx, c.ExecutablePath, commands...)

	// Set working directory
	cmd.Dir = outputDir

	// Capture output for progress parsing
	// SteamCMD outputs progress like: "Downloading item 123456... (50%)"
	startTime := time.Now()

	// Create a pipe to capture output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start steamcmd: %w", err)
	}

	// Parse output for progress (simplified - real implementation would parse steamcmd output)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if err != nil {
				break
			}
			// Parse steamcmd output for progress
			_ = string(buf[:n]) // In real implementation, parse for "Downloading... X%"
		}
	}()

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		// Check for specific steamcmd errors
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			stderrStr := ""
			if stderr != nil {
				if data, err := os.ReadFile("/dev/stderr"); err == nil {
					stderrStr = string(data)
				}
			}
			return "", fmt.Errorf("steamcmd failed (exit %d): %s", exitErr.ExitCode(), stderrStr)
		}
		return "", fmt.Errorf("steamcmd execution failed: %w", err)
	}

	// Find the downloaded file
	// SteamCMD downloads to: outputDir/steamapps/workshop/content/<appID>/<workshopID>/
	workshopPath := filepath.Join(outputDir, "steamapps", "workshop", "content",
		fmt.Sprintf("%d", appID), workshopID)

	files, err := os.ReadDir(workshopPath)
	if err != nil {
		return "", fmt.Errorf("locate downloaded file: %w", err)
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no files downloaded")
	}

	// Return path to first file (or directory if multiple files)
	downloadedPath := filepath.Join(workshopPath, files[0].Name())

	// Report final progress
	if progress != nil {
		info, _ := os.Stat(downloadedPath)
		var size int64 = 0
		if info != nil {
			size = info.Size()
		}
		progress(ProgressEvent{
			PackageID:       workshopID,
			Provider:        "steamcmd",
			BytesDownloaded: size,
			BytesTotal:      size,
			SpeedBps:        float64(size) / time.Since(startTime).Seconds(),
			Percent:         100.0,
			Timestamp:       time.Now(),
		})
	}

	return downloadedPath, nil
}

// IsAvailable checks if steamcmd is installed and working
func (c *SteamCMDClient) IsAvailable() bool {
	if err := c.validateExecutable(); err != nil {
		return false
	}

	// Try to run steamcmd with "version" command
	cmd := exec.Command(c.ExecutablePath, "+quit")
	if err := cmd.Run(); err != nil {
		return false
	}

	return true
}

// validateExecutable checks if steamcmd executable exists
func (c *SteamCMDClient) validateExecutable() error {
	if c.ExecutablePath == "" {
		return fmt.Errorf("steamcmd executable path not set")
	}

	if _, err := os.Stat(c.ExecutablePath); err != nil {
		return fmt.Errorf("steamcmd not found at %s: %w", c.ExecutablePath, err)
	}

	return nil
}

// buildDownloadCommands creates the steamcmd command sequence
func (c *SteamCMDClient) buildDownloadCommands(appID int, workshopID, outputDir string) []string {
	commands := []string{}

	// Login
	if c.Anonymous {
		commands = append(commands, "+login", "anonymous")
	} else {
		commands = append(commands, "+login", c.SteamUsername)
		if c.SteamPassword != "" {
			// Note: In production, use Steam Guard or alternative auth
			commands = append(commands, c.SteamPassword)
		}
	}

	// Set download directory
	commands = append(commands, "+force_install_dir", outputDir)

	// Download workshop item
	// Syntax: workshop_download_item <appid> <workshopid>
	commands = append(commands, "+workshop_download_item", fmt.Sprintf("%d", appID), workshopID)

	// Quit
	commands = append(commands, "+quit")

	return commands
}

// FindSteamCMD attempts to locate steamcmd in common paths
func FindSteamCMD() string {
	// Common paths
	paths := []string{
		"steamcmd", // In PATH
		"/usr/local/bin/steamcmd",
		"/usr/bin/steamcmd",
		"C:\\Program Files (x86)\\Steam\\steamcmd.exe",
		"C:\\Steam\\steamcmd.exe",
		"./steamcmd.sh",
		"./steamcmd",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
		// Also try with .exe on Windows
		if !strings.HasSuffix(path, ".exe") {
			winPath := path + ".exe"
			if _, err := os.Stat(winPath); err == nil {
				return winPath
			}
		}
	}

	return ""
}
