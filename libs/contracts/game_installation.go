package contracts

type GameInstallation struct {
	Path           string `json:"path"`
	Version        string `json:"version"`
	SteamAppID     string `json:"steamAppId,omitempty"`
	LaunchURI      string `json:"launchUri,omitempty"`
	Platform       string `json:"platform,omitempty"`
	IsSteamInstall bool   `json:"isSteamInstall"`
}
