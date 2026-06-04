package session

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"pzlauncher/libs/contracts"
	"pzlauncher/libs/providers"
)

// SteamExecutor is a real Steam Workshop executor for production use
// It implements a hardened multi-strategy download chain:
//  1. Workshop ID mapping (mod name -> Steam ID)
//  2. Rate limiting (prevents API bans)
//  3. Steam Web API (direct download URL)
//  4. SteamCMD fallback (when API unavailable)
//  5. Failure injection (for testing chaos scenarios)
//  6. Cached file verification (idempotency)
//
// All complexity lives here — session manager remains agnostic
type SteamExecutor struct {
	CacheDir        string
	MappingService  *MappingService
	RateLimiter     *RateLimiter
	FailureInjector *FailureInjector
	APIClient       *SteamAPIClient
	CMDClient       *SteamCMDClient
	HTTPClient      *http.Client
	MaxRetries      int
	RetryDelay      time.Duration
	RetryBudget     int
	Progress        ProgressCallback
}

// NewSteamExecutor creates a new Steam executor with sensible defaults
func NewSteamExecutor(cacheDir string) *SteamExecutor {
	return &SteamExecutor{
		CacheDir:        cacheDir,
		MappingService:  NewMappingService(cacheDir),
		RateLimiter:     NewRateLimiter().WithSteamLimits(),
		FailureInjector: NewFailureInjector(),  // Disabled by default
		APIClient:       NewSteamAPIClient(""), // No API key for public items
		CMDClient:       nil,                   // Will be auto-detected if available
		HTTPClient:      &http.Client{Timeout: 5 * time.Minute},
		MaxRetries:      3,
		RetryDelay:      2 * time.Second,
		RetryBudget:     10, // Global retry budget per session
		Progress:        NopProgressCallback,
	}
}

// WithSteamCMD configures the executor to use steamcmd as fallback
func (e *SteamExecutor) WithSteamCMD(executablePath string) *SteamExecutor {
	e.CMDClient = NewSteamCMDClient(executablePath)
	return e
}

// WithProgress sets a progress callback for download events
func (e *SteamExecutor) WithProgress(cb ProgressCallback) *SteamExecutor {
	e.Progress = cb
	return e
}

// WithFailureInjector enables chaos testing for this executor
func (e *SteamExecutor) WithFailureInjector(injector *FailureInjector) *SteamExecutor {
	e.FailureInjector = injector
	return e
}

// WithRateLimiter sets a custom rate limiter
func (e *SteamExecutor) WithRateLimiter(limiter *RateLimiter) *SteamExecutor {
	e.RateLimiter = limiter
	return e
}

// Execute downloads a package from Steam Workshop with full observability
func (e *SteamExecutor) Execute(ctx context.Context, exec *contracts.PackageExecution) (*contracts.PackageExecution, error) {
	decision := exec.ProviderDecision

	// Validate this is a Steam decision
	if decision.ChosenProvider != "Steam" {
		return nil, fmt.Errorf("steam executor cannot handle provider: %s", decision.ChosenProvider)
	}

	// Build target path in cache
	targetPath := filepath.Join(e.CacheDir, decision.PackageSHA256)

	// Check if already exists (idempotency at file level)
	if _, err := os.Stat(targetPath); err == nil {
		// File exists — verify hash
		if err := e.verifyHash(targetPath, decision.PackageSHA256); err == nil {
			exec.State = contracts.PackageStateComplete
			exec.CompletedAt = time.Now()
			exec.CachePath = targetPath
			return exec, nil
		}
		// Hash mismatch — will overwrite
	}

	// Resolve Workshop ID from package metadata
	workshopID, err := e.resolveWorkshopID(decision.PackageID, decision.PackageVersion)
	if err != nil {
		exec.State = contracts.PackageStateFailed
		exec.Error = fmt.Sprintf("resolve workshop id: %v", err)
		return exec, err
	}

	// Execute download with retry
	startTime := time.Now()
	var lastErr error

	for attempt := 0; attempt < e.MaxRetries; attempt++ {
		if attempt > 0 {
			exec.Attempts = attempt + 1
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(e.RetryDelay * time.Duration(attempt)):
			}
		}

		err := e.downloadWithFallback(ctx, workshopID, targetPath, decision.PackageSHA256)
		if err == nil {
			// Success
			exec.State = contracts.PackageStateComplete
			exec.CompletedAt = time.Now()
			exec.DurationMs = time.Since(startTime).Milliseconds()
			exec.CachePath = targetPath
			exec.Attempts = attempt + 1
			return exec, nil
		}

		lastErr = err

		// Check if error is retryable
		if !e.isRetryable(err) {
			break
		}
	}

	// All retries failed
	exec.State = contracts.PackageStateFailed
	exec.Error = fmt.Sprintf("download failed after %d attempts: %v", exec.Attempts, lastErr)
	exec.DurationMs = time.Since(startTime).Milliseconds()
	return exec, fmt.Errorf("steam download failed: %w", lastErr)
}

// resolveWorkshopID maps a mod ID to Steam Workshop ID
// Priority: 1) MappingService (local cache), 2) Direct numeric ID, 3) SteamAPI resolver
func (e *SteamExecutor) resolveWorkshopID(packageID, version string) (string, error) {
	// Check rate limiter first
	if e.RateLimiter != nil {
		if err := e.RateLimiter.Acquire(context.Background()); err != nil {
			return "", fmt.Errorf("rate limited: %w", err)
		}
	}

	// Inject failure if testing
	if e.FailureInjector != nil {
		if err := e.FailureInjector.MaybeFail(context.Background(), "resolve"); err != nil {
			return "", err
		}
	}

	// 1. Try MappingService (local cache + registry)
	if e.MappingService != nil {
		if workshopID, err := e.MappingService.Resolve(packageID); err == nil {
			return workshopID, nil
		}
	}

	// 2. If it's already numeric, it's likely a workshop ID
	if isNumeric(packageID) {
		return packageID, nil
	}

	// 3. Fallback to Steam API
	if e.APIClient != nil {
		return e.APIClient.ResolveWorkshopID(packageID, version)
	}

	return "", fmt.Errorf("cannot resolve mod %s to workshop id: no resolver available", packageID)
}

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// downloadWithFallback attempts download using multiple strategies:
// 1. Steam Web API (direct download URL)
// 2. SteamCMD (fallback when API unavailable or no URL provided)
func (e *SteamExecutor) downloadWithFallback(ctx context.Context, workshopID, targetPath, expectedSHA string) error {
	// Strategy 1: Try Steam Web API for direct download URL
	if e.APIClient != nil && e.APIClient.IsAvailable() {
		url, size, err := e.APIClient.GetDownloadURL(workshopID)
		if err == nil && url != "" {
			// API returned a direct URL - use HTTP download
			return e.downloadHTTP(ctx, url, size, targetPath, expectedSHA)
		}
		// API available but no URL - item requires steamcmd
	}

	// Strategy 2: Fallback to SteamCMD
	if e.CMDClient != nil && e.CMDClient.IsAvailable() {
		return e.downloadSteamCMD(ctx, workshopID, targetPath, expectedSHA)
	}

	return fmt.Errorf("no download method available: steam api failed and steamcmd not available")
}

// downloadHTTP downloads via HTTP with progress tracking
func (e *SteamExecutor) downloadHTTP(ctx context.Context, url string, totalSize int64, targetPath, expectedSHA string) error {
	// Ensure cache directory exists
	if err := os.MkdirAll(e.CacheDir, 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	// Create temp file
	tempFile := targetPath + ".tmp"
	f, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tempFile)

	// Start download
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := e.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http error %d: %s", resp.StatusCode, resp.Status)
	}

	// Stream download with progress and hash verification
	hasher := sha256.New()
	var downloaded int64 = 0
	startTime := time.Now()
	buf := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			f.Write(buf[:n])
			hasher.Write(buf[:n])
			downloaded += int64(n)

			// Report progress
			if totalSize > 0 {
				e.Progress(ProgressEvent{
					BytesDownloaded: downloaded,
					BytesTotal:      totalSize,
					Percent:         float64(downloaded) / float64(totalSize) * 100,
					SpeedBps:        float64(downloaded) / time.Since(startTime).Seconds(),
				})
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("download stream: %w", err)
		}
	}

	f.Close()

	// Verify hash
	actualSHA := hex.EncodeToString(hasher.Sum(nil))
	if actualSHA != expectedSHA {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedSHA[:16], actualSHA[:16])
	}

	// Atomic move
	if err := os.Rename(tempFile, targetPath); err != nil {
		return fmt.Errorf("finalize download: %w", err)
	}

	return nil
}

// downloadSteamCMD downloads using steamcmd as fallback
func (e *SteamExecutor) downloadSteamCMD(ctx context.Context, workshopID, targetPath, expectedSHA string) error {
	// Project Zomboid app ID
	const pzAppID = 108600

	// Download to temp directory
	tempDir := filepath.Join(e.CacheDir, ".steamcmd-temp", workshopID)
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir) // Cleanup

	// Execute steamcmd download
	downloadedPath, err := e.CMDClient.DownloadWorkshopItem(ctx, pzAppID, workshopID, tempDir, e.Progress)
	if err != nil {
		return fmt.Errorf("steamcmd download: %w", err)
	}

	// Verify hash of downloaded file
	if err := e.verifyHash(downloadedPath, expectedSHA); err != nil {
		return fmt.Errorf("steamcmd download hash mismatch: %w", err)
	}

	// Move to final location
	if err := os.Rename(downloadedPath, targetPath); err != nil {
		return fmt.Errorf("finalize steamcmd download: %w", err)
	}

	return nil
}

// verifyHash checks if a file matches the expected SHA256
func (e *SteamExecutor) verifyHash(filePath, expectedSHA string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actualSHA := hex.EncodeToString(h.Sum(nil))
	if actualSHA != expectedSHA {
		return fmt.Errorf("hash mismatch")
	}
	return nil
}

// isRetryable determines if an error warrants a retry
func (e *SteamExecutor) isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Network errors are retryable
	// Hash mismatches are NOT retryable (indicates corrupt source)
	// Context cancellation is NOT retryable

	errStr := err.Error()

	// Non-retryable errors
	if errStr == "hash mismatch" {
		return false // Source is corrupt
	}

	// Context errors
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}

	// Everything else is retryable (network, IO, etc.)
	return true
}

// Provider returns the underlying provider for this executor
// This allows the executor to participate in provider selection
func (e *SteamExecutor) Provider() providers.Provider {
	return providers.NewSteamProvider()
}
