package contracts

import "time"

type PackageMetadata struct {
	ID           string            `json:"id"`
	Version      string            `json:"version"`
	SHA256       string            `json:"sha256"`
	Size         int64             `json:"size"`
	Provider     string            `json:"provider"`
	OriginURL    string            `json:"originUrl,omitempty"`
	Dependencies []string          `json:"dependencies,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	CreatedAt    time.Time         `json:"createdAt,omitempty"`
}
