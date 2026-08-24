// Package tokenstore persists the Agent's Backend access token to local
// disk so a restart does not force re-enrollment. Storage constraints:
//
//   - One file per serverId, permission 0600 (owner read/write only).
//   - No encryption: the token is a bearer credential scoped to a single
//     serverId (see RFC-0056 invariant 5), equivalent in sensitivity to an
//     API key already accepted as plaintext-on-disk elsewhere in this repo.
package tokenstore

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// unsafeChars is stripped from serverId before using it in a filename.
var unsafeChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// DefaultDir returns the directory tokens are stored in when no explicit
// path is configured: <os.UserConfigDir()>/pz-agent/tokens.
// Falls back to a temp-adjacent dir if UserConfigDir is unavailable.
func DefaultDir() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "pz-agent", "tokens")
}

// Path returns the token file path for a given serverId under dir.
func Path(dir, serverID string) string {
	safe := unsafeChars.ReplaceAllString(serverID, "_")
	if safe == "" {
		safe = "default"
	}
	return filepath.Join(dir, safe+".token")
}

// Load reads a previously saved token for serverID. Returns "" (no error)
// if no token file exists yet — that is the expected first-run state.
func Load(dir, serverID string) (string, error) {
	data, err := os.ReadFile(Path(dir, serverID))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("tokenstore: load: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// Save writes token for serverID, creating dir if needed. The file is
// written with 0600 permissions; an empty token deletes any existing file.
func Save(dir, serverID, token string) error {
	path := Path(dir, serverID)
	if token == "" {
		err := os.Remove(path)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("tokenstore: clear: %w", err)
		}
		return nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("tokenstore: mkdir: %w", err)
	}
	if err := os.WriteFile(path, []byte(token), 0o600); err != nil {
		return fmt.Errorf("tokenstore: save: %w", err)
	}
	return nil
}
