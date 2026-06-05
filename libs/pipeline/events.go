package pipeline

// Event mirrors launcher-ui UIEvent for Wails bridge.
type Event struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"sessionId"`
	PackageID string                 `json:"packageId,omitempty"`
	Progress  *Progress              `json:"progress,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type Progress struct {
	Current int64 `json:"current"`
	Total   int64 `json:"total"`
	Percent int   `json:"percent"`
	Speed   int64 `json:"speed,omitempty"`
	ETA     int   `json:"eta,omitempty"`
}

type Emitter func(Event)
