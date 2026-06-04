package contracts

import "time"

// PackageExecutionState represents the execution state of a single package
type PackageExecutionState string

const (
	PackageStatePending     PackageExecutionState = "pending"     // Awaiting execution
	PackageStateSkipped     PackageExecutionState = "skipped"     // Already cached, no download needed
	PackageStateQueued      PackageExecutionState = "queued"      // Ready for download
	PackageStateDownloading PackageExecutionState = "downloading" // In progress
	PackageStateVerifying   PackageExecutionState = "verifying"   // Checking integrity
	PackageStateComplete    PackageExecutionState = "complete"    // Downloaded and verified
	PackageStateFailed      PackageExecutionState = "failed"      // Error occurred
)

// PackageExecution represents the execution trace for a single package
type PackageExecution struct {
	PackageID        string                `json:"packageId"`
	ProviderDecision ProviderDecision      `json:"providerDecision"` // Input: the decision that led here
	State            PackageExecutionState `json:"state"`
	StartedAt        time.Time             `json:"startedAt,omitempty"`
	CompletedAt      time.Time             `json:"completedAt,omitempty"`
	DurationMs       int64                 `json:"durationMs"`
	BytesDownloaded  int64                 `json:"bytesDownloaded,omitempty"`
	BytesTotal       int64                 `json:"bytesTotal,omitempty"`
	Error            string                `json:"error,omitempty"`
	Attempts         int                   `json:"attempts"`            // Retry count
	CachePath        string                `json:"cachePath,omitempty"` // Where it was stored
}

// SessionSummary provides high-level session statistics
type SessionSummary struct {
	TotalPackages   int   `json:"totalPackages"`
	SkippedCount    int   `json:"skippedCount"`  // Already cached
	DownloadCount   int   `json:"downloadCount"` // Needed download
	CompletedCount  int   `json:"completedCount"`
	FailedCount     int   `json:"failedCount"`
	TotalDurationMs int64 `json:"totalDurationMs"`
}

// DownloadSession is the execution session for materializing provider decisions
type DownloadSession struct {
	ID          string             `json:"id"` // Session ID (deterministic from inputs)
	ServerID    string             `json:"serverId"`
	ProfileID   string             `json:"profileId"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt"`
	InputHash   string             `json:"inputHash"`  // Hash of provider decisions (for idempotency)
	Executions  []PackageExecution `json:"executions"` // Per-package execution states
	Summary     SessionSummary     `json:"summary"`
	IsComplete  bool               `json:"isComplete"`
	IsResumable bool               `json:"isResumable"` // Can this session be resumed?
}

// SessionTrace combines provider trace with execution trace
type SessionTrace struct {
	SessionID         string             `json:"sessionId"`
	ProviderDecisions []ProviderDecision `json:"providerDecisions"`
	Executions        []PackageExecution `json:"executions"`
	Summary           SessionSummary     `json:"summary"`
	Timeline          []TraceEvent       `json:"timeline"` // Chronological events
}

// TraceEvent represents a single event in the session timeline
type TraceEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "session_start", "package_start", "package_complete", "session_end", etc.
	PackageID string    `json:"packageId,omitempty"`
	Message   string    `json:"message"`
	State     string    `json:"state,omitempty"`
}
