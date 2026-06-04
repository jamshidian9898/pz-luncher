package contracts

type LaunchRequest struct {
	ServerID   string `json:"serverId"`
	ProfileID  string `json:"profileId"`
	ManifestID string `json:"manifestId"`
	LaunchArgs string `json:"launchArgs,omitempty"`
}
