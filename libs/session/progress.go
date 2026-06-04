package session

import "time"

// ProgressEvent represents a download progress update
// Used for internal observability, not UI directly
type ProgressEvent struct {
	PackageID       string    `json:"packageId"`
	Provider        string    `json:"provider"`
	BytesDownloaded int64     `json:"bytesDownloaded"`
	BytesTotal      int64     `json:"bytesTotal"`
	SpeedBps        float64   `json:"speedBps"` // bytes per second
	Percent         float64   `json:"percent"`
	Timestamp       time.Time `json:"timestamp"`
}

// ProgressCallback is called periodically during download
// Returns error to cancel download, nil to continue
type ProgressCallback func(event ProgressEvent) error

// NopProgressCallback is a no-op callback for when progress isn't needed
func NopProgressCallback(event ProgressEvent) error {
	return nil
}
