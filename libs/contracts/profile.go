package contracts

type ProfileStatus string

const (
	ProfileStatusReady   ProfileStatus = "ready"
	ProfileStatusSyncing ProfileStatus = "syncing"
	ProfileStatusError   ProfileStatus = "error"
)

type Profile struct {
	ID              string        `json:"id"`
	ServerID        string        `json:"serverId"`
	ManifestVersion int           `json:"manifestVersion"`
	Path            string        `json:"path"`
	Status          ProfileStatus `json:"status"`
}
