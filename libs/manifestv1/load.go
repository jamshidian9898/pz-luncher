package manifestv1

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

func LoadFile(path string) (*ServerManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	return Parse(data)
}

func Parse(data []byte) (*ServerManifest, error) {
	var m ServerManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	// Accept legacy RFC-0001 shape when serverId missing at root but present via migration
	if m.ServerID == "" {
		var legacy struct {
			ServerID        string `json:"serverId"`
			ManifestVersion int    `json:"manifestVersion"`
			Version         int    `json:"version"`
		}
		if err := json.Unmarshal(data, &legacy); err == nil && legacy.ServerID != "" {
			m.ServerID = legacy.ServerID
			if m.Version == "" {
				if legacy.ManifestVersion != 0 {
					m.Version = strconv.Itoa(legacy.ManifestVersion)
				} else if legacy.Version != 0 {
					m.Version = strconv.Itoa(legacy.Version)
				}
			}
		}
	}
	for i := range m.Mods {
		if m.Mods[i].Name == "" {
			m.Mods[i].Name = m.Mods[i].ID
		}
		if m.Mods[i].Dependencies == nil {
			m.Mods[i].Dependencies = []string{}
		}
	}
	if err := Validate(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

func LoadRegistry(path string) (*ServerRegistry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var reg ServerRegistry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, err
	}
	return &reg, nil
}

func FindServer(reg *ServerRegistry, serverID string) (*ServerDescriptor, error) {
	for i := range reg.Servers {
		if reg.Servers[i].ID == serverID {
			return &reg.Servers[i], nil
		}
	}
	return nil, fmt.Errorf("server not found: %s", serverID)
}

// SaveSnapshot writes manifest copy into profile dir (RFC-0034).
func SaveSnapshot(profileDir string, m *ServerManifest) error {
	name := fmt.Sprintf("manifest-%s.json", m.Version)
	path := filepath.Join(profileDir, name)
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
