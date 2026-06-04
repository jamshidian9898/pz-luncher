package contracts

import "time"

type ServerStatus string

const (
	ServerStatusOnline      ServerStatus = "online"
	ServerStatusOffline     ServerStatus = "offline"
	ServerStatusMaintenance ServerStatus = "maintenance"
)

type Server struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Description   string       `json:"description,omitempty"`
	Region        string       `json:"region,omitempty"`
	Tags          []string     `json:"tags,omitempty"`
	ManifestID    string       `json:"manifestId,omitempty"`
	Status        ServerStatus `json:"status"`
	PlayerCount   int          `json:"playerCount,omitempty"`
	MaxPlayers    int          `json:"maxPlayers,omitempty"`
	LastHeartbeat time.Time    `json:"lastHeartbeat,omitempty"`
}
