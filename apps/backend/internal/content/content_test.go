package content

import (
	"path/filepath"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	tmpdir := t.TempDir()
	reg, err := NewRegistry(filepath.Join(tmpdir, "test.json"), 3)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}
	if reg == nil {
		t.Fatal("registry is nil")
	}
}

func TestSubmit_NewHash(t *testing.T) {
	tmpdir := t.TempDir()
	reg, _ := NewRegistry(filepath.Join(tmpdir, "test.json"), 3)

	result := reg.Submit("pz", "42.16", "windows-x64", "abc123", 1000, "192.168.1.1:12345")
	if result.Status != "upload_required" {
		t.Errorf("expected upload_required, got %s", result.Status)
	}
	if result.TrustLevel != TrustPending {
		t.Errorf("expected pending, got %s", result.TrustLevel)
	}
	if result.UploadCount != 1 {
		t.Errorf("expected 1 unique submitter after first submission, got %d", result.UploadCount)
	}
}

func TestSubmit_DuplicateSubnet(t *testing.T) {
	tmpdir := t.TempDir()
	reg, _ := NewRegistry(filepath.Join(tmpdir, "test.json"), 3)

	// First submission
	reg.Submit("pz", "42.16", "windows-x64", "abc123", 1000, "192.168.1.10")

	// Same subnet — should not double count
	result := reg.Submit("pz", "42.16", "windows-x64", "abc123", 1000, "192.168.1.20")
	if result.UploadCount != 1 {
		t.Errorf("expected 1 upload count (deduped subnet), got %d", result.UploadCount)
	}
}

func TestSubmit_HashConflict(t *testing.T) {
	tmpdir := t.TempDir()
	reg, _ := NewRegistry(filepath.Join(tmpdir, "test.json"), 3)

	// First hash
	reg.Submit("pz", "42.16", "windows-x64", "abc123", 1000, "192.168.1.1")

	// Different hash, same version+platform → conflict
	result := reg.Submit("pz", "42.16", "windows-x64", "def456", 1000, "192.168.1.2")
	if result.Status != "conflict" {
		t.Errorf("expected conflict, got %s", result.Status)
	}
	if result.TrustLevel != TrustRejected {
		t.Errorf("expected rejected trust level on conflict, got %s", result.TrustLevel)
	}
}

func TestSubmit_VerificationThreshold(t *testing.T) {
	tmpdir := t.TempDir()
	reg, _ := NewRegistry(filepath.Join(tmpdir, "test.json"), 3)

	sha256 := "abc123"
	version := "42.16"

	// Submit from 3 different subnets
	subnets := []string{"192.168.1.10", "10.0.0.1", "172.16.0.5"}
	for _, subnet := range subnets {
		result := reg.Submit("pz", version, "windows-x64", sha256, 1000, subnet)
		if result.TrustLevel == TrustVerified {
			// Should be verified after 3rd submission
			break
		}
	}

	// Check final state
	record := reg.Get("pz", version, "windows-x64")
	if record == nil {
		t.Fatal("record not found after submission")
	}
	if record.TrustLevel != TrustVerified {
		t.Errorf("expected verified after 3 submissions, got %s", record.TrustLevel)
	}
	if record.UploadCount != 3 {
		t.Errorf("expected 3 uploads, got %d", record.UploadCount)
	}
}

func TestGet_NoMatch(t *testing.T) {
	tmpdir := t.TempDir()
	reg, _ := NewRegistry(filepath.Join(tmpdir, "test.json"), 3)

	record := reg.Get("pz", "42.16", "windows-x64")
	if record != nil {
		t.Fatal("expected nil for non-existent record")
	}
}

func TestList_Empty(t *testing.T) {
	tmpdir := t.TempDir()
	reg, _ := NewRegistry(filepath.Join(tmpdir, "test.json"), 3)

	records := reg.List("pz")
	if len(records) != 0 {
		t.Errorf("expected empty list, got %d records", len(records))
	}
}

func TestList_Filter(t *testing.T) {
	tmpdir := t.TempDir()
	reg, _ := NewRegistry(filepath.Join(tmpdir, "test.json"), 3)

	// Add two games
	reg.Submit("pz", "42.16", "windows-x64", "abc123", 1000, "192.168.1.1")
	reg.Submit("minetest", "5.7.0", "windows-x64", "def456", 2000, "192.168.1.1")

	// List only pz
	records := reg.List("pz")
	if len(records) != 1 {
		t.Errorf("expected 1 pz record, got %d", len(records))
	}
	if records[0].GameID != "pz" {
		t.Errorf("expected pz, got %s", records[0].GameID)
	}

	// List all
	allRecords := reg.List("")
	if len(allRecords) != 2 {
		t.Errorf("expected 2 total records, got %d", len(allRecords))
	}
}

func TestPersist_LoadFromDisk(t *testing.T) {
	tmpdir := t.TempDir()
	path := filepath.Join(tmpdir, "test.json")

	// Create and populate registry
	reg1, _ := NewRegistry(path, 3)
	reg1.Submit("pz", "42.16", "windows-x64", "abc123", 1000, "192.168.1.1")

	// Create new registry from same file
	reg2, err := NewRegistry(path, 3)
	if err != nil {
		t.Fatalf("failed to load from disk: %v", err)
	}

	record := reg2.Get("pz", "42.16", "windows-x64")
	if record == nil {
		t.Fatal("record not persisted to disk")
	}
	if record.SHA256 != "abc123" {
		t.Errorf("expected abc123, got %s", record.SHA256)
	}
}

func TestIPSubnet_IPv4(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"192.168.1.10:8080", "192.168.1.0/24"},
		{"10.0.0.1", "10.0.0.0/24"},
		{"172.16.255.254", "172.16.255.0/24"},
	}

	for _, tt := range tests {
		result := ipSubnet(tt.input)
		if result != tt.expected {
			t.Errorf("ipSubnet(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
