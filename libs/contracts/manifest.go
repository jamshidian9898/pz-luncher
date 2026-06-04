package contracts

import "time"

type ManifestMod struct {
	ID           string   `json:"id"`
	Version      string   `json:"version"`
	SHA256       string   `json:"sha256"`
	DownloadURL  string   `json:"downloadUrl"`
	Dependencies []string `json:"dependencies,omitempty"`
}

type Manifest struct {
	ID          string        `json:"id"`
	ServerID    string        `json:"serverId"`
	Version     int           `json:"version"`
	GameVersion string        `json:"gameVersion"`
	CreatedAt   time.Time     `json:"createdAt,omitempty"`
	Checksum    string        `json:"checksum"`
	Mods        []ManifestMod `json:"mods"`
}
