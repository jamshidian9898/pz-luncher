package pipeline

// Event mirrors launcher-ui UIEvent for Wails bridge.
type Event struct {
	Type      string
	SessionID string
	PackageID string
	Progress  *Progress
	Error     string
	Metadata  map[string]interface{}
}

type Progress struct {
	Current int64
	Total   int64
	Percent int
	Speed   int64
	ETA     int
}

type Emitter func(Event)
