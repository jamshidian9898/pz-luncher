package session

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// WorkshopMapping maps mod identifiers to Steam Workshop IDs
// This is the critical layer that bridges "mod name" to "Steam artifact"
type WorkshopMapping struct {
	ModID       string `json:"modId"`       // e.g., "Brita", "ORGM"
	WorkshopID  string `json:"workshopId"`  // e.g., "123456789"
	Version     string `json:"version"`     // optional version constraint
	DisplayName string `json:"displayName"` // human-readable name
	Source      string `json:"source"`      // "registry", "steam", "manual"
}

// MappingService resolves mod names to Workshop IDs
// Supports multiple sources with fallback chain
type MappingService struct {
	mu       sync.RWMutex
	mappings map[string]WorkshopMapping // modId -> mapping
	cacheDir string
	client   *http.Client
}

// NewMappingService creates a new mapping service
func NewMappingService(cacheDir string) *MappingService {
	return &MappingService{
		mappings: make(map[string]WorkshopMapping),
		cacheDir: cacheDir,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

// Resolve looks up a mod ID and returns the Workshop ID
// Priority: 1) Local cache, 2) Numeric ID (assume workshop), 3) Registry API
func (s *MappingService) Resolve(modID string) (string, error) {
	// Already numeric? Assume it's a workshop ID
	if isNumeric(modID) {
		return modID, nil
	}

	s.mu.RLock()
	mapping, exists := s.mappings[modID]
	s.mu.RUnlock()

	if exists {
		return mapping.WorkshopID, nil
	}

	// Try to load from cache file
	if cached, err := s.loadFromCache(modID); err == nil {
		s.mu.Lock()
		s.mappings[modID] = *cached
		s.mu.Unlock()
		return cached.WorkshopID, nil
	}

	// Try registry API
	if workshopID, err := s.queryRegistry(modID); err == nil {
		return workshopID, nil
	}

	return "", fmt.Errorf("cannot resolve mod %s: no mapping found", modID)
}

// LoadFromFile loads mappings from a JSON file
// Format: {"mappings": [{"modId": "Brita", "workshopId": "123456789", ...}]}
func (s *MappingService) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read mapping file: %w", err)
	}

	var file struct {
		Mappings []WorkshopMapping `json:"mappings"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("parse mapping file: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, m := range file.Mappings {
		s.mappings[strings.ToLower(m.ModID)] = m
	}

	return nil
}

// AddManualMapping adds a manually configured mapping
func (s *MappingService) AddManualMapping(modID, workshopID, displayName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.mappings[strings.ToLower(modID)] = WorkshopMapping{
		ModID:       modID,
		WorkshopID:  workshopID,
		DisplayName: displayName,
		Source:      "manual",
	}
}

// loadFromCache tries to load a cached mapping
func (s *MappingService) loadFromCache(modID string) (*WorkshopMapping, error) {
	cacheFile := filepath.Join(s.cacheDir, "workshop-mappings", fmt.Sprintf("%s.json", modID))
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}

	var mapping WorkshopMapping
	if err := json.Unmarshal(data, &mapping); err != nil {
		return nil, err
	}

	return &mapping, nil
}

// saveToCache persists a mapping to local cache
func (s *MappingService) saveToCache(mapping WorkshopMapping) error {
	dir := filepath.Join(s.cacheDir, "workshop-mappings")
	os.MkdirAll(dir, 0755)

	cacheFile := filepath.Join(dir, fmt.Sprintf("%s.json", mapping.ModID))
	data, err := json.Marshal(mapping)
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0644)
}

// queryRegistry queries a registry service for mod mapping
// This is a placeholder - real implementation calls your registry API
func (s *MappingService) queryRegistry(modID string) (string, error) {
	// Placeholder: In production, this calls your registry service
	// e.g., GET https://registry.pzlauncher.com/v1/mods/{modID}

	// For now, return error to indicate not implemented
	return "", fmt.Errorf("registry query not implemented for: %s", modID)
}

// QuerySteam tries to resolve via Steam API directly
// Some mods can be found by searching Steam Workshop
func (s *MappingService) QuerySteam(modID string) (string, error) {
	// This would use Steam Web API to search for mods
	// For now, placeholder
	return "", fmt.Errorf("steam search not implemented for: %s", modID)
}

// GetStats returns statistics about the mapping service
func (s *MappingService) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sources := make(map[string]int)
	for _, m := range s.mappings {
		sources[m.Source]++
	}

	return map[string]interface{}{
		"totalMappings": len(s.mappings),
		"sources":       sources,
	}
}
