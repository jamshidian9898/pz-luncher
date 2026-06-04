package contracts

type ProfileBuildRequest struct {
	ProfileID   string            `json:"profileId"`
	ServerID    string            `json:"serverId"`
	ManifestID  string            `json:"manifestId"`
	Packages    []ResolvedPackage `json:"packages"`
	ProfilePath string            `json:"profilePath"`
}

type ProfileBuildResult struct {
	ProfileID   string `json:"profileId"`
	ProfilePath string `json:"profilePath"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
}
