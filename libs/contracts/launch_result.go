package contracts

type LaunchResult struct {
	Success    bool   `json:"success"`
	ProfileID  string `json:"profileId,omitempty"`
	LaunchArgs string `json:"launchArgs,omitempty"`
	Error      string `json:"error,omitempty"`
}
