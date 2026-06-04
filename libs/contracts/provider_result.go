package contracts

type ProviderResult struct {
	ProviderName string `json:"providerName"`
	PackageHash  string `json:"packageHash"`
	Exists       bool   `json:"exists"`
	Error        string `json:"error,omitempty"`
	Priority     int    `json:"priority"`
}
