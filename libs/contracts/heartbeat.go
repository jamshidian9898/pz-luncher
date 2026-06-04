package contracts

import "time"

type HeartbeatStatus string

const (
	HeartbeatStatusOnline      HeartbeatStatus = "online"
	HeartbeatStatusOffline     HeartbeatStatus = "offline"
	HeartbeatStatusMaintenance HeartbeatStatus = "maintenance"
)

type Heartbeat struct {
	ID          string            `json:"id"`
	ServerID    string            `json:"serverId"`
	Timestamp   time.Time         `json:"timestamp"`
	Status      HeartbeatStatus   `json:"status"`
	PlayerCount int               `json:"playerCount,omitempty"`
	MaxPlayers  int               `json:"maxPlayers,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}
