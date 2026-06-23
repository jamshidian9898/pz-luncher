// Package content implements the Content Registry (RFC-0059).
//
// It tracks metadata and trust levels for game client binaries submitted by
// community members. Actual blob storage is delegated to storage.Store (RFC-0054).
//
// Trust model: N independent submitters with matching SHA256 → Verified.
// Independence is approximated by /24 IP subnet deduplication.
package content

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TrustLevel describes how validated a piece of content is.
type TrustLevel string

const (
	TrustPending  TrustLevel = "pending"
	TrustVerified TrustLevel = "verified"
	TrustRejected TrustLevel = "rejected"

	DefaultThreshold = 3 // independent submitters needed for Verified
)

// ContentRecord is the metadata entry for one (game, version, platform) binary.
type ContentRecord struct {
	SHA256          string     `json:"sha256"`
	ContentType     string     `json:"contentType"` // "game-binary"
	GameID          string     `json:"gameId"`
	GameVersion     string     `json:"gameVersion"`
	Platform        string     `json:"platform"`
	SizeBytes       int64      `json:"sizeBytes"`
	TrustLevel      TrustLevel `json:"trustLevel"`
	UploadCount     int        `json:"uploadCount"`
	FirstUploadedAt string     `json:"firstUploadedAt"`
	LastVerifiedAt  string     `json:"lastVerifiedAt,omitempty"`
}

// versionKey uniquely identifies a (game, version, platform) tuple.
func versionKey(gameID, gameVersion, platform string) string {
	return gameID + "/" + gameVersion + "/" + platform
}

// submissionKey tracks which subnets submitted a given SHA256.
func submissionKey(sha256 string) string {
	return "sub/" + sha256
}

// persistedState is the JSON structure written to disk.
type persistedState struct {
	Records     []*ContentRecord      `json:"records"`
	Submissions map[string][]string   `json:"submissions"` // sha256 → []subnet
}

// Registry holds all content records in memory and persists them to a JSON file.
type Registry struct {
	mu          sync.RWMutex
	records     map[string]*ContentRecord // key: versionKey
	submissions map[string][]string       // key: sha256 → []subnet
	dataPath    string
	threshold   int
}

// NewRegistry creates a Registry backed by dataPath.
// If dataPath exists it is loaded; otherwise an empty registry is created.
func NewRegistry(dataPath string, threshold int) (*Registry, error) {
	if threshold <= 0 {
		threshold = DefaultThreshold
	}
	r := &Registry{
		records:     make(map[string]*ContentRecord),
		submissions: make(map[string][]string),
		dataPath:    dataPath,
		threshold:   threshold,
	}
	if err := r.load(); err != nil {
		return nil, err
	}
	return r, nil
}

// SubmitResult is returned from Submit.
type SubmitResult struct {
	Status      string     // "upload_required" | "counted" | "already_stored"
	TrustLevel  TrustLevel
	UploadCount int
	Conflict    bool // true when a different SHA256 already exists for this version+platform
}

// Submit records that remoteAddr has seen sha256 for the given version+platform.
//
//   - If no blob exists for this key yet and the hash is new → UploadRequired.
//   - If a blob exists with the same hash → count the submitter, recompute trust.
//   - If a blob exists with a DIFFERENT hash → mark both Rejected, return Conflict.
func (r *Registry) Submit(gameID, gameVersion, platform, sha256hex string, sizeBytes int64, remoteAddr string) SubmitResult {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := versionKey(gameID, gameVersion, platform)
	subnet := ipSubnet(remoteAddr)
	now := time.Now().UTC().Format(time.RFC3339)

	existing, exists := r.records[key]

	if exists && existing.SHA256 != sha256hex {
		// Hash conflict — mark both as rejected.
		existing.TrustLevel = TrustRejected
		_ = r.persist()
		return SubmitResult{Status: "conflict", TrustLevel: TrustRejected, Conflict: true}
	}

	if !exists {
		// First submission for this version+platform.
		rec := &ContentRecord{
			SHA256:          sha256hex,
			ContentType:     "game-binary",
			GameID:          gameID,
			GameVersion:     gameVersion,
			Platform:        platform,
			SizeBytes:       sizeBytes,
			TrustLevel:      TrustPending,
			UploadCount:     0,
			FirstUploadedAt: now,
		}
		r.records[key] = rec
		r.addSubmitter(sha256hex, subnet)
		_ = r.persist()
		return SubmitResult{Status: "upload_required", TrustLevel: TrustPending, UploadCount: 0}
	}

	// Same hash as existing record.
	if r.hasSubmitter(sha256hex, subnet) {
		// Same subnet already submitted — don't double-count.
		return SubmitResult{Status: "already_counted", TrustLevel: existing.TrustLevel, UploadCount: existing.UploadCount}
	}

	r.addSubmitter(sha256hex, subnet)
	count := len(r.submissions[submissionKey(sha256hex)])
	existing.UploadCount = count

	if count >= r.threshold && existing.TrustLevel == TrustPending {
		existing.TrustLevel = TrustVerified
		existing.LastVerifiedAt = now
	}
	_ = r.persist()

	status := "counted"
	if !r.blobStored(sha256hex) {
		status = "upload_required"
	}
	return SubmitResult{Status: status, TrustLevel: existing.TrustLevel, UploadCount: count}
}

// MarkUploaded records that a blob was successfully stored for sha256.
// It increments the upload count for the submitter's subnet and recomputes trust.
func (r *Registry) MarkUploaded(sha256hex, remoteAddr string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	subnet := ipSubnet(remoteAddr)
	r.addSubmitter(sha256hex, subnet)

	for _, rec := range r.records {
		if rec.SHA256 == sha256hex && rec.UploadCount == 0 {
			rec.UploadCount = len(r.submissions[submissionKey(sha256hex)])
			if rec.UploadCount >= r.threshold {
				rec.TrustLevel = TrustVerified
				rec.LastVerifiedAt = time.Now().UTC().Format(time.RFC3339)
			}
		}
	}
	_ = r.persist()
}

// Get returns the ContentRecord for (gameID, gameVersion, platform), or nil.
func (r *Registry) Get(gameID, gameVersion, platform string) *ContentRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec := r.records[versionKey(gameID, gameVersion, platform)]
	if rec == nil {
		return nil
	}
	copy := *rec
	return &copy
}

// List returns all records, optionally filtered by gameID. Pass "" for all.
func (r *Registry) List(gameID string) []*ContentRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ContentRecord, 0, len(r.records))
	for _, rec := range r.records {
		if gameID != "" && rec.GameID != gameID {
			continue
		}
		copy := *rec
		out = append(out, &copy)
	}
	return out
}

// --- persistence ---

func (r *Registry) load() error {
	if r.dataPath == "" {
		return nil
	}
	data, err := os.ReadFile(r.dataPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("content: load %q: %w", r.dataPath, err)
	}
	var state persistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("content: parse %q: %w", r.dataPath, err)
	}
	for _, rec := range state.Records {
		r.records[versionKey(rec.GameID, rec.GameVersion, rec.Platform)] = rec
	}
	if state.Submissions != nil {
		r.submissions = state.Submissions
	}
	return nil
}

func (r *Registry) persist() error {
	if r.dataPath == "" {
		return nil
	}
	recs := make([]*ContentRecord, 0, len(r.records))
	for _, rec := range r.records {
		recs = append(recs, rec)
	}
	state := persistedState{Records: recs, Submissions: r.submissions}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(r.dataPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(r.dataPath, data, 0o644)
}

// --- internal helpers ---

func (r *Registry) addSubmitter(sha256hex, subnet string) {
	k := submissionKey(sha256hex)
	for _, s := range r.submissions[k] {
		if s == subnet {
			return
		}
	}
	r.submissions[k] = append(r.submissions[k], subnet)
}

func (r *Registry) hasSubmitter(sha256hex, subnet string) bool {
	for _, s := range r.submissions[submissionKey(sha256hex)] {
		if s == subnet {
			return true
		}
	}
	return false
}

func (r *Registry) blobStored(sha256hex string) bool {
	// Blob presence is checked by the caller via storage.Store.Has().
	// This method intentionally always returns false — the router checks storage directly.
	_ = sha256hex
	return false
}

// ipSubnet returns the /24 subnet string for a remoteAddr (host:port or host).
// Used to approximate uploader independence — not a security guarantee.
func ipSubnet(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return host
	}
	if ip4 := ip.To4(); ip4 != nil {
		return fmt.Sprintf("%d.%d.%d.0/24", ip4[0], ip4[1], ip4[2])
	}
	// IPv6: use /48
	masked := ip.Mask(net.CIDRMask(48, 128))
	return masked.String() + "/48"
}
