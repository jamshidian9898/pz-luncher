package contracts

import "time"

// ProviderAttempt represents a single provider check attempt
type ProviderAttempt struct {
	ProviderName string        `json:"providerName"`
	CheckedAt    time.Time     `json:"checkedAt"`
	Exists       bool          `json:"exists"`
	Error        string        `json:"error,omitempty"`
	DurationMs   int64         `json:"durationMs"`
	CachePath    string        `json:"cachePath,omitempty"` // e.g., "cache/sha256/abc123"
}

// ProviderDecision is a complete decision trace for a package
type ProviderDecision struct {
	PackageID       string            `json:"packageId"`
	PackageVersion  string            `json:"packageVersion"`
	PackageSHA256   string            `json:"packageSha256,omitempty"`
	ChosenProvider  string            `json:"chosenProvider"`
	Cached          bool              `json:"cached"`
	DecisionAt      time.Time         `json:"decisionAt"`
	TotalDurationMs int64             `json:"totalDurationMs"`
	FinalReason     string            `json:"finalReason"` // e.g., "found in LocalCache", "cache miss → fallback to Steam"
	Attempts        []ProviderAttempt `json:"attempts"`    // chronological attempts
	FallbackChain   []string          `json:"fallbackChain,omitempty"`
}
