package contracts

type ResolvedPackage struct {
	ID           string   `json:"id"`
	Version      string   `json:"version"`
	SHA256       string   `json:"sha256"`
	Size         int64    `json:"size"`
	WorkshopID   string   `json:"workshopId,omitempty"`
	ProviderName string   `json:"providerName"`
	DownloadURL  string   `json:"downloadUrl"`
	Dependencies []string `json:"dependencies,omitempty"`
	Cached       bool     `json:"cached"`
}
