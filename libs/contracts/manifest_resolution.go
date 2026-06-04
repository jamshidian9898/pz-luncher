package contracts

type ManifestResolutionResult struct {
	ServerID     string              `json:"serverId"`
	ManifestID   string              `json:"manifestId"`
	Packages     []ResolvedPackage   `json:"packages"`
	Dependencies map[string][]string `json:"dependencies"`
	Errors       []string            `json:"errors,omitempty"`
}
