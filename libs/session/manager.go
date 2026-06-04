package session

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"pzlauncher/libs/contracts"
)

// Manager handles download session lifecycle
type Manager interface {
	// CreateSession creates a new session from provider decisions (idempotent)
	CreateSession(serverID, profileID string, decisions []contracts.ProviderDecision) (*contracts.DownloadSession, error)

	// LoadSession loads an existing session from disk
	LoadSession(sessionID string) (*contracts.DownloadSession, error)

	// Execute runs the session, resuming from previous state if available
	Execute(ctx context.Context, session *contracts.DownloadSession, executor Executor) error

	// Save persists session state to disk
	Save(session *contracts.DownloadSession) error

	// GetTrace returns the combined provider + execution trace
	GetTrace(session *contracts.DownloadSession) contracts.SessionTrace
}

// Executor handles the actual download/verification of a single package
type Executor interface {
	// Execute processes a single package based on its provider decision
	// Returns: updated execution state, isComplete, error
	Execute(ctx context.Context, exec *contracts.PackageExecution) (*contracts.PackageExecution, error)
}

// SimpleManager is a file-based session manager
type SimpleManager struct {
	SessionsDir string // Where to persist sessions
}

// NewSimpleManager creates a new session manager
func NewSimpleManager(sessionsDir string) *SimpleManager {
	return &SimpleManager{SessionsDir: sessionsDir}
}

// CreateSession creates a deterministic session from provider decisions
func (m *SimpleManager) CreateSession(serverID, profileID string, decisions []contracts.ProviderDecision) (*contracts.DownloadSession, error) {
	// Generate deterministic session ID from inputs
	inputHash := hashDecisions(decisions)
	sessionID := fmt.Sprintf("%s-%s", serverID, inputHash[:16])

	// Check if session already exists (idempotency)
	existing, err := m.LoadSession(sessionID)
	if err == nil && existing != nil {
		// Session exists - check if it's resumable
		if existing.IsResumable && !existing.IsComplete {
			return existing, nil // Return existing session for resume
		}
		if existing.IsComplete {
			// Already complete - return as-is (idempotent)
			return existing, nil
		}
	}

	// Create executions from decisions
	executions := make([]contracts.PackageExecution, len(decisions))
	var downloadCount, skippedCount int

	for i, d := range decisions {
		exec := contracts.PackageExecution{
			PackageID:        d.PackageID,
			ProviderDecision: d,
			State:            contracts.PackageStatePending,
			Attempts:         0,
		}

		// If already cached, mark as skipped
		if d.Cached {
			exec.State = contracts.PackageStateSkipped
			exec.CompletedAt = time.Now()
			skippedCount++
		} else {
			downloadCount++
		}

		executions[i] = exec
	}

	session := &contracts.DownloadSession{
		ID:          sessionID,
		ServerID:    serverID,
		ProfileID:   profileID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		InputHash:   inputHash,
		Executions:  executions,
		IsComplete:  false,
		IsResumable: true,
		Summary: contracts.SessionSummary{
			TotalPackages:  len(decisions),
			SkippedCount:   skippedCount,
			DownloadCount:  downloadCount,
			CompletedCount: skippedCount, // Skipped = already complete
		},
	}

	// Save initial state
	if err := m.Save(session); err != nil {
		return nil, fmt.Errorf("save initial session: %w", err)
	}

	return session, nil
}

// LoadSession loads a session from disk
func (m *SimpleManager) LoadSession(sessionID string) (*contracts.DownloadSession, error) {
	path := filepath.Join(m.SessionsDir, sessionID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var session contracts.DownloadSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}

	return &session, nil
}

// Execute runs the session, processing pending packages
func (m *SimpleManager) Execute(ctx context.Context, session *contracts.DownloadSession, executor Executor) error {
	startTime := time.Now()

	// Record session start in timeline
	timeline := []contracts.TraceEvent{{
		Timestamp: time.Now(),
		Type:      "session_start",
		Message:   fmt.Sprintf("Session %s started (%d packages, %d to download)", session.ID, session.Summary.TotalPackages, session.Summary.DownloadCount),
	}}

	// Process each package
	for i := range session.Executions {
		select {
		case <-ctx.Done():
			session.IsResumable = true
			session.UpdatedAt = time.Now()
			session.Summary.TotalDurationMs = time.Since(startTime).Milliseconds()
			_ = m.Save(session)
			return ctx.Err()
		default:
		}

		pkgExec := &session.Executions[i]

		// Skip if already complete or skipped
		if pkgExec.State == contracts.PackageStateSkipped ||
			pkgExec.State == contracts.PackageStateComplete {
			timeline = append(timeline, contracts.TraceEvent{
				Timestamp: time.Now(),
				Type:      "package_skip",
				PackageID: pkgExec.PackageID,
				Message:   fmt.Sprintf("Package %s already %s", pkgExec.PackageID, pkgExec.State),
			})
			continue
		}

		// Execute the package
		pkgExec.StartedAt = time.Now()
		pkgExec.State = contracts.PackageStateDownloading
		pkgExec.Attempts++

		timeline = append(timeline, contracts.TraceEvent{
			Timestamp: time.Now(),
			Type:      "package_start",
			PackageID: pkgExec.PackageID,
			Message:   fmt.Sprintf("Starting download from %s", pkgExec.ProviderDecision.ChosenProvider),
			State:     string(pkgExec.State),
		})

		// Call the executor
		updated, err := executor.Execute(ctx, pkgExec)
		if err != nil {
			pkgExec.State = contracts.PackageStateFailed
			pkgExec.Error = err.Error()
			session.Summary.FailedCount++
			timeline = append(timeline, contracts.TraceEvent{
				Timestamp: time.Now(),
				Type:      "package_failed",
				PackageID: pkgExec.PackageID,
				Message:   fmt.Sprintf("Failed: %v", err),
				State:     string(pkgExec.State),
			})
		} else {
			*pkgExec = *updated
			if pkgExec.State == contracts.PackageStateComplete {
				session.Summary.CompletedCount++
				timeline = append(timeline, contracts.TraceEvent{
					Timestamp: time.Now(),
					Type:      "package_complete",
					PackageID: pkgExec.PackageID,
					Message:   fmt.Sprintf("Complete in %dms", pkgExec.DurationMs),
					State:     string(pkgExec.State),
				})
			}
		}

		pkgExec.CompletedAt = time.Now()
		pkgExec.DurationMs = time.Since(pkgExec.StartedAt).Milliseconds()

		// Save progress after each package (for resume)
		session.UpdatedAt = time.Now()
		_ = m.Save(session)
	}

	// Mark session complete
	session.IsComplete = session.Summary.CompletedCount == session.Summary.TotalPackages
	session.IsResumable = !session.IsComplete || session.Summary.FailedCount > 0
	session.UpdatedAt = time.Now()
	session.Summary.TotalDurationMs = time.Since(startTime).Milliseconds()

	timeline = append(timeline, contracts.TraceEvent{
		Timestamp: time.Now(),
		Type:      "session_end",
		Message: fmt.Sprintf("Session complete: %d/%d packages, %d failed, %dms total",
			session.Summary.CompletedCount, session.Summary.TotalPackages,
			session.Summary.FailedCount, session.Summary.TotalDurationMs),
	})

	// Store timeline (we'll add this to the session struct if needed)
	_ = timeline

	return m.Save(session)
}

// Save persists session to disk
func (m *SimpleManager) Save(session *contracts.DownloadSession) error {
	if err := os.MkdirAll(m.SessionsDir, 0755); err != nil {
		return fmt.Errorf("create sessions dir: %w", err)
	}

	path := filepath.Join(m.SessionsDir, session.ID+".json")
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// GetTrace returns combined provider + execution trace
func (m *SimpleManager) GetTrace(session *contracts.DownloadSession) contracts.SessionTrace {
	decisions := make([]contracts.ProviderDecision, len(session.Executions))
	for i, e := range session.Executions {
		decisions[i] = e.ProviderDecision
	}

	return contracts.SessionTrace{
		SessionID:         session.ID,
		ProviderDecisions: decisions,
		Executions:        session.Executions,
		Summary:           session.Summary,
	}
}

// hashDecisions creates a deterministic hash of provider decisions
func hashDecisions(decisions []contracts.ProviderDecision) string {
	h := sha256.New()
	for _, d := range decisions {
		h.Write([]byte(d.PackageID))
		h.Write([]byte(d.PackageVersion))
		h.Write([]byte(d.PackageSHA256))
		h.Write([]byte(d.ChosenProvider))
	}
	return hex.EncodeToString(h.Sum(nil))
}
